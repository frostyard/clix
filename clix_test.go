package clix

import (
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
