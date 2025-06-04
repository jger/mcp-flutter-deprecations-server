package handlers

import (
	"fmt"
	"sort"

	"github.com/example/flutter-deprecations-server/internal/models"
	"github.com/example/flutter-deprecations-server/internal/services"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

// MCPHandlers contains all MCP tool handlers
type MCPHandlers struct {
	deprecationService  services.DeprecationServiceInterface
	versionInfoService  services.VersionInfoServiceInterface
	cacheService        services.CacheServiceInterface
}

// NewMCPHandlers creates a new MCP handlers instance
func NewMCPHandlers(deprecationService services.DeprecationServiceInterface, versionInfoService services.VersionInfoServiceInterface, cacheService services.CacheServiceInterface) *MCPHandlers {
	return &MCPHandlers{
		deprecationService: deprecationService,
		versionInfoService: versionInfoService,
		cacheService:       cacheService,
	}
}

// CheckFlutterDeprecations handles the check_flutter_deprecations tool
func (h *MCPHandlers) CheckFlutterDeprecations(args models.CheckCodeArgs) (*mcp_golang.ToolResponse, error) {
	deprecations := h.deprecationService.CheckCodeForDeprecations(args.Code)
	
	if len(deprecations) == 0 {
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent("No deprecated APIs found in the provided code."),
		), nil
	}

	result := "Found deprecated APIs:\n\n"
	for i, dep := range deprecations {
		result += fmt.Sprintf("%d. **%s**\n", i+1, dep.API)
		if dep.Replacement != "" {
			result += fmt.Sprintf("   - Replacement: %s\n", dep.Replacement)
		}
		result += fmt.Sprintf("   - Description: %s\n", dep.Description)
		if dep.Example != "" {
			result += fmt.Sprintf("   - Example: %s\n", dep.Example)
		}
		if dep.Version != "" {
			result += fmt.Sprintf("   - Since version: %s\n", dep.Version)
		}
		result += "\n"
	}

	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(result),
	), nil
}

// ListFlutterDeprecations handles the list_flutter_deprecations tool
func (h *MCPHandlers) ListFlutterDeprecations(args models.NoArguments) (*mcp_golang.ToolResponse, error) {
	cache, err := h.cacheService.Load()
	if err != nil {
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent(fmt.Sprintf("Error loading deprecations: %v", err)),
		), nil
	}

	if len(cache.Deprecations) == 0 {
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent("No deprecations found in cache. Try updating the cache first."),
		), nil
	}

	result := fmt.Sprintf("Flutter Deprecations (Last updated: %s)\n\n", cache.LastUpdated.Format("2006-01-02 15:04:05"))
	
	sort.Slice(cache.Deprecations, func(i, j int) bool {
		return cache.Deprecations[i].API < cache.Deprecations[j].API
	})

	for i, dep := range cache.Deprecations {
		result += fmt.Sprintf("%d. **%s**\n", i+1, dep.API)
		if dep.Replacement != "" {
			result += fmt.Sprintf("   - Replacement: %s\n", dep.Replacement)
		}
		result += fmt.Sprintf("   - Description: %s\n", dep.Description)
		if dep.Example != "" {
			result += fmt.Sprintf("   - Example: %s\n", dep.Example)
		}
		if dep.Version != "" {
			result += fmt.Sprintf("   - Since version: %s\n", dep.Version)
		}
		result += "\n"
	}

	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(result),
	), nil
}


// CheckFlutterVersionInfo handles the check_flutter_version_info tool
func (h *MCPHandlers) CheckFlutterVersionInfo(args models.NoArguments) (*mcp_golang.ToolResponse, error) {
	info, err := h.versionInfoService.GetFlutterVersionInfo()
	if err != nil {
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent(fmt.Sprintf("Error getting Flutter version info: %v", err)),
		), nil
	}

	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(info.Details),
	), nil
}