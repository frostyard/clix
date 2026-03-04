package clix

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Common flag values, populated when Run() registers persistent flags.
var (
	JSONOutput bool // --json flag value
	Verbose    bool // --verbose / -v flag value
	DryRun     bool // --dry-run / -n flag value
)

// registerFlags adds --json, --verbose, and --dry-run as persistent flags on cmd.
func registerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "output in JSON format")
	cmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().BoolVarP(&DryRun, "dry-run", "n", false, "dry run mode (no actual changes)")
}

// BindViper binds the common flags (--json, --verbose, --dry-run) to viper.
// Call this in a PersistentPreRunE if your app uses viper for config management.
func BindViper(cmd *cobra.Command) error {
	for _, name := range []string{"json", "verbose", "dry-run"} {
		if err := viper.BindPFlag(name, cmd.PersistentFlags().Lookup(name)); err != nil {
			return err
		}
	}
	return nil
}
