package app

import (
	"fmt"
	"runtime"
)

// Build information. Populated at build-time.
var (
	Version   = "unknown"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
	Platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// VersionInfo holds the version information.
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetVersion returns the version string.
func GetVersion() string {
	return Version
}

// GetVersionInfo returns the full version information.
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

// String returns version info as a string.
func (v VersionInfo) String() string {
	return fmt.Sprintf(
		"Version: %s\nGit Commit: %s\nBuild Date: %s\nGo Version: %s\nPlatform: %s",
		v.Version, v.GitCommit, v.BuildDate, v.GoVersion, v.Platform,
	)
}
