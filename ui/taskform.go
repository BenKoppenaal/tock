package ui

import (
	"tock/db"
	"tock/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type taskCreatedMsg struct{ task model.Task }
type taskUpdatedMsg struct{ task model.Task }
type openNewTaskMsg struct{}

type field int

const (
	fieldName field = iota
	fieldNote
)

type TaskFormModel struct {
	db     *db.DB
	taskID int64 // 0 = new, >0 = edit
	name   textinput.Model
	note   textinput.Model
	active field
	err    string
}

func NewTaskFormModel(database *db.DB) TaskFormModel {
	name := textinput.New()
	name.Placeholder = "Task name"
	name.CharLimit = 100

	note := textinput.New()
	note.Placeholder = "Note (optional)"
	note.CharLimit = 200

	return TaskFormModel{db: database, name: name, note: note, active: fieldName}
}

func NewTaskEditFormModel(database *db.DB, task model.Task) TaskFormModel {
	name := textinput.New()
	name.Placeholder = "Task name"
	name.CharLimit = 100
	name.SetValue(task.Name)

	note := textinput.New()
	note.Placeholder = "Note (optional)"
	note.CharLimit = 200
	note.SetValue(task.Note)

	return TaskFormModel{db: database, taskID: task.ID, name: name, note: note, active: fieldName}
}

func (m TaskFormModel) Init() tea.Cmd {
	return m.name.Focus()
}

func (m TaskFormModel) Update(msg tea.Msg) (TaskFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			if m.active == fieldName {
				m.name.Blur()
				m.active = fieldNote
				return m, m.note.Focus()
			}
		case "shift+tab", "up":
			if m.active == fieldNote {
				m.note.Blur()
				m.active = fieldName
				return m, m.name.Focus()
			}
		case "enter":
			if m.active == fieldName && m.note.Value() == "" {
				m.name.Blur()
				m.active = fieldNote
				return m, m.note.Focus()
			}
			name := m.name.Value()
			if name == "" {
				m.err = "Name is required"
				return m, nil
			}
			if m.taskID == 0 {
				task, err := m.db.CreateTask(name, m.note.Value())
				if err != nil {
					m.err = err.Error()
					return m, nil
				}
				return m, func() tea.Msg { return taskCreatedMsg{task: task} }
			}
			if err := m.db.UpdateTask(m.taskID, name, m.note.Value()); err != nil {
				m.err = err.Error()
				return m, nil
			}
			task := model.Task{ID: m.taskID, Name: name, Note: m.note.Value()}
			return m, func() tea.Msg { return taskUpdatedMsg{task: task} }
		}
	}

	var cmd tea.Cmd
	if m.active == fieldName {
		m.name, cmd = m.name.Update(msg)
	} else {
		m.note, cmd = m.note.Update(msg)
	}
	return m, cmd
}

func (m TaskFormModel) View() string {
	title := "New task"
	if m.taskID != 0 {
		title = "Edit task"
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorHighlight).
		Padding(1, 3).
		Width(50)

	errLine := ""
	if m.err != "" {
		errLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render(m.err) + "\n"
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		"",
		m.name.View(),
		m.note.View(),
		"",
		errLine+helpStyle.Render("tab: next field  enter: save  esc: cancel"),
	)

	return lipgloss.NewStyle().Padding(4, 8).Render(box.Render(content))
}
