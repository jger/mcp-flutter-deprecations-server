package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jger/mcp-flutter-deprecations-server/internal/models"
	"github.com/jger/mcp-flutter-deprecations-server/pkg/config"
)

// FlutterAPIService handles Flutter API interactions
type FlutterAPIService struct{}

// NewFlutterAPIService creates a new Flutter API service instance
func NewFlutterAPIService() *FlutterAPIService {
	return &FlutterAPIService{}
}

// FetchReleases fetches Flutter releases from GitHub API
func (f *FlutterAPIService) FetchReleases() ([]models.FlutterRelease, error) {
	resp, err := http.Get(config.FLUTTER_API_URL + fmt.Sprintf("?per_page=%d", config.MAX_RELEASES))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == 403 || resp.StatusCode == 401 || resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		var errorResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errorResp) == nil && strings.Contains(errorResp.Message, "API rate limit exceeded") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Please wait before retrying or authenticate with a GitHub token")
		}
		return nil, fmt.Errorf("GitHub API access forbidden (403): %s", errorResp.Message)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []models.FlutterRelease
	err = json.Unmarshal(body, &releases)
	if err != nil {
		return nil, err
	}

	// Sort by published date, newest first
	sort.Slice(releases, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, releases[i].PublishedAt)
		timeJ, errJ := time.Parse(time.RFC3339, releases[j].PublishedAt)
		if errI != nil || errJ != nil {
			return false
		}
		return timeI.After(timeJ)
	})

	return releases, nil
}

// ParseVersionFromRelease extracts version string from release tag
func (f *FlutterAPIService) ParseVersionFromRelease(release models.FlutterRelease) string {
	version := strings.TrimPrefix(release.TagName, "v")
	return version
}

// GetLatestStableVersion finds the latest stable Flutter version
func (f *FlutterAPIService) GetLatestStableVersion() (string, error) {
	releases, err := f.FetchReleases()
	if err != nil {
		return "", err
	}

	// Find latest stable version
	for _, release := range releases {
		tagLower := strings.ToLower(release.TagName)
		version := f.ParseVersionFromRelease(release)

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
			return version, nil
		}
	}

	// If no stable release found, return the latest release regardless
	if len(releases) > 0 {
		return f.ParseVersionFromRelease(releases[0]), nil
	}

	return "", fmt.Errorf("no releases found")
}

// CheckFVMInstalled checks if FVM is installed on the system
func (f *FlutterAPIService) CheckFVMInstalled() bool {
	cmd := exec.Command("fvm", "--version")
	return cmd.Run() == nil
}

// CheckFVMVersionExists checks if a specific Flutter version exists in FVM
func (f *FlutterAPIService) CheckFVMVersionExists(version string) bool {
	if !f.CheckFVMInstalled() {
		return false
	}

	cmd := exec.Command("fvm", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), version)
}

// CheckDockerImageExists checks if a Docker image exists for a specific tag
func (f *FlutterAPIService) CheckDockerImageExists(image string, tag string) bool {
	// Handle different registries
	if strings.HasPrefix(image, "ghcr.io/") {
		// GitHub Container Registry
		return f.checkGHCRImageExists(image, tag)
	} else {
		// Docker Hub
		return f.checkDockerHubImageExists(image, tag)
	}
}

// checkDockerHubImageExists checks Docker Hub for image availability
func (f *FlutterAPIService) checkDockerHubImageExists(image string, tag string) bool {
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags/%s", image, tag)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// checkGHCRImageExists checks GitHub Container Registry for image availability
func (f *FlutterAPIService) checkGHCRImageExists(image string, tag string) bool {
	// For GHCR, we'll try to check the GitHub repository instead
	// ghcr.io/cirruslabs/flutter -> check cirruslabs/docker-images-flutter repo
	if image == "ghcr.io/cirruslabs/flutter" {
		// Check if the package exists in GitHub packages
		url := "https://api.github.com/users/cirruslabs/packages/container/flutter/versions"
		resp, err := http.Get(url)
		if err != nil {
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return false
		}

		// For simplicity, if the package exists, assume the tag exists
		// In a real implementation, you'd parse the JSON and check for the specific tag
		return true
	}

	return false
}

// FetchFlutterSourceDeprecations fetches @Deprecated annotations from Flutter source on GitHub
func (f *FlutterAPIService) FetchFlutterSourceDeprecations() ([]models.Deprecation, error) {
	// Base URL for Flutter source code on GitHub
	baseURL := "https://raw.githubusercontent.com/flutter/flutter/master/packages/flutter/lib/src/"

	// Key directories to search for deprecations
	directories := []string{
		"widgets/",
		"material/",
		"cupertino/",
		"services/",
		"rendering/",
		"foundation/",
		"painting/",
		"gestures/",
		"animation/",
	}

	var deprecations []models.Deprecation

	// For each directory, we'll fetch a directory listing and then scan files
	for _, dir := range directories {
		dirDeprecations, err := f.scanDirectoryForDeprecations(baseURL + dir)
		if err != nil {
			// Log error but continue with other directories
			fmt.Printf("Warning: Failed to scan directory %s: %v\n", dir, err)
			continue
		}
		deprecations = append(deprecations, dirDeprecations...)
	}

	return deprecations, nil
}

// scanDirectoryForDeprecations scans a directory for Dart files and extracts @Deprecated annotations
func (f *FlutterAPIService) scanDirectoryForDeprecations(baseURL string) ([]models.Deprecation, error) {
	// Since we can't easily list directory contents via GitHub raw URLs,
	// we'll use the GitHub API to get directory contents first
	apiURL := strings.Replace(baseURL, "https://raw.githubusercontent.com/", "https://api.github.com/repos/", 1)
	apiURL = strings.Replace(apiURL, "/master/", "/contents/", 1)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == 403 {
		body, _ := ioutil.ReadAll(resp.Body)
		var errorResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errorResp) == nil && strings.Contains(errorResp.Message, "API rate limit exceeded") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Please wait before retrying or authenticate with a GitHub token")
		}
		return nil, fmt.Errorf("GitHub API access forbidden (403): %s", errorResp.Message)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch directory listing: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var files []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(body, &files); err != nil {
		return nil, err
	}

	var deprecations []models.Deprecation

	// Process each Dart file
	for _, file := range files {
		if file.Type == "file" && strings.HasSuffix(file.Name, ".dart") {
			fileURL := baseURL + file.Name
			fileDeprecations, err := f.ScanFileForDeprecations(fileURL)
			if err != nil {
				fmt.Printf("Warning: Failed to scan file %s: %v\n", file.Name, err)
				continue
			}
			deprecations = append(deprecations, fileDeprecations...)
		}
	}

	return deprecations, nil
}

// ScanFileForDeprecations scans a single Dart file for @Deprecated annotations (exported for testing)
func (f *FlutterAPIService) ScanFileForDeprecations(fileURL string) ([]models.Deprecation, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch file: %d", resp.StatusCode)
	}

	var deprecations []models.Deprecation
	scanner := bufio.NewScanner(resp.Body)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Enhanced pattern matching for @Deprecated annotations
	deprecatedPattern := regexp.MustCompile(`@[Dd]eprecated\s*\(\s*['"](.+?)['"]`)
	
	// More comprehensive patterns for different Dart constructs
	classPattern := regexp.MustCompile(`(?:abstract\s+)?(?:class|enum|mixin)\s+(\w+)`)
	methodPattern := regexp.MustCompile(`(?:(?:static|final|const)\s+)*(?:[\w<>?]+\s+)?(\w+)\s*\(`)
	constructorPattern := regexp.MustCompile(`(\w+)\s*\.\s*(\w+)\s*\(`)
	propertyPattern := regexp.MustCompile(`(?:(?:static|final|const)\s+)*(?:[\w<>?]+\s+)+(get\s+)?(\w+)(?:\s*[;=]|\s*=>)`)
	getterPattern := regexp.MustCompile(`(?:[\w<>?]+\s+)?get\s+(\w+)\s*(?:=>|{)`)
	setterPattern := regexp.MustCompile(`set\s+(\w+)\s*\(`)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Look for @Deprecated annotation
		if matches := deprecatedPattern.FindStringSubmatch(line); len(matches) > 1 {
			description := matches[1]
			
			// Look ahead for the deprecated item (next few lines)
			var apiName string
			var className string
			
			// Get current class context by looking backward
			for j := i - 1; j >= 0 && j >= i-50; j-- {
				if classMatches := classPattern.FindStringSubmatch(strings.TrimSpace(lines[j])); len(classMatches) > 1 {
					className = classMatches[1]
					break
				}
			}
			
			// Look ahead for the deprecated item
			for j := i + 1; j < len(lines) && j <= i+10; j++ {
				nextLine := strings.TrimSpace(lines[j])
				
				// Skip empty lines, comments, and annotations
				if nextLine == "" || strings.HasPrefix(nextLine, "//") || 
				   strings.HasPrefix(nextLine, "/*") || strings.HasPrefix(nextLine, "@") {
					continue
				}

				// Try to match different constructs
				if matches := classPattern.FindStringSubmatch(nextLine); len(matches) > 1 {
					apiName = matches[1]
					break
				} else if matches := constructorPattern.FindStringSubmatch(nextLine); len(matches) > 2 {
					apiName = matches[1] + "." + matches[2]
					break
				} else if matches := getterPattern.FindStringSubmatch(nextLine); len(matches) > 1 {
					if className != "" {
						apiName = className + "." + matches[1]
					} else {
						apiName = matches[1]
					}
					break
				} else if matches := setterPattern.FindStringSubmatch(nextLine); len(matches) > 1 {
					if className != "" {
						apiName = className + "." + matches[1]
					} else {
						apiName = matches[1]
					}
					break
				} else if matches := methodPattern.FindStringSubmatch(nextLine); len(matches) > 1 {
					methodName := matches[1]
					// Filter out common non-method words
					if methodName != "if" && methodName != "for" && methodName != "while" && 
					   methodName != "switch" && methodName != "return" && methodName != "throw" {
						if className != "" && methodName != className {
							apiName = className + "." + methodName
						} else {
							apiName = methodName
						}
						break
					}
				} else if matches := propertyPattern.FindStringSubmatch(nextLine); len(matches) > 2 {
					propertyName := matches[2]
					if className != "" {
						apiName = className + "." + propertyName
					} else {
						apiName = propertyName
					}
					break
				}
			}

			if apiName != "" {
				deprecation := models.Deprecation{
					API:         apiName,
					Description: description,
				}

				// Enhanced replacement extraction
				replacement := f.extractReplacement(description)
				if replacement != "" {
					deprecation.Replacement = replacement
				}

				// Try to infer better replacement based on context
				if deprecation.Replacement == "" {
					deprecation.Replacement = f.InferReplacement(apiName, description)
				}

				deprecations = append(deprecations, deprecation)
			}
		}
	}

	return deprecations, nil
}

// extractReplacement tries to extract replacement suggestions from deprecation messages
func (f *FlutterAPIService) extractReplacement(description string) string {
	// Pattern 1: "Use X instead"
	useInsteadPattern := regexp.MustCompile(`(?i)use\s+([A-Za-z0-9_.()]+)(?:\s+instead)?`)
	if matches := useInsteadPattern.FindStringSubmatch(description); len(matches) > 1 {
		return matches[1]
	}
	
	// Pattern 2: "Replaced by X"
	replacedByPattern := regexp.MustCompile(`(?i)replaced\s+by\s+([A-Za-z0-9_.()]+)`)
	if matches := replacedByPattern.FindStringSubmatch(description); len(matches) > 1 {
		return matches[1]
	}
	
	// Pattern 3: "Use X() method"
	useMethodPattern := regexp.MustCompile(`(?i)use\s+(?:the\s+)?([A-Za-z0-9_.()]+)\s+method`)
	if matches := useMethodPattern.FindStringSubmatch(description); len(matches) > 1 {
		return matches[1]
	}
	
	// Pattern 4: "Prefer X"
	preferPattern := regexp.MustCompile(`(?i)prefer\s+([A-Za-z0-9_.()]+)`)
	if matches := preferPattern.FindStringSubmatch(description); len(matches) > 1 {
		return matches[1]
	}
	
	return ""
}

// InferReplacement tries to infer replacement based on common Flutter patterns and contextual analysis (exported for testing)
func (f *FlutterAPIService) InferReplacement(apiName, description string) string {
	desc := strings.ToLower(description)
	api := strings.ToLower(apiName)
	
	// Enhanced pattern-based replacements with contextual understanding
	patterns := map[string]string{
		// Color patterns
		"withopacity":        "withValues(alpha: value)",
		"color.withopacity":  "color.withValues(alpha: value)",
		
		// Button patterns  
		"raisedbutton":       "ElevatedButton",
		"flatbutton":         "TextButton", 
		"outlinebutton":      "OutlinedButton",
		"materialbutton":     "ElevatedButton, TextButton, or OutlinedButton",
		
		// Material patterns
		"floatingactionbutton.mini": "FloatingActionButton(mini: true)",
		
		// Navigator patterns
		"navigator.of(context).push": "Navigator.push(context, route)",
		"navigator.of(context).pop":  "Navigator.pop(context)",
		
		// Scaffold patterns
		"scaffold.of(context).showsnackbar": "ScaffoldMessenger.of(context).showSnackBar",
		
		// Text patterns
		"text.overflow":      "Text with overflow parameter",
		"textstyle.height":   "TextStyle.height or TextHeightBehavior",
		
		// Widget patterns
		"wrap.direction":     "Wrap.direction parameter",
		"flex.direction":     "Flex.direction parameter",
		
		// Animation patterns
		"animationcontroller.reset": "AnimationController.reset() alternative",
		"tween.animate":             "Tween.animate() or AnimatedBuilder",
		
		// Layout patterns
		"positioned.fill":    "Positioned.fill() constructor",
		"expanded.flex":      "Expanded(flex: value)",
		"flexible.flex":      "Flexible(flex: value)",
	}
	
	// Check direct API patterns
	for pattern, replacement := range patterns {
		if strings.Contains(api, pattern) {
			return replacement
		}
	}
	
	// Analyze description for contextual clues
	if strings.Contains(desc, "will lead to bugs") || strings.Contains(desc, "causes issues") {
		if strings.Contains(api, "jump") || strings.Contains(api, "scroll") {
			return "Use ScrollController methods or ScrollPosition alternatives"
		}
		return "Alternative implementation recommended - see Flutter documentation"
	}
	
	if strings.Contains(desc, "performance") {
		return "More efficient alternative available - check Flutter performance guide"
	}
	
	if strings.Contains(desc, "accessibility") {
		return "Use semantically improved alternative for better accessibility"
	}
	
	// Method-specific patterns
	if strings.HasSuffix(api, "withoutsettling") {
		return "Use standard navigation/animation methods that properly settle"
	}
	
	if strings.Contains(api, "copywidth") || strings.Contains(api, "copyheight") {
		return "Use copyWith() with specific dimension parameters"
	}
	
	// Generic fallbacks based on API type
	if strings.Contains(api, "button") {
		return "Use Material 3 button alternatives (ElevatedButton, TextButton, OutlinedButton)"
	}
	
	if strings.Contains(api, "color") {
		return "Use updated Color API with values() constructor"
	}
	
	if strings.Contains(api, "theme") {
		return "Use Material 3 ThemeData with updated color scheme"
	}
	
	return ""
}

// FetchFlutterSourceDeprecationsWithProgress fetches @Deprecated annotations with progress reporting
func (f *FlutterAPIService) FetchFlutterSourceDeprecationsWithProgress(progressCallback func(string), verbose bool) ([]models.Deprecation, error) {
	// Base URL for Flutter source code on GitHub
	baseURL := "https://raw.githubusercontent.com/flutter/flutter/master/packages/flutter/lib/src/"

	// Key directories to search for deprecations
	directories := []string{
		"widgets/",
		"material/",
		"cupertino/",
		"services/",
		"rendering/",
		"foundation/",
		"painting/",
		"gestures/",
		"animation/",
	}

	var deprecations []models.Deprecation

	// For each directory, we'll fetch a directory listing and then scan files
	for i, dir := range directories {
		progressCallback(fmt.Sprintf("📂 Scanning directory %d/%d: %s", i+1, len(directories), dir))
		if verbose {
			log.Printf("Scanning directory: %s", dir)
		}

		dirDeprecations, err := f.scanDirectoryForDeprecationsWithProgress(baseURL+dir, progressCallback, verbose)
		if err != nil {
			// Log error but continue with other directories
			if verbose {
				log.Printf("Warning: Failed to scan directory %s: %v", dir, err)
			}
			progressCallback(fmt.Sprintf("⚠️ Warning: Failed to scan directory %s", dir))
			continue
		}
		deprecations = append(deprecations, dirDeprecations...)

		if verbose {
			log.Printf("Found %d deprecations in directory %s", len(dirDeprecations), dir)
		}
	}

	progressCallback(fmt.Sprintf("✅ Completed scanning %d directories", len(directories)))
	return deprecations, nil
}

// scanDirectoryForDeprecationsWithProgress scans a directory with progress reporting
func (f *FlutterAPIService) scanDirectoryForDeprecationsWithProgress(baseURL string, progressCallback func(string), verbose bool) ([]models.Deprecation, error) {
	// Since we cannot easily list directory contents via GitHub raw URLs,
	// we'll use the GitHub API to get directory contents first
	apiURL := strings.Replace(baseURL, "https://raw.githubusercontent.com/", "https://api.github.com/repos/", 1)
	apiURL = strings.Replace(apiURL, "/master/", "/contents/", 1)

	if verbose {
		log.Printf("Fetching directory listing from: %s", apiURL)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == 403 {
		body, _ := ioutil.ReadAll(resp.Body)
		var errorResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errorResp) == nil && strings.Contains(errorResp.Message, "API rate limit exceeded") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Please wait before retrying or authenticate with a GitHub token")
		}
		return nil, fmt.Errorf("GitHub API access forbidden (403): %s", errorResp.Message)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch directory listing: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var files []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(body, &files); err != nil {
		return nil, err
	}

	var deprecations []models.Deprecation
	dartFiles := make([]string, 0)

	// Count Dart files first
	for _, file := range files {
		if file.Type == "file" && strings.HasSuffix(file.Name, ".dart") {
			dartFiles = append(dartFiles, file.Name)
		}
	}

	if len(dartFiles) > 0 {
		progressCallback(fmt.Sprintf("  📜 Found %d Dart files to scan", len(dartFiles)))
	}

	// Process each Dart file
	for i, fileName := range dartFiles {
		if verbose {
			log.Printf("Scanning file %d/%d: %s", i+1, len(dartFiles), fileName)
		}

		fileURL := baseURL + fileName
		fileDeprecations, err := f.ScanFileForDeprecations(fileURL)
		if err != nil {
			if verbose {
				log.Printf("Warning: Failed to scan file %s: %v", fileName, err)
			}
			continue
		}
		deprecations = append(deprecations, fileDeprecations...)

		if len(fileDeprecations) > 0 {
			progressCallback(fmt.Sprintf("  🔍 Found %d deprecations in %s", len(fileDeprecations), fileName))
		}
	}

	return deprecations, nil
}
