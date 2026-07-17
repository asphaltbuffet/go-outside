package version

import "fmt"

const unknownBuildValue = "unknown"

var (
	// Version is the semantic version (injected via ldflags).
	Version = "dev"

	// GitCommit is the short git commit hash (injected via ldflags).
	GitCommit = "unknown"

	// BuildDate is the build timestamp ISO 8601 (injected via ldflags).
	BuildDate = "unknown"
)

// FullVersion returns the version string including commit hash.
func FullVersion() string {
	return fmt.Sprintf("%s+%s", Version, GitCommit)
}

// ShortVersion returns only the semantic version.
func ShortVersion() string {
	return Version
}

// BuildInfo returns build metadata including commit and timestamp.
func BuildInfo() string {
	if GitCommit == unknownBuildValue && BuildDate == unknownBuildValue {
		return ""
	}
	if BuildDate == unknownBuildValue {
		return GitCommit
	}
	return fmt.Sprintf("%s (%s)", GitCommit, BuildDate)
}
