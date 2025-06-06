# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based MCP (Model Context Protocol) server that tracks Flutter API deprecations by scanning Flutter's GitHub source code and provides tools to analyze Flutter code for deprecated APIs and suggest replacements.

## Development Commands

**Build and Run:**
```bash
make build          # Build the server binary
make run           # Run the server directly
make dev           # Build and run together
```

**Testing:**
```bash
make test          # Run all tests
make test-coverage # Run tests with coverage report (generates coverage.html)
```

**Code Quality:**
```bash
make fmt           # Format Go code
make vet           # Run Go vet analysis
make deps          # Install/update dependencies (go mod tidy)
```

**Binary Management:**
```bash
make clean         # Remove bin/ directory
make build-all     # Cross-compile for multiple platforms (linux, darwin, windows)
```

**Cache Operations:**
```bash
# Update deprecations cache
./bin/flutter-deprecations-server --update
./bin/flutter-deprecations-server -u -vvv  # With verbose logging

# Show current cache contents
./bin/flutter-deprecations-server --show-cache
./bin/flutter-deprecations-server -sc      # Short version

# Clear cache
./bin/flutter-deprecations-server --clear-cache
```

## Architecture

**Core Services (Dependency Injection Pattern):**
- `CacheService`: Local file-based caching with 24-hour TTL at `~/.flutter-deprecations/`
- `FlutterAPIService`: GitHub API client with rate limiting, source code scanning, Docker registry checks
- `DeprecationService`: Analysis engine that scans Flutter source for `@Deprecated` annotations
- `VersionInfoService`: Multi-tier Flutter version detection (CLI first, GitHub API fallback)
- `FlutterVersionService`: Direct Flutter CLI integration for version/channel detection

**Handler Layer:**
- `MCPHandlers`: Implements MCP protocol tools, coordinates service calls with dependency injection

**Data Models:**
- All models in `internal/models/flutter.go` define API contracts
- Uses interfaces in `internal/services/interfaces.go` for testability

**MCP Tools Provided:**
1. `check_flutter_deprecations` - Analyzes code snippets
2. `list_flutter_deprecations` - Lists all cached deprecations  
3. `check_flutter_version_info` - Version/availability checking

## Key Implementation Details

**Source Code Scanning:** Directly parses Flutter's GitHub repository source files for `@Deprecated` annotations rather than relying on release notes, providing more comprehensive coverage.

**Rate Limiting:** GitHub API calls include exponential backoff and graceful degradation when rate limits are hit.

**Multi-Platform Version Detection:** Checks Flutter CLI, FVM, and Docker image availability across multiple registries (Docker Hub, GitHub Container Registry).

**Caching Strategy:** 24-hour local cache with manual update/clear commands. Cache automatically refreshes on first tool use if stale.

## Testing

**Test Structure:**
- Each service has corresponding `*_test.go` files
- Uses dependency injection with interfaces for easy mocking
- Testdata in `internal/handlers/testdata/` and `internal/services/testdata/`
- Run single test: `go test -v ./internal/services -run TestSpecificFunction`

**Module:** `github.com/jger/mcp-flutter-deprecations-server` (Go 1.24.3)
**Main Dependency:** `github.com/metoro-io/mcp-golang v0.13.0`