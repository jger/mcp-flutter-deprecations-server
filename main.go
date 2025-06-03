package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

type FlutterRelease struct {
	Name        string `json:"name"`
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	Prerelease  bool   `json:"prerelease"`
}

type Deprecation struct {
	API         string `json:"api"`
	Replacement string `json:"replacement"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

type DeprecationCache struct {
	LastUpdated  time.Time     `json:"last_updated"`
	Deprecations []Deprecation `json:"deprecations"`
}

type CheckCodeArgs struct {
	Code string `json:"code"`
}

type FlutterVersionInfo struct {
	LatestVersion    string `json:"latest_version"`
	FVMInstalled     bool   `json:"fvm_installed"`
	FVMVersionExists bool   `json:"fvm_version_exists"`
	DockerImages     struct {
		Instrumentisto bool `json:"instrumentisto"`
		Cirrusci       bool `json:"cirrusci"`
	} `json:"docker_images"`
	Details string `json:"details"`
}

type NoArguments struct{}

const (
	CACHE_FILE      = "flutter_deprecations.json"
	CACHE_DURATION  = 24 * time.Hour
	FLUTTER_API_URL = "https://api.github.com/repos/flutter/flutter/releases"
)

var deprecationPatterns = map[string]Deprecation{
	`Color\.\w+\.withOpacity\(([^)]+)\)`: {
		API:         "Color.withOpacity",
		Replacement: "Color.withValues(alpha: $1)",
		Description: "withOpacity is deprecated, use withValues instead",
		Example:     "Color.red.withOpacity(0.5) → Color.red.withValues(alpha: 0.5)",
	},
	`RaisedButton`: {
		API:         "RaisedButton",
		Replacement: "ElevatedButton",
		Description: "RaisedButton is deprecated, use ElevatedButton instead",
		Example:     "RaisedButton → ElevatedButton",
	},
	`FlatButton`: {
		API:         "FlatButton",
		Replacement: "TextButton",
		Description: "FlatButton is deprecated, use TextButton instead",
		Example:     "FlatButton → TextButton",
	},
	`OutlineButton`: {
		API:         "OutlineButton",
		Replacement: "OutlinedButton",
		Description: "OutlineButton is deprecated, use OutlinedButton instead",
		Example:     "OutlineButton → OutlinedButton",
	},
	`Scaffold\.of\(context\)\.showSnackBar`: {
		API:         "Scaffold.of(context).showSnackBar",
		Replacement: "ScaffoldMessenger.of(context).showSnackBar",
		Description: "Direct showSnackBar on Scaffold is deprecated",
		Example:     "Scaffold.of(context).showSnackBar → ScaffoldMessenger.of(context).showSnackBar",
	},
	`FloatingActionButton\(child:`: {
		API:         "FloatingActionButton(child:",
		Replacement: "FloatingActionButton with specific constructors",
		Description: "Consider using FloatingActionButton.extended or other specific constructors",
	},
}

func getCacheDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".flutter-deprecations")
}

func ensureCacheDir() error {
	return os.MkdirAll(getCacheDir(), 0755)
}

func loadCache() (*DeprecationCache, error) {
	cachePath := filepath.Join(getCacheDir(), CACHE_FILE)
	
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return &DeprecationCache{Deprecations: []Deprecation{}}, nil
	}

	data, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache DeprecationCache
	err = json.Unmarshal(data, &cache)
	if err != nil {
		return &DeprecationCache{Deprecations: []Deprecation{}}, nil
	}

	return &cache, nil
}

func saveCache(cache *DeprecationCache) error {
	if err := ensureCacheDir(); err != nil {
		return err
	}

	cachePath := filepath.Join(getCacheDir(), CACHE_FILE)
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cachePath, data, 0644)
}

func fetchFlutterReleases() ([]FlutterRelease, error) {
	// Get more releases to ensure we find stable ones
	resp, err := http.Get(FLUTTER_API_URL + "?per_page=100")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []FlutterRelease
	err = json.Unmarshal(body, &releases)
	if err != nil {
		return nil, err
	}

	// GitHub returns releases sorted by creation date, but we want by tag version
	// Filter and sort stable releases properly
	sort.Slice(releases, func(i, j int) bool {
		// Sort by published date, newest first
		timeI, errI := time.Parse(time.RFC3339, releases[i].PublishedAt)
		timeJ, errJ := time.Parse(time.RFC3339, releases[j].PublishedAt)
		if errI != nil || errJ != nil {
			return false
		}
		return timeI.After(timeJ)
	})

	return releases, nil
}

func parseVersionFromRelease(release FlutterRelease) string {
	version := strings.TrimPrefix(release.TagName, "v")
	return version
}

func isVersionFromLast18Months(publishedAt string) bool {
	publishTime, err := time.Parse(time.RFC3339, publishedAt)
	if err != nil {
		return false
	}
	
	cutoff := time.Now().AddDate(0, -18, 0)
	return publishTime.After(cutoff)
}

func extractDeprecationsFromReleaseNotes(releases []FlutterRelease) []Deprecation {
	var deprecations []Deprecation
	
	// More specific patterns for real Flutter API deprecations
	patterns := []string{
		`(?i)deprecated[:\s]+([A-Z][a-zA-Z0-9_.]*)\s*(?:in favor of|replaced by|use)\s+([A-Z][a-zA-Z0-9_.]*)`,
		`(?i)([A-Z][a-zA-Z0-9_.]*)\s+(?:is\s+)?deprecated[,\s]*(?:use|replaced by)\s+([A-Z][a-zA-Z0-9_.]*)`,
		`(?i)\*\*Breaking change\*\*[^*]*deprecated[^*]*([A-Z][a-zA-Z0-9_.]*)[^*]*([A-Z][a-zA-Z0-9_.]*)?`,
	}

	for _, release := range releases {
		if !isVersionFromLast18Months(release.PublishedAt) {
			continue
		}

		version := parseVersionFromRelease(release)
		body := release.Body

		for _, pattern := range patterns {
			regex := regexp.MustCompile(pattern)
			matches := regex.FindAllStringSubmatch(body, -1)
			for _, match := range matches {
				if len(match) >= 2 {
					api := strings.TrimSpace(match[1])
					replacement := ""
					if len(match) >= 3 && match[2] != "" {
						replacement = strings.TrimSpace(match[2])
					}

					// Filter out obviously wrong matches
					if len(api) < 3 || !strings.Contains(api, ".") && len(api) < 5 {
						continue
					}

					deprecation := Deprecation{
						API:         api,
						Replacement: replacement,
						Version:     version,
						Description: fmt.Sprintf("Deprecated in Flutter %s", version),
					}
					deprecations = append(deprecations, deprecation)
				}
			}
		}
	}

	// Add the known deprecation patterns
	for _, templateDep := range deprecationPatterns {
		dep := templateDep
		dep.Version = "Multiple versions"
		deprecations = append(deprecations, dep)
	}

	return deprecations
}

func updateDeprecationsCache() error {
	cache, err := loadCache()
	if err != nil {
		return err
	}

	if time.Since(cache.LastUpdated) < CACHE_DURATION {
		return nil
	}

	releases, err := fetchFlutterReleases()
	if err != nil {
		return err
	}

	deprecations := extractDeprecationsFromReleaseNotes(releases)
	
	cache.Deprecations = deprecations
	cache.LastUpdated = time.Now()

	return saveCache(cache)
}

func checkCodeForDeprecations(code string) []Deprecation {
	var foundDeprecations []Deprecation

	for regexPattern, deprecation := range deprecationPatterns {
		regex := regexp.MustCompile(regexPattern)
		if regex.MatchString(code) {
			foundDeprecations = append(foundDeprecations, deprecation)
		}
	}

	cache, err := loadCache()
	if err == nil {
		for _, dep := range cache.Deprecations {
			if dep.API != "" && strings.Contains(code, dep.API) {
				foundDeprecations = append(foundDeprecations, dep)
			}
		}
	}

	return foundDeprecations
}

func getLatestFlutterVersion() (string, error) {
	releases, err := fetchFlutterReleases()
	if err != nil {
		return "", err
	}

	// First try to find a stable release
	for _, release := range releases {
		tagLower := strings.ToLower(release.TagName)
		version := parseVersionFromRelease(release)
		
		// Check if this is a stable release
		isStable := !release.Prerelease &&
			!strings.Contains(tagLower, "beta") && 
			!strings.Contains(tagLower, "dev") && 
			!strings.Contains(tagLower, "pre") &&
			!strings.Contains(tagLower, "rc") &&
			!strings.Contains(tagLower, "alpha") &&
			!strings.Contains(tagLower, "hotfix") &&
			// Check for version patterns like "3.19.0-0.1.pre"
			!strings.Contains(version, "-") &&
			// Stable versions should match pattern like "3.32.0"
			regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(version)
			
		if isStable {
			return version, nil
		}
	}
	
	// If no stable release found, return the latest release regardless
	if len(releases) > 0 {
		return parseVersionFromRelease(releases[0]), nil
	}
	
	return "", fmt.Errorf("no releases found")
}

func checkFVMInstalled() bool {
	cmd := exec.Command("fvm", "--version")
	return cmd.Run() == nil
}

func checkFVMVersionExists(version string) bool {
	if !checkFVMInstalled() {
		return false
	}
	
	cmd := exec.Command("fvm", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	return strings.Contains(string(output), version)
}

func checkDockerImageExists(image string, tag string) bool {
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags/%s", image, tag)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == 200
}

func getFlutterVersionInfo() (*FlutterVersionInfo, error) {
	// Force fresh fetch from GitHub API
	releases, err := fetchFlutterReleases()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Flutter releases from GitHub: %v", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no Flutter releases found")
	}

	// Find latest stable version
	var latestVersion string
	for _, release := range releases {
		tagLower := strings.ToLower(release.TagName)
		version := parseVersionFromRelease(release)
		
		// Check if this is a stable release
		isStable := !release.Prerelease &&
			!strings.Contains(tagLower, "beta") && 
			!strings.Contains(tagLower, "dev") && 
			!strings.Contains(tagLower, "pre") &&
			!strings.Contains(tagLower, "rc") &&
			!strings.Contains(tagLower, "alpha") &&
			!strings.Contains(tagLower, "hotfix") &&
			!strings.Contains(version, "-") &&
			regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(version)
			
		if isStable {
			latestVersion = version
			break
		}
	}

	// If no stable found, use the most recent release
	if latestVersion == "" {
		latestVersion = parseVersionFromRelease(releases[0])
	}

	info := &FlutterVersionInfo{
		LatestVersion: latestVersion,
		FVMInstalled:  checkFVMInstalled(),
	}

	if info.FVMInstalled {
		info.FVMVersionExists = checkFVMVersionExists(latestVersion)
	}

	// Check Docker images availability
	info.DockerImages.Instrumentisto = checkDockerImageExists("instrumentisto/flutter", latestVersion)
	info.DockerImages.Cirrusci = checkDockerImageExists("cirrusci/flutter", latestVersion)

	// Build details string
	details := fmt.Sprintf("Latest Flutter Version: %s\n\n", latestVersion)
	
	if info.FVMInstalled {
		details += "FVM Status: ✅ Installed\n"
		if info.FVMVersionExists {
			details += fmt.Sprintf("  - Version %s: ✅ Available locally\n", latestVersion)
		} else {
			details += fmt.Sprintf("  - Version %s: ❌ Not installed locally\n", latestVersion)
			details += fmt.Sprintf("  - Install with: fvm install %s\n", latestVersion)
		}
	} else {
		details += "FVM Status: ❌ Not installed\n"
		details += "  - Install FVM: https://fvm.app/docs/getting_started/installation\n"
	}

	details += "\nDocker Images:\n"
	if info.DockerImages.Instrumentisto {
		details += fmt.Sprintf("  - instrumentisto/flutter:%s ✅ Available\n", latestVersion)
	} else {
		details += fmt.Sprintf("  - instrumentisto/flutter:%s ❌ Not available\n", latestVersion)
	}
	
	if info.DockerImages.Cirrusci {
		details += fmt.Sprintf("  - cirrusci/flutter:%s ✅ Available\n", latestVersion)
	} else {
		details += fmt.Sprintf("  - cirrusci/flutter:%s ❌ Not available\n", latestVersion)
	}

	details += "\nUsage Examples:\n"
	if info.FVMInstalled {
		details += fmt.Sprintf("  - FVM: fvm use %s\n", latestVersion)
	}
	details += fmt.Sprintf("  - Docker (instrumentisto): docker run -it instrumentisto/flutter:%s\n", latestVersion)
	details += fmt.Sprintf("  - Docker (cirrusci): docker run -it cirrusci/flutter:%s\n", latestVersion)

	// Add debug info about releases found
	details += fmt.Sprintf("\n--- Debug Info ---\n")
	details += fmt.Sprintf("Total releases fetched: %d\n", len(releases))
	if len(releases) > 0 {
		details += fmt.Sprintf("Most recent release: %s (prerelease: %v)\n", releases[0].TagName, releases[0].Prerelease)
	}
	if len(releases) > 1 {
		details += fmt.Sprintf("Second release: %s (prerelease: %v)\n", releases[1].TagName, releases[1].Prerelease)
	}

	info.Details = details
	return info, nil
}

func main() {
	done := make(chan struct{})

	if err := updateDeprecationsCache(); err != nil {
		fmt.Printf("Warning: Failed to update deprecations cache: %v\n", err)
	}

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	err := server.RegisterTool(
		"check_flutter_deprecations",
		"Check Flutter code for deprecated APIs and get suggestions for replacements. Provide the code snippet to analyze.",
		func(args CheckCodeArgs) (*mcp_golang.ToolResponse, error) {
			deprecations := checkCodeForDeprecations(args.Code)
			
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
		})
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"list_flutter_deprecations",
		"Get a list of all known Flutter deprecations from the cache. Optionally filter by version or API name.",
		func(args NoArguments) (*mcp_golang.ToolResponse, error) {
			cache, err := loadCache()
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
		})
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"update_flutter_deprecations",
		"Manually update the Flutter deprecations cache by fetching the latest release information from GitHub.",
		func(args NoArguments) (*mcp_golang.ToolResponse, error) {
			err := updateDeprecationsCache()
			if err != nil {
				return mcp_golang.NewToolResponse(
					mcp_golang.NewTextContent(fmt.Sprintf("Error updating deprecations cache: %v", err)),
				), nil
			}

			cache, err := loadCache()
			if err != nil {
				return mcp_golang.NewToolResponse(
					mcp_golang.NewTextContent("Cache updated but failed to load for verification."),
				), nil
			}

			return mcp_golang.NewToolResponse(
				mcp_golang.NewTextContent(fmt.Sprintf("Successfully updated deprecations cache. Found %d deprecations. Last updated: %s", 
					len(cache.Deprecations), cache.LastUpdated.Format("2006-01-02 15:04:05"))),
			), nil
		})
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"check_flutter_version_info",
		"Get the latest Flutter version and check availability in FVM and Docker images (instrumentisto/flutter and cirrusci/flutter).",
		func(args NoArguments) (*mcp_golang.ToolResponse, error) {
			info, err := getFlutterVersionInfo()
			if err != nil {
				return mcp_golang.NewToolResponse(
					mcp_golang.NewTextContent(fmt.Sprintf("Error getting Flutter version info: %v", err)),
				), nil
			}

			return mcp_golang.NewToolResponse(
				mcp_golang.NewTextContent(info.Details),
			), nil
		})
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