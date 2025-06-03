package services

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/example/flutter-deprecations-server/internal/models"
	"github.com/example/flutter-deprecations-server/pkg/config"
)

// CacheService handles local cache operations
type CacheService struct{}

// NewCacheService creates a new cache service instance
func NewCacheService() *CacheService {
	return &CacheService{}
}

// getCacheDir returns the cache directory path
func (c *CacheService) getCacheDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".flutter-deprecations")
}

// ensureCacheDir creates the cache directory if it doesn't exist
func (c *CacheService) ensureCacheDir() error {
	return os.MkdirAll(c.getCacheDir(), 0755)
}

// Load loads the deprecation cache from disk
func (c *CacheService) Load() (*models.DeprecationCache, error) {
	cachePath := filepath.Join(c.getCacheDir(), config.CACHE_FILE)
	
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

// Save saves the deprecation cache to disk
func (c *CacheService) Save(cache *models.DeprecationCache) error {
	if err := c.ensureCacheDir(); err != nil {
		return err
	}

	cachePath := filepath.Join(c.getCacheDir(), config.CACHE_FILE)
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cachePath, data, 0644)
}