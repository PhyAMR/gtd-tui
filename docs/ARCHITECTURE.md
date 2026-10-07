# Architecture Overview

## System Design

GTD is built with a clean architecture separating concerns:

```
┌─────────────┐
│     CLI     │  (cobra commands)
└──────┬──────┘
       │
┌──────▼──────┐
│     TUI     │  (Bubble Tea)
└──────┬──────┘
       │
┌──────┴───────┬──────────┬───────────┬──────────┐
│              │          │           │          │
▼              ▼          ▼           ▼          ▼
Calendar    Scheduler   Database   Pomodoro   Config
```

## Component Responsibilities

### CLI Layer (`cmd/gtd`)
- Parse command-line arguments
- Route to appropriate handler
- Format output for terminal

### UI Layer (`internal/ui`)
- Bubble Tea models and updates
- Render 5-pane dashboard
- Handle keyboard input
- Focus management
- Styling with Lip Gloss

### Calendar (`internal/calendar`)
- OAuth2 authentication
- Google Calendar API client
- Event CRUD operations
- Sync logic

### Scheduler (`internal/scheduler`)
- Spaced repetition algorithm
- Task lifecycle management
- Review session generation
- Integration with plan script

### Database (`internal/db`)
- BoltDB wrapper
- Task persistence
- Event mappings
- Pomodoro history
- Settings storage

### Pomodoro (`internal/pomodoro`)
- Timer state machine
- Cycle management
- Break/work transitions
- Log keeping

### Config (`internal/config`)
- Configuration loading/saving
- Default values
- Path management

## Data Flow

### Adding a Task

```
User Input → CLI → Scheduler → Database → Calendar API
                              ↓
                         Local DB saved
                              ↓
                    Event created/updated
                              ↓
                         UI refreshes
```

### Syncing Data

```
Timer tick → Sync → Calendar API → Parse events
                         ↓
                    Compare with DB
                         ↓
                   Update local state
                         ↓
                    Re-render UI
```

### Pomodoro Flow

```
User starts → Update state → Start ticker
                  ↓
             Tick every 1s
                  ↓
          Decrement remaining
                  ↓
           Re-render timer
                  ↓
        Remaining reaches 0?
                  ↓
         Transition to next phase
                  ↓
              Log entry
```

## State Management

### TUI State
- Managed by Bubble Tea's Elm architecture
- Immutable updates via Update() function
- Single source of truth in Model

### Persistent State
- Tasks, events, settings in BoltDB
- Tokens in JSON file (secure permissions)
- Config in JSON file

### Remote State
- Google Calendar is canonical for events
- Local DB caches and extends with task metadata
- Periodic sync reconciles differences

## Error Handling

### Network Errors
- Retry with exponential backoff
- Degrade gracefully to offline mode
- Show user-friendly messages

### Database Errors
- Transaction rollback
- Corruption detection
- Auto-recovery where possible

### Calendar API Errors
- Rate limit handling
- OAuth refresh
- Quota management

## Security

### Token Storage
- OAuth tokens in `~/.config/gtd/` with 0600 perms
- Never logged or displayed
- Auto-refresh when expired

### Credentials
- User provides own Google API credentials
- Not bundled with application
- Clear setup documentation

### Data Privacy
- All data stored locally
- Only syncs with user's own calendars
- No telemetry or external servers

## Testing Strategy

### Unit Tests
- Pure functions (scheduler logic, date math)
- Database operations (with temp DB)
- Config parsing

### Integration Tests
- Mock Calendar API (recorded fixtures)
- End-to-end CLI commands
- TUI interactions (where feasible)

### Manual Testing
- Real Google Calendar
- Various terminal sizes
- Different OSes (Linux, macOS)

## Performance Considerations

### Database
- BoltDB is fast for reads
- Batch writes where possible
- Indexes not needed for small datasets

### UI Rendering
- Bubble Tea handles efficiently
- Avoid re-rendering unchanged panes
- Debounce sync operations

### API Calls
- Batch calendar queries
- Cache responses
- Respect rate limits

## Future Enhancements

Potential additions:
- Export to HTML/PDF
- Desktop notifications
- Mobile companion app
- Themes (light/dark/custom)
- Plugin system
- Multi-user support (shared calendars)
