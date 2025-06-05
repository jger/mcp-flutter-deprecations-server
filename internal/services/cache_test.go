package services

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jger/mcp-flutter-deprecations-server/internal/models"
)

// TestCacheServiceImpl implements CacheServiceInterface for testing
type TestCacheServiceImpl struct {
	tempDir string
}

func (t *TestCacheServiceImpl) getCacheDir() string {
	return t.tempDir
}

func (t *TestCacheServiceImpl) ensureCacheDir() error {
	return os.MkdirAll(t.getCacheDir(), 0755)
}

func (t *TestCacheServiceImpl) Load() (*models.DeprecationCache, error) {
	cachePath := filepath.Join(t.getCacheDir(), "flutter_deprecations.json")

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return &models.DeprecationCache{Deprecations: []models.Deprecation{}}, nil
	}

	data, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache models.DeprecationCache
	err = json.Unmarshal(data, &cache)
	if err != nil {
		return &models.DeprecationCache{Deprecations: []models.Deprecation{}}, nil
	}

	return &cache, nil
}

func (t *TestCacheServiceImpl) Save(cache *models.DeprecationCache) error {
	if err := t.ensureCacheDir(); err != nil {
		return err
	}

	cachePath := filepath.Join(t.getCacheDir(), "flutter_deprecations.json")
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cachePath, data, 0644)
}

func TestCacheService(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create cache service with custom cache dir
	cacheService := &TestCacheServiceImpl{
		tempDir: tempDir,
	}

	t.Run("Load empty cache", func(t *testing.T) {
		cache, err := cacheService.Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if cache == nil {
			t.Fatal("Expected cache to not be nil")
		}
		// When no cache file exists, should return empty cache
		if len(cache.Deprecations) != 0 {
			t.Errorf("Expected empty deprecations, got %d", len(cache.Deprecations))
		}
	})

	t.Run("Save and load cache", func(t *testing.T) {
		testCache := &models.DeprecationCache{
			LastUpdated: time.Now(),
			Deprecations: []models.Deprecation{
				{
					API:         "TestAPI",
					Replacement: "NewAPI",
					Version:     "1.0.0",
					Description: "Test deprecation",
					Example:     "TestAPI() → NewAPI()",
				},
			},
		}

		// Save cache
		err := cacheService.Save(testCache)
		if err != nil {
			t.Fatalf("Expected no error saving cache, got %v", err)
		}

		// Load cache
		loadedCache, err := cacheService.Load()
		if err != nil {
			t.Fatalf("Expected no error loading cache, got %v", err)
		}

		if len(loadedCache.Deprecations) != 1 {
			t.Errorf("Expected 1 deprecation, got %d", len(loadedCache.Deprecations))
		}

		dep := loadedCache.Deprecations[0]
		if dep.API != "TestAPI" {
			t.Errorf("Expected API 'TestAPI', got '%s'", dep.API)
		}
		if dep.Replacement != "NewAPI" {
			t.Errorf("Expected Replacement 'NewAPI', got '%s'", dep.Replacement)
		}
	})

	t.Run("Cache directory creation", func(t *testing.T) {
		// Remove the temp directory
		os.RemoveAll(tempDir)

		// Try to save cache (should create directory)
		testCache := &models.DeprecationCache{
			LastUpdated:  time.Now(),
			Deprecations: []models.Deprecation{},
		}

		err := cacheService.Save(testCache)
		if err != nil {
			t.Fatalf("Expected no error creating cache directory, got %v", err)
		}

		// Verify directory was created
		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Error("Expected cache directory to be created")
		}
	})
}
