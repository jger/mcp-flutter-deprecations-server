package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jger/mcp-flutter-deprecations-server/internal/handlers"
	"github.com/jger/mcp-flutter-deprecations-server/internal/services"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func main() {
	// Parse command line flags
	update := flag.Bool("update", false, "Update the Flutter deprecations cache and exit")
	updateShort := flag.Bool("u", false, "Update the Flutter deprecations cache and exit (short)")
	clearCache := flag.Bool("clear-cache", false, "Clear the Flutter deprecations cache and exit")
	clearCacheShort := flag.Bool("cc", false, "Clear the Flutter deprecations cache and exit (short)")
	showCache := flag.Bool("show-cache", false, "Display the current Flutter deprecations cache and exit")
	showCacheShort := flag.Bool("sc", false, "Display the current Flutter deprecations cache and exit (short)")
	help := flag.Bool("help", false, "Show help information")
	helpShort := flag.Bool("h", false, "Show help information (short)")
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

	// Handle help flag
	if *help || *helpShort {
		fmt.Println("Flutter Deprecations MCP Server")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  server [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --update, -u       Update the Flutter deprecations cache and exit")
		fmt.Println("  --clear-cache, -cc Clear the Flutter deprecations cache and exit")
		fmt.Println("  --show-cache, -sc  Display the current Flutter deprecations cache and exit")
		fmt.Println("  --help, -h         Show this help information")
		fmt.Println("  --vvv              Enable verbose logging")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  server             Start the MCP server")
		fmt.Println("  server -u          Update deprecations cache")
		fmt.Println("  server -cc         Clear deprecations cache")
		fmt.Println("  server -sc         Show current cache contents")
		fmt.Println("  server --vvv       Start with verbose logging")
		return
	}

	// Handle clear cache flag
	if *clearCache || *clearCacheShort {
		fmt.Println("🗑️ Clearing Flutter deprecations cache...")

		if err := cacheService.Clear(); err != nil {
			fmt.Printf("❌ Error clearing deprecations cache: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Successfully cleared deprecations cache")
		return
	}

	// Handle show cache flag
	if *showCache || *showCacheShort {
		fmt.Println("📋 Flutter Deprecations Cache Contents")
		fmt.Println("=" + strings.Repeat("=", 40))

		cache, err := cacheService.Load()
		if err != nil {
			fmt.Printf("❌ Error loading deprecations cache: %v\n", err)
			fmt.Println("💡 Try running with --update to create the cache first")
			os.Exit(1)
		}

		if len(cache.Deprecations) == 0 {
			fmt.Println("📭 No deprecations found in cache")
			fmt.Println("💡 Try running with --update to populate the cache")
			return
		}

		fmt.Printf("📊 Cache Info:\n")
		fmt.Printf("  Last Updated: %s\n", cache.LastUpdated.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Total Deprecations: %d\n", len(cache.Deprecations))
		fmt.Println()

		// Group deprecations by API name for better display
		fmt.Printf("📜 Deprecations:\n")
		for i, dep := range cache.Deprecations {
			fmt.Printf("%d. 🔴 %s\n", i+1, dep.API)
			if dep.Description != "" {
				fmt.Printf("   📝 Description: %s\n", dep.Description)
			}
			if dep.Replacement != "" {
				fmt.Printf("   ✅ Replacement: %s\n", dep.Replacement)
			}
			if dep.Version != "" {
				fmt.Printf("   📅 Since version: %s\n", dep.Version)
			}
			if dep.Example != "" {
				fmt.Printf("   💡 Example: %s\n", dep.Example)
			}
			fmt.Println()
		}

		fmt.Printf("✨ Total: %d deprecations found\n", len(cache.Deprecations))
		return
	}

	// Handle update flag
	if *update || *updateShort {
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
