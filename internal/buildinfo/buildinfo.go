// Package buildinfo provides build version and metadata information.
package buildinfo

// Version metadata is injected at build time via ldflags.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

// Summary returns a human-readable version summary string.
func Summary() string {
	version := Version
	if version == "" {
		version = "dev"
	}
	parts := version
	if Commit != "" {
		parts += " (" + Commit
		if Date != "" {
			parts += " " + Date
		}
		parts += ")"
	} else if Date != "" {
		parts += " (" + Date + ")"
	}
	return parts
}
