package clix

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionString(t *testing.T) {
	app := App{
		Version: "1.2.3",
		Commit:  "abc123",
		Date:    "2026-03-04",
		BuiltBy: "ci",
	}
	got := app.VersionString()
	want := "1.2.3 (Commit: abc123) (Date: 2026-03-04) (Built by: ci)"
	if got != want {
		t.Errorf("VersionString() = %q, want %q", got, want)
	}
}

func TestVersionStringDefaults(t *testing.T) {
	app := App{}
	got := app.VersionString()
	want := "dev (Commit: none) (Date: unknown) (Built by: local)"
	if got != want {
		t.Errorf("VersionString() = %q, want %q", got, want)
	}
}

func TestRunRegistersFlags(t *testing.T) {
	// Reset package-level flag state
	defer func() {
		JSONOutput = false
		Verbose = false
		DryRun = false
		Silent = false
	}()

	ran := false
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			ran = true
			return nil
		},
	}

	app := App{Version: "1.0.0"}
	err := app.Run(cmd)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ran {
		t.Error("command RunE was not called")
	}

	if cmd.PersistentFlags().Lookup("json") == nil {
		t.Error("--json flag not registered")
	}
	if cmd.PersistentFlags().Lookup("verbose") == nil {
		t.Error("--verbose flag not registered")
	}
	if cmd.PersistentFlags().Lookup("dry-run") == nil {
		t.Error("--dry-run flag not registered")
	}
	if cmd.PersistentFlags().Lookup("silent") == nil {
		t.Error("--silent flag not registered")
	}
}

// TestRunTwiceOnSameRootIsRepeatable pins that App.Run can execute the same
// cobra root more than once. cobra merges a root's persistent flags into its
// local flag set the first time it executes, so after that first run the
// reserved flags clix registered turn up in both flag sets; a naive second
// registerFlags call would see them as a collision against itself and refuse
// to run. Both executions must succeed, and the flags must keep working.
func TestRunTwiceOnSameRootIsRepeatable(t *testing.T) {
	t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })

	var seenJSON []bool
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(*cobra.Command, []string) error {
			seenJSON = append(seenJSON, JSONOutput)
			return nil
		},
	}

	app := &App{Version: "1.0.0"}

	cmd.SetArgs(nil)
	if err := runNoPanic(t, app, cmd); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	cmd.SetArgs([]string{"--json"})
	if err := runNoPanic(t, app, cmd); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if len(seenJSON) != 2 {
		t.Fatalf("RunE ran %d times, want 2", len(seenJSON))
	}
	if seenJSON[0] {
		t.Errorf("JSONOutput = true on first run, want false (no --json passed)")
	}
	if !seenJSON[1] {
		t.Errorf("JSONOutput = false on second run, want true (--json passed)")
	}
}

// TestRunTwiceOnSameRootJSONTrueThenFalseIsolatesState pins the fix for a
// reliability gap: a --json invocation must not leak into a later
// invocation on the same root that passes no --json flag. Before the fix,
// registerFlags reused the earlier invocation's *pflag.Flag without
// resetting its value or Changed bit, so the second invocation observed
// JSONOutput == true even though it never asked for JSON output.
func TestRunTwiceOnSameRootJSONTrueThenFalseIsolatesState(t *testing.T) {
	t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })

	var seenJSON []bool
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(*cobra.Command, []string) error {
			seenJSON = append(seenJSON, JSONOutput)
			return nil
		},
	}

	app := &App{Version: "1.0.0"}

	cmd.SetArgs([]string{"--json"})
	if err := runNoPanic(t, app, cmd); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	cmd.SetArgs(nil)
	if err := runNoPanic(t, app, cmd); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if len(seenJSON) != 2 {
		t.Fatalf("RunE ran %d times, want 2", len(seenJSON))
	}
	if !seenJSON[0] {
		t.Errorf("JSONOutput = false on first run, want true (--json passed)")
	}
	if seenJSON[1] {
		t.Errorf("JSONOutput = true on second run, want false (no --json passed); stale state leaked across invocations")
	}
	if got := cmd.PersistentFlags().Lookup("json").Changed; got {
		t.Errorf("--json Changed = true after the second run, want false: stale Changed leaked across invocations")
	}
}

// TestRunTwice_StaleChangedDoesNotSuppressViper pins that a stale Changed bit
// carried over from an earlier invocation cannot suppress a later
// invocation's viper-provided value, while an explicit flag on the current
// invocation still wins over viper (the command-line-over-viper precedence
// BindViper documents).
func TestRunTwice_StaleChangedDoesNotSuppressViper(t *testing.T) {
	restoreFlagState(t)
	writeViperConfig(t, "json: true\n")

	var seenJSON []bool
	cmd := &cobra.Command{
		Use: "test",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return BindViper(cmd)
		},
		RunE: func(*cobra.Command, []string) error {
			seenJSON = append(seenJSON, JSONOutput)
			return nil
		},
	}

	app := &App{Version: "1.0.0"}

	// First invocation: an explicit --json=false beats viper's json: true
	// and leaves the flag Changed.
	cmd.SetArgs([]string{"--json=false"})
	if err := runNoPanic(t, app, cmd); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	// Second invocation: no --json flag at all. Without resetting the stale
	// Changed=true left by the first invocation, BindViper's guard would
	// wrongly treat this invocation as having an explicit flag too,
	// suppressing viper's json: true.
	cmd.SetArgs(nil)
	if err := runNoPanic(t, app, cmd); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	// Third invocation: an explicit --json=false must still beat viper.
	cmd.SetArgs([]string{"--json=false"})
	if err := runNoPanic(t, app, cmd); err != nil {
		t.Fatalf("third Run() error = %v", err)
	}

	if len(seenJSON) != 3 {
		t.Fatalf("RunE ran %d times, want 3", len(seenJSON))
	}
	if seenJSON[0] {
		t.Errorf("JSONOutput = true on first run, want false (explicit --json=false beats viper)")
	}
	if !seenJSON[1] {
		t.Errorf("JSONOutput = false on second run, want true (viper's json: true should apply once stale Changed is reset)")
	}
	if seenJSON[2] {
		t.Errorf("JSONOutput = true on third run, want false (explicit --json=false beats viper again)")
	}
}

func TestRunContextPropagatesCancellation(t *testing.T) {
	defer func() {
		JSONOutput = false
		Verbose = false
		DryRun = false
		Silent = false
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the command runs

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Context().Err()
		},
	}

	app := App{Version: "1.0.0"}
	err := app.RunContext(ctx, cmd)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunContext() error = %v, want context.Canceled", err)
	}
}

// runNoPanic calls app.Run(cmd) and converts a panic into an explicit test
// failure, so a regression to pflag's "flag redefined" panic is reported
// rather than crashing the test binary.
func runNoPanic(t *testing.T, app *App, cmd *cobra.Command) (err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Run() panicked: %v", r)
		}
	}()
	return app.Run(cmd)
}

func TestRunNilCommandReturnsError(t *testing.T) {
	err := runNoPanic(t, &App{}, nil)
	if err == nil {
		t.Fatal("Run(nil) = nil error, want contextual error")
	}
	if got, want := err.Error(), "clix: Run: root command is nil"; got != want {
		t.Errorf("Run(nil) error = %q, want %q", got, want)
	}
}

func TestRunReservedShorthandCollisionReturnsError(t *testing.T) {
	t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })

	var name string
	cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.PersistentFlags().StringVarP(&name, "name", "n", "", "consumer flag using clix's -n")

	err := runNoPanic(t, &App{Version: "1.0.0"}, cmd)
	if err == nil {
		t.Fatal("Run() = nil error, want reserved-shorthand collision error")
	}
	for _, want := range []string{"clix:", "-n", "--name"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Run() error = %q, want it to mention %q", err, want)
		}
	}
	if cmd.PersistentFlags().Lookup("dry-run") != nil {
		t.Error("--dry-run was registered despite the shorthand collision")
	}
}

func TestRunReservedNameCollisionReturnsError(t *testing.T) {
	t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })

	var jsonPath string
	cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.PersistentFlags().StringVar(&jsonPath, "json", "", "consumer flag using clix's --json")

	err := runNoPanic(t, &App{Version: "1.0.0"}, cmd)
	if err == nil {
		t.Fatal("Run() = nil error, want reserved-flag collision error")
	}
	if !strings.Contains(err.Error(), "--json") || !strings.Contains(err.Error(), "clix:") {
		t.Errorf("Run() error = %q, want it to name --json", err)
	}
}

func TestRunReservedLocalFlagCollisionReturnsError(t *testing.T) {
	t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })

	// A root-local (non-persistent) flag collides too: cobra merges the local
	// and persistent sets at parse time, so pflag would panic there instead.
	var silent bool
	cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.Flags().BoolVarP(&silent, "quiet", "s", false, "consumer flag using clix's -s")

	err := runNoPanic(t, &App{Version: "1.0.0"}, cmd)
	if err == nil {
		t.Fatal("Run() = nil error, want reserved-shorthand collision error for local flag")
	}
	if !strings.Contains(err.Error(), "-s") || !strings.Contains(err.Error(), "--quiet") {
		t.Errorf("Run() error = %q, want it to mention -s and --quiet", err)
	}
}
