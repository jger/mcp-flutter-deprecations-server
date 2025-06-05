package services

import (
	"strings"
	"testing"

	"github.com/jger/mcp-flutter-deprecations-server/internal/models"
)

// MockFlutterAPIService for testing
type MockFlutterAPIService struct {
	releases         []models.FlutterRelease
	fvmInstalled     bool
	fvmVersionExists bool
	dockerResults    map[string]bool
}

func (m *MockFlutterAPIService) FetchReleases() ([]models.FlutterRelease, error) {
	return m.releases, nil
}

func (m *MockFlutterAPIService) ParseVersionFromRelease(release models.FlutterRelease) string {
	return strings.TrimPrefix(release.TagName, "v")
}

func (m *MockFlutterAPIService) GetLatestStableVersion() (string, error) {
	// Not used in VersionInfoService, so can be empty
	return "", nil
}

func (m *MockFlutterAPIService) CheckFVMInstalled() bool {
	return m.fvmInstalled
}

func (m *MockFlutterAPIService) CheckFVMVersionExists(version string) bool {
	return m.fvmVersionExists
}

func (m *MockFlutterAPIService) CheckDockerImageExists(image string, tag string) bool {
	key := image + ":" + tag
	if result, exists := m.dockerResults[key]; exists {
		return result
	}
	return false
}

func TestVersionInfoService(t *testing.T) {
	t.Run("GetFlutterVersionInfo with stable version", func(t *testing.T) {
		mockAPI := &MockFlutterAPIService{
			releases: []models.FlutterRelease{
				{TagName: "3.19.0-0.1.pre", Prerelease: true, PublishedAt: "2024-12-02T10:00:00Z"},
				{TagName: "3.32.0", Prerelease: false, PublishedAt: "2024-12-01T10:00:00Z"},
				{TagName: "3.31.0", Prerelease: false, PublishedAt: "2024-11-15T10:00:00Z"},
			},
			fvmInstalled:     true,
			fvmVersionExists: true,
			dockerResults: map[string]bool{
				"instrumentisto/flutter:3.32.0": true,
				"cirrusci/flutter:3.32.0":       false,
			},
		}

		versionService := NewVersionInfoService(mockAPI)
		info, err := versionService.GetFlutterVersionInfo()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if info.LatestVersion != "3.32.0" {
			t.Errorf("Expected latest version to be 3.32.0, got %s", info.LatestVersion)
		}

		if !info.FVMInstalled {
			t.Error("Expected FVM to be installed")
		}

		if !info.FVMVersionExists {
			t.Error("Expected FVM version to exist")
		}

		if !info.DockerImages.Instrumentisto {
			t.Error("Expected instrumentisto docker image to be available")
		}

		if info.DockerImages.Cirrusci {
			t.Error("Expected cirrusci docker image to not be available")
		}

		// Check that details contain expected information
		if !strings.Contains(info.Details, "3.32.0") {
			t.Error("Expected details to contain version 3.32.0")
		}

		if !strings.Contains(info.Details, "FVM Status: ✅ Installed") {
			t.Error("Expected details to show FVM installed")
		}
	})

	t.Run("GetFlutterVersionInfo with no stable version", func(t *testing.T) {
		mockAPI := &MockFlutterAPIService{
			releases: []models.FlutterRelease{
				{TagName: "3.19.0-0.1.pre", Prerelease: true, PublishedAt: "2024-12-02T10:00:00Z"},
				{TagName: "3.18.0-beta.1", Prerelease: true, PublishedAt: "2024-11-01T10:00:00Z"},
			},
			fvmInstalled:     false,
			fvmVersionExists: false,
			dockerResults:    map[string]bool{},
		}

		versionService := NewVersionInfoService(mockAPI)
		info, err := versionService.GetFlutterVersionInfo()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should fall back to latest release
		if info.LatestVersion != "3.19.0-0.1.pre" {
			t.Errorf("Expected latest version to be 3.19.0-0.1.pre, got %s", info.LatestVersion)
		}

		if info.FVMInstalled {
			t.Error("Expected FVM to not be installed")
		}

		if !strings.Contains(info.Details, "FVM Status: ❌ Not installed") {
			t.Error("Expected details to show FVM not installed")
		}
	})

	t.Run("GetFlutterVersionInfo with complex version patterns", func(t *testing.T) {
		mockAPI := &MockFlutterAPIService{
			releases: []models.FlutterRelease{
				{TagName: "v3.33.0-rc.1", Prerelease: true, PublishedAt: "2024-12-03T10:00:00Z"},
				{TagName: "v3.32.0-hotfix.1", Prerelease: false, PublishedAt: "2024-12-02T10:00:00Z"},
				{TagName: "v3.32.0", Prerelease: false, PublishedAt: "2024-12-01T10:00:00Z"},
				{TagName: "v3.31.0", Prerelease: false, PublishedAt: "2024-11-15T10:00:00Z"},
			},
			fvmInstalled:     true,
			fvmVersionExists: false,
			dockerResults:    map[string]bool{},
		}

		versionService := NewVersionInfoService(mockAPI)
		info, err := versionService.GetFlutterVersionInfo()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should find 3.32.0 as the latest stable (skipping hotfix and rc versions)
		if info.LatestVersion != "3.32.0" {
			t.Errorf("Expected latest version to be 3.32.0, got %s", info.LatestVersion)
		}

		if info.FVMVersionExists {
			t.Error("Expected FVM version to not exist")
		}

		if !strings.Contains(info.Details, "❌ Not installed locally") {
			t.Error("Expected details to show version not installed locally")
		}
	})

	t.Run("GetFlutterVersionInfo with no releases", func(t *testing.T) {
		mockAPI := &MockFlutterAPIService{
			releases:         []models.FlutterRelease{},
			fvmInstalled:     false,
			fvmVersionExists: false,
			dockerResults:    map[string]bool{},
		}

		versionService := NewVersionInfoService(mockAPI)
		_, err := versionService.GetFlutterVersionInfo()

		if err == nil {
			t.Error("Expected error when no releases found")
		}

		if !strings.Contains(err.Error(), "no Flutter releases found") {
			t.Errorf("Expected error message about no releases, got %v", err)
		}
	})
}
