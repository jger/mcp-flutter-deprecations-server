package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/example/flutter-deprecations-server/internal/models"
)

// VersionInfoService handles Flutter version information
type VersionInfoService struct {
	apiService FlutterAPIServiceInterface
}

// NewVersionInfoService creates a new version info service instance
func NewVersionInfoService(apiService FlutterAPIServiceInterface) *VersionInfoService {
	return &VersionInfoService{
		apiService: apiService,
	}
}

// GetFlutterVersionInfo gets comprehensive Flutter version information
func (v *VersionInfoService) GetFlutterVersionInfo() (*models.FlutterVersionInfo, error) {
	// Force fresh fetch from GitHub API - bypass any potential caching
	releases, err := v.apiService.FetchReleases()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Flutter releases from GitHub: %v", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no Flutter releases found")
	}

	// Find latest stable version
	var latestVersion string
	var debugInfo []string
	
	for i, release := range releases {
		if i < 10 { // Collect debug info for first 10 releases
			debugInfo = append(debugInfo, fmt.Sprintf("Release %d: %s (prerelease: %v)", i, release.TagName, release.Prerelease))
		}
		
		// Special check for version 3.32.0
		if strings.Contains(release.TagName, "3.32.0") {
			debugInfo = append(debugInfo, fmt.Sprintf("FOUND 3.32.0: %s (prerelease: %v)", release.TagName, release.Prerelease))
		}
		
		tagLower := strings.ToLower(release.TagName)
		version := v.apiService.ParseVersionFromRelease(release)
		
		// More strict stable release detection
		isStable := !release.Prerelease &&
			!strings.Contains(tagLower, "beta") && 
			!strings.Contains(tagLower, "dev") && 
			!strings.Contains(tagLower, "pre") &&
			!strings.Contains(tagLower, "rc") &&
			!strings.Contains(tagLower, "alpha") &&
			!strings.Contains(tagLower, "hotfix") &&
			!strings.Contains(version, "-") &&
			// Ensure it's a pure semantic version (no suffixes)
			regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(version) &&
			// Additional check: tag should not contain pre-release indicators
			!strings.Contains(release.TagName, "-") &&
			!strings.Contains(release.TagName, ".pre") &&
			!strings.Contains(release.TagName, ".rc") &&
			!strings.Contains(release.TagName, ".beta") &&
			!strings.Contains(release.TagName, ".alpha")
			
		if isStable {
			latestVersion = version
			debugInfo = append(debugInfo, fmt.Sprintf("FOUND STABLE: %s", version))
			break
		}
	}

	// If no stable found, use the most recent release
	if latestVersion == "" {
		latestVersion = v.apiService.ParseVersionFromRelease(releases[0])
	}

	info := &models.FlutterVersionInfo{
		LatestVersion: latestVersion,
		FVMInstalled:  v.apiService.CheckFVMInstalled(),
	}

	if info.FVMInstalled {
		info.FVMVersionExists = v.apiService.CheckFVMVersionExists(latestVersion)
	}

	// Check Docker images availability
	info.DockerImages.Instrumentisto = v.apiService.CheckDockerImageExists("instrumentisto/flutter", latestVersion)
	info.DockerImages.Cirrusci = v.apiService.CheckDockerImageExists("cirrusci/flutter", latestVersion)

	// Build details string
	details := v.buildDetailsString(info, releases, debugInfo)
	info.Details = details

	return info, nil
}

// buildDetailsString creates the formatted details string
func (v *VersionInfoService) buildDetailsString(info *models.FlutterVersionInfo, releases []models.FlutterRelease, debugInfo []string) string {
	details := fmt.Sprintf("Latest Flutter Version: %s (Checked: %s)\n\n", info.LatestVersion, time.Now().Format("2006-01-02 15:04:05"))
	
	if info.FVMInstalled {
		details += "FVM Status: ✅ Installed\n"
		if info.FVMVersionExists {
			details += fmt.Sprintf("  - Version %s: ✅ Available locally\n", info.LatestVersion)
		} else {
			details += fmt.Sprintf("  - Version %s: ❌ Not installed locally\n", info.LatestVersion)
			details += fmt.Sprintf("  - Install with: fvm install %s\n", info.LatestVersion)
		}
	} else {
		details += "FVM Status: ❌ Not installed\n"
		details += "  - Install FVM: https://fvm.app/docs/getting_started/installation\n"
	}

	details += "\nDocker Images:\n"
	if info.DockerImages.Instrumentisto {
		details += fmt.Sprintf("  - instrumentisto/flutter:%s ✅ Available\n", info.LatestVersion)
	} else {
		details += fmt.Sprintf("  - instrumentisto/flutter:%s ❌ Not available\n", info.LatestVersion)
	}
	
	if info.DockerImages.Cirrusci {
		details += fmt.Sprintf("  - cirrusci/flutter:%s ✅ Available\n", info.LatestVersion)
	} else {
		details += fmt.Sprintf("  - cirrusci/flutter:%s ❌ Not available\n", info.LatestVersion)
	}

	details += "\nUsage Examples:\n"
	if info.FVMInstalled {
		details += fmt.Sprintf("  - FVM: fvm use %s\n", info.LatestVersion)
	}
	details += fmt.Sprintf("  - Docker (instrumentisto): docker run -it instrumentisto/flutter:%s\n", info.LatestVersion)
	details += fmt.Sprintf("  - Docker (cirrusci): docker run -it cirrusci/flutter:%s\n", info.LatestVersion)

	// Add debug info about releases found
	details += fmt.Sprintf("\n--- Debug Info ---\n")
	details += fmt.Sprintf("Total releases fetched: %d\n", len(releases))
	
	for _, info := range debugInfo {
		details += fmt.Sprintf("%s\n", info)
	}

	return details
}