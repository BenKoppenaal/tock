package ui

import (
	"tock/db"
	"tock/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type entrySavedMsg struct{ entry *model.Entry }
type entryStoppedMsg struct{}

type EntryFormModel struct {
	db          *db.DB
	task        model.Task
	activeEntry *model.Entry
	comment     textinput.Model
	stopping    bool
}

func NewEntryFormModel(database *db.DB, task model.Task, active *model.Entry) EntryFormModel {
	comment := textinput.New()
	comment.Placeholder = "Comment (optional)"
	comment.CharLimit = 200
	comment.Focus()

	stopping := active != nil && active.TaskID == task.ID

	return EntryFormModel{
		db:          database,
		task:        task,
		activeEntry: active,
		comment:     comment,
		stopping:    stopping,
	}
}

func (m EntryFormModel) Init() tea.Cmd {
	return m.comment.Focus() // returns cursor blink cmd
}

func (m EntryFormModel) Update(msg tea.Msg) (EntryFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.stopping {
				_ = m.db.StopEntry(m.activeEntry.ID, m.comment.Value())
				return m, func() tea.Msg { return entryStoppedMsg{} }
			}
			// stop any active entry first, then start new one
			if m.activeEntry != nil {
				_ = m.db.StopEntry(m.activeEntry.ID, "")
			}
			entry, _ := m.db.StartEntry(m.task.ID)
			return m, func() tea.Msg { return entrySavedMsg{entry: &entry} }
		}
	}
	var cmd tea.Cmd
	m.comment, cmd = m.comment.Update(msg)
	return m, cmd
}

func (m EntryFormModel) View() string {
	action := "Start tracking"
	if m.stopping {
		action = "Stop tracking"
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorHighlight).
		Padding(1, 3).
		Width(50)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(action+": "+m.task.Name),
		"",
		m.comment.View(),
		"",
		helpStyle.Render("enter: confirm  esc: cancel"),
	)

	return lipgloss.NewStyle().Padding(4, 8).Render(box.Render(content))
}
