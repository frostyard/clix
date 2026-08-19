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

// BindViper binds the common flags (--json, --verbose, --dry-run, --silent) to
// viper keys of the same names. Call it from a PersistentPreRunE if your app
// uses viper for config management. cobra runs the root's PersistentPreRunE
// with cmd set to the command actually executing, so BindViper resolves each
// flag through that command's local, persistent, and inherited flag sets — it
// is safe to call from any command in the tree, root or subcommand. It returns
// a clix-namespaced error when a flag is not registered on cmd, which means
// App.Run has not registered the flags on the root command yet.
func BindViper(cmd *cobra.Command) error {
	for _, name := range []string{"json", "verbose", "dry-run", "silent"} {
		flag := lookupFlag(cmd, name)
		if flag == nil {
			return fmt.Errorf("clix: BindViper: --%s is not registered on %q; call App.Run on the root command first", name, cmd.Name())
		}
		if err := viper.BindPFlag(name, flag); err != nil {
			return err
		}
	}
	return nil
}

// lookupFlag resolves name on cmd the way cobra will at parse time: the
// command's own flag set (which holds merged persistent flags once parsing
// has run), then its persistent set (before parsing), then the flags it
// inherits from its ancestors (a subcommand seeing the root's clix flags).
func lookupFlag(cmd *cobra.Command, name string) *pflag.Flag {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		return flag
	}
	return cmd.InheritedFlags().Lookup(name)
}
