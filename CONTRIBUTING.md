# Contributing to Notion TUI

Thank you for considering contributing to Notion TUI! We welcome contributions from the community and are excited to see what you'll build.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Guidelines](#coding-guidelines)
- [Testing Requirements](#testing-requirements)
- [Submitting Changes](#submitting-changes)
- [Project Structure](#project-structure)
- [Communication](#communication)

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code. Please be respectful, inclusive, and considerate of others.

**Core Principles:**
- Be respectful and welcoming to all contributors
- Focus on constructive feedback
- Assume good intentions
- Respect differing viewpoints and experiences

## Getting Started

### Prerequisites

- **Go 1.21 or later** - [Download Go](https://golang.org/dl/)
- **Git** - [Download Git](https://git-scm.com/downloads)
- **A Notion account** with integration token for testing
- **golangci-lint** (optional but recommended) - [Installation](https://golangci-lint.run/usage/install/)

### First-Time Contributors

If you're new to open source or Go, we recommend:

1. Start with issues labeled `good first issue` or `help wanted`
2. Read through existing code to understand the architecture
3. Ask questions in GitHub Discussions if anything is unclear
4. Make small, focused changes to start

## Development Setup

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/notion-tui.git
cd notion-tui

# Add upstream remote
git remote add upstream https://github.com/Panandika/notion-tui.git
```

### 2. Install Dependencies

```bash
# Download Go modules
go mod download

# (Optional) Install development tools
make install-tools

# Or manually:
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install gotest.tools/gotestsum@latest
```

### 3. Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your Notion credentials
# NOTION_TUI_NOTION_TOKEN=secret_xxxxx
# NOTION_TUI_DATABASE_ID=xxxxx
```

**Security Note:** Never commit `.env` with real credentials!

### 4. Build and Run

```bash
# Build the application
go build -o notion-tui main.go

# Run
./notion-tui

# Or use go run for development
go run main.go
```

### 5. Run Tests

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check existing issues to avoid duplicates
2. Verify the bug with the latest version
3. Collect debug logs (`NOTION_TUI_DEBUG=true notion-tui`)

**Bug Report Template:**
```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce:
1. Go to '...'
2. Click on '...'
3. See error

**Expected behavior**
What you expected to happen.

**Screenshots/Logs**
If applicable, add screenshots or debug logs.

**Environment:**
- OS: [e.g., macOS 14.0, Ubuntu 22.04, Windows 11]
- Notion TUI version: [e.g., v1.0.0]
- Go version: [e.g., 1.21.0]
```

### Suggesting Features

We love new ideas! Before suggesting a feature:
1. Check if it's already been proposed
2. Consider if it fits the project's scope
3. Think about implementation complexity

**Feature Request Template:**
```markdown
**Problem Statement**
What problem does this feature solve?

**Proposed Solution**
How would you like to see this implemented?

**Alternatives Considered**
What other approaches did you consider?

**Additional Context**
Any mockups, examples, or references?
```

### Contributing Code

1. **Find or Create an Issue**
   - Comment on an issue to claim it
   - Get feedback on your approach before coding

2. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/bug-description
   ```

3. **Make Your Changes**
   - Follow coding guidelines (see below)
   - Add tests for new functionality
   - Update documentation as needed

4. **Test Your Changes**
   ```bash
   # Run tests
   go test ./...

   # Run linters
   go fmt ./...
   go vet ./...
   golangci-lint run
   ```

5. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add amazing feature"
   ```

6. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   # Then create a Pull Request on GitHub
   ```

## Coding Guidelines

We follow the guidelines outlined in [CLAUDE.md](CLAUDE.md). Key principles:

### Go Best Practices

**MUST Follow (Enforced by CI):**

- **CS-1:** Use `gofmt` and `go vet`
- **CS-2:** Avoid stutter in names (e.g., `package cache; type Cache` not `CacheCache`)
- **CS-5:** Use input structs for functions with >2 parameters
- **ERR-1:** Wrap errors with context: `fmt.Errorf("action: %w", err)`
- **ERR-2:** Use `errors.Is`/`errors.As` for control flow
- **CC-1:** Only senders close channels
- **CC-2:** Tie goroutine lifetime to `context.Context`
- **CC-3:** Protect shared state with proper synchronization
- **CTX-1:** Context must be first parameter
- **T-1:** Use table-driven tests
- **T-2:** Run tests with `-race` flag
- **API-1:** Document all exported items

**SHOULD Follow (Strong Recommendations):**

- **CS-3:** Small interfaces near consumers
- **CS-4:** Avoid reflection on hot paths
- **PERF-2:** Avoid allocations on hot paths
- **CFG-2:** Treat config as immutable after init

### Code Style

```go
// Good: Clear, documented, testable
type FetchPageInput struct {
    PageID    string
    UseCache  bool
    Timeout   time.Duration
}

func (c *Client) FetchPage(ctx context.Context, input FetchPageInput) (*Page, error) {
    if input.PageID == "" {
        return nil, fmt.Errorf("page ID is required")
    }

    // Implementation...
}

// Bad: Too many parameters, no context
func (c *Client) FetchPage(pageID string, cache bool, timeout int) *Page {
    // Implementation...
}
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation only
- `style:` Code style (formatting, missing semi colons, etc.)
- `refactor:` Code change that neither fixes a bug nor adds a feature
- `perf:` Performance improvement
- `test:` Adding or updating tests
- `chore:` Maintenance tasks

**Examples:**
```
feat(editor): add block type transformation shortcuts

fix(cache): prevent race condition in cache writes

docs(readme): update installation instructions

test(notion): add tests for rate limiter
```

## Testing Requirements

### Unit Tests

- **Location:** `*_test.go` files alongside source code
- **Coverage:** Aim for >70% coverage for new code
- **Pattern:** Use table-driven tests

```go
func TestFetchPage(t *testing.T) {
    tests := []struct {
        name    string
        input   FetchPageInput
        want    *Page
        wantErr bool
    }{
        {
            name: "valid page ID",
            input: FetchPageInput{PageID: "valid-id"},
            want: &Page{ID: "valid-id"},
            wantErr: false,
        },
        {
            name: "empty page ID",
            input: FetchPageInput{PageID: ""},
            want: nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FetchPage(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FetchPage() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // Assert on got vs tt.want
        })
    }
}
```

### Integration Tests

- Use mocks for external dependencies (Notion API)
- Test realistic workflows end-to-end
- Use `httptest` for HTTP mocking

### TUI Component Tests

- Use `teatest` from Bubble Tea for component testing
- Test keyboard input sequences
- Verify view output with golden files

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/notion

# With verbose output
go test -v ./...

# With race detection
go test -race ./...

# With coverage
go test -cover ./...
```

## Submitting Changes

### Pull Request Process

1. **Update Documentation**
   - Update README.md if adding features
   - Add/update comments for exported functions
   - Update CLAUDE.md if changing architecture

2. **Ensure Tests Pass**
   ```bash
   go test -race ./...
   golangci-lint run
   ```

3. **Create Pull Request**
   - Use a clear, descriptive title
   - Reference related issues (e.g., "Fixes #123")
   - Describe what changed and why
   - Add screenshots for UI changes

4. **Respond to Feedback**
   - Address reviewer comments
   - Push additional commits to the same branch
   - Mark conversations as resolved

### Pull Request Template

```markdown
## Description
Brief description of changes.

## Related Issue
Fixes #123

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings generated
- [ ] Tests pass locally
- [ ] Dependent changes merged
```

## Project Structure

Understanding the codebase structure:

```
notion-tui/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command and config
│   └── version.go         # Version command
├── internal/
│   ├── config/            # Configuration management
│   ├── notion/            # Notion API client
│   │   ├── client.go      # HTTP client with rate limiting
│   │   └── models.go      # Data structures
│   ├── cache/             # File-based cache
│   ├── version/           # Version information
│   └── ui/                # TUI implementation
│       ├── model.go       # Root app model
│       ├── update.go      # Update logic (Elm architecture)
│       ├── view.go        # View rendering
│       ├── keys.go        # Key bindings
│       ├── navigation.go  # Page routing
│       ├── components/    # Reusable UI components
│       │   ├── sidebar.go
│       │   ├── editor.go
│       │   └── viewer.go
│       └── pages/         # Page implementations
│           ├── list.go    # Database page list
│           ├── detail.go  # Page viewer
│           ├── edit.go    # Page editor
│           ├── search.go  # Search interface
│           └── dblist.go  # Database selector
├── e2e/                   # End-to-end tests
│   └── tapes/            # VHS tape recordings
├── testdata/              # Test fixtures
├── main.go                # Entry point
└── go.mod                 # Dependencies
```

### Key Packages

- **cmd:** CLI interface using Cobra
- **internal/config:** Viper-based configuration
- **internal/notion:** Notion API client with rate limiting
- **internal/cache:** Local file cache for offline support
- **internal/ui:** Bubble Tea TUI components
- **internal/ui/pages:** Full-screen page views
- **internal/ui/components:** Reusable widgets

### Architecture Patterns

- **Elm Architecture:** Model-View-Update via Bubble Tea
- **Command Pattern:** All I/O operations return `tea.Cmd`
- **Interface-Driven:** Mock `notionapi.Client` for testing
- **Dependency Injection:** Pass dependencies via constructors

## Communication

### GitHub Discussions

For questions, ideas, and general discussion:
- [GitHub Discussions](https://github.com/Panandika/notion-tui/discussions)

### GitHub Issues

For bug reports and feature requests:
- [GitHub Issues](https://github.com/Panandika/notion-tui/issues)

### Pull Requests

For code reviews and contributions:
- [GitHub Pull Requests](https://github.com/Panandika/notion-tui/pulls)

### Response Times

- We aim to respond to issues within 48 hours
- Pull requests are typically reviewed within a week
- Complex changes may take longer

## Recognition

Contributors will be:
- Listed in release notes
- Mentioned in the README
- Credited in commit messages

Thank you for contributing to Notion TUI!

---

**Questions?** Don't hesitate to ask in [GitHub Discussions](https://github.com/Panandika/notion-tui/discussions) or comment on an issue.
