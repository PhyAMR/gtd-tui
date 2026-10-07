# Study Pomodoro — Omarchy plugin on top of OmaCal

[OmaCal](https://omacal.app)'s bar widget with two additions. A pomodoro timer
sits beside OmaCal's calendar mark in the bar. The popup gains the timer
controls and today's **study sessions**: the 📝 Review, 🎯 Practice Problems,
🔄 Spaced Review and 🔀 Integrated Review events that `study_scheduler.py`
creates each week.

It is built on OmaCal and doesn't work without it:

- **Calendar data** comes from the feed the OmaCal app writes
  (`~/.local/state/omacal/upcoming.json`). The widget reads it through OmaCal's
  own `read-feed.py`, exactly as `omacal.upcoming` does. The plugin never talks
  to Google, and there are no API keys or tokens to manage.
- **Shared QML/JS** (`Model.js`, `Timeline.mjs`, `DayView.qml`, …) is imported
  directly from the installed `omacal.upcoming` plugin, so OmaCal's fixes arrive
  with OmaCal's updates.
- **`Panel.qml` is a fork** of `omacal.upcoming`'s (plugin version 1.11.2). Every
  change is marked `Study Pomodoro:`. It replaces OmaCal's widget in the bar
  (disabled, not removed) and keeps its `omacal.upcoming` IPC target, so the
  app's own refreshes still arrive.
- **The planner** (`planner/study_planner.py`) reads lectures, busy time and
  exams with `omacal events list` and creates sessions with `omacal events create`.

## How sessions are planned

Every lecture gets its own review ladder. Twice a day (06:30 and 14:00) the
planner turns what falls due in the next 7 days into sessions and places
them in free time between 08:30 and 20:30. It keeps 30 min around lectures
and other events, and 15 min between study sessions.

| Session | When | Length |
|---|---|---|
| 📝 Review: *course* | same day as the lecture (by next noon at the latest) | ¼ of the lecture, in pomodoros (2 h → 25 min) |
| 📝 Review: *course* (n lectures) | 2nd pass, ~2 days later, bundled per course | 15 min per lecture |
| 🔄 Spaced / 🔀 Interleaved review | later passes; courses that fall due together share a session | 10 min per lecture |
| 🎯 Practice: *course* | mornings, after that week's first lecture; 0–2 a week by difficulty (1 → 0, 2–3 → 1, 4–5 → 2) | 2 pomodoros |
| 🎯 Exam prep: *course* | up to 3 weeks before an exam whose title has the course's code | 2 pomodoros |

When a review ends, rate it in the popup: **again / hard / good / easy**. The
grade sets the next interval for the lectures it covered: again brings them
back tomorrow, while good roughly triples the interval (2 → 6 → 18 days).
Harder courses come back sooner. An unrated review counts as *good* after 20 h.
Once an exam is known, no review is scheduled for later than two days before it.

Sessions are fixed once created, with three exceptions:
- one more than a day away that now clashes with a new event is moved;
- if you drag a session to another time, the planner follows it;
- if you delete one, its lectures are queued again.

At most 3 h of self-study a day (`daily_cap_minutes`). 45 min of each future
day stays free of practice for reviews that come due later.

## Setup

1. Install OmaCal, open it, and connect your Google account:
   ```bash
   curl -fsSL https://omacal.app/install.sh | sh
   omacal
   ```
2. Install the plugin:
   ```bash
   cd ~/Scripts/gtd-tui/omarchy-plugin
   ./install.sh
   ```

The installer is safe to re-run. It:

- copies the plugin to `~/.config/omarchy/plugins/phuniverse.study-pomodoro/`,
  puts it where OmaCal's widget was, and restarts the Omarchy shell;
- links the `study-pomodoro` CLI into `~/.local/bin`;
- adds keybindings to `~/.config/hypr/bindings.lua`, after backing the file up;
- runs `study-pomodoro setup`, which asks for calendars and study hours, then
  each course's difficulty (1–5) and exam code (optional; add it once the exam
  shows up in your exam calendar);
- previews the next 7 days and asks before creating anything;
- installs a systemd user timer that plans at 06:30 and 14:00. OmaCal must be
  running to create events; the timer retries every 10 minutes.

After editing the plugin, run `./install.sh --plugin-only` to update the installed copy.
`./uninstall.sh` removes it and puts OmaCal's own widget back.

When OmaCal updates `omacal.upcoming` to a new minor version, the installer
warns you. The fork may then need its `Study Pomodoro:` changes re-applied to
the new `Panel.qml`.

## Using it

| Where | Action |
|---|---|
| OmaCal mark | Same as OmaCal: left popup, right quick-add, middle open app |
| 🍅 in the bar: left / middle / right click | Popup / start-pause / skip phase |
| `Super+Alt+P` · `Super+Shift+Alt+P` · `Super+Ctrl+Alt+P` | Start-pause · skip · popup |
| In the popup | `p`/Space start-pause · `n` skip · `x` reset; OmaCal's keys still work (`o` `s` `r` `j` `q`) |

Clicking a study session opens OmaCal on that day. The timer shows only 🍅
while idle and counts down (dimmed when paused) otherwise. Its cycle matches
the old TUI: focus, then a short break, with a long break after 4 rounds.
When a phase ends you get a notification, and the timer survives shell restarts.

Timer settings go on the widget's entry in `~/.config/omarchy/shell.json`:

```json
{ "id": "phuniverse.study-pomodoro", "focusMins": 50, "shortBreakMins": 10,
  "longBreakMins": 20, "cycles": 4, "notify": true, "icon": "🍅" }
```

### CLI

```
study-pomodoro setup             calendars, study hours, course difficulty + exam codes
study-pomodoro plan [--dry-run]  plan the next 7 days now
study-pomodoro progress          per-course overview
study-pomodoro calendars         OmaCal calendars and their ids
study-pomodoro toggle|reset|skip|status|popup
study-pomodoro legacy-plan       the old weekly study_scheduler.py
```

Settings: `~/.config/study-pomodoro/planner.json`. The planner's memory:
`~/.local/state/study-pomodoro/planner-state.json`.

Tests: `node tests/model.test.js` (timer, feed) and `python3 -m unittest discover tests` (planner).
