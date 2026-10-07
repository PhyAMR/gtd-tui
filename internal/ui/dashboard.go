package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/phuniverse/gtd/internal/calendar"
	"github.com/phuniverse/gtd/internal/config"
	"github.com/phuniverse/gtd/internal/db"
	"github.com/phuniverse/gtd/internal/pomodoro"
	"github.com/phuniverse/gtd/internal/scheduler"
)

type FocusPane int

const (
	FocusTodo FocusPane = iota
	FocusAcademic
	FocusOther
	FocusPomodoro
	FocusCommand
)

type Model struct {
	db        *db.Database
	cal       *calendar.Client
	sched     *scheduler.Scheduler
	config    *config.Config
	
	focus     FocusPane
	width     int
	height    int
	
	// Pane A - Todo List
	todos     []*db.Task
	todoCursor int
	todoProgress float64
	intervalInput string
	
	// Pane B - Academic Events
	academicEvents []*calendar.Event
	
	// Pane C - Other Events
	otherEvents []*calendar.Event
	
	// Pane D - Pomodoro
	pomodoro  pomodoro.Model
	
	// Pane E - Command
	commandInput textinput.Model
	commandHistory []string
	
	quitting  bool
	err       error
}

type syncMsg struct {
	todos     []*db.Task
	academic  []*calendar.Event
	other     []*calendar.Event
}

func RunDashboard(database *db.Database, calClient *calendar.Client, sched *scheduler.Scheduler, cfg *config.Config) error {
	cmdInput := textinput.New()
	cmdInput.Placeholder = "gtd command..."
	cmdInput.CharLimit = 100

	m := Model{
		db:           database,
		cal:          calClient,
		sched:        sched,
		config:       cfg,
		focus:        FocusTodo,
		pomodoro:     pomodoro.NewModel(cfg.PomodoroWorkMins, cfg.PomodoroBreakMins, cfg.PomodoroLongBreak),
		commandInput: cmdInput,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.syncData(),
		tea.EnterAltScreen,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.focus != FocusCommand {
				m.quitting = true
				return m, tea.Quit
			}
		
		case "tab":
			m.focus = (m.focus + 1) % 5
			if m.focus == FocusCommand {
				m.commandInput.Focus()
			} else {
				m.commandInput.Blur()
			}
		
		case "enter":
			if m.focus == FocusCommand {
				return m.handleCommand()
			} else if m.focus == FocusTodo {
				return m.toggleTodo()
			}
		
		default:
			return m.handleFocusedInput(msg)
		}

	case syncMsg:
		m.todos = msg.todos
		m.academic = msg.academic
		m.other = msg.other
		m.updateProgress()

	case pomodoro.TickMsg:
		m.pomodoro, cmd = m.pomodoro.Update(msg)
		return m, cmd
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	topHeight := m.height * 50 / 100
	bottomHeight := m.height - topHeight - 2

	// Top three panes
	paneWidth := m.width / 3
	todoPane := m.renderTodoPane(paneWidth, topHeight)
	academicPane := m.renderAcademicPane(paneWidth, topHeight)
	otherPane := m.renderOtherPane(paneWidth, topHeight)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, todoPane, academicPane, otherPane)

	// Bottom two panes
	pomodoroWidth := m.width * 3 / 4
	commandWidth := m.width - pomodoroWidth

	pomodoroPane := m.renderPomodoroPane(pomodoroWidth, bottomHeight)
	commandPane := m.renderCommandPane(commandWidth, bottomHeight)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, pomodoroPane, commandPane)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

func (m *Model) syncData() tea.Cmd {
	return func() tea.Msg {
		today := time.Now()
		start := today.Truncate(24 * time.Hour)
		end := start.Add(24 * time.Hour)

		todos, _ := m.db.GetTasksDueToday()
		academic, _ := m.cal.GetStudyEvents(start, end)
		other, _ := m.cal.GetEvents("primary", start, end)

		return syncMsg{
			todos:    todos,
			academic: academic,
			other:    other,
		}
	}
}

func (m *Model) updateProgress() {
	if len(m.todos) == 0 {
		m.todoProgress = 0
		return
	}

	completed := 0
	for _, task := range m.todos {
		if task.Status == "done" {
			completed++
		}
	}

	m.todoProgress = float64(completed) / float64(len(m.todos))
}

func (m *Model) handleCommand() (tea.Model, tea.Cmd) {
	cmd := m.commandInput.Value()
	m.commandHistory = append(m.commandHistory, cmd)
	m.commandInput.SetValue("")
	
	return m, m.syncData()
}

func (m *Model) toggleTodo() (tea.Model, tea.Cmd) {
	if m.todoCursor >= 0 && m.todoCursor < len(m.todos) {
		task := m.todos[m.todoCursor]
		if task.Status == "pending" {
			m.sched.MarkTaskComplete(task.ID, 1)
		}
	}
	return m, m.syncData()
}

func (m Model) handleFocusedInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focus {
	case FocusTodo:
		return m.handleTodoInput(msg)
	case FocusPomodoro:
		return m.handlePomodoroInput(msg)
	case FocusCommand:
		m.commandInput, cmd = m.commandInput.Update(msg)
	}

	return m, cmd
}

func (m Model) handleTodoInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.todoCursor < len(m.todos)-1 {
			m.todoCursor++
		}
	case "k", "up":
		if m.todoCursor > 0 {
			m.todoCursor--
		}
	case "-":
		m.intervalInput = "-"
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if m.intervalInput == "-" {
			interval := int(msg.Runes[0] - '0')
			if m.todoCursor >= 0 && m.todoCursor < len(m.todos) {
				task := m.todos[m.todoCursor]
				m.sched.MarkTaskComplete(task.ID, interval)
				m.intervalInput = ""
				return m, m.syncData()
			}
		}
	}
	return m, nil
}

func (m Model) handlePomodoroInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		if m.pomodoro.Running {
			m.pomodoro = m.pomodoro.Pause()
		} else {
			return m.pomodoro.Start()
		}
	case "r":
		m.pomodoro = m.pomodoro.Reset()
	case "n":
		return m.pomodoro.Skip()
	}
	return m, nil
}

func getFocusStyle(focused bool) lipgloss.Style {
	if focused {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)
}
