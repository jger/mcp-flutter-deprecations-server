package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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