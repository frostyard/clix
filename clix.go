// Package clix provides CLI convenience functions for Frostyard tools,
// wrapping charm.land/fang/v2 and spf13/cobra with standardized version
// injection, common flags, JSON output helpers, and reporter factory.
package clix

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
)

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

// Run registers common persistent flags on cmd, then executes the command
// via fang.Execute with the formatted version string and signal handling.
//
// The flags --json, --verbose/-v, --dry-run/-n, and --silent/-s are reserved
// by clix on the root command. If cmd already defines one of those names or
// shorthands, Run returns an error naming the collision before executing
// anything, rather than letting pflag panic.
//
// Run is a thin wrapper around RunContext using context.Background(); use
// RunContext directly when the caller needs to bound execution with a
// deadline or let a supervising process cancel the run.
func (a *App) Run(cmd *cobra.Command) error {
	return a.RunContext(context.Background(), cmd)
}

// RunContext is Run with a caller-supplied context: it registers common
// persistent flags on cmd, then executes the command via fang.Execute with
// the formatted version string and signal handling, running under ctx.
//
// Prefer RunContext over Run when the caller wants to bound execution with a
// deadline or timeout, or let a supervising process (a test, a parent
// command, an orchestrator) cancel the run instead of relying solely on the
// OS signal handling fang.Execute installs for SIGINT/SIGTERM.
//
// The flags --json, --verbose/-v, --dry-run/-n, and --silent/-s are reserved
// by clix on the root command. If cmd already defines one of those names or
// shorthands, RunContext returns an error naming the collision before
// executing anything, rather than letting pflag panic.
func (a *App) RunContext(ctx context.Context, cmd *cobra.Command) error {
	if cmd == nil {
		return fmt.Errorf("clix: Run: root command is nil")
	}
	a.defaults()
	if err := registerFlags(cmd); err != nil {
		return err
	}
	return fang.Execute(
		ctx,
		cmd,
		fang.WithVersion(a.VersionString()),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	)
}
