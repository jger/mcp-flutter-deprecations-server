package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/example/flutter-deprecations-server/internal/models"
	"github.com/example/flutter-deprecations-server/pkg/config"
)

// DeprecationService handles deprecation analysis and management
type DeprecationService struct {
	cacheService CacheServiceInterface
	apiService   FlutterAPIServiceInterface
}

// NewDeprecationService creates a new deprecation service instance
func NewDeprecationService(cacheService CacheServiceInterface, apiService FlutterAPIServiceInterface) *DeprecationService {
	return &DeprecationService{
		cacheService: cacheService,
		apiService:   apiService,
	}
}

// getDeprecationPatterns returns known deprecation patterns
func (d *DeprecationService) getDeprecationPatterns() map[string]models.Deprecation {
	return map[string]models.Deprecation{
		`Color\.\w+\.withOpacity\(([^)]+)\)`: {
			API:         "Color.withOpacity",
			Replacement: "Color.withValues(alpha: $1)",
			Description: "withOpacity is deprecated, use withValues instead",
			Example:     "Color.red.withOpacity(0.5) → Color.red.withValues(alpha: 0.5)",
		},
		`RaisedButton`: {
			API:         "RaisedButton",
			Replacement: "ElevatedButton",
			Description: "RaisedButton is deprecated, use ElevatedButton instead",
			Example:     "RaisedButton → ElevatedButton",
		},
		`FlatButton`: {
			API:         "FlatButton",
			Replacement: "TextButton",
			Description: "FlatButton is deprecated, use TextButton instead",
			Example:     "FlatButton → TextButton",
		},
		`OutlineButton`: {
			API:         "OutlineButton",
			Replacement: "OutlinedButton",
			Description: "OutlineButton is deprecated, use OutlinedButton instead",
			Example:     "OutlineButton → OutlinedButton",
		},
		`Scaffold\.of\(context\)\.showSnackBar`: {
			API:         "Scaffold.of(context).showSnackBar",
			Replacement: "ScaffoldMessenger.of(context).showSnackBar",
			Description: "Direct showSnackBar on Scaffold is deprecated",
			Example:     "Scaffold.of(context).showSnackBar → ScaffoldMessenger.of(context).showSnackBar",
		},
		`FloatingActionButton\(child:`: {
			API:         "FloatingActionButton(child:",
			Replacement: "FloatingActionButton with specific constructors",
			Description: "Consider using FloatingActionButton.extended or other specific constructors",
		},
	}
}

// isVersionFromLast18Months checks if a version is from the last 18 months
func (d *DeprecationService) isVersionFromLast18Months(publishedAt string) bool {
	publishTime, err := time.Parse(time.RFC3339, publishedAt)
	if err != nil {
		return false
	}
	
	cutoff := time.Now().AddDate(0, -18, 0)
	return publishTime.After(cutoff)
}

// ExtractDeprecationsFromReleaseNotes extracts deprecations from Flutter release notes
func (d *DeprecationService) ExtractDeprecationsFromReleaseNotes(releases []models.FlutterRelease) []models.Deprecation {
	var deprecations []models.Deprecation
	
	// More specific patterns for real Flutter API deprecations
	patterns := []string{
		`(?i)deprecated[:\s]+([A-Z][a-zA-Z0-9_.]*)\s*(?:in favor of|replaced by|use)\s+([A-Z][a-zA-Z0-9_.]*)`,
		`(?i)([A-Z][a-zA-Z0-9_.]*)\s+(?:is\s+)?deprecated[,\s]*(?:use|replaced by)\s+([A-Z][a-zA-Z0-9_.]*)`,
		`(?i)\*\*Breaking change\*\*[^*]*deprecated[^*]*([A-Z][a-zA-Z0-9_.]*)[^*]*([A-Z][a-zA-Z0-9_.]*)?`,
	}

	for _, release := range releases {
		if !d.isVersionFromLast18Months(release.PublishedAt) {
			continue
		}

		version := d.apiService.ParseVersionFromRelease(release)
		body := release.Body

		for _, pattern := range patterns {
			regex := regexp.MustCompile(pattern)
			matches := regex.FindAllStringSubmatch(body, -1)
			for _, match := range matches {
				if len(match) >= 2 {
					api := strings.TrimSpace(match[1])
					replacement := ""
					if len(match) >= 3 && match[2] != "" {
						replacement = strings.TrimSpace(match[2])
					}

					// Filter out obviously wrong matches
					if len(api) < 3 || !strings.Contains(api, ".") && len(api) < 5 {
						continue
					}

					deprecation := models.Deprecation{
						API:         api,
						Replacement: replacement,
						Version:     version,
						Description: fmt.Sprintf("Deprecated in Flutter %s", version),
					}
					deprecations = append(deprecations, deprecation)
				}
			}
		}
	}

	// Add the known deprecation patterns
	for _, templateDep := range d.getDeprecationPatterns() {
		dep := templateDep
		dep.Version = "Multiple versions"
		deprecations = append(deprecations, dep)
	}

	return deprecations
}

// CheckCodeForDeprecations analyzes code for deprecated APIs
func (d *DeprecationService) CheckCodeForDeprecations(code string) []models.Deprecation {
	var foundDeprecations []models.Deprecation

	for regexPattern, deprecation := range d.getDeprecationPatterns() {
		regex := regexp.MustCompile(regexPattern)
		if regex.MatchString(code) {
			foundDeprecations = append(foundDeprecations, deprecation)
		}
	}

	cache, err := d.cacheService.Load()
	if err == nil {
		for _, dep := range cache.Deprecations {
			if dep.API != "" && strings.Contains(code, dep.API) {
				foundDeprecations = append(foundDeprecations, dep)
			}
		}
	}

	return foundDeprecations
}

// UpdateCache updates the deprecations cache
func (d *DeprecationService) UpdateCache() error {
	cache, err := d.cacheService.Load()
	if err != nil {
		return err
	}

	if time.Since(cache.LastUpdated) < config.CACHE_DURATION {
		return nil
	}

	releases, err := d.apiService.FetchReleases()
	if err != nil {
		return err
	}

	deprecations := d.ExtractDeprecationsFromReleaseNotes(releases)
	
	cache.Deprecations = deprecations
	cache.LastUpdated = time.Now()

	return d.cacheService.Save(cache)
}