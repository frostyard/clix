# Clix

CLI convenience module for Frostyard tools.

Wraps [charmbracelet/fang](https://github.com/charmbracelet/fang) and [spf13/cobra](https://github.com/spf13/cobra) to provide standardized version strings, common flags, JSON output helpers, and a reporter factory. Consuming CLIs only need to define their own commands.

## Install

```bash
go get github.com/frostyard/clix
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/frostyard/clix"
	"github.com/spf13/cobra"
)

// Set via ldflags at build time.
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
		Use:   "mytool",
		Short: "An example CLI built with clix",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use the reporter for progress output
			r := clix.NewReporter()
			r.Info("doing work...")

			result := map[string]string{"status": "ok"}

			// OutputJSON writes to stdout when --json is set
			if written, err := clix.OutputJSON(result); written {
				return err
			}

			// Fall through to text output
			fmt.Println("done")
			return nil
		},
	}

	if err := app.Run(rootCmd); err != nil {
		clix.OutputJSONError("command failed", err)
	}
}
```

Build with ldflags for version injection:

```bash
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -I) -X main.builtBy=ci"
```

## Flags

`App.Run()` automatically registers these persistent flags on the root command:

| Flag | Short | Description |
|---|---|---|
| `--json` | | Output in JSON format |
| `--verbose` | `-v` | Verbose output |
| `--dry-run` | `-n` | Dry run mode (no actual changes) |
| `--silent` | `-s` | Suppress all progress output |

Flag values are available as package-level variables: `clix.JSONOutput`, `clix.Verbose`, `clix.DryRun`, `clix.Silent`.

## Reporter

`clix.NewReporter()` returns a reporter based on the active flags:

| Priority | Condition | Reporter | Output |
|---|---|---|---|
| 1 (highest) | `--silent` | NoopReporter | none |
| 2 | `--json` | JSONReporter | stdout |
| 3 (default) | neither | TextReporter | stderr |

Silent always takes priority over JSON. Text output goes to stderr to keep stdout clean for data.

## Viper Integration

If your CLI uses [spf13/viper](https://github.com/spf13/viper) for config, bind the common flags in `PersistentPreRunE`:

```go
rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
	return clix.BindViper(cmd)
}
```

This binds `--json`, `--verbose`, `--dry-run`, and `--silent` to viper keys so they can be set via config files or environment variables.

## Development

```bash
make test            # run all tests
make lint            # run golangci-lint
make check           # fmt + lint + test (pre-commit gate)
make bump            # tag next semver with svu and push
```
