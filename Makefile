.PHONY: build build-server build-cli build-all clean test install docker-build help

VERSION := 1.1.0
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

# Default target
all: build

# Build both server and CLI
build: build-server build-cli

# Build server
build-server:
	@echo "Building GONK server..."
	@mkdir -p bin
	cd cmd/gonk && go build -ldflags "$(LDFLAGS)" -o ../../bin/gonk

# Build CLI
build-cli:
	@echo "Building GONK CLI..."
	@mkdir -p bin
	cd cmd/gonk-cli && go build -ldflags "$(LDFLAGS)" -o ../../bin/gonk-cli

# Build for all platforms (GitHub releases)
build-all:
	@echo "Building for all platforms..."
	@mkdir -p bin
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-linux-amd64 ./cmd/gonk
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-cli-linux-amd64 ./cmd/gonk-cli
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-darwin-amd64 ./cmd/gonk
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-cli-darwin-amd64 ./cmd/gonk-cli
	# macOS ARM64 (M1/M2)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-darwin-arm64 ./cmd/gonk
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-cli-darwin-arm64 ./cmd/gonk-cli
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-windows-amd64.exe ./cmd/gonk
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-cli-windows-amd64.exe ./cmd/gonk-cli
	# Linux ARM64 (Raspberry Pi 4, etc)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-linux-arm64 ./cmd/gonk
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/gonk-cli-linux-arm64 ./cmd/gonk-cli
	@echo "✅ All binaries built in bin/"
	@ls -lh bin/ 2>/dev/null || dir bin

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/ 2>/dev/null || rmdir /s /q bin 2>nul || echo "Clean complete"
	@go clean

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Install locally (Linux/macOS only)
install: build
	@echo "Installing GONK..."
	@cp bin/gonk /usr/local/bin/ || echo "Error: Use 'sudo make install' or install manually"
	@cp bin/gonk-cli /usr/local/bin/ || echo "Error: Use 'sudo make install' or install manually"
	@echo "✅ Installed to /usr/local/bin/"

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t gonk:$(VERSION) .
	@echo "✅ Docker image built: gonk:$(VERSION)"

# Show help
help:
	@echo "GONK v$(VERSION) Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build        - Build both server and CLI"
	@echo "  make build-server - Build server only"
	@echo "  make build-cli    - Build CLI only"
	@echo "  make build-all    - Build for all platforms (releases)"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install      - Install to /usr/local/bin (Linux/macOS)"
	@echo "  make docker-build - Build Docker image"
	@echo "  make help         - Show this help"
	@echo ""
	@echo "Note: On Windows, use Git Bash or WSL to run make commands"