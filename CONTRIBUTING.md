# Contributing to GTD

Thank you for your interest in contributing! This document provides guidelines for contributing to the GTD project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/gtd`
3. Create a branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `make test`
6. Run linter: `make lint`
7. Commit your changes: `git commit -am 'Add some feature'`
8. Push to the branch: `git push origin feature/your-feature-name`
9. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Make
- golangci-lint (for linting)

### Building

```bash
make build
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests
make test-integration

# With coverage
go test -cover ./...
```

### Linting

```bash
make lint
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting (automatically run by `make fmt`)
- Write clear, self-documenting code
- Add comments for exported functions
- Keep functions small and focused

## Testing Guidelines

- Write unit tests for new functionality
- Maintain or improve code coverage
- Use table-driven tests where appropriate
- Mock external dependencies (Calendar API, etc.)

Example:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"case 1", "input1", "output1", false},
        {"case 2", "input2", "output2", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := YourFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Commit Messages

Follow conventional commits format:

- `feat: add new feature`
- `fix: resolve bug`
- `docs: update documentation`
- `test: add tests`
- `refactor: restructure code`
- `chore: maintenance tasks`

## Pull Request Process

1. Update the README.md with details of changes if needed
2. Update documentation in docs/ if you're changing behavior
3. Add tests for new functionality
4. Ensure all tests pass and linter is happy
5. Request review from maintainers
6. Address review feedback
7. Once approved, your PR will be merged

## Bug Reports

When reporting bugs, please include:

- OS and version
- Go version
- Steps to reproduce
- Expected behavior
- Actual behavior
- Relevant logs or screenshots

Use the GitHub issue template.

## Feature Requests

We welcome feature requests! Please:

- Check if it's already been requested
- Clearly describe the use case
- Explain why it would be useful
- Consider submitting a PR yourself!

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Assume good intentions

## Questions?

- Open an issue with the "question" label
- Check existing issues and documentation first

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
