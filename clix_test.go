package clix

import (
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
