# Developer Guide

## Local Development Setup

### Prerequisites

1. **Go 1.21+**
   ```bash
   go version  # Should output 1.21 or higher
   ```

2. **golangci-lint** (optional but recommended)
   ```bash
   # macOS
   brew install golangci-lint

   # Linux
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
   ```

3. **Make**
   ```bash
   make --version
   ```

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/phuniverse/gtd
cd gtd

# Download dependencies
go mod download

# Verify everything compiles
go build ./...

# Run tests
go test ./...

# Build the binary
make build
```

## Project Layout

### Directory Structure

```
cmd/gtd/              - CLI entrypoint, cobra commands
internal/             - Private application code
  ├── calendar/       - Google Calendar API client
  ├── config/         - Configuration management  
  ├── db/             - BoltDB wrapper for persistence
  ├── pomodoro/       - Pomodoro timer state machine
  ├── scheduler/      - Spaced repetition logic
  └── ui/             - Bubble Tea TUI components
tests/                - Integration tests & fixtures
docs/                 - Documentation
examples/             - Example configurations
scripts/              - Helper scripts
```

### Key Files

- `cmd/gtd/main.go` - CLI entrypoint, command definitions
- `internal/ui/dashboard.go` - Main TUI model and update logic
- `internal/ui/panes.go` - Rendering functions for each pane
- `internal/scheduler/scheduler.go` - Core scheduling algorithm
- `internal/db/db.go` - Database operations
- `internal/calendar/client.go` - Calendar API wrapper

## Development Workflow

### Running Locally

```bash
# Run the dashboard directly
go run ./cmd/gtd dashboard

# Or use make
make run

# Run with environment variables
GTD_DEBUG=true go run ./cmd/gtd dashboard
```

### Making Changes

1. **Create a feature branch**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes**
   - Edit code in `internal/`
   - Add tests in `*_test.go` files
   - Update docs if needed

3. **Test your changes**
   ```bash
   # Run unit tests
   make test

   # Run linter
   make lint

   # Format code
   make fmt
   ```

4. **Commit and push**
   ```bash
   git add .
   git commit -m "feat: add my feature"
   git push origin feature/my-feature
   ```

5. **Open a Pull Request**

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/scheduler/...

# Verbose output
go test -v ./...
```

### Integration Tests

Integration tests require Google Calendar credentials:

```bash
# Set environment variables
export GTD_TEST_CREDENTIALS=~/.credentials/credentials.json
export GTD_TEST_CLASS_CAL=primary
export GTD_TEST_STUDY_CAL=primary

# Run integration tests
go test -tags=integration ./tests/...
```

### Writing Tests

Example unit test:
```go
func TestScheduler_AddTask(t *testing.T) {
    // Setup
    db, _ := setupTestDB(t)
    cal := &MockCalendarClient{}
    sched := New(db, cal, &config.Config{})

    // Execute
    err := sched.AddTask("Test task", "Test class", false)

    // Assert
    if err != nil {
        t.Errorf("AddTask failed: %v", err)
    }

    tasks, _ := db.GetTasksByClass("Test class")
    if len(tasks) != 1 {
        t.Errorf("Expected 1 task, got %d", len(tasks))
    }
}
```

## Debugging

### Debug Logging

Add debug prints:
```go
fmt.Printf("[DEBUG] Variable: %+v\n", myVariable)
```

### Debugging TUI

The TUI captures stdout, so use a debug file:
```go
f, _ := os.OpenFile("/tmp/gtd-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
fmt.Fprintf(f, "Debug: %+v\n", data)
```

### Using Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug ./cmd/gtd -- dashboard

# Set breakpoint
(dlv) break internal/scheduler/scheduler.go:42
(dlv) continue
```

## Common Tasks

### Adding a New CLI Command

1. Edit `cmd/gtd/main.go`
2. Define a new cobra.Command
3. Add it to rootCmd in init()

Example:
```go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(myCmd)
}
```

### Adding a New Pane to TUI

1. Add pane to FocusPane enum in `ui/dashboard.go`
2. Update tab navigation logic
3. Add rendering function in `ui/panes.go`
4. Update layout in `View()` method

### Adding Database Fields

1. Update struct in `db/db.go`
2. Update Save/Get methods
3. Handle migration for existing data
4. Add tests

### Changing Calendar Sync Logic

1. Edit `calendar/client.go`
2. Update `GetEvents()` or related methods
3. Update sync function in `ui/dashboard.go`
4. Test with real calendar

## Code Style

### Go Conventions

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Document exported functions
- Keep functions small (<50 lines ideally)
- Use table-driven tests

### Naming

- **Packages**: lowercase, single word (e.g., `scheduler`)
- **Files**: lowercase with underscores (e.g., `dashboard_test.go`)
- **Types**: PascalCase (e.g., `TodoItem`)
- **Functions**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to save task: %w", err)
}

// Check errors immediately
result, err := doSomething()
if err != nil {
    return err
}
```

## Performance

### Profiling

```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

### Optimization Tips

- Avoid allocations in hot paths
- Use sync.Pool for frequently allocated objects
- Batch database writes
- Cache calendar API responses
- Debounce UI updates

## Troubleshooting

### "go: command not found"

Install Go or add it to PATH:
```bash
export PATH=$PATH:/usr/local/go/bin
```

### "database is locked"

Only one process can access BoltDB at a time:
```bash
# Find process
lsof ~/.config/gtd/gtd.db

# Kill it
killall gtd
```

### "failed to authenticate"

Delete tokens and re-auth:
```bash
rm ~/.config/gtd/tokens.json
./bin/gtd dashboard
```

### Build Errors

```bash
# Clean and rebuild
make clean
go clean -cache
make build
```

## Release Process

1. **Update version**
   - Edit `cmd/gtd/main.go` version constant
   - Update CHANGELOG.md

2. **Tag release**
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

3. **Build binaries**
   ```bash
   # Linux amd64
   GOOS=linux GOARCH=amd64 go build -o bin/gtd-linux-amd64 ./cmd/gtd
   
   # macOS amd64
   GOOS=darwin GOARCH=amd64 go build -o bin/gtd-darwin-amd64 ./cmd/gtd
   
   # macOS arm64
   GOOS=darwin GOARCH=arm64 go build -o bin/gtd-darwin-arm64 ./cmd/gtd
   ```

4. **Create GitHub release**
   - Attach binaries
   - Include changelog

## Resources

- [Bubble Tea Docs](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss Examples](https://github.com/charmbracelet/lipgloss/tree/master/examples)
- [Google Calendar API Reference](https://developers.google.com/calendar/api/v3/reference)
- [BoltDB Documentation](https://github.com/etcd-io/bbolt)
- [Cobra User Guide](https://github.com/spf13/cobra/blob/master/user_guide.md)

## Getting Help

- Check existing GitHub issues
- Read the documentation in `docs/`
- Ask in discussions
- Review code examples in `tests/fixtures/`

## Tips for Contributors

1. **Start small** - Fix a typo, improve docs, add a test
2. **Read existing code** - Understand patterns before changing
3. **Ask questions** - Open an issue or discussion
4. **Test thoroughly** - Run tests, try edge cases
5. **Document changes** - Update README if behavior changes
6. **Keep PRs focused** - One feature/fix per PR

Happy coding! 🚀
