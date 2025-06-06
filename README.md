# Flutter Deprecations MCP Server

This MCP server tracks Flutter API deprecations and provides tools to check code for deprecated APIs and suggest replacements.

## Project Structure

```
flutter-deprecations-server/
├── cmd/
│   └── server/          # Main application entry point
│       └── main.go
├── internal/            # Private application code
│   ├── handlers/        # MCP tool handlers
│   │   ├── mcp_handlers.go
│   │   ├── mcp_handlers_test.go
│   │   └── testdata/
│   ├── models/          # Data structures
│   │   └── flutter.go
│   └── services/        # Business logic
│       ├── cache.go
│       ├── cache_test.go
│       ├── deprecations.go
│       ├── deprecations_test.go
│       ├── flutter_api.go
│       ├── flutter_api_test.go
│       ├── flutter_version.go
│       ├── interfaces.go
│       ├── version_info.go
│       ├── version_info_test.go
│       └── testdata/
├── pkg/                 # Public libraries
│   └── config/          # Configuration constants
│       └── config.go
├── bin/                 # Compiled binaries
├── Makefile            # Build automation
├── go.mod              # Go module definition
├── go.sum              # Dependency checksums
└── README.md
```

## Features

- **Source-based deprecation tracking**: Directly scans Flutter's GitHub source code for `@Deprecated` annotations
- **Local caching**: Stores deprecations locally with 24-hour cache duration
- **Code analysis**: Analyzes Flutter code snippets for deprecated APIs
- **Replacement suggestions**: Provides modern alternatives for deprecated APIs
- **Comprehensive scanning**: Scans key Flutter directories (widgets, material, cupertino, services, etc.)
- **Version checking**: Gets latest Flutter version using Flutter CLI (most reliable) with GitHub API fallback
- **Multi-platform support**: Checks FVM and Docker image availability
- **Command-line cache management**: Manual cache updates and clearing with progress reporting
- **Short command options**: Support for both long and short command flags
- **Rate limit handling**: Graceful handling of GitHub API rate limits with helpful error messages
- **Verbose logging**: Detailed logging with `-vvv` flag for troubleshooting

## MCP Tools

### 1. `check_flutter_deprecations`
Analyzes provided Flutter code for deprecated APIs and suggests replacements.

**Parameters:**
- `code` (string): Flutter code snippet to analyze

**Example:**
```dart
Color.red.withOpacity(0.5)  // Will suggest Color.red.withValues(alpha: 0.5)
```

### 2. `list_flutter_deprecations`
Lists all known Flutter deprecations from the cache.

**Parameters:** None

**Returns:** Complete list of deprecations with replacements and version information.

### 3. `check_flutter_version_info`
Gets the latest stable Flutter version and checks availability across different tools and platforms.

**Parameters:** None

**Returns:**
- Latest stable Flutter version (using Flutter CLI when available, GitHub API fallback)
- Flutter CLI installation status and channel information
- FVM installation status and version availability
- Docker image availability for `instrumentisto/flutter` and `ghcr.io/cirruslabs/flutter`
- Usage examples and installation commands

## Known Deprecations

The server includes built-in patterns for common deprecations:

- `Color.withOpacity()` → `Color.withValues(alpha:)`
- `RaisedButton` → `ElevatedButton`
- `FlatButton` → `TextButton`
- `OutlineButton` → `OutlinedButton`
- `Scaffold.of(context).showSnackBar` → `ScaffoldMessenger.of(context).showSnackBar`

## Installation

### Using Makefile (Recommended)

```bash
# Install dependencies
make deps

# Build the server
make build

# Run the server
make run

# Build for multiple platforms
make build-all
```

### Manual Build

```bash
# Install dependencies
go mod tidy

# Build the server
go build -o bin/flutter-deprecations-server ./cmd/server

# Run the server
./bin/flutter-deprecations-server

# Update deprecations cache manually
./bin/flutter-deprecations-server --update

# Update with verbose logging  
./bin/flutter-deprecations-server --update -vvv

# Clear deprecations cache
./bin/flutter-deprecations-server --clear-cache

# Show current cache contents
./bin/flutter-deprecations-server --show-cache

# Show help information
./bin/flutter-deprecations-server --help
```

## Development

```bash
# Format code
make fmt

# Vet code
make vet

# Run tests
make test

# Development build and run
make dev
```

## Configuration with AI Assistants

Add to your MCP configuration (e.g., `mcp.json`):

```json
{
  "mcpServers": {
    "flutter-deprecations": {
      "command": "/path/to/flutter-deprecations-server/bin/flutter-deprecations-server",
      "args": [],
      "env": {}
    }
  }
}
```

## Version Detection

The server uses a reliable multi-tier approach to detect the latest Flutter version:

1. **Primary**: Flutter CLI (`flutter --version`) - Most accurate, matches developer environment
2. **Fallback**: GitHub API releases - Used when Flutter CLI not available
3. **Channel Detection**: Identifies stable/beta/dev channels
4. **Docker Registry Support**: Checks both Docker Hub and GitHub Container Registry

## Cache Location

Deprecations are cached at: `~/.flutter-deprecations/flutter_deprecations.json`

The cache is automatically updated every 24 hours when tools are used.

## Usage Examples

Ask your AI assistant:
- "Check this Flutter code for deprecations: `Color.red.withOpacity(0.5)`"
- "List all Flutter deprecations"
- "What should I use instead of RaisedButton?"
- "What's the latest Flutter version and is it available in FVM and Docker?"
- "Check Flutter version info"

## Command Line Usage

The server supports several command-line options for cache management:

```bash
# Show help and available options
./bin/flutter-deprecations-server --help
./bin/flutter-deprecations-server -h        # Short version

# Update deprecations cache with progress reporting
./bin/flutter-deprecations-server --update
./bin/flutter-deprecations-server -u        # Short version

# Update with verbose logging for troubleshooting
./bin/flutter-deprecations-server --update -vvv
./bin/flutter-deprecations-server -u -vvv   # Short version

# Clear deprecations cache
./bin/flutter-deprecations-server --clear-cache
./bin/flutter-deprecations-server -cc       # Short version

# Show current cache contents
./bin/flutter-deprecations-server --show-cache
./bin/flutter-deprecations-server -sc       # Short version

# Start the MCP server (default behavior)
./bin/flutter-deprecations-server
```

### Available Options

- `--help, -h`: Show help information with usage examples
- `--update, -u`: Update the Flutter deprecations cache and exit
- `--clear-cache, -cc`: Clear the Flutter deprecations cache and exit
- `--show-cache, -sc`: Display the current Flutter deprecations cache and exit
- `--vvv`: Enable verbose logging for detailed troubleshooting

## Architecture

The project follows Go best practices with a clean architecture:

- **cmd/**: Application entry points
- **internal/**: Private application code (cannot be imported by other projects)
- **pkg/**: Public libraries that can be imported
- **Makefile**: Build automation and common tasks

### Services Layer

- **CacheService**: Handles local file caching with clear functionality
- **FlutterAPIService**: Manages GitHub API interactions with rate limit handling, source code scanning, and Docker registry checks
- **FlutterVersionService**: Gets Flutter version directly from Flutter CLI
- **DeprecationService**: Analyzes and manages deprecation data from Flutter source code
- **VersionInfoService**: Provides comprehensive version and availability information

### Handlers Layer

- **MCPHandlers**: Implements MCP tool interfaces and coordinates service calls

### Testing

- **Comprehensive test suite**: Unit tests for all services and handlers
- **Mock services**: Test infrastructure with dependency injection
- **Test coverage**: Run `make test-coverage` to generate coverage reports