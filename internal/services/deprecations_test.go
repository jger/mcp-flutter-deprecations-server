package services

import (
	"testing"
	"time"

	"github.com/example/flutter-deprecations-server/internal/models"
)

func TestDeprecationService(t *testing.T) {
	// Create mock services
	cacheService := NewCacheService()
	apiService := NewFlutterAPIService()
	depService := NewDeprecationService(cacheService, apiService)

	t.Run("CheckCodeForDeprecations", func(t *testing.T) {
		testCases := []struct {
			name          string
			code          string
			expectedCount int
			expectedAPIs  []string
		}{
			{
				name:          "Color.withOpacity deprecation",
				code:          "Color.red.withOpacity(0.5)",
				expectedCount: 1,
				expectedAPIs:  []string{"Color.withOpacity"},
			},
			{
				name:          "RaisedButton deprecation",
				code:          "RaisedButton(onPressed: () {}, child: Text('Click'))",
				expectedCount: 1,
				expectedAPIs:  []string{"RaisedButton"},
			},
			{
				name:          "Multiple deprecations",
				code:          "RaisedButton(child: Text('Click')) and FlatButton(child: Text('Flat'))",
				expectedCount: 2,
				expectedAPIs:  []string{"RaisedButton", "FlatButton"},
			},
			{
				name:          "No deprecations",
				code:          "ElevatedButton(onPressed: () {}, child: Text('Modern'))",
				expectedCount: 0,
				expectedAPIs:  []string{},
			},
			{
				name:          "Scaffold showSnackBar deprecation",
				code:          "Scaffold.of(context).showSnackBar(SnackBar(content: Text('Test')))",
				expectedCount: 1,
				expectedAPIs:  []string{"Scaffold.of(context).showSnackBar"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				deprecations := depService.CheckCodeForDeprecations(tc.code)
				
				if len(deprecations) != tc.expectedCount {
					t.Errorf("Expected %d deprecations, got %d", tc.expectedCount, len(deprecations))
				}

				// Check that expected APIs are found
				foundAPIs := make(map[string]bool)
				for _, dep := range deprecations {
					foundAPIs[dep.API] = true
				}

				for _, expectedAPI := range tc.expectedAPIs {
					if !foundAPIs[expectedAPI] {
						t.Errorf("Expected to find API '%s' but didn't", expectedAPI)
					}
				}
			})
		}
	})

	t.Run("ExtractDeprecationsFromReleaseNotes", func(t *testing.T) {
		testReleases := []models.FlutterRelease{
			{
				TagName:     "3.31.0",
				PublishedAt: time.Now().AddDate(0, -6, 0).Format(time.RFC3339), // 6 months ago
				Body:        "RaisedButton is deprecated, use ElevatedButton instead. Also deprecated ColorScheme.background in favor of ColorScheme.surface.",
				Prerelease:  false,
			},
			{
				TagName:     "2.0.0",
				PublishedAt: time.Now().AddDate(-2, 0, 0).Format(time.RFC3339), // 2 years ago (should be filtered out)
				Body:        "OldWidget is deprecated, use NewWidget instead.",
				Prerelease:  false,
			},
		}

		deprecations := depService.ExtractDeprecationsFromReleaseNotes(testReleases)

		// Should include built-in patterns plus any extracted from recent releases
		if len(deprecations) < 6 { // At least the 6 built-in patterns
			t.Errorf("Expected at least 6 deprecations (built-in patterns), got %d", len(deprecations))
		}

		// Check that built-in patterns are included
		hasColorWithOpacity := false
		hasRaisedButton := false
		for _, dep := range deprecations {
			if dep.API == "Color.withOpacity" {
				hasColorWithOpacity = true
			}
			if dep.API == "RaisedButton" {
				hasRaisedButton = true
			}
		}

		if !hasColorWithOpacity {
			t.Error("Expected to find Color.withOpacity in deprecations")
		}
		if !hasRaisedButton {
			t.Error("Expected to find RaisedButton in deprecations")
		}
	})

	t.Run("isVersionFromLast18Months", func(t *testing.T) {
		testCases := []struct {
			name        string
			publishedAt string
			expected    bool
		}{
			{
				name:        "6 months ago",
				publishedAt: time.Now().AddDate(0, -6, 0).Format(time.RFC3339),
				expected:    true,
			},
			{
				name:        "12 months ago",
				publishedAt: time.Now().AddDate(0, -12, 0).Format(time.RFC3339),
				expected:    true,
			},
			{
				name:        "24 months ago",
				publishedAt: time.Now().AddDate(0, -24, 0).Format(time.RFC3339),
				expected:    false,
			},
			{
				name:        "Invalid date",
				publishedAt: "invalid-date",
				expected:    false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := depService.isVersionFromLast18Months(tc.publishedAt)
				if result != tc.expected {
					t.Errorf("Expected %v, got %v for date %s", tc.expected, result, tc.publishedAt)
				}
			})
		}
	})
}