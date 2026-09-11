package ui

import (
	"time"
	"tock/db"
	"tock/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type entrySavedMsg struct{ entry *model.Entry }
type entryStoppedMsg struct{}
type entryEditedMsg struct{}

const timeInputLayout = "15:04"

type formField struct {
	label  string
	input  textinput.Model
	isTime bool
}

type EntryFormModel struct {
	db          *db.DB
	task        model.Task
	activeEntry *model.Entry
	fields      []formField
	focused     int
	stopping    bool
	editing     bool
	stopTime    time.Time // captured when form opens
}

func newTimeField(label, value string) formField {
	ti := textinput.New()
	ti.Placeholder = "HH:MM"
	ti.CharLimit = 5
	ti.Width = 10
	ti.SetValue(value)
	return formField{label: label, input: ti, isTime: true}
}

func newCommentField(value string) formField {
	ti := textinput.New()
	ti.Placeholder = "Comment (optional)"
	ti.CharLimit = 200
	ti.SetValue(value)
	return formField{label: "Comment", input: ti}
}

func NewEntryFormModel(database *db.DB, task model.Task, active *model.Entry) EntryFormModel {
	stopping := active != nil && active.TaskID == task.ID
	now := time.Now()

	var fields []formField
	if stopping {
		fields = []formField{
			newTimeField("Start time", active.StartTime.Format(timeInputLayout)),
			newTimeField("End time", now.Format(timeInputLayout)),
			newCommentField(active.Comment),
		}
	} else {
		fields = []formField{
			newTimeField("Start time", now.Format(timeInputLayout)),
			newCommentField(""),
		}
	}
	fields[0].input.Focus()

	return EntryFormModel{
		db:          database,
		task:        task,
		activeEntry: active,
		fields:      fields,
		focused:     0,
		stopping:    stopping,
		stopTime:    now,
	}
}

func NewEntryEditFormModel(database *db.DB, entry model.Entry) EntryFormModel {
	endVal := ""
	if entry.EndTime != nil {
		endVal = entry.EndTime.Format(timeInputLayout)
	}
	fields := []formField{
		newTimeField("Start time", entry.StartTime.Format(timeInputLayout)),
		newTimeField("End time", endVal),
		newCommentField(entry.Comment),
	}
	fields[0].input.Focus()
	return EntryFormModel{
		db:          database,
		task:        model.Task{ID: entry.TaskID, Name: entry.TaskName},
		activeEntry: &entry,
		fields:      fields,
		focused:     0,
		editing:     true,
		stopTime:    time.Now(),
	}
}

func (m EntryFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m EntryFormModel) parseTime(s string, ref time.Time) time.Time {
	t, err := time.ParseInLocation(timeInputLayout, s, ref.Location())
	if err != nil {
		return ref
	}
	return time.Date(ref.Year(), ref.Month(), ref.Day(), t.Hour(), t.Minute(), 0, 0, ref.Location())
}

func (m EntryFormModel) Update(msg tea.Msg) (EntryFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "down", "shift+up", "shift+down":
			if m.fields[m.focused].isTime {
				delta := time.Minute
				if msg.String() == "shift+up" || msg.String() == "shift+down" {
					delta = 5 * time.Minute
				}
				if msg.String() == "down" || msg.String() == "shift+down" {
					delta = -delta
				}
				ref := time.Now()
				t := m.parseTime(m.fields[m.focused].input.Value(), ref)
				t = t.Add(delta)
				m.fields[m.focused].input.SetValue(t.Format(timeInputLayout))
				return m, nil
			}

		case "tab", "shift+tab":
			m.fields[m.focused].input.Blur()
			if msg.String() == "tab" {
				m.focused = (m.focused + 1) % len(m.fields)
			} else {
				m.focused = (m.focused - 1 + len(m.fields)) % len(m.fields)
			}
			cmd := m.fields[m.focused].input.Focus()
			return m, cmd

		case "enter":
			if m.editing {
				startTime := m.parseTime(m.fields[0].input.Value(), m.activeEntry.StartTime)
				ref := m.stopTime
				if m.activeEntry.EndTime != nil {
					ref = *m.activeEntry.EndTime
				}
				endTime := m.parseTime(m.fields[1].input.Value(), ref)
				comment := m.fields[2].input.Value()
				_ = m.db.UpdateEntry(m.activeEntry.ID, startTime, endTime, comment)
				return m, func() tea.Msg { return entryEditedMsg{} }
			}
			if m.stopping {
				startTime := m.parseTime(m.fields[0].input.Value(), m.activeEntry.StartTime)
				endTime := m.parseTime(m.fields[1].input.Value(), m.stopTime)
				comment := m.fields[2].input.Value()
				_ = m.db.StopEntry(m.activeEntry.ID, startTime, endTime, comment)
				return m, func() tea.Msg { return entryStoppedMsg{} }
			}
			startTime := m.parseTime(m.fields[0].input.Value(), time.Now())
			comment := m.fields[1].input.Value()
			if m.activeEntry != nil {
				_ = m.db.StopEntry(m.activeEntry.ID, m.activeEntry.StartTime, time.Now(), "")
			}
			entry, _ := m.db.StartEntry(m.task.ID, startTime, comment)
			return m, func() tea.Msg { return entrySavedMsg{entry: &entry} }
		}
	}

	if m.fields[m.focused].isTime {
		return m, nil
	}
	var cmd tea.Cmd
	m.fields[m.focused].input, cmd = m.fields[m.focused].input.Update(msg)
	return m, cmd
}

func (m EntryFormModel) View() string {
	action := "Start tracking"
	switch {
	case m.editing:
		action = "Edit entry"
	case m.stopping:
		action = "Stop tracking"
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorHighlight).
		Padding(1, 3).
		Width(50)

	labelStyle := lipgloss.NewStyle().Foreground(colorMuted)

	lines := []string{
		titleStyle.Render(action + ": " + m.task.Name),
		"",
	}
	for _, f := range m.fields {
		lines = append(lines, labelStyle.Render(f.label))
		lines = append(lines, f.input.View())
		lines = append(lines, "")
	}
	lines = append(lines, helpStyle.Render("enter: confirm  tab: next  ↑↓: ±1m  shift+↑↓: ±5m  esc: cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.NewStyle().Padding(4, 8).Render(box.Render(content))
}
