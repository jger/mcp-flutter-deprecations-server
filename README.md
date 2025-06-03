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
│   │   └── mcp_handlers.go
│   ├── models/          # Data structures
│   │   └── flutter.go
│   └── services/        # Business logic
│       ├── cache.go
│       ├── deprecations.go
│       ├── flutter_api.go
│       └── version_info.go
├── pkg/                 # Public libraries
│   └── config/          # Configuration constants
│       └── config.go
├── bin/                 # Compiled binaries
├── Makefile            # Build automation
├── go.mod              # Go module definition
└── README.md
```

## Features

- **Automatic deprecation tracking**: Fetches Flutter releases from GitHub and extracts deprecation information
- **Local caching**: Stores deprecations locally with 24-hour cache duration
- **Code analysis**: Analyzes Flutter code snippets for deprecated APIs
- **Replacement suggestions**: Provides modern alternatives for deprecated APIs
- **Historical data**: Tracks deprecations from the last 1.5 years
- **Version checking**: Gets latest Flutter version and checks FVM/Docker availability

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

### 3. `update_flutter_deprecations`
Manually updates the deprecations cache by fetching the latest Flutter releases.

**Parameters:** None

### 4. `check_flutter_version_info`
Gets the latest stable Flutter version and checks availability across different tools and platforms.

**Parameters:** None

**Returns:**
- Latest stable Flutter version
- FVM installation status and version availability
- Docker image availability for `instrumentisto/flutter` and `cirrusci/flutter`
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

## Cache Location

Deprecations are cached at: `~/.flutter-deprecations/flutter_deprecations.json`

The cache is automatically updated every 24 hours when tools are used.

## Usage Examples

Ask your AI assistant:
- "Check this Flutter code for deprecations: `Color.red.withOpacity(0.5)`"
- "List all Flutter deprecations"
- "Update the Flutter deprecations cache"
- "What should I use instead of RaisedButton?"
- "What's the latest Flutter version and is it available in FVM and Docker?"
- "Check Flutter version info"

## Architecture

The project follows Go best practices with a clean architecture:

- **cmd/**: Application entry points
- **internal/**: Private application code (cannot be imported by other projects)
- **pkg/**: Public libraries that can be imported
- **Makefile**: Build automation and common tasks

### Services Layer

- **CacheService**: Handles local file caching
- **FlutterAPIService**: Manages GitHub API interactions
- **DeprecationService**: Analyzes and manages deprecation data
- **VersionInfoService**: Provides version and availability information

### Handlers Layer

- **MCPHandlers**: Implements MCP tool interfaces and coordinates service calls