package ui

import (
	"tock/model"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type taskItem struct{ model.Task }

func (t taskItem) Title() string       { return t.Name }
func (t taskItem) Description() string { return t.Note }
func (t taskItem) FilterValue() string { return t.Name }

type startEntryMsg struct{ task model.Task }

type TaskListModel struct {
	list   list.Model
	search textinput.Model
}

func NewTaskListModel(tasks []model.Task) TaskListModel {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{t}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	search := textinput.New()
	search.Placeholder = "/ to search tasks..."
	search.CharLimit = 100

	return TaskListModel{list: l, search: search}
}

func (m TaskListModel) setSize(w, h int) TaskListModel {
	m.list.SetSize(w, h-4)
	return m
}

func (m TaskListModel) Update(msg tea.Msg) (TaskListModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			if !m.search.Focused() {
				return m, m.search.Focus()
			}
		case "esc":
			if m.search.Focused() {
				m.search.Blur()
				m.search.SetValue("")
				return m, nil
			}
		case "enter":
			if m.search.Focused() {
				m.search.Blur()
				return m, nil
			}
			if item, ok := m.list.SelectedItem().(taskItem); ok {
				return m, func() tea.Msg { return startEntryMsg{task: item.Task} }
			}
		}
	}

	if m.search.Focused() {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m TaskListModel) View() string {
	help := helpStyle.Render("n: new task  / search  enter: start/stop  q: quit")
	return lipgloss.JoinVertical(lipgloss.Left,
		m.search.View(),
		m.list.View(),
		help,
	)
}
