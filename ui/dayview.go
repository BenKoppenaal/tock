package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"tock/db"
	"tock/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	dayStartHour = 0
	dayEndHour   = 24
	rowsPerHour  = 4 // 15-min resolution per row
)

type dayLoadedMsg struct {
	entries []model.Entry
	day     time.Time
}

type editEntryMsg struct{ entry model.Entry }
type entryDeletedMsg struct{}

type DayViewModel struct {
	db            *db.DB
	day           time.Time
	entries       []model.Entry
	width         int
	height        int
	scrollOffset  int
	selectedIdx   int // -1 = none
	confirmDelete bool
}

func NewDayViewModel(database *db.DB) DayViewModel {
	return DayViewModel{
		db:          database,
		day:         time.Now(),
		selectedIdx: -1,
	}
}

func (m DayViewModel) load() tea.Cmd {
	day := m.day
	return func() tea.Msg {
		entries, _ := m.db.EntriesForDay(day)
		return dayLoadedMsg{entries: entries, day: day}
	}
}

func (m DayViewModel) setSize(w, h int) DayViewModel {
	m.width = w
	m.height = h
	return m
}

func (m DayViewModel) scrollForNow() int {
	row := m.timeToRow(time.Now())
	visibleRows := m.height - 4
	if visibleRows < 1 {
		visibleRows = 10
	}
	totalRows := (dayEndHour - dayStartHour) * rowsPerHour
	offset := row - visibleRows/3
	if offset < 0 {
		offset = 0
	}
	if offset > totalRows-visibleRows {
		offset = totalRows - visibleRows
	}
	return offset
}

func (m DayViewModel) scrollToSelected() DayViewModel {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return m
	}
	entryRow := m.timeToRow(m.entries[m.selectedIdx].StartTime)
	visibleRows := m.height - 4
	if visibleRows < 1 {
		visibleRows = 1
	}
	if entryRow < m.scrollOffset {
		m.scrollOffset = entryRow
	} else if entryRow >= m.scrollOffset+visibleRows {
		m.scrollOffset = entryRow - visibleRows + 1
	}
	return m
}

func (m DayViewModel) Update(msg tea.Msg) (DayViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case dayLoadedMsg:
		m.entries = msg.entries
		m.day = msg.day
		m.scrollOffset = m.scrollForNow()
		m.selectedIdx = -1
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.day = m.day.AddDate(0, 0, -1)
			m.selectedIdx = -1
			return m, m.load()
		case "right", "l":
			m.day = m.day.AddDate(0, 0, 1)
			m.selectedIdx = -1
			return m, m.load()
		case "t":
			m.day = time.Now()
			m.selectedIdx = -1
			return m, m.load()
		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "down", "j":
			totalRows := (dayEndHour - dayStartHour) * rowsPerHour
			visibleRows := m.height - 4
			if visibleRows < 1 {
				visibleRows = 1
			}
			if m.scrollOffset < totalRows-visibleRows {
				m.scrollOffset++
			}
		case "tab":
			if len(m.entries) > 0 {
				if m.selectedIdx < 0 {
					m.selectedIdx = 0
				} else {
					m.selectedIdx = (m.selectedIdx + 1) % len(m.entries)
				}
				m = m.scrollToSelected()
			}
		case "shift+tab":
			if len(m.entries) > 0 {
				if m.selectedIdx < 0 {
					m.selectedIdx = len(m.entries) - 1
				} else {
					m.selectedIdx = (m.selectedIdx - 1 + len(m.entries)) % len(m.entries)
				}
				m = m.scrollToSelected()
			}
		case "e":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.entries) {
				entry := m.entries[m.selectedIdx]
				return m, func() tea.Msg { return editEntryMsg{entry: entry} }
			}
		case "d":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.entries) {
				m.confirmDelete = true
			}
		case "y":
			if m.confirmDelete && m.selectedIdx >= 0 && m.selectedIdx < len(m.entries) {
				entry := m.entries[m.selectedIdx]
				m.db.DeleteEntry(entry.ID) //nolint:errcheck
				m.confirmDelete = false
				m.selectedIdx = -1
				return m, tea.Batch(m.load(), func() tea.Msg { return entryDeletedMsg{} })
			}
		case "esc":
			if m.confirmDelete {
				m.confirmDelete = false
			} else {
				m.selectedIdx = -1
			}
		}
	}
	return m, nil
}

func (m DayViewModel) View() string {
	totalRows := (dayEndHour - dayStartHour) * rowsPerHour
	gutterW := 6
	marginLeft := 2
	blockW := m.width - gutterW - 1 - marginLeft
	if blockW < 10 {
		blockW = 10
	}

	type span struct {
		entry    model.Entry
		colorIdx int
		startRow int
		endRow   int
		col      int
	}

	spans := make([]span, 0, len(m.entries))
	for i, e := range m.entries {
		sr := m.timeToRow(e.StartTime)
		er := m.timeToEndRow(m.entryEnd(e))
		if er <= sr {
			er = sr + 1
		}
		if sr < 0 {
			sr = 0
		}
		if er > totalRows {
			er = totalRows
		}
		spans = append(spans, span{entry: e, colorIdx: i, startRow: sr, endRow: er})
	}

	// Assign columns: greedy lowest-free-column among overlapping spans
	for i := range spans {
		used := map[int]bool{}
		for j := 0; j < i; j++ {
			if spans[j].startRow < spans[i].endRow && spans[i].startRow < spans[j].endRow {
				used[spans[j].col] = true
			}
		}
		col := 0
		for used[col] {
			col++
		}
		spans[i].col = col
	}

	now := time.Now()
	isToday := m.day.Year() == now.Year() && m.day.YearDay() == now.YearDay()
	nowRow := m.timeToRow(now)

	nowIndicator := lipgloss.NewStyle().Foreground(colorActive).Bold(true).Render("▸")

	rows := make([]string, totalRows)
	for row := 0; row < totalRows; row++ {
		totalMins := row * (60 / rowsPerHour)
		h := dayStartHour + totalMins/60
		min := totalMins % 60

		indicator := " "
		if isToday && row == nowRow {
			indicator = nowIndicator
		}

		var gutter string
		if min == 0 {
			gutter = timeGutterStyle.Render(fmt.Sprintf("%02d:%02d", h, min))
		} else {
			tick := " "
			if min == 30 {
				tick = "·"
			}
			gutter = timeGutterStyle.Render("  " + tick + "   ")
		}

		// Collect spans active in this row, sorted by column
		var active []span
		for _, s := range spans {
			if row >= s.startRow && row < s.endRow {
				active = append(active, s)
			}
		}
		sort.Slice(active, func(i, j int) bool { return active[i].col < active[j].col })

		var cell string
		if len(active) == 0 {
			cell = strings.Repeat(" ", blockW)
		} else {
			numCols := len(active)
			baseW := blockW / numCols
			var parts []string
			for ci, s := range active {
				w := baseW
				if ci == numCols-1 {
					w = blockW - baseW*(numCols-1)
				}
				isSelected := s.colorIdx == m.selectedIdx
				color := blockColors[s.colorIdx%len(blockColors)]
				style := lipgloss.NewStyle().
					Background(color).
					Foreground(lipgloss.Color("#ffffff")).
					Bold(isSelected).
					Width(w)

				var content string
				if row == s.startRow {
					dur := s.entry.Duration().Round(time.Second)
					prefix := " "
					if isSelected {
						prefix = "▸"
					}
					label := fmt.Sprintf("%s%s  %s", prefix, s.entry.TaskName, formatDuration(dur))
					if len(label) > w {
						label = prefix + s.entry.TaskName
						if len(label) > w {
							label = label[:w]
						}
					}
					content = label
				}
				parts = append(parts, style.Render(content))
			}
			cell = strings.Join(parts, "")
		}

		rows[row] = indicator + gutter + cell
	}

	visibleRows := m.height - 4 // header + blank line + help + slack
	if visibleRows < 1 {
		visibleRows = 10
	}
	start := m.scrollOffset
	end := start + visibleRows
	if end > totalRows {
		end = totalRows
	}

	header := lipgloss.NewStyle().PaddingLeft(1).Render(titleStyle.Render(m.day.Format("Monday, January 2 2006")))
	var helpText string
	if m.confirmDelete {
		helpText = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Bold(true).Render("Delete entry? y to confirm  esc to cancel")
	} else {
		helpText = helpStyle.Render("← → days  t: today  ↑↓ scroll  tab: select  e: edit  d: delete  1: tasks")
	}
	timeline := lipgloss.NewStyle().PaddingLeft(marginLeft - 1).Render(strings.Join(rows[start:end], "\n"))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		timeline,
		helpText,
	)
}

func (m DayViewModel) timeToRow(t time.Time) int {
	totalMins := (t.Hour()-dayStartHour)*60 + t.Minute()
	return totalMins * rowsPerHour / 60
}

func (m DayViewModel) timeToEndRow(t time.Time) int {
	secsPerRow := 3600 / rowsPerHour
	totalSecs := (t.Hour()-dayStartHour)*3600 + t.Minute()*60 + t.Second()
	return (totalSecs + secsPerRow - 1) / secsPerRow
}

func (m DayViewModel) entryEnd(e model.Entry) time.Time {
	if e.EndTime != nil {
		return *e.EndTime
	}
	return time.Now()
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
