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

// TestRegisterFlags_CommandTree pins that the reserved-flag guard covers
// every subcommand, not just the root: cobra merges each command's local
// flags with its ancestors' persistent flags at parse time, so a subcommand
// shorthand collision panics in pflag when that subcommand runs, and a name
// collision silently shadows clix's flag. registerFlags must refuse both
// before anything executes, naming the command path and the flag.
func TestRegisterFlags_CommandTree(t *testing.T) {
	cases := []struct {
		name    string
		build   func() *cobra.Command
		wantErr []string // substrings; empty means no error
	}{
		{
			name: "subcommand local shorthand collision",
			build: func() *cobra.Command {
				root := &cobra.Command{Use: "probe"}
				sub := &cobra.Command{Use: "sub", RunE: func(*cobra.Command, []string) error { return nil }}
				var name string
				sub.Flags().StringVarP(&name, "name", "n", "", "local -n on the subcommand")
				root.AddCommand(sub)
				return root
			},
			wantErr: []string{`clix: command "probe sub" already defines shorthand -n (used by --name)`},
		},
		{
			name: "nested sub-subcommand persistent name collision",
			build: func() *cobra.Command {
				root := &cobra.Command{Use: "probe"}
				sub := &cobra.Command{Use: "sub"}
				leaf := &cobra.Command{Use: "leaf", RunE: func(*cobra.Command, []string) error { return nil }}
				var verbose bool
				leaf.PersistentFlags().BoolVar(&verbose, "verbose", false, "persistent --verbose two levels down")
				sub.AddCommand(leaf)
				root.AddCommand(sub)
				return root
			},
			wantErr: []string{`clix: command "probe sub leaf" already defines flag --verbose`},
		},
		{
			name: "subcommand with non-colliding flags",
			build: func() *cobra.Command {
				root := &cobra.Command{Use: "probe"}
				sub := &cobra.Command{Use: "sub", RunE: func(*cobra.Command, []string) error { return nil }}
				var name string
				var force bool
				sub.Flags().StringVarP(&name, "name", "m", "", "a shorthand clix does not reserve")
				sub.PersistentFlags().BoolVarP(&force, "force", "f", false, "another one")
				root.AddCommand(sub)
				return root
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })
			root := tc.build()
			err := registerFlags(root)
			if len(tc.wantErr) == 0 {
				if err != nil {
					t.Fatalf("registerFlags() error = %v, want nil", err)
				}
				if root.PersistentFlags().Lookup("dry-run") == nil {
					t.Error("--dry-run not registered on the root")
				}
				return
			}
			if err == nil {
				t.Fatal("registerFlags() = nil error, want a reserved-flag collision")
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("registerFlags() error = %q, want it to contain %q", err, want)
				}
			}
			// The guard refuses before registering: nothing was added to the root.
			if root.PersistentFlags().Lookup("json") != nil {
				t.Error("clix flags were registered despite the collision")
			}
		})
	}
}

// TestRunSubcommandCollisionReturnsBeforeExecuting pins the App.Run contract
// for the subcommand case: the error surfaces from Run itself, and the
// subcommand never executes (against main this reached pflag's
// "unable to redefine 'n' shorthand" panic when the subcommand ran).
func TestRunSubcommandCollisionReturnsBeforeExecuting(t *testing.T) {
	t.Cleanup(func() { JSONOutput, Verbose, DryRun, Silent = false, false, false, false })

	ran := false
	root := &cobra.Command{Use: "probe"}
	sub := &cobra.Command{Use: "sub", RunE: func(*cobra.Command, []string) error { ran = true; return nil }}
	var name string
	sub.Flags().StringVarP(&name, "name", "n", "", "local -n on the subcommand")
	root.AddCommand(sub)
	root.SetArgs([]string{"sub"})

	err := runNoPanic(t, &App{Version: "1.0.0"}, root)
	if err == nil {
		t.Fatal("Run() = nil error, want reserved-shorthand collision error for the subcommand")
	}
	if !strings.Contains(err.Error(), `command "probe sub"`) || !strings.Contains(err.Error(), "-n") {
		t.Errorf("Run() error = %q, want it to name the subcommand and -n", err)
	}
	if ran {
		t.Error("subcommand executed despite the collision")
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
