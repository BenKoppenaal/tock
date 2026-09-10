package ui

import (
	"fmt"
	"io"
	"strings"
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

type taskDelegate struct {
	activeTaskID int64
}

func (d taskDelegate) Height() int                                { return 2 }
func (d taskDelegate) Spacing() int                               { return 1 }
func (d taskDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d taskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	t, ok := item.(taskItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := d.activeTaskID != 0 && t.ID == d.activeTaskID

	var titleStyle, descStyle lipgloss.Style

	leftBorder := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		Padding(0, 0, 0, 1)

	switch {
	case isActive:
		titleStyle = leftBorder.
			BorderForeground(colorActive).
			Foreground(colorActive).
			Bold(isSelected)
		descStyle = lipgloss.NewStyle().
			Foreground(colorActive).
			Padding(0, 0, 0, 2)
	case isSelected:
		titleStyle = leftBorder.
			BorderForeground(colorHighlight).
			Foreground(colorHighlight)
		descStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Padding(0, 0, 0, 2)
	default:
		titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
			Padding(0, 0, 0, 2)
		descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
			Padding(0, 0, 0, 2)
	}

	fmt.Fprintf(w, "%s\n%s", titleStyle.Render(t.Name), descStyle.Render(t.Note)) //nolint:errcheck
}

type TaskListModel struct {
	list         list.Model
	search       textinput.Model
	allTasks     []model.Task
	activeTaskID int64
}

func NewTaskListModel(tasks []model.Task, activeTaskID int64) TaskListModel {
	items := toListItems(activeFirstTasks(tasks, activeTaskID))

	l := list.New(items, taskDelegate{activeTaskID: activeTaskID}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetStatusBarItemName("task", "tasks")

	search := textinput.New()
	search.Placeholder = "Search tasks..."
	search.CharLimit = 100

	return TaskListModel{list: l, search: search, allTasks: tasks, activeTaskID: activeTaskID}
}

func (m TaskListModel) setSize(w, h int) TaskListModel {
	m.list.SetSize(w, h-4)
	return m
}

func (m TaskListModel) applyFilter() TaskListModel {
	q := strings.ToLower(m.search.Value())
	filtered := make([]model.Task, 0, len(m.allTasks))
	for _, t := range m.allTasks {
		if q == "" || strings.Contains(strings.ToLower(t.Name), q) {
			filtered = append(filtered, t)
		}
	}
	m.list.SetItems(toListItems(activeFirstTasks(filtered, m.activeTaskID)))
	return m
}

func activeFirstTasks(tasks []model.Task, activeID int64) []model.Task {
	if activeID == 0 {
		return tasks
	}
	out := make([]model.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.ID == activeID {
			out = append([]model.Task{t}, out...)
		} else {
			out = append(out, t)
		}
	}
	return out
}

func toListItems(tasks []model.Task) []list.Item {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{t}
	}
	return items
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
				m = m.applyFilter()
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
		m = m.applyFilter()
	} else {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m TaskListModel) View() string {
	help := helpStyle.Render("n: new task  / search  enter: start/stop  q: quit")
	if m.search.Focused() {
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().PaddingLeft(2).Render(m.search.View()),
			m.list.View(),
			help,
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.list.View(),
		help,
	)
}
