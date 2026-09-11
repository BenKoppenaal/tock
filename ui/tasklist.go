package ui

import (
	"fmt"
	"io"
	"strings"
	"time"
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
type editTaskMsg struct{ task model.Task }

type taskDelegate struct {
	activeTaskID int64
	totals       map[int64]time.Duration
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

	// Determine per-state colors
	var nameFg, descFg lipgloss.TerminalColor
	switch {
	case isActive:
		nameFg = colorActive
		descFg = colorActive
	case isSelected:
		nameFg = colorHighlight
		descFg = colorSubtle
	default:
		nameFg = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}
		descFg = lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	}

	// Left decoration is always 2 cols (border+pad or just pad)
	avail := m.Width() - 2
	if avail < 8 {
		avail = 8
	}

	// Build title: name left, total right-aligned
	dur := d.totals[t.ID]
	durStr := ""
	if dur > 0 {
		durStr = formatDuration(dur)
	}

	var titleContent string
	nameStyle := lipgloss.NewStyle().Foreground(nameFg).Bold(isActive && isSelected)
	durStyle := lipgloss.NewStyle().Foreground(colorMuted)

	if durStr != "" {
		nameAvail := avail - len(durStr) - 1
		name := t.Name
		if len(name) > nameAvail {
			name = name[:max(0, nameAvail-1)] + "…"
		}
		gap := nameAvail - len(name)
		if gap < 0 {
			gap = 0
		}
		titleContent = nameStyle.Render(name) + strings.Repeat(" ", gap) + " " + durStyle.Render(durStr)
	} else {
		titleContent = nameStyle.Render(t.Name)
	}

	// Container provides left border or padding
	var container lipgloss.Style
	if isActive || isSelected {
		container = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(nameFg).
			Padding(0, 0, 0, 1)
	} else {
		container = lipgloss.NewStyle().Padding(0, 0, 0, 2)
	}

	descStyle := lipgloss.NewStyle().Foreground(descFg).Padding(0, 0, 0, 2)
	fmt.Fprintf(w, "%s\n%s", container.Render(titleContent), descStyle.Render(t.Note)) //nolint:errcheck
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type TaskListModel struct {
	list         list.Model
	search       textinput.Model
	allTasks     []model.Task
	activeTaskID int64
	totals       map[int64]time.Duration
}

func NewTaskListModel(tasks []model.Task, activeTaskID int64, totals map[int64]time.Duration) TaskListModel {
	items := toListItems(activeFirstTasks(tasks, activeTaskID))

	l := list.New(items, taskDelegate{activeTaskID: activeTaskID, totals: totals}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetStatusBarItemName("task", "tasks")

	search := textinput.New()
	search.Placeholder = "Search tasks..."
	search.CharLimit = 100

	return TaskListModel{list: l, search: search, allTasks: tasks, activeTaskID: activeTaskID, totals: totals}
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
		case "e":
			if !m.search.Focused() {
				if item, ok := m.list.SelectedItem().(taskItem); ok {
					return m, func() tea.Msg { return editTaskMsg{task: item.Task} }
				}
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
	help := helpStyle.Render("n: new  e: edit  / search  enter: start/stop  q: quit")
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
