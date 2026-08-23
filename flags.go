package clix

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

// BindViper binds the common flags (--json, --verbose, --dry-run, --silent) to
// viper keys of the same names. Call it from a PersistentPreRunE if your app
// uses viper for config management. cobra runs the root's PersistentPreRunE
// with cmd set to the command actually executing, so BindViper resolves each
// flag through that command's local, persistent, and inherited flag sets — it
// is safe to call from any command in the tree, root or subcommand. It returns
// a clix-namespaced error when a flag is not registered on cmd, which means
// App.Run has not registered the flags on the root command yet.
func BindViper(cmd *cobra.Command) error {
	for _, f := range commonFlags() {
		name := f.name
		set, flag := lookupFlag(cmd, name)
		if flag == nil {
			return fmt.Errorf("clix: BindViper: --%s is not registered on %q; call App.Run on the root command first", name, cmd.CommandPath())
		}
		if err := viper.BindPFlag(name, flag); err != nil {
			return err
		}
		// Binding alone leaves a config-file or environment value inside
		// viper. clix and its consumers read the package-level variables
		// (JSONOutput, Verbose, DryRun, Silent), which are the flags'
		// targets, so without this write-back a `json: true` config would
		// change nothing. Precedence is command line > viper > default: an
		// explicit flag has Changed set, and is never overwritten — so
		// `--json=false` against `json: true` in config stays false.
		if flag.Changed || !viper.GetBool(name) {
			continue
		}
		if err := set.Set(name, "true"); err != nil {
			return err
		}
	}
	return nil
}

// lookupFlag resolves name on cmd the way cobra will at parse time: the
// command's own flag set (which holds merged persistent flags once parsing
// has run), then its persistent set (before parsing), then the flags it
// inherits from its ancestors (a subcommand seeing the root's clix flags).
// It returns the owning set alongside the flag, because writing a viper
// value back goes through the set — FlagSet.Set records that the value was
// set, where assigning the flag's Value alone would not. Every set here
// holds the same *pflag.Flag pointer, so the write reaches the same target
// variable whichever one resolved it.
func lookupFlag(cmd *cobra.Command, name string) (*pflag.FlagSet, *pflag.Flag) {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return cmd.Flags(), flag
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		return cmd.PersistentFlags(), flag
	}
	if flag := cmd.InheritedFlags().Lookup(name); flag != nil {
		return cmd.InheritedFlags(), flag
	}
	return nil, nil
}
