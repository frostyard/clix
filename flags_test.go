package clix

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestRegisterFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	if err := registerFlags(cmd); err != nil {
		t.Fatalf("registerFlags() error = %v", err)
	}

	flags := []struct {
		name      string
		shorthand string
	}{
		{"json", ""},
		{"verbose", "v"},
		{"dry-run", "n"},
		{"silent", "s"},
	}

	for _, f := range flags {
		pf := cmd.PersistentFlags().Lookup(f.name)
		if pf == nil {
			t.Errorf("flag --%s not registered", f.name)
			continue
		}
		if f.shorthand != "" && pf.Shorthand != f.shorthand {
			t.Errorf("flag --%s shorthand = %q, want %q", f.name, pf.Shorthand, f.shorthand)
		}
	}
}

func TestBindViper(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cmd := &cobra.Command{Use: "test"}
	if err := registerFlags(cmd); err != nil {
		t.Fatalf("registerFlags() error = %v", err)
	}
	// registerFlags binds the flags to the package-level variables; setting
	// them below mutates that state, so restore it for the other tests.
	t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })

	err := BindViper(cmd)
	if err != nil {
		t.Fatalf("BindViper() error = %v", err)
	}

	// Pin the bound key set: each common flag must reach viper under its own key.
	for _, name := range []string{"json", "verbose", "dry-run", "silent"} {
		if viper.GetBool(name) {
			t.Fatalf("viper key %q unexpectedly true before the flag was set", name)
		}
		if err := cmd.PersistentFlags().Set(name, "true"); err != nil {
			t.Fatalf("Set(%q) error = %v", name, err)
		}
		if !viper.GetBool(name) {
			t.Errorf("viper key %q not bound to --%s: GetBool = false after flag set", name, name)
		}
	}
}

// TestBindViper_FromSubcommandPreRun pins the wiring README.md documents:
// clix.BindViper installed as the root's PersistentPreRunE, which cobra runs
// with cmd set to the *executing* subcommand. The clix flags live on the
// root's persistent set and reach the subcommand only as inherited flags, so
// BindViper must resolve through the executing command's local, persistent,
// and inherited sets rather than cmd.PersistentFlags() alone.
func TestBindViper_FromSubcommandPreRun(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })

	ran := false
	root := &cobra.Command{Use: "probe"}
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error { return BindViper(cmd) }
	sub := &cobra.Command{Use: "sub", RunE: func(*cobra.Command, []string) error { ran = true; return nil }}
	root.AddCommand(sub)
	if err := registerFlags(root); err != nil {
		t.Fatalf("registerFlags() error = %v", err)
	}
	root.SetArgs([]string{"sub", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !ran {
		t.Fatal("subcommand RunE did not run")
	}
	if !viper.GetBool("json") {
		t.Error(`viper.GetBool("json") = false after "sub --json"; BindViper did not bind the inherited flag`)
	}
}

// TestBindViper_UnregisteredFlagError pins the clix-namespaced error for a
// command tree App.Run has not registered the flags on.
func TestBindViper_UnregisteredFlagError(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cmd := &cobra.Command{Use: "bare"}
	err := BindViper(cmd)
	if err == nil {
		t.Fatal("BindViper() = nil error, want an error naming the missing flag")
	}
	for _, want := range []string{"clix: BindViper", "--json", `"bare"`, "App.Run"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("BindViper() error = %q, want it to contain %q", err, want)
		}
	}
}
