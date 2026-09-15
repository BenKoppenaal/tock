package ui

import (
	"strings"
	"time"
	"tock/db"
	"tock/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

type viewKind int

const (
	viewTaskList viewKind = iota
	viewDay
	viewEntryForm
	viewEditEntry
	viewNewTask
	viewEditTask
)

type App struct {
	db          *db.DB
	view        viewKind
	taskList    TaskListModel
	dayView     DayViewModel
	entryForm   EntryFormModel
	taskForm    TaskFormModel
	activeEntry *model.Entry
	width       int
	height      int
}

func NewApp(database *db.DB) (*App, error) {
	active, err := database.ActiveEntry()
	if err != nil {
		return nil, err
	}
	tasks, err := database.ListTasks("")
	if err != nil {
		return nil, err
	}
	totals, err := database.TotalTimeByTask()
	if err != nil {
		return nil, err
	}
	return &App{
		db:          database,
		view:        viewTaskList,
		activeEntry: active,
		taskList:    NewTaskListModel(database, tasks, activeTaskID(active), totals),
		dayView:     NewDayViewModel(database),
	}, nil
}

// refreshTaskList reloads tasks and totals from the DB and rebuilds the task list model.
func (a *App) refreshTaskList() {
	tasks, _ := a.db.ListTasks("")
	totals, _ := a.db.TotalTimeByTask()
	a.taskList = NewTaskListModel(a.db, tasks, activeTaskID(a.activeEntry), totals)
	a.taskList = a.taskList.setSize(a.width, a.height-5)
}

func (a *App) Init() tea.Cmd {
	return tick()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		inner := msg.Height - 3 // tabs (3 rows with padding)
		a.taskList = a.taskList.setSize(msg.Width, inner)
		a.dayView = a.dayView.setSize(msg.Width, inner)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "q":
			if a.view == viewTaskList || a.view == viewDay {
				return a, tea.Quit
			}
		case "1":
			if a.view == viewTaskList || a.view == viewDay {
				a.view = viewTaskList
				return a, nil
			}
		case "2":
			if a.view == viewTaskList || a.view == viewDay {
				a.view = viewDay
				return a, a.dayView.load()
			}
		case "n":
			if a.view == viewTaskList {
				a.taskForm = NewTaskFormModel(a.db)
				a.view = viewNewTask
				return a, a.taskForm.Init()
			}
		case "esc":
			if a.view == viewEntryForm || a.view == viewNewTask || a.view == viewEditTask {
				a.view = viewTaskList
				return a, nil
			}
			if a.view == viewEditEntry {
				a.view = viewDay
				return a, nil
			}
		}

	case openNewTaskMsg:
		a.taskForm = NewTaskFormModel(a.db)
		a.view = viewNewTask
		return a, a.taskForm.Init()

	case editTaskMsg:
		a.taskForm = NewTaskEditFormModel(a.db, msg.task)
		a.view = viewEditTask
		return a, a.taskForm.Init()

	case taskDeletedMsg:
		a.refreshTaskList()
		return a, nil

	case taskCreatedMsg:
		a.view = viewTaskList
		a.refreshTaskList()
		return a, nil

	case taskUpdatedMsg:
		a.view = viewTaskList
		a.refreshTaskList()
		return a, nil

	case entryDeletedMsg:
		a.refreshTaskList()
		return a, nil

	case editEntryMsg:
		a.entryForm = NewEntryEditFormModel(a.db, msg.entry)
		a.view = viewEditEntry
		return a, a.entryForm.Init()

	case entryEditedMsg:
		a.view = viewDay
		a.refreshTaskList()
		return a, a.dayView.load()

	case startEntryMsg:
		a.entryForm = NewEntryFormModel(a.db, msg.task, a.activeEntry)
		a.view = viewEntryForm
		return a, a.entryForm.Init()

	case entrySavedMsg:
		a.activeEntry = msg.entry
		a.view = viewTaskList
		a.refreshTaskList()
		return a, nil

	case entryStoppedMsg:
		a.activeEntry = nil
		a.view = viewTaskList
		a.refreshTaskList()
		return a, nil

	case tickMsg:
		return a, tick()
	}

	var cmd tea.Cmd
	switch a.view {
	case viewTaskList:
		a.taskList, cmd = a.taskList.Update(msg)
	case viewDay:
		a.dayView, cmd = a.dayView.Update(msg)
	case viewEntryForm, viewEditEntry:
		a.entryForm, cmd = a.entryForm.Update(msg)
	case viewNewTask, viewEditTask:
		a.taskForm, cmd = a.taskForm.Update(msg)
	}
	return a, cmd
}

func (a *App) View() string {
	tabsView := a.renderTabs()

	var content string
	var helpText string
	switch a.view {
	case viewTaskList:
		content = a.taskList.View()
	case viewDay:
		content = a.dayView.View()
	case viewEntryForm, viewEditEntry:
		content = a.entryForm.View()
		helpText = "enter: confirm  tab: next  ↑↓: ±1m  shift+↑↓: ±5m  esc: cancel"
	case viewNewTask, viewEditTask:
		content = a.taskForm.View()
		helpText = "tab: next field  enter: save  esc: cancel"
	}

	if helpText == "" {
		return lipgloss.JoinVertical(lipgloss.Left, tabsView, content)
	}

	helpBar := lipgloss.PlaceHorizontal(a.width, lipgloss.Center, helpStyle.Render(helpText))
	used := lipgloss.Height(tabsView) + lipgloss.Height(content) + 1
	spacerH := a.height - used
	if spacerH <= 0 {
		return lipgloss.JoinVertical(lipgloss.Left, tabsView, content, helpBar)
	}
	spacer := strings.Repeat("\n", spacerH-1)
	return lipgloss.JoinVertical(lipgloss.Left, tabsView, content, spacer, helpBar)
}

func (a *App) renderTabs() string {
	mk := func(label string, active bool) string {
		if active {
			return lipgloss.NewStyle().
				Padding(0, 1).
				Margin(1, 1, 1, 2).
				Bold(true).
				Background(colorHighlight).
				Foreground(lipgloss.Color("#ffffff")).
				Render(label)
		}
		return lipgloss.NewStyle().
			Padding(0, 1).
			Margin(1, 1, 1, 2).
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
			Render(label)
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Top,
		mk("[1] Tasks", a.view == viewTaskList || a.view == viewEntryForm || a.view == viewNewTask || a.view == viewEditTask),
		mk("[2] Day", a.view == viewDay),
	)
	title := lipgloss.NewStyle().
		Padding(0, 1).
		Margin(1, 2, 1, 1).
		Bold(true).
		Background(colorHighlight).
		Foreground(lipgloss.Color("#ffffff")).
		Render("Tock")

	var activeInfo string
	if a.activeEntry != nil {
		name := a.activeEntry.TaskName
		if len(name) > 20 {
			name = name[:19] + "…"
		}
		elapsed := a.activeEntry.Duration().Round(1e9)
		activeInfo = lipgloss.NewStyle().
			Margin(1, 1, 1, 0).
			Foreground(colorActive).
			Render("● " + name + " " + formatDuration(elapsed))
	}

	rightWidth := lipgloss.Width(activeInfo) + lipgloss.Width(title)
	gap := a.width - lipgloss.Width(tabs) - rightWidth
	if gap < 0 {
		gap = 0
	}
	spacer := lipgloss.NewStyle().Width(gap).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs, spacer, activeInfo, title)
}

func activeTaskID(e *model.Entry) int64 {
	if e == nil {
		return 0
	}
	return e.TaskID
}
