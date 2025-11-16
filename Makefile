.PHONY: help build run test test-race lint fmt vet cover clean

BINARY_NAME=notion-tui
GO_FILES=$(shell find . -name '*.go' -type f)
GOFLAGS=-trimpath
LDFLAGS=-ldflags "-X main.version=dev"

help:
	@echo "notion-tui development tasks:"
	@echo "  make build         - Build the binary"
	@echo "  make run           - Run the application"
	@echo "  make test          - Run unit tests"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make cover         - Generate coverage report"
	@echo "  make lint          - Run linters (golangci-lint, vet, fmt)"
	@echo "  make fmt           - Format code with gofmt"
	@echo "  make vet           - Run go vet"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make install-tools - Install dev tools (golangci-lint, etc)"

install-tools:
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing gofumpt..."
	go install mvdan.cc/gofumpt@latest
	@echo "Installing gotestsum..."
	go install gotest.tools/gotestsum@latest
	@echo "Installing mockgen..."
	go install github.com/golang/mock/mockgen@latest

build: fmt vet
	@echo "Building ${BINARY_NAME}..."
	go build ${GOFLAGS} ${LDFLAGS} -o ${BINARY_NAME} ./cmd/main.go

run: build
	@echo "Running ${BINARY_NAME}..."
	./${BINARY_NAME}

test:
	@echo "Running tests..."
	go test -v -cover ./...

test-race:
	@echo "Running tests with race detector..."
	go test -v -race ./...

cover:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: fmt vet golangci-lint-check

fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofumpt -w .

vet:
	@echo "Running go vet..."
	go vet ./...

golangci-lint-check:
	@echo "Running golangci-lint..."
	golangci-lint run ./...

clean:
	@echo "Cleaning up..."
	rm -f ${BINARY_NAME}
	rm -f coverage.out coverage.html
	rm -f debug.log
	go clean -testcache
