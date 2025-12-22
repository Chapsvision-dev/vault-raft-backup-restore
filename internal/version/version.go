package version

var (
	// Version is the semantic version (injected at build time).
	Version = "dev"
	// Commit is the git commit SHA (injected at build time).
	Commit = "unknown"
	// BuildDate is the build timestamp (injected at build time).
	BuildDate = "unknown"
)

// Info returns formatted version information.
func Info() string {
	return Version + " (" + Commit + ", built " + BuildDate + ")"
}
