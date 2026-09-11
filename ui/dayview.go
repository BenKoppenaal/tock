package ui

import (
	"fmt"
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

type DayViewModel struct {
	db           *db.DB
	day          time.Time
	entries      []model.Entry
	width        int
	height       int
	scrollOffset int
}

func NewDayViewModel(database *db.DB) DayViewModel {
	return DayViewModel{
		db:  database,
		day: time.Now(),
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

func (m DayViewModel) Update(msg tea.Msg) (DayViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case dayLoadedMsg:
		m.entries = msg.entries
		m.day = msg.day
		m.scrollOffset = m.scrollForNow()
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.day = m.day.AddDate(0, 0, -1)
			return m, m.load()
		case "right", "l":
			m.day = m.day.AddDate(0, 0, 1)
			return m, m.load()
		case "t":
			m.day = time.Now()
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
	}

	spans := make([]span, 0, len(m.entries))
	for i, e := range m.entries {
		sr := m.timeToRow(e.StartTime)
		er := m.timeToRow(m.entryEnd(e))
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

		cell := strings.Repeat(" ", blockW)
		for _, s := range spans {
			if row < s.startRow || row >= s.endRow {
				continue
			}
			color := blockColors[s.colorIdx%len(blockColors)]
			style := lipgloss.NewStyle().
				Background(color).
				Foreground(lipgloss.Color("#ffffff")).
				Width(blockW)

			if row == s.startRow {
				dur := s.entry.Duration().Round(time.Minute)
				label := fmt.Sprintf(" %s  %s", s.entry.TaskName, formatDuration(dur))
				if len(label) > blockW {
					label = " " + s.entry.TaskName
					if len(label) > blockW {
						label = label[:blockW]
					}
				}
				cell = style.Render(label)
			} else {
				cell = style.Render("")
			}
			break
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
	help := helpStyle.Render("← → days  t: today  ↑↓ scroll  1: tasks")
	timeline := lipgloss.NewStyle().PaddingLeft(marginLeft - 1).Render(strings.Join(rows[start:end], "\n"))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		timeline,
		help,
	)
}

func (m DayViewModel) timeToRow(t time.Time) int {
	totalMins := (t.Hour()-dayStartHour)*60 + t.Minute()
	return totalMins * rowsPerHour / 60
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
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
