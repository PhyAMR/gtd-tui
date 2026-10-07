package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderTodoPane(width, height int) string {
	style := getFocusStyle(m.focus == FocusTodo)
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("📚 To-Do List"))
	content.WriteString("\n\n")

	if len(m.todos) == 0 {
		content.WriteString(lipgloss.NewStyle().Faint(true).Render("No tasks for today"))
	} else {
		// Progress bar
		barWidth := 20
		filled := int(m.todoProgress * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		content.WriteString(fmt.Sprintf("Progress: %s %.0f%%\n\n", bar, m.todoProgress*100))

		// Task list
		for i, task := range m.todos {
			cursor := " "
			if i == m.todoCursor {
				cursor = "▸"
			}

			status := "[ ]"
			if task.Status == "done" {
				status = "[✓]"
			}

			taskStyle := lipgloss.NewStyle()
			if task.Status == "done" {
				taskStyle = taskStyle.Strikethrough(true).Faint(true)
			}
			if i == m.todoCursor {
				taskStyle = taskStyle.Bold(true)
			}

			line := fmt.Sprintf("%s %s %s", cursor, status, task.Description)
			content.WriteString(taskStyle.Render(line))
			content.WriteString("\n")
		}

		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Faint(true).Render("j/k: move • enter: toggle • -N: mark done w/ interval"))
	}

	return style.Width(width - 4).Height(height - 4).Render(content.String())
}

func (m Model) renderAcademicPane(width, height int) string {
	style := getFocusStyle(m.focus == FocusAcademic)
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("🗓️  Academic Events"))
	content.WriteString("\n\n")

	if len(m.academicEvents) == 0 {
		content.WriteString(lipgloss.NewStyle().Faint(true).Render("No events today"))
	} else {
		for _, event := range m.academicEvents {
			timeStr := event.Start.Format("15:04") + " - " + event.End.Format("15:04")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Render(timeStr))
			content.WriteString("\n")
			content.WriteString(fmt.Sprintf("  %s\n", event.Summary))
			if event.Location != "" {
				content.WriteString(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("  📍 %s\n", event.Location)))
			}
			content.WriteString("\n")
		}
	}

	return style.Width(width - 4).Height(height - 4).Render(content.String())
}

func (m Model) renderOtherPane(width, height int) string {
	style := getFocusStyle(m.focus == FocusOther)
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("🎉 Other Events"))
	content.WriteString("\n\n")

	if len(m.otherEvents) == 0 {
		content.WriteString(lipgloss.NewStyle().Faint(true).Render("No events today"))
	} else {
		for _, event := range m.otherEvents {
			// Skip study events
			if strings.Contains(event.Summary, "📚") || strings.Contains(event.Summary, "Study") {
				continue
			}

			timeStr := event.Start.Format("15:04")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("200")).Render(timeStr))
			content.WriteString(fmt.Sprintf("  %s\n", event.Summary))
		}
	}

	return style.Width(width - 4).Height(height - 4).Render(content.String())
}

func (m Model) renderPomodoroPane(width, height int) string {
	style := getFocusStyle(m.focus == FocusPomodoro)
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("🍅 Pomodoro Timer"))
	content.WriteString("\n\n")

	// Current status
	statusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	content.WriteString(statusStyle.Render(fmt.Sprintf("CURRENT STATUS: %s\n", m.pomodoro.StateName())))
	content.WriteString("\n")

	// Time remaining
	mins := int(m.pomodoro.Remaining.Minutes())
	secs := int(m.pomodoro.Remaining.Seconds()) % 60
	content.WriteString(fmt.Sprintf("REMAINING: %02d:%02d (of %.0f:00)\n", 
		mins, secs, m.pomodoro.Duration.Minutes()))

	// Progress bar
	barWidth := 40
	filled := int(m.pomodoro.ProgressPercent() * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	content.WriteString(fmt.Sprintf("%s\n\n", bar))

	// Cycle progress
	content.WriteString(fmt.Sprintf("CYCLE PROGRESS: %d/%d Pomodoros Completed\n", 
		m.pomodoro.CycleCount, m.pomodoro.TotalCycles))
	content.WriteString(fmt.Sprintf("NEXT UP: %s\n\n", m.pomodoro.NextStateName()))

	// Controls
	content.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"CONTROLS: s = start/pause • r = reset • n = skip\n\n"))

	// Settings
	content.WriteString(lipgloss.NewStyle().Faint(true).Render(
		fmt.Sprintf("SETTINGS: Work: %d min, Short Break: %d min, Long Break: %d min\n\n",
			int(m.pomodoro.WorkDuration.Minutes()),
			int(m.pomodoro.ShortBreak.Minutes()),
			int(m.pomodoro.LongBreak.Minutes()))))

	// Recent logs
	if len(m.pomodoro.Logs) > 0 {
		content.WriteString("LOG:\n")
		for _, log := range m.pomodoro.Logs {
			content.WriteString(lipgloss.NewStyle().Faint(true).Render(
				fmt.Sprintf("  %s at %s (%.0f min)\n", 
					log.Type, 
					log.StartedAt.Format("15:04:05"),
					log.Duration.Minutes())))
		}
	}

	return style.Width(width - 4).Height(height - 4).Render(content.String())
}

func (m Model) renderCommandPane(width, height int) string {
	style := getFocusStyle(m.focus == FocusCommand)
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("⌨  Command"))
	content.WriteString("\n\n")

	content.WriteString(m.commandInput.View())
	content.WriteString("\n\n")

	if len(m.commandHistory) > 0 {
		content.WriteString(lipgloss.NewStyle().Faint(true).Render("Recent:\n"))
		start := len(m.commandHistory) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(m.commandHistory); i++ {
			content.WriteString(lipgloss.NewStyle().Faint(true).Render(
				fmt.Sprintf("  > %s\n", m.commandHistory[i])))
		}
	}

	return style.Width(width - 4).Height(height - 4).Render(content.String())
}
