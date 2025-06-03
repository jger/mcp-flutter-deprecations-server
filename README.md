# Flutter Deprecations MCP Server

This MCP server tracks Flutter API deprecations and provides tools to check code for deprecated APIs and suggest replacements.

## Features

- **Automatic deprecation tracking**: Fetches Flutter releases from GitHub and extracts deprecation information
- **Local caching**: Stores deprecations locally with 24-hour cache duration
- **Code analysis**: Analyzes Flutter code snippets for deprecated APIs
- **Replacement suggestions**: Provides modern alternatives for deprecated APIs
- **Historical data**: Tracks deprecations from the last 1.5 years

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

## Known Deprecations

The server includes built-in patterns for common deprecations:

- `Color.withOpacity()` → `Color.withValues(alpha:)`
- `RaisedButton` → `ElevatedButton`
- `FlatButton` → `TextButton`
- `OutlineButton` → `OutlinedButton`
- `Scaffold.of(context).showSnackBar` → `ScaffoldMessenger.of(context).showSnackBar`

## Installation

1. **Build the server:**
   ```bash
   cd flutter-deprecations-server
   go mod tidy
   go build -o flutter-deprecations-server
   ```

2. **Run the server:**
   ```bash
   ./flutter-deprecations-server
   ```

## Configuration with AI Assistants

Add to your MCP configuration (e.g., `mcp.json`):

```json
{
  "mcpServers": {
    "flutter-deprecations": {
      "command": "/path/to/flutter-deprecations-server",
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