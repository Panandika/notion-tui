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
