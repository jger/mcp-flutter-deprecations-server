package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/jger/mcp-flutter-deprecations-server/internal/models"
)

// MockCacheService for testing
type MockCacheService struct {
	cache *models.DeprecationCache
}

func (m *MockCacheService) Load() (*models.DeprecationCache, error) {
	if m.cache == nil {
		return &models.DeprecationCache{
			LastUpdated:  time.Now(),
			Deprecations: []models.Deprecation{},
		}, nil
	}
	return m.cache, nil
}

func (m *MockCacheService) Save(cache *models.DeprecationCache) error {
	m.cache = cache
	return nil
}

// MockDeprecationService for testing
type MockDeprecationService struct {
	deprecations []models.Deprecation
}

func (m *MockDeprecationService) CheckCodeForDeprecations(code string) []models.Deprecation {
	return m.deprecations
}

func (m *MockDeprecationService) UpdateCache() error {
	return nil
}

func (m *MockDeprecationService) ExtractDeprecationsFromReleaseNotes(releases []models.FlutterRelease) []models.Deprecation {
	return m.deprecations
}

// MockVersionInfoService for testing
type MockVersionInfoService struct {
	versionInfo *models.FlutterVersionInfo
	err         error
}

func (m *MockVersionInfoService) GetFlutterVersionInfo() (*models.FlutterVersionInfo, error) {
	return m.versionInfo, m.err
}

func TestMCPHandlers(t *testing.T) {
	t.Run("CheckFlutterDeprecations - with deprecations found", func(t *testing.T) {
		mockDepService := &MockDeprecationService{
			deprecations: []models.Deprecation{
				{
					API:         "Color.withOpacity",
					Replacement: "Color.withValues(alpha: $1)",
					Description: "withOpacity is deprecated",
					Example:     "Color.red.withOpacity(0.5) → Color.red.withValues(alpha: 0.5)",
					Version:     "Multiple versions",
				},
			},
		}

		handlers := NewMCPHandlers(mockDepService, nil, nil)

		args := models.CheckCodeArgs{Code: "Color.red.withOpacity(0.5)"}
		response, err := handlers.CheckFlutterDeprecations(args)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content := response.Content[0].TextContent.Text
		if !strings.Contains(content, "Found deprecated APIs") {
			t.Error("Expected response to mention found deprecated APIs")
		}
		if !strings.Contains(content, "Color.withOpacity") {
			t.Error("Expected response to mention Color.withOpacity")
		}
		if !strings.Contains(content, "withOpacity is deprecated") {
			t.Error("Expected response to mention deprecation description")
		}
	})

	t.Run("CheckFlutterDeprecations - no deprecations found", func(t *testing.T) {
		mockDepService := &MockDeprecationService{
			deprecations: []models.Deprecation{},
		}

		handlers := NewMCPHandlers(mockDepService, nil, nil)

		args := models.CheckCodeArgs{Code: "ElevatedButton()"}
		response, err := handlers.CheckFlutterDeprecations(args)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content := response.Content[0].TextContent.Text
		if !strings.Contains(content, "No deprecated APIs found") {
			t.Error("Expected response to mention no deprecated APIs found")
		}
	})

	t.Run("ListFlutterDeprecations - with cache data", func(t *testing.T) {
		mockCache := &MockCacheService{
			cache: &models.DeprecationCache{
				LastUpdated: time.Now(),
				Deprecations: []models.Deprecation{
					{
						API:         "RaisedButton",
						Replacement: "ElevatedButton",
						Description: "RaisedButton is deprecated",
						Version:     "Multiple versions",
					},
					{
						API:         "FlatButton",
						Replacement: "TextButton",
						Description: "FlatButton is deprecated",
						Version:     "Multiple versions",
					},
				},
			},
		}

		handlers := NewMCPHandlers(nil, nil, mockCache)

		args := models.NoArguments{}
		response, err := handlers.ListFlutterDeprecations(args)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content := response.Content[0].TextContent.Text
		if !strings.Contains(content, "Flutter Deprecations") {
			t.Error("Expected response to mention Flutter Deprecations")
		}
		if !strings.Contains(content, "RaisedButton") {
			t.Error("Expected response to mention RaisedButton")
		}
		if !strings.Contains(content, "FlatButton") {
			t.Error("Expected response to mention FlatButton")
		}
	})

	t.Run("ListFlutterDeprecations - empty cache", func(t *testing.T) {
		mockCache := &MockCacheService{
			cache: &models.DeprecationCache{
				LastUpdated:  time.Now(),
				Deprecations: []models.Deprecation{},
			},
		}

		handlers := NewMCPHandlers(nil, nil, mockCache)

		args := models.NoArguments{}
		response, err := handlers.ListFlutterDeprecations(args)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content := response.Content[0].TextContent.Text
		if !strings.Contains(content, "No deprecations found in cache") {
			t.Error("Expected response to mention no deprecations found")
		}
	})

	t.Run("UpdateFlutterDeprecations - success", func(t *testing.T) {
		mockDepService := &MockDeprecationService{}
		mockCache := &MockCacheService{
			cache: &models.DeprecationCache{
				LastUpdated: time.Now(),
				Deprecations: []models.Deprecation{
					{API: "TestAPI", Replacement: "NewAPI"},
				},
			},
		}

		handlers := NewMCPHandlers(mockDepService, nil, mockCache)

		args := models.NoArguments{}
		response, err := handlers.UpdateFlutterDeprecations(args)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content := response.Content[0].TextContent.Text
		if !strings.Contains(content, "Successfully updated deprecations cache") {
			t.Error("Expected response to mention successful update")
		}
		if !strings.Contains(content, "Found 1 deprecations") {
			t.Error("Expected response to mention found deprecations count")
		}
	})

	t.Run("CheckFlutterVersionInfo - success", func(t *testing.T) {
		mockVersionService := &MockVersionInfoService{
			versionInfo: &models.FlutterVersionInfo{
				LatestVersion:    "3.32.0",
				FVMInstalled:     true,
				FVMVersionExists: true,
				Details:          "Latest Flutter Version: 3.32.0\n\nFVM Status: ✅ Installed",
			},
		}

		handlers := NewMCPHandlers(nil, mockVersionService, nil)

		args := models.NoArguments{}
		response, err := handlers.CheckFlutterVersionInfo(args)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content := response.Content[0].TextContent.Text
		if !strings.Contains(content, "3.32.0") {
			t.Error("Expected response to mention version 3.32.0")
		}
		if !strings.Contains(content, "FVM Status: ✅ Installed") {
			t.Error("Expected response to mention FVM status")
		}
	})

	t.Run("CheckFlutterVersionInfo - error", func(t *testing.T) {
		mockVersionService := &MockVersionInfoService{
			err: &MockError{message: "GitHub API failed"},
		}

		handlers := NewMCPHandlers(nil, mockVersionService, nil)

		args := models.NoArguments{}
		response, err := handlers.CheckFlutterVersionInfo(args)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content := response.Content[0].TextContent.Text
		if !strings.Contains(content, "Error getting Flutter version info") {
			t.Error("Expected response to mention error")
		}
		if !strings.Contains(content, "GitHub API failed") {
			t.Error("Expected response to mention specific error")
		}
	})
}

// MockError for testing
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}
