// Command exampletool is the consumer program clix's end-to-end suite builds
// and runs as a subprocess. It wires clix exactly the way README.md tells a
// Frostyard CLI to: build metadata via ldflags, App.Run on a cobra root, the
// reporter for progress, OutputJSON/OutputJSONError for data.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/frostyard/clix"
	"github.com/spf13/cobra"
)

// Set via ldflags at build time (the e2e suite injects them).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "local"
)

func main() {
	app := clix.App{
		Version: version,
		Commit:  commit,
		Date:    date,
		BuiltBy: builtBy,
	}

	rootCmd := &cobra.Command{
		Use:           "exampletool",
		Short:         "clix end-to-end example consumer",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "greet [name]",
		Short: "Print a greeting (JSON with --json)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "world"
			if len(args) == 1 {
				name = args[0]
			}
			r := clix.NewReporter()
			r.Message("greeting %s", name)

			result := map[string]any{
				"greeting": "hello " + name,
				"verbose":  clix.Verbose,
				"dry_run":  clix.DryRun,
			}
			if written, err := clix.OutputJSON(result); written {
				return err
			}
			fmt.Printf("hello %s\n", name)
			return nil
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "fail",
		Short: "Exit non-zero (JSON error envelope with --json)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return clix.OutputJSONError("fail command failed", errors.New("boom"))
		},
	})

	if err := app.Run(rootCmd); err != nil {
		os.Exit(1)
	}
}
