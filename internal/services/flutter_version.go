package services

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// FlutterVersionService handles getting Flutter version directly from Flutter CLI
type FlutterVersionService struct{}

// NewFlutterVersionService creates a new Flutter version service
func NewFlutterVersionService() *FlutterVersionService {
	return &FlutterVersionService{}
}

// GetInstalledFlutterVersion gets the Flutter version from the installed Flutter CLI
func (f *FlutterVersionService) GetInstalledFlutterVersion() (string, error) {
	cmd := exec.Command("flutter", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output like: "Flutter 3.32.0 • channel stable • https://github.com/flutter/flutter.git"
	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no output from flutter --version")
	}

	firstLine := lines[0]
	
	// Extract version using regex: "Flutter X.Y.Z"
	versionRegex := regexp.MustCompile(`Flutter (\d+\.\d+\.\d+)`)
	matches := versionRegex.FindStringSubmatch(firstLine)
	
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse version from: %s", firstLine)
	}

	return matches[1], nil
}

// IsFlutterInstalled checks if Flutter CLI is available
func (f *FlutterVersionService) IsFlutterInstalled() bool {
	cmd := exec.Command("flutter", "--version")
	return cmd.Run() == nil
}

// GetFlutterChannel gets the Flutter channel (stable, beta, dev)
func (f *FlutterVersionService) GetFlutterChannel() (string, error) {
	cmd := exec.Command("flutter", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no output from flutter --version")
	}

	firstLine := lines[0]
	
	// Extract channel using regex: "• channel stable •"
	channelRegex := regexp.MustCompile(`• channel (\w+) •`)
	matches := channelRegex.FindStringSubmatch(firstLine)
	
	if len(matches) < 2 {
		return "unknown", nil
	}

	return matches[1], nil
}