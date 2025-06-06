package config

import "time"

const (
	// Cache configuration
	CACHE_FILE     = "flutter_deprecations.json"
	CACHE_DURATION = 24 * time.Hour

	// API endpoints
	FLUTTER_API_URL = "https://api.github.com/repos/flutter/flutter/releases"
	FLUTTER_RELEASES_URL = "https://storage.googleapis.com/flutter_infra_release/releases/releases_linux.json"

	// API limits
	MAX_RELEASES = 100
)