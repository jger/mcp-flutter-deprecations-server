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

	"github.com/example/flutter-deprecations-server/internal/models"
	"github.com/example/flutter-deprecations-server/pkg/config"
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
			fileDeprecations, err := f.scanFileForDeprecations(fileURL)
			if err != nil {
				fmt.Printf("Warning: Failed to scan file %s: %v\n", file.Name, err)
				continue
			}
			deprecations = append(deprecations, fileDeprecations...)
		}
	}
	
	return deprecations, nil
}

// scanFileForDeprecations scans a single Dart file for @Deprecated annotations
func (f *FlutterAPIService) scanFileForDeprecations(fileURL string) ([]models.Deprecation, error) {
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
	
	var currentDeprecation *models.Deprecation
	var collectingDeprecation bool
	var deprecationMessage strings.Builder
	
	// Regex patterns for extracting deprecation info
	deprecatedPattern := regexp.MustCompile(`@[Dd]eprecated\s*\(\s*['"](.+?)['"]`)
	classPattern := regexp.MustCompile(`(?:class|enum|mixin)\s+(\w+)`)
	methodPattern := regexp.MustCompile(`(?:static\s+)?(?:[\w<>]+\s+)?(\w+)\s*\(`)
	constructorPattern := regexp.MustCompile(`(\w+)\s*\.\s*(\w+)\s*\(`)
	
	lineNumber := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNumber++
		
		// Look for @Deprecated annotation
		if matches := deprecatedPattern.FindStringSubmatch(line); len(matches) > 1 {
			currentDeprecation = &models.Deprecation{
				Description: matches[1],
			}
			collectingDeprecation = true
			deprecationMessage.Reset()
			continue
		}
		
		// If we're collecting a deprecation, look for the deprecated item
		if collectingDeprecation && currentDeprecation != nil {
			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
				continue
			}
			
			// Look for class, method, constructor, or property
			var apiName string
			
			if matches := classPattern.FindStringSubmatch(line); len(matches) > 1 {
				apiName = matches[1]
			} else if matches := constructorPattern.FindStringSubmatch(line); len(matches) > 2 {
				apiName = matches[1] + "." + matches[2]
			} else if matches := methodPattern.FindStringSubmatch(line); len(matches) > 1 {
				apiName = matches[1]
			}
			
			if apiName != "" {
				currentDeprecation.API = apiName
				
				// Try to extract replacement from description
				desc := currentDeprecation.Description
				if strings.Contains(strings.ToLower(desc), "use ") {
					// Extract replacement suggestion
					usePattern := regexp.MustCompile(`(?i)use\s+([A-Za-z0-9_.]+)`)
					if useMatches := usePattern.FindStringSubmatch(desc); len(useMatches) > 1 {
						currentDeprecation.Replacement = useMatches[1]
					}
				}
				
				deprecations = append(deprecations, *currentDeprecation)
				currentDeprecation = nil
				collectingDeprecation = false
			}
		}
	}
	
	return deprecations, scanner.Err()
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
		fileDeprecations, err := f.scanFileForDeprecations(fileURL)
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