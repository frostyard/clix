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
// consumer's root command, or any subcommand at any depth, already defines
// one of those names or shorthands on either its persistent or its local
// flag set. cobra merges a command's local flags with every ancestor's
// persistent flags at parse time: a colliding shorthand panics in pflag when
// that command runs, and a colliding name silently shadows clix's flag so
// the package-level variable is never set for that command.
func registerFlags(cmd *cobra.Command) error {
	if err := checkCommandTree(cmd, cmd); err != nil {
		return err
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

// checkCommandTree checks cmd and every descendant, depth first, against the
// reserved flags. root identifies the command whose collisions are reported
// with the original "root command" wording.
func checkCommandTree(root, cmd *cobra.Command) error {
	for _, f := range commonFlags() {
		for _, persistent := range []bool{true, false} {
			if err := checkReserved(root, cmd, persistent, f); err != nil {
				return err
			}
		}
	}
	for _, sub := range cmd.Commands() {
		if err := checkCommandTree(root, sub); err != nil {
			return err
		}
	}
	return nil
}

// checkReserved reports a collision between f and a flag already defined on
// cmd's persistent (persistent=true) or local flag set. The root command is
// named "root command"; a subcommand is named by its command path.
func checkReserved(root, cmd *cobra.Command, persistent bool, f commonFlag) error {
	set := cmd.Flags()
	if persistent {
		set = cmd.PersistentFlags()
	}
	who := "root command"
	if cmd != root {
		who = fmt.Sprintf("command %q", cmd.CommandPath())
	}
	if set.Lookup(f.name) != nil {
		return fmt.Errorf("clix: %s already defines flag --%s", who, f.name)
	}
	if f.shorthand == "" {
		return nil
	}
	if other := set.ShorthandLookup(f.shorthand); other != nil {
		return fmt.Errorf("clix: %s already defines shorthand -%s (used by --%s)", who, f.shorthand, other.Name)
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
