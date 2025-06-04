package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/example/flutter-deprecations-server/internal/handlers"
	"github.com/example/flutter-deprecations-server/internal/services"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func main() {
	// Parse command line flags
	update := flag.Bool("update", false, "Update the Flutter deprecations cache and exit")
	verbose := flag.Bool("vvv", false, "Enable verbose logging")
	flag.Parse()

	// Configure logging based on verbose flag
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Verbose logging enabled")
	} else {
		// Disable logging for normal operation
		log.SetOutput(os.Stderr)
	}

	done := make(chan struct{})

	// Initialize services
	cacheService := services.NewCacheService()
	apiService := services.NewFlutterAPIService()
	deprecationService := services.NewDeprecationService(cacheService, apiService)
	versionInfoService := services.NewVersionInfoService(apiService)

	// Handle update flag
	if *update {
		fmt.Println("🔄 Updating Flutter deprecations cache...")
		
		// Create a progress callback
		progressCallback := func(message string) {
			fmt.Printf("  %s\n", message)
		}
		
		if err := deprecationService.UpdateCacheWithProgress(progressCallback, *verbose); err != nil {
			fmt.Printf("❌ Error updating deprecations cache: %v\n", err)
			os.Exit(1)
		}
		
		cache, err := cacheService.Load()
		if err != nil {
			fmt.Printf("❌ Cache updated but failed to load for verification: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("✅ Successfully updated deprecations cache. Found %d deprecations. Last updated: %s\n",
			len(cache.Deprecations), cache.LastUpdated.Format("2006-01-02 15:04:05"))
		return
	}

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