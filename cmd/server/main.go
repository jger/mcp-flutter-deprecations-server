package main

import (
	"fmt"

	"github.com/example/flutter-deprecations-server/internal/handlers"
	"github.com/example/flutter-deprecations-server/internal/services"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func main() {
	done := make(chan struct{})

	// Initialize services
	cacheService := services.NewCacheService()
	apiService := services.NewFlutterAPIService()
	deprecationService := services.NewDeprecationService(cacheService, apiService)
	versionInfoService := services.NewVersionInfoService(apiService)

	// Initialize handlers
	mcpHandlers := handlers.NewMCPHandlers(deprecationService, versionInfoService, cacheService)

	// Initialize MCP server
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	// Update deprecations cache on startup
	if err := deprecationService.UpdateCache(); err != nil {
		fmt.Printf("Warning: Failed to update deprecations cache: %v\n", err)
	}

	// Register MCP tools
	err := server.RegisterTool(
		"check_flutter_deprecations",
		"Check Flutter code for deprecated APIs and get suggestions for replacements. Provide the code snippet to analyze.",
		mcpHandlers.CheckFlutterDeprecations)
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"list_flutter_deprecations",
		"Get a list of all known Flutter deprecations from the cache. Optionally filter by version or API name.",
		mcpHandlers.ListFlutterDeprecations)
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"update_flutter_deprecations",
		"Manually update the Flutter deprecations cache by fetching the latest release information from GitHub.",
		mcpHandlers.UpdateFlutterDeprecations)
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"check_flutter_version_info",
		"Get the latest Flutter version and check availability in FVM and Docker images (instrumentisto/flutter and cirrusci/flutter).",
		mcpHandlers.CheckFlutterVersionInfo)
	if err != nil {
		panic(err)
	}

	fmt.Println("Flutter Deprecations MCP Server started. Waiting for requests...")
	err = server.Serve()
	if err != nil {
		panic(err)
	}

	<-done
}