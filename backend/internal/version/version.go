// Package version surfaces build-time identity for the running backend.
//
// Backend and Frontend are semver constants bumped by the save-project skill
// (.claude/skills/save-project/SKILL.md). They are the authoritative source —
// package.json "version" must mirror Frontend exactly.
//
// GitSHA and BuiltAt are injected at build time via -ldflags. When unset
// (e.g. `go run` during dev) they fall back to "dev" / "unknown".
package version

const (
	Backend  = "0.5.0"
	Frontend = "1.6.0"
)

var (
	GitSHA  = "dev"
	BuiltAt = "unknown"
)
