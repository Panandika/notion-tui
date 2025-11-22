package version

import "fmt"

// Version information set at build time via ldflags.
var (
	// Version is the semantic version (e.g., "1.0.0" or "dev").
	Version = "dev"

	// Commit is the git commit hash.
	Commit = "unknown"

	// Date is the build date in RFC3339 format.
	Date = "unknown"

	// BuiltBy is the builder identifier.
	BuiltBy = "unknown"
)

// Info returns a formatted version string with all build information.
func Info() string {
	return fmt.Sprintf(
		"Notion TUI %s\nCommit: %s\nBuilt: %s\nBuilt by: %s",
		Version,
		Commit,
		Date,
		BuiltBy,
	)
}

// Short returns a short version string (just the version number).
func Short() string {
	if Version == "dev" {
		return fmt.Sprintf("%s-%s", Version, Commit)
	}
	return Version
}
