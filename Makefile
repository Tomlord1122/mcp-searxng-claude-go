.PHONY: help build run test clean docker-build docker-up docker-down docker-logs install deps

# Default target
help:
	@echo "MCP SearXNG Go - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make deps          - Download Go dependencies"
	@echo "  make build         - Build the Go binary"
	@echo "  make run           - Run the server locally (requires SEARXNG_URL)"
	@echo "  make test          - Run tests"
	@echo "  make clean         - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build  - Build Docker images"
	@echo "  make docker-up     - Start Docker containers"
	@echo "  make docker-down   - Stop Docker containers"
	@echo "  make docker-logs   - View Docker logs"
	@echo "  make docker-clean  - Remove containers and volumes"
	@echo ""
	@echo "Quick Start:"
	@echo "  make install       - Install executable to /usr/local/bin"
	@echo "  make start         - Start the MCP server (via script)"
	@echo "  make stop          - Stop the MCP server"

# Go build
deps:
	go mod download
	go mod tidy

build:
	go build -o mcp-searxng-go .

run: build
	@if [ -z "$(SEARXNG_URL)" ]; then \
		echo "Error: SEARXNG_URL not set"; \
		echo "Usage: SEARXNG_URL=http://localhost:8080 make run"; \
		exit 1; \
	fi
	./mcp-searxng-go

test:
	go test -v ./...

clean:
	rm -f mcp-searxng-go
	rm -rf searxng/
	go clean

# Docker operations
docker-build:
	docker compose build --no-cache

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-clean:
	docker compose down -v
	rm -rf searxng/

# Installation
install: build
	@echo "Installing mcp-searxng-go to /usr/local/bin..."
	@sudo cp mcp-searxng-go /usr/local/bin/
	@echo "Installation complete! Run 'mcp-searxng-go' from anywhere."

# Quick start/stop
start:
	./start-mcp.sh

stop:
	./stop-mcp.sh

# Development with hot reload (requires entr)
watch:
	@echo "Watching Go files for changes..."
	@echo "Install entr if needed: brew install entr (macOS) or apt-get install entr (Linux)"
	@find . -name "*.go" | entr -r sh -c 'make build && echo "Rebuild complete!"'

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Lint (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: brew install golangci-lint"; exit 1)
	golangci-lint run

# Generate go.sum
tidy:
	go mod tidy

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/mcp-searxng-go-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o dist/mcp-searxng-go-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/mcp-searxng-go-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/mcp-searxng-go-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o dist/mcp-searxng-go-windows-amd64.exe .
	@echo "Binaries built in dist/ directory"

# Check for updates
update:
	go get -u ./...
	go mod tidy

# Run with race detector
run-race: build
	@if [ -z "$(SEARXNG_URL)" ]; then \
		echo "Error: SEARXNG_URL not set"; \
		exit 1; \
	fi
	go run -race .

# Generate coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Quick setup
setup: deps
	@echo "Generating .env file..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "SEARXNG_SECRET=$$(openssl rand -hex 16)" >> .env; \
		echo ".env file created!"; \
	else \
		echo ".env file already exists"; \
	fi
	@chmod +x start-mcp.sh stop-mcp.sh
	@echo "Setup complete! Run 'make start' to begin."
