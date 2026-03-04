// Package clix provides CLI convenience functions for Frostyard tools,
// wrapping charmbracelet/fang and spf13/cobra with standardized version
// injection, common flags, JSON output helpers, and reporter factory.
package clix

import "fmt"

// App holds build-time metadata for a CLI application.
// Create one in main() and call Run() to execute the root command.
type App struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

// defaults fills zero-value fields with sensible defaults.
func (a *App) defaults() {
	if a.Version == "" {
		a.Version = "dev"
	}
	if a.Commit == "" {
		a.Commit = "none"
	}
	if a.Date == "" {
		a.Date = "unknown"
	}
	if a.BuiltBy == "" {
		a.BuiltBy = "local"
	}
}

// VersionString returns a formatted version string including commit, date,
// and builder info. Example: "1.2.3 (Commit: abc) (Date: 2026-01-01) (Built by: ci)"
func (a *App) VersionString() string {
	a.defaults()
	return fmt.Sprintf("%s (Commit: %s) (Date: %s) (Built by: %s)",
		a.Version, a.Commit, a.Date, a.BuiltBy)
}
