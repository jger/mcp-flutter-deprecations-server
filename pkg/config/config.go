package config

import "time"

const (
	// Cache configuration
	CACHE_FILE     = "flutter_deprecations.json"
	CACHE_DURATION = 24 * time.Hour

	// API endpoints
	FLUTTER_API_URL = "https://api.github.com/repos/flutter/flutter/releases"

	// API limits
	MAX_RELEASES = 100
)