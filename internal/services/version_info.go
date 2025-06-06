package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jger/mcp-flutter-deprecations-server/internal/models"
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
	var latestVersion string
	var debugInfo []string
	var flutterInstalled bool
	var installedVersion string
	var channel string

	// First, try to get version from installed Flutter CLI (most reliable)
	flutterVersionService := NewFlutterVersionService()
	flutterInstalled = flutterVersionService.IsFlutterInstalled()

	if flutterInstalled {
		var err error
		installedVersion, err = flutterVersionService.GetInstalledFlutterVersion()
		if err == nil {
			latestVersion = installedVersion
			debugInfo = append(debugInfo, fmt.Sprintf("Using installed Flutter version: %s", installedVersion))

			// Get channel info
			channel, _ = flutterVersionService.GetFlutterChannel()
			debugInfo = append(debugInfo, fmt.Sprintf("Flutter channel: %s", channel))
		} else {
			debugInfo = append(debugInfo, fmt.Sprintf("Error getting installed Flutter version: %v", err))
		}
	} else {
		debugInfo = append(debugInfo, "Flutter CLI not installed, falling back to GitHub API")
	}

	// If Flutter not installed or failed, fall back to official releases API, then GitHub API
	if latestVersion == "" {
		// Try official releases API first
		officialReleases, err := v.apiService.FetchOfficialReleases()
		if err == nil && len(officialReleases.Releases) > 0 {
			debugInfo = append(debugInfo, "Using official Flutter releases API")
			
			// Find latest stable release
			for _, release := range officialReleases.Releases {
				if release.Channel == "stable" {
					latestVersion = release.Version
					debugInfo = append(debugInfo, fmt.Sprintf("Official API: Found stable version: %s", release.Version))
					break
				}
			}
		} else {
			debugInfo = append(debugInfo, fmt.Sprintf("Official API failed: %v, falling back to GitHub API", err))
		}

		// If official API failed or no stable found, fall back to GitHub API
		if latestVersion == "" {
			releases, err := v.apiService.FetchReleases()
			if err != nil {
				return nil, fmt.Errorf("failed to fetch Flutter releases from GitHub: %v", err)
			}

			if len(releases) == 0 {
				return nil, fmt.Errorf("no Flutter releases found")
			}

			debugInfo = append(debugInfo, "Falling back to GitHub API releases")

			for i, release := range releases {
				if i < 5 { // Collect debug info for first 5 releases
					debugInfo = append(debugInfo, fmt.Sprintf("GitHub Release %d: %s (prerelease: %v)", i, release.TagName, release.Prerelease))
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
					debugInfo = append(debugInfo, fmt.Sprintf("GitHub: Found stable version: %s", version))
					break
				}
			}

			// If no stable found, use the most recent release
			if latestVersion == "" {
				latestVersion = v.apiService.ParseVersionFromRelease(releases[0])
				debugInfo = append(debugInfo, fmt.Sprintf("GitHub: No stable found, using latest: %s", latestVersion))
			}
		}
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
	info.DockerImages.CirrusLabs = v.apiService.CheckDockerImageExists("ghcr.io/cirruslabs/flutter", latestVersion)

	// Build details string
	details := v.buildDetailsString(info, flutterInstalled, installedVersion, channel, debugInfo)
	info.Details = details

	return info, nil
}

// buildDetailsString creates the formatted details string
func (v *VersionInfoService) buildDetailsString(info *models.FlutterVersionInfo, flutterInstalled bool, installedVersion, channel string, debugInfo []string) string {
	details := fmt.Sprintf("Latest Flutter Version: %s (Checked: %s)\n\n", info.LatestVersion, time.Now().Format("2006-01-02 15:04:05"))

	// Flutter CLI status
	if flutterInstalled {
		details += "Flutter CLI: ✅ Installed\n"
		if installedVersion != "" {
			details += fmt.Sprintf("  - Installed Version: %s\n", installedVersion)
			if channel != "" {
				details += fmt.Sprintf("  - Channel: %s\n", channel)
			}
		}
	} else {
		details += "Flutter CLI: ❌ Not installed\n"
		details += "  - Install Flutter: https://docs.flutter.dev/get-started/install\n"
	}
	details += "\n"

	// FVM status
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

	if info.DockerImages.CirrusLabs {
		details += fmt.Sprintf("  - ghcr.io/cirruslabs/flutter:%s ✅ Available\n", info.LatestVersion)
	} else {
		details += fmt.Sprintf("  - ghcr.io/cirruslabs/flutter:%s ❌ Not available\n", info.LatestVersion)
	}

	details += "\nUsage Examples:\n"
	if info.FVMInstalled {
		details += fmt.Sprintf("  - FVM: fvm use %s\n", info.LatestVersion)
	}
	details += fmt.Sprintf("  - Docker (instrumentisto): docker run -it instrumentisto/flutter:%s\n", info.LatestVersion)
	details += fmt.Sprintf("  - Docker (cirruslabs): docker run -it ghcr.io/cirruslabs/flutter:%s\n", info.LatestVersion)

	// Add debug info
	details += fmt.Sprintf("\n--- Debug Info ---\n")

	for _, info := range debugInfo {
		details += fmt.Sprintf("%s\n", info)
	}

	return details
}
