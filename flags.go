package clix

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Common flag values, populated when Run() registers persistent flags.
var (
	JSONOutput bool // --json flag value
	Verbose    bool // --verbose / -v flag value
	DryRun     bool // --dry-run / -n flag value
	Silent     bool // --silent / -s flag value
)

// commonFlag describes one of the flags clix reserves on the root command.
type commonFlag struct {
	target    *bool
	name      string
	shorthand string
	usage     string
}

// commonFlags lists the flags registerFlags reserves, in registration order.
func commonFlags() []commonFlag {
	return []commonFlag{
		{&JSONOutput, "json", "", "output in JSON format"},
		{&Verbose, "verbose", "v", "verbose output"},
		{&DryRun, "dry-run", "n", "dry run mode (no actual changes)"},
		{&Silent, "silent", "s", "suppress all progress output"},
	}
}

// registerFlags adds --json, --verbose, --dry-run, and --silent as persistent
// flags on cmd. It returns an error instead of letting pflag panic when the
// consumer's root command already defines one of those names or shorthands,
// on either its persistent or its local flag set (cobra merges both at parse
// time, so a local collision would still panic).
func registerFlags(cmd *cobra.Command) error {
	for _, f := range commonFlags() {
		for _, persistent := range []bool{true, false} {
			if err := checkReserved(cmd, persistent, f); err != nil {
				return err
			}
		}
	}
	for _, f := range commonFlags() {
		if f.shorthand == "" {
			cmd.PersistentFlags().BoolVar(f.target, f.name, false, f.usage)
			continue
		}
		cmd.PersistentFlags().BoolVarP(f.target, f.name, f.shorthand, false, f.usage)
	}
	return nil
}

// checkReserved reports a collision between f and a flag already defined on
// cmd's persistent (persistent=true) or local flag set.
func checkReserved(cmd *cobra.Command, persistent bool, f commonFlag) error {
	set := cmd.Flags()
	if persistent {
		set = cmd.PersistentFlags()
	}
	if set.Lookup(f.name) != nil {
		return fmt.Errorf("clix: root command already defines flag --%s", f.name)
	}
	if f.shorthand == "" {
		return nil
	}
	if other := set.ShorthandLookup(f.shorthand); other != nil {
		return fmt.Errorf("clix: root command already defines shorthand -%s (used by --%s)", f.shorthand, other.Name)
	}
	return nil
}

// BindViper binds the common flags (--json, --verbose, --dry-run, --silent) to viper.
// Call this in a PersistentPreRunE if your app uses viper for config management.
func BindViper(cmd *cobra.Command) error {
	for _, name := range []string{"json", "verbose", "dry-run", "silent"} {
		if err := viper.BindPFlag(name, cmd.PersistentFlags().Lookup(name)); err != nil {
			return err
		}
	}
	return nil
}
