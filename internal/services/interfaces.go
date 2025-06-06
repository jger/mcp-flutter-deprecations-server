package services

import "github.com/jger/mcp-flutter-deprecations-server/internal/models"

// CacheServiceInterface defines the cache service contract
type CacheServiceInterface interface {
	Load() (*models.DeprecationCache, error)
	Save(cache *models.DeprecationCache) error
}

// FlutterAPIServiceInterface defines the Flutter API service contract
type FlutterAPIServiceInterface interface {
	FetchReleases() ([]models.FlutterRelease, error)
	FetchOfficialReleases() (*models.FlutterReleasesResponse, error)
	ParseVersionFromRelease(release models.FlutterRelease) string
	GetLatestStableVersion() (string, error)
	CheckFVMInstalled() bool
	CheckFVMVersionExists(version string) bool
	CheckDockerImageExists(image string, tag string) bool
	FetchFlutterSourceDeprecations() ([]models.Deprecation, error)
	FetchFlutterSourceDeprecationsWithProgress(progressCallback func(string), verbose bool) ([]models.Deprecation, error)
}

// DeprecationServiceInterface defines the deprecation service contract
type DeprecationServiceInterface interface {
	CheckCodeForDeprecations(code string) []models.Deprecation
	UpdateCache() error
	ExtractDeprecationsFromReleaseNotes(releases []models.FlutterRelease) []models.Deprecation
}

// VersionInfoServiceInterface defines the version info service contract
type VersionInfoServiceInterface interface {
	GetFlutterVersionInfo() (*models.FlutterVersionInfo, error)
}

// FlutterVersionServiceInterface defines the Flutter version detection contract
type FlutterVersionServiceInterface interface {
	GetInstalledFlutterVersion() (string, error)
	IsFlutterInstalled() bool
	GetFlutterChannel() (string, error)
}
