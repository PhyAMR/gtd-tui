# Quick Start Guide

Get up and running with GTD in 5 minutes!

## Installation

### Option 1: Build from source

```bash
git clone https://github.com/phuniverse/gtd
cd gtd
make build
sudo cp bin/gtd /usr/local/bin/
```

### Option 2: Go install

```bash
go install github.com/phuniverse/gtd/cmd/gtd@latest
```

## Google Calendar Setup (5 minutes)

1. **Get Google API credentials**
   ```bash
   ./scripts/setup-credentials.sh
   ```
   Follow the interactive prompts to download and install your credentials.

2. **First run**
   ```bash
   gtd dashboard
   ```
   
   You'll see a URL. Copy it to your browser, authorize the app, and paste the code back.

## Your First Task

1. **Add a task**
   ```bash
   gtd add "Read Chapter 5" -c "Calculus I"
   ```

2. **List today's reviews**
   ```bash
   gtd list -r
   ```

3. **Open the dashboard**
   ```bash
   gtd dashboard
   ```

## Dashboard Basics

### Navigation
- Press `Tab` to move between panes
- Press `q` to quit

### To-Do List (Top Left)
- `j` / `k` - Move down/up
- `Enter` - Mark task done (interval = 1 day)
- `-3` then a number - Mark done with custom interval
  - `-0` = Never review again
  - `-1` = Review tomorrow
  - `-3` = Review in 1 week

### Pomodoro Timer (Bottom)
- `s` - Start/Pause
- `r` - Reset
- `n` - Skip to next phase

### Command Pane (Bottom Right)
- Type any gtd command
- Press `Enter` to execute

## Example Workflow

### Morning Review
```bash
# See what's scheduled today
gtd list -r

# Open dashboard
gtd dashboard
```

In the dashboard:
1. Tab to To-Do pane
2. Use `j`/`k` to navigate tasks
3. For each task:
   - Press `Enter` if easy (review tomorrow)
   - Press `-3` if medium (review in 1 week)
   - Press `-0` if mastered (never review)

### Adding Tasks During Class
```bash
gtd add "Review today's lecture notes" -c "Physics 101"
gtd add "Complete problem set 3" -c "Physics 101"
```

### Study Session
```bash
gtd dashboard
```

1. Tab to Pomodoro pane
2. Press `s` to start 25-minute focus session
3. Tab to To-Do pane
4. Work through tasks
5. When Pomodoro beeps, take a break!

## Calendar Integration

GTD creates two types of events:

**Class Calendar** (you manage):
- Your lecture times
- Lab sessions
- Office hours

**Study Calendar** (GTD manages):
- Review sessions (after classes)
- Problem sessions
- Study blocks

## Tips

### Customize Settings
Edit `~/.config/gtd/config.json`:
```json
{
  "pomodoro_work_mins": 50,
  "pomodoro_break_mins": 10,
  "review_ratio": 0.5
}
```

### Run Plan Script
```bash
gtd plan
```
This analyzes your class schedule and creates optimal study sessions.

### Dry Run
Test commands without making changes:
```bash
gtd add "Task" -c "Class" --dry-run
gtd plan --dry-run
```

## Troubleshooting

### "Failed to load config"
First run creates default config automatically. Check `~/.config/gtd/`

### "Unable to read credentials"
Run `./scripts/setup-credentials.sh` to set up Google API credentials.

### "Database locked"
Only one instance can run at a time. Close other gtd processes.

## Next Steps

- Read the full [README](README.md)
- Check [Google Calendar Setup](docs/GOOGLE_CALENDAR_SETUP.md)
- Explore [Architecture](docs/ARCHITECTURE.md)
- Configure keybindings to your preference

## Need Help?

- Check [GitHub Issues](https://github.com/phuniverse/gtd/issues)
- Read the documentation in `docs/`
- Open a new issue with details

Happy studying! 📚
