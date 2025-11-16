# Prerequisites Checklist

Before starting development on notion-tui, complete the following steps:

## 📦 Project Setup

### Step 1: Clone Repository
```bash
git clone https://github.com/Panandika/notion-tui.git
cd notion-tui
```

### Step 2: Verify Go Installation
```bash
go version  # Should show go1.25.4 or later
go env GOPATH
```

### Step 3: Download Dependencies
```bash
go mod download
go mod tidy  # Optional: clean up unused dependencies
```

### Step 4: (Optional but Recommended) Install Dev Tools
```bash
make install-tools
```

This installs:
- `golangci-lint` - Comprehensive Go linter
- `gofumpt` - Strict Go formatter
- `gotestsum` - Better test output
- `mockgen` - Mock code generator

If `make install-tools` fails, you can install individually:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install gotest.tools/gotestsum@latest
go install github.com/golang/mock/mockgen@latest
```

### Step 5: Set Up Environment Variables
```bash
# Copy template
cp .env.example .env

# Edit .env and add your Notion API token
# Get token from: https://www.notion.so/profile/integrations
# Token format: secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**Important:** Never commit `.env` with real credentials!

### Step 6: (Optional) Set Up Git Hooks
```bash
bash scripts/setup-hooks.sh
```

This ensures code quality checks run before each commit.

### Step 7: Verify Setup
```bash
# Format check
go fmt ./...

# Lint check
go vet ./...

# Run tests (if any exist)
go test ./...

# Build the project
make build
```

## 📋 Configuration Files Created

| File | Purpose |
|------|---------|
| `CLAUDE.md` | Development guidelines and architecture |
| `SETUP.md` | Detailed setup instructions |
| `Makefile` | Common development commands |
| `.golangci.yml` | Linter configuration |
| `.env.example` | Environment variable template |
| `.pre-commit-config.yaml` | Git hook configuration |
| `.github/workflows/test.yml` | CI test workflow |
| `.github/workflows/release.yml` | Release automation |
| `.goreleaser.yaml` | Release build configuration |
| `scripts/setup-hooks.sh` | Hook installation script |

## ✅ Pre-Development Checklist

Before writing code, confirm:

- [ ] Go 1.25.4+ is installed (`go version`)
- [ ] Dependencies downloaded (`go mod download`)
- [ ] `.env` file created with valid `NOTION_TOKEN`
- [ ] No linting errors (`go vet ./...`)
- [ ] Code formats correctly (`go fmt ./...`)
- [ ] Tests pass (if any exist) (`go test ./...`)
- [ ] Build succeeds (`make build`)
- [ ] (Optional) Git hooks installed (`bash scripts/setup-hooks.sh`)

## 🚦 Ready to Code

Once all checklist items are complete, you're ready to start development!

### Quick Reference
```bash
# Daily development commands
make build          # Build binary
make test           # Run tests
make test-race      # Run with race detector
make lint           # Full linting
make fmt            # Format code
make cover          # Coverage report
```

### Next Steps
1. Read `CLAUDE.md` for architecture and standards
2. Review `plan/claude_convo.md` for implementation roadmap
3. Check out Phase 1 (Foundation) in the roadmap
4. Start with `cmd/main.go`

## 🔐 Security Notes

- **Never** commit `.env` file to Git (it's in `.gitignore`)
- **Always** use environment variables for sensitive data
- Notion tokens start with `secret_`
- See `plan/claude_convo.md` section 1 for secure credential storage options

## 🆘 Troubleshooting

### "go: command not found"
- Ensure Go is installed: `go version`
- Add Go to PATH: See https://go.dev/doc/install

### "Permission denied" on scripts
```bash
chmod +x scripts/*.sh
```

### Make command not found
- Install GNU Make
- Or use commands directly (e.g., `go test ./...`)

### golangci-lint installation fails
```bash
# Try with explicit version
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

# Or download binary from: https://github.com/golangci/golangci-lint/releases
```

### NOTION_TOKEN validation error
- Verify token format: `secret_` prefix
- Get new token from: https://www.notion.so/profile/integrations
- Ensure .env file is in project root
- Check that .env is not committed to Git

## 📖 Further Reading

- **Go Best Practices**: https://go.dev/doc/effective_go
- **Bubble Tea Framework**: https://github.com/charmbracelet/bubbletea
- **jomei/notionapi**: https://github.com/jomei/notionapi
- **Notion API Docs**: https://developers.notion.com/reference

---

**Status**: All prerequisites configured ✅

Start development when all checklist items are complete!