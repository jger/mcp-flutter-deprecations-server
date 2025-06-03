# Flutter Deprecations MCP Server Makefile

.PHONY: build run clean test fmt vet

# Build the server
build:
	go build -o bin/flutter-deprecations-server ./cmd/server

# Run the server
run:
	go run ./cmd/server

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Install dependencies
deps:
	go mod tidy

# Build for multiple platforms
build-all:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/flutter-deprecations-server-linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o bin/flutter-deprecations-server-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build -o bin/flutter-deprecations-server-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build -o bin/flutter-deprecations-server-windows-amd64.exe ./cmd/server

# Development build and run
dev: build run