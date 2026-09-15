package ui

import (
	"fmt"
	"io"
	"strings"
	"time"
	"tock/db"
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
type taskDeletedMsg struct{}
type taskUnarchivedMsg struct{}
type reloadTasksMsg struct {
	index  int
	search string
}

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
	isArchived := t.Archived

	// Determine per-state colors
	var nameFg, descFg lipgloss.TerminalColor
	switch {
	case isActive:
		nameFg = colorActive
		descFg = colorActive
	case isArchived && isSelected:
		nameFg = colorMuted
		descFg = lipgloss.AdaptiveColor{Light: "#C0BBBC", Dark: "#555555"}
	case isArchived:
		nameFg = lipgloss.AdaptiveColor{Light: "#C0BBBC", Dark: "#555555"}
		descFg = lipgloss.AdaptiveColor{Light: "#D0CCCD", Dark: "#444444"}
	case isSelected:
		nameFg = colorHighlight
		descFg = lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	default:
		nameFg = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}
		descFg = lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	}

	// Left decoration is always 2 cols (border+pad or just pad); 2 cols right margin
	avail := m.Width() - 4
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

	rightLabel := durStr
	rightStyle := durStyle
	if rightLabel == "" {
		rightLabel = "n/a"
		rightStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#C0BBBC", Dark: "#555555"})
	}
	nameAvail := avail - len(rightLabel) - 1
	name := t.Name
	if len(name) > nameAvail {
		name = name[:max(0, nameAvail-1)] + "…"
	}
	gap := nameAvail - len(name)
	if gap < 0 {
		gap = 0
	}
	titleContent = nameStyle.Render(name) + strings.Repeat(" ", gap) + " " + rightStyle.Render(rightLabel)

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

	// Build note line: note left, right label (archived marker or last-tracked date)
	var line2 string
	descAvail := avail
	rightAnnotation := ""
	if isArchived {
		rightAnnotation = "archived"
	} else if t.LastTracked != nil {
		rightAnnotation = t.LastTracked.Format("2006-01-02")
	}
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#C0BBBC", Dark: "#555555"})
	if rightAnnotation != "" {
		noteAvail := descAvail - len(rightAnnotation) - 1
		note := t.Note
		if len(note) > noteAvail {
			note = note[:max(0, noteAvail-1)] + "…"
		}
		gap := noteAvail - len(note)
		if gap < 0 {
			gap = 0
		}
		noteStyled := lipgloss.NewStyle().Foreground(descFg).Render(note)
		rightStyled := dimStyle.Render(rightAnnotation)
		line2 = noteStyled + strings.Repeat(" ", gap) + " " + rightStyled
	} else {
		line2 = lipgloss.NewStyle().Foreground(descFg).Render(t.Note)
	}
	fmt.Fprintf(w, "%s", container.Render(titleContent+"\n"+line2)) //nolint:errcheck
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type taskConfirm int

const (
	confirmNone taskConfirm = iota
	confirmTaskDelete
	confirmTaskArchive
)

type TaskListModel struct {
	db           *db.DB
	list         list.Model
	search       textinput.Model
	allTasks     []model.Task
	activeTaskID int64
	totals       map[int64]time.Duration
	confirm      taskConfirm
	confirmTask  model.Task
	showArchived bool
	width        int
	height       int
}

func NewTaskListModel(database *db.DB, tasks []model.Task, activeTaskID int64, totals map[int64]time.Duration) TaskListModel {
	items := toListItems(activeFirstTasks(tasks, activeTaskID))

	l := list.New(items, taskDelegate{activeTaskID: activeTaskID, totals: totals}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetStatusBarItemName("task", "tasks")

	search := textinput.New()
	search.Placeholder = "Search tasks..."
	search.CharLimit = 100

	return TaskListModel{db: database, list: l, search: search, allTasks: tasks, activeTaskID: activeTaskID, totals: totals}
}

func (m TaskListModel) setSize(w, h int) TaskListModel {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
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
			if m.confirm != confirmNone {
				m.confirm = confirmNone
				return m, nil
			}
			if m.search.Focused() {
				m.search.Blur()
				m.search.SetValue("")
				m = m.applyFilter()
				return m, nil
			}
		case "a":
			if !m.search.Focused() && m.confirm == confirmNone {
				m.showArchived = !m.showArchived
				idx := m.list.Index()
				q := m.search.Value()
				return m, func() tea.Msg { return reloadTasksMsg{index: idx, search: q} }
			}
		case "u":
			if !m.search.Focused() && m.confirm == confirmNone {
				if item, ok := m.list.SelectedItem().(taskItem); ok {
					if item.Archived {
						m.db.UnarchiveTask(item.ID) //nolint:errcheck
						return m, func() tea.Msg { return taskUnarchivedMsg{} }
					}
				}
			}
		case "d":
			if !m.search.Focused() && m.confirm == confirmNone {
				if item, ok := m.list.SelectedItem().(taskItem); ok {
					if !item.Archived {
						m.confirmTask = item.Task
						if m.totals[item.ID] > 0 {
							m.confirm = confirmTaskArchive
						} else {
							m.confirm = confirmTaskDelete
						}
						return m, nil
					}
				}
			}
		case "y":
			if m.confirm == confirmTaskDelete {
				m.db.DeleteTask(m.confirmTask.ID) //nolint:errcheck
				m.confirm = confirmNone
				return m, func() tea.Msg { return taskDeletedMsg{} }
			}
			if m.confirm == confirmTaskArchive {
				m.db.ArchiveTask(m.confirmTask.ID) //nolint:errcheck
				m.confirm = confirmNone
				return m, func() tea.Msg { return taskDeletedMsg{} }
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

func (m TaskListModel) HelpText() string {
	switch m.confirm {
	case confirmTaskDelete:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Bold(true).
			Render(fmt.Sprintf("Delete \"%s\"? y to confirm  esc to cancel", m.confirmTask.Name))
	case confirmTaskArchive:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Bold(true).
			Render(fmt.Sprintf("Archive \"%s\"? y to confirm  esc to cancel", m.confirmTask.Name))
	default:
		if m.showArchived {
			return helpStyle.Render("n: new  e: edit  d: delete/archive  u: unarchive  a: hide archived  / search  enter: start/stop  q: quit")
		}
		return helpStyle.Render("n: new  e: edit  d: delete/archive  a: show archived  / search  enter: start/stop  q: quit")
	}
}

func (m TaskListModel) View() string {
	if m.search.Focused() {
		l := m.list
		l.SetSize(m.width, m.height-1)
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().PaddingLeft(2).Render(m.search.View()),
			l.View(),
		)
	}
	return m.list.View()
}
