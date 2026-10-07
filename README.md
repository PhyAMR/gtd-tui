# GTD - Getting Things Done TUI

A production-ready TUI (Terminal User Interface) application for managing study schedules using Google Calendar and spaced repetition techniques. Built with Go and the Bubble Tea ecosystem.

## Features

- **📚 Smart Task Management**: Spaced repetition system for optimal learning
- **🗓️ Google Calendar Integration**: Automatic sync with Class and Study calendars
- **🍅 Built-in Pomodoro Timer**: Stay focused with integrated time management
- **⌨️ Keyboard-First Interface**: Vim-like navigation for efficiency
- **💾 Local Database**: BoltDB for fast, reliable task persistence
- **🎨 Beautiful TUI**: Built with Bubble Tea, Lip Gloss, and Bubbles

## Screenshots

```
┌─── 📚 To-Do List ───┬─── 🗓️  Academic Events ───┬─── 🎉 Other Events ───┐
│ Progress: ████░░░░░ │ 18:00 - 19:00             │ 18:00  Dinner         │
│                     │   Study Group - Calculus  │ 20:00  Gym Workout    │
│ ▸ [ ] Finish Ch. 3  │ 20:00 - 21:00             │                       │
│   [ ] Lab Report    │   Problem Session - Phys  │                       │
└─────────────────────┴───────────────────────────┴───────────────────────┘
┌─────────────── 🍅 Pomodoro Timer ──────────────────┬─── ⌨ Command ───┐
│ CURRENT STATUS: Focus Time!                        │                   │
│ REMAINING: 24:35 (of 25:00)                       │ > gtd add ...     │
│ ████████████████████░░░░░░░░░░░░░░░░░░░░░░░░      │                   │
│ CYCLE PROGRESS: 2/4 Pomodoros Completed           │                   │
└────────────────────────────────────────────────────┴───────────────────┘
```

## Installation

### Prerequisites

- Go 1.21 or higher
- Google Calendar API credentials (see Setup section)

### Build from source

```bash
git clone https://github.com/phuniverse/gtd
cd gtd
make build
```

Or install directly:

```bash
go install ./cmd/gtd
```

## Setup

### 1. Google Calendar API Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable the Google Calendar API
4. Create OAuth 2.0 credentials (Desktop app)
5. Download the credentials as `credentials.json`
6. Place it in `~/.credentials/credentials.json`

### 2. First Run

```bash
gtd dashboard
```

On first run, you'll be prompted to authorize the application. Follow the OAuth flow in your browser.

### 3. Configuration

The default configuration is created at `~/.config/gtd/config.json`. You can customize:

```json
{
  "class_calendar_id": "primary",
  "study_calendar_id": "primary",
  "review_ratio": 0.75,
  "pomodoro_work_mins": 25,
  "pomodoro_break_mins": 5,
  "pomodoro_long_break": 15,
  "theme": "default"
}
```

## Usage

### Dashboard

Launch the interactive TUI:

```bash
gtd dashboard
```

#### Keybindings

**Global:**
- `Tab` - Cycle focus between panes
- `q` or `Ctrl+C` - Quit

**To-Do List Pane:**
- `j/k` - Navigate up/down
- `Enter` - Toggle task completion
- `-N` (e.g., `-3`) - Mark done with interval N
- `a` - Add new task
- `d` - Delete task

**Pomodoro Pane:**
- `s` - Start/Pause
- `r` - Reset
- `n` - Skip to next phase

### CLI Commands

List today's review sessions:
```bash
gtd list -r
```

List all classes:
```bash
gtd list -c
```

Add a task to a class:
```bash
gtd add "Read Chapter 5" -c "Calculus I"
```

Remove a task:
```bash
gtd rm "Read Chapter 5" -c "Calculus I"
```

Run the planning script:
```bash
gtd plan
```

Use `--dry-run` to preview changes:
```bash
gtd add "Task" -c "Class" --dry-run
```

## Spaced Repetition System

When you mark a task as done, you can specify an interval:

- `-0` - Never review again (task complete)
- `-1` - Review tomorrow (1 day)
- `-2` - Review in 3 days
- `-3` - Review in 1 week
- `-4` - Review in 2 weeks
- `-5+` - Review in N weeks

Tasks left undone are automatically rescheduled to the next review session.

## Development

### Project Structure

```
gtd/
├── cmd/gtd/              # CLI entrypoint
├── internal/
│   ├── ui/              # TUI components (Bubble Tea)
│   ├── pomodoro/        # Pomodoro timer logic
│   ├── calendar/        # Google Calendar client
│   ├── scheduler/       # Spaced repetition scheduler
│   ├── db/              # BoltDB wrapper
│   └── config/          # Configuration management
├── docs/                # Documentation
├── tests/               # Integration tests
└── examples/            # Example configs and data
```

### Running Tests

```bash
make test
```

Run integration tests (requires Google Calendar setup):
```bash
make test-integration
```

### Linting

```bash
make lint
```

### Building

```bash
make build
```

The binary will be in `./bin/gtd`.

## Integration with Existing Pomodoro

This project integrates the Pomodoro timer from [Bahaaio/pomo](https://github.com/Bahaaio/pomo). The core timer logic is adapted and embedded into the TUI.

## Integration with Study Scheduler

The planning functionality uses your existing `study_scheduler.go` script. Set the path in `~/.config/gtd/config.json`:

```json
{
  "plan_script_path": "/path/to/study_scheduler.go"
}
```

## Troubleshooting

### OAuth Issues

If you see authentication errors:
1. Delete `~/.config/gtd/tokens.json`
2. Run `gtd dashboard` again
3. Complete the OAuth flow

### Calendar Not Syncing

- Check internet connection
- Verify calendar IDs in config
- Check API quota in Google Cloud Console

### Database Locked

If you see "database locked" errors:
- Ensure only one instance of gtd is running
- Delete `~/.config/gtd/gtd.db` to reset (will lose local data)

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Run linter and tests
5. Submit a pull request

## License

MIT License - see LICENSE file

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [pomo](https://github.com/Bahaaio/pomo) - Pomodoro timer inspiration
- Google Calendar API

## Support

For issues and questions:
- Open an issue on GitHub
- Check the docs/ folder for detailed guides

---

**Made with ❤️ for focused, distraction-free studying**
