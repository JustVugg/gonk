.PHONY: help build test run clean docker lint fmt

# Variables
BINARY_NAME=gonk
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"
DOCKER_IMAGE=gonk:${VERSION}

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	@go build ${LDFLAGS} -o bin/${BINARY_NAME} cmd/gonk/main.go
	@echo "Build complete: bin/${BINARY_NAME}"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

run: ## Run with example config
	@go run cmd/gonk/main.go -config configs/gonk.example.yaml

dev: ## Run in development mode with hot reload
	@air -c .air.toml

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ dist/ coverage.out coverage.html

docker: ## Build Docker image
	@echo "Building Docker image ${DOCKER_IMAGE}..."
	@docker build -t ${DOCKER_IMAGE} -f deployments/docker/Dockerfile .

docker-run: docker ## Run Docker container
	@docker run -p 8080:8080 -v $(PWD)/configs/gonk.example.yaml:/etc/gonk/gonk.yaml ${DOCKER_IMAGE}

lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

install: build ## Install binary to $GOPATH/bin
	@echo "Installing ${BINARY_NAME}..."
	@cp bin/${BINARY_NAME} $(GOPATH)/bin/

# Cross-compilation
build-linux: ## Build for Linux
	@GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 cmd/gonk/main.go

build-darwin: ## Build for macOS
	@GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 cmd/gonk/main.go

build-windows: ## Build for Windows
	@GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-windows-amd64.exe cmd/gonk/main.go

build-arm: ## Build for ARM (Raspberry Pi)
	@GOOS=linux GOARCH=arm GOARM=7 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-arm cmd/gonk/main.go

build-all: build-linux build-darwin build-windows build-arm ## Build for all platforms

release: build-all ## Create release artifacts
	@mkdir -p dist
	@tar czf dist/${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz -C bin ${BINARY_NAME}-linux-amd64
	@tar czf dist/${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz -C bin ${BINARY_NAME}-darwin-amd64
	@zip dist/${BINARY_NAME}-${VERSION}-windows-amd64.zip bin/${BINARY_NAME}-windows-amd64.exe
	@tar czf dist/${BINARY_NAME}-${VERSION}-linux-arm.tar.gz -C bin ${BINARY_NAME}-linux-arm
	@echo "Release artifacts created in dist/"