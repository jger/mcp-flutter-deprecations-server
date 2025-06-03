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
		Cirrusci       bool `json:"cirrusci"`
	} `json:"docker_images"`
	Details string `json:"details"`
}

// CheckCodeArgs represents the input for code checking
type CheckCodeArgs struct {
	Code string `json:"code"`
}

// NoArguments represents empty arguments for tools that don't need parameters
type NoArguments struct{}