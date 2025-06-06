package models

import "time"

// FlutterRelease represents a Flutter release from GitHub API
type FlutterRelease struct {
	Name        string `json:"name"`
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	Prerelease  bool   `json:"prerelease"`
}

// FlutterOfficialRelease represents a release from the official Flutter releases API
type FlutterOfficialRelease struct {
	Hash           string `json:"hash"`
	Channel        string `json:"channel"`
	Version        string `json:"version"`
	DartSDKVersion string `json:"dart_sdk_version"`
	DartSDKArch    string `json:"dart_sdk_arch"`
	ReleaseDate    string `json:"release_date"`
	Archive        string `json:"archive"`
	SHA256         string `json:"sha256"`
}

// FlutterReleasesResponse represents the complete response from the official Flutter releases API
type FlutterReleasesResponse struct {
	BaseURL        string `json:"base_url"`
	CurrentRelease struct {
		Beta   string `json:"beta"`
		Dev    string `json:"dev"`
		Stable string `json:"stable"`
	} `json:"current_release"`
	Releases []FlutterOfficialRelease `json:"releases"`
}

// Deprecation represents a deprecated Flutter API
type Deprecation struct {
	API         string `json:"api"`
	Replacement string `json:"replacement"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

// DeprecationCache represents the local cache structure
type DeprecationCache struct {
	LastUpdated  time.Time     `json:"last_updated"`
	Deprecations []Deprecation `json:"deprecations"`
}

// FlutterVersionInfo contains version and availability information
type FlutterVersionInfo struct {
	LatestVersion    string `json:"latest_version"`
	FVMInstalled     bool   `json:"fvm_installed"`
	FVMVersionExists bool   `json:"fvm_version_exists"`
	DockerImages     struct {
		Instrumentisto bool `json:"instrumentisto"`
		CirrusLabs     bool `json:"cirruslabs"`
	} `json:"docker_images"`
	Details string `json:"details"`
}

// CheckCodeArgs represents the input for code checking
type CheckCodeArgs struct {
	Code string `json:"code"`
}

// NoArguments represents empty arguments for tools that don't need parameters
type NoArguments struct{}