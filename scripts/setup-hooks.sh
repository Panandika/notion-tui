#!/bin/bash
# Setup Git pre-commit hooks for development

set -e

HOOK_DIR=".git/hooks"
HOOK_FILE="$HOOK_DIR/pre-commit"

echo "Setting up Git pre-commit hooks..."

# Create hooks directory if it doesn't exist
mkdir -p "$HOOK_DIR"

# Create pre-commit hook
cat > "$HOOK_FILE" << 'EOF'
#!/bin/bash
# Pre-commit hook for notion-tui
# Runs formatting, linting, and tests before commit

set -e

echo " pre-commit checks..."
Running
# Get list of staged Go files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -n "$STAGED_GO_FILES" ]; then
    echo "Formatting code..."
    echo "$STAGED_GO_FILES" | xargs gofmt -w
    echo "$STAGED_GO_FILES" | xargs git add

    echo "Running go vet..."
    go vet ./...

    echo "Running tests..."
    go test -race -short ./...
fi

echo "Pre-commit checks passed!"
EOF

# Make the hook executable
chmod +x "$HOOK_FILE"

echo "✓ Pre-commit hook installed at $HOOK_FILE"
echo ""
echo "The hook will run on every commit to check:"
echo "  1. Code formatting (gofmt)"
echo "  2. Linting (go vet)"
echo "  3. Tests (go test -race -short ./...)"
echo ""
echo "To skip pre-commit hooks, use: git commit --no-verify"
