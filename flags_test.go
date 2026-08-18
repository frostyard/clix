package clix

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestRegisterFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	registerFlags(cmd)

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
	registerFlags(cmd)
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
