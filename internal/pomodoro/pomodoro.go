package pomodoro

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type State int

const (
	StateFocus State = iota
	StateShortBreak
	StateLongBreak
	StatePaused
)

type Model struct {
	State          State
	Duration       time.Duration
	Remaining      time.Duration
	CycleCount     int
	TotalCycles    int
	Running        bool
	WorkDuration   time.Duration
	ShortBreak     time.Duration
	LongBreak      time.Duration
	Logs           []LogEntry
}

type LogEntry struct {
	Type      string
	StartedAt time.Time
	Duration  time.Duration
}

type TickMsg time.Time

func NewModel(workMins, shortBreakMins, longBreakMins int) Model {
	workDur := time.Duration(workMins) * time.Minute
	return Model{
		State:        StateFocus,
		Duration:     workDur,
		Remaining:    workDur,
		CycleCount:   0,
		TotalCycles:  4,
		Running:      false,
		WorkDuration: workDur,
		ShortBreak:   time.Duration(shortBreakMins) * time.Minute,
		LongBreak:    time.Duration(longBreakMins) * time.Minute,
		Logs:         []LogEntry{},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		if !m.Running {
			return m, nil
		}

		m.Remaining -= time.Second
		if m.Remaining <= 0 {
			return m.nextPhase(), tick()
		}
		return m, tick()
	}
	return m, nil
}

func (m Model) Start() (Model, tea.Cmd) {
	m.Running = true
	return m, tick()
}

func (m Model) Pause() Model {
	m.Running = false
	return m
}

func (m Model) Reset() Model {
	m.Remaining = m.Duration
	m.Running = false
	return m
}

func (m Model) Skip() (Model, tea.Cmd) {
	m = m.nextPhase()
	if m.Running {
		return m, tick()
	}
	return m, nil
}

func (m Model) nextPhase() Model {
	m.addLog()

	switch m.State {
	case StateFocus:
		m.CycleCount++
		if m.CycleCount >= m.TotalCycles {
			m.State = StateLongBreak
			m.Duration = m.LongBreak
			m.CycleCount = 0
		} else {
			m.State = StateShortBreak
			m.Duration = m.ShortBreak
		}
	case StateShortBreak, StateLongBreak:
		m.State = StateFocus
		m.Duration = m.WorkDuration
	}

	m.Remaining = m.Duration
	return m
}

func (m *Model) addLog() {
	entry := LogEntry{
		Type:      m.StateName(),
		StartedAt: time.Now().Add(-m.Duration + m.Remaining),
		Duration:  m.Duration - m.Remaining,
	}
	m.Logs = append([]LogEntry{entry}, m.Logs...)
	if len(m.Logs) > 3 {
		m.Logs = m.Logs[:3]
	}
}

func (m Model) StateName() string {
	switch m.State {
	case StateFocus:
		return "Focus"
	case StateShortBreak:
		return "Short Break"
	case StateLongBreak:
		return "Long Break"
	case StatePaused:
		return "Paused"
	default:
		return "Unknown"
	}
}

func (m Model) NextStateName() string {
	switch m.State {
	case StateFocus:
		if m.CycleCount+1 >= m.TotalCycles {
			return "Long Break"
		}
		return "Short Break"
	case StateShortBreak, StateLongBreak:
		return "Focus"
	default:
		return "Focus"
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) ProgressPercent() float64 {
	if m.Duration == 0 {
		return 0
	}
	elapsed := m.Duration - m.Remaining
	return float64(elapsed) / float64(m.Duration)
}
