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
			r.Message("doing work...")

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

These four flags — `--json`, `--verbose`/`-v`, `--dry-run`/`-n`, and `--silent`/`-s` — are reserved by clix on the root command: if your root command already defines one of those names or shorthands (persistent or local), `App.Run()` returns an error naming the collision instead of panicking, before anything executes.

## Reporter

`clix.NewReporter()` returns a reporter based on the active flags:

| Priority | Condition | Reporter | Output |
|---|---|---|---|
| 1 (highest) | `--silent` | NoopReporter | none |
| 2 | `--json` | JSONReporter | stdout |
| 3 (default) | neither | TextReporter | stderr |

Silent always takes priority over JSON. Text output goes to stderr to keep stdout clean for data.

## Testing output

Every clix output path — `OutputJSON`, `OutputJSONError`, and the reporters
from `NewReporter` — writes through two package-level writer seams,
`clix.Stdout` and `clix.Stderr`. Both default to `nil`, meaning "the current
`os.Stdout` / `os.Stderr` at call time", so production behavior is unchanged
and consumers that already swap `os.Stdout` in tests keep working. `Stdout` /
`Stderr` are shared package-level variables, so tests that set them should not
run in parallel unless they coordinate access. In tests, assign a
`bytes.Buffer` instead of touching the process globals:
assign a `bytes.Buffer` instead of touching the process globals:

```go
func TestStatusJSON(t *testing.T) {
	var out bytes.Buffer
	clix.Stdout = &out
	defer func() { clix.Stdout = nil }()
	clix.JSONOutput = true
	defer func() { clix.JSONOutput = false }()

	// ... run the command ...

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
}
```

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
make bump            # tag next semver with svu and push; the tag push publishes the GitHub release
```

The e2e suite in [`tests/e2e/`](tests/e2e/README.md) builds a consumer program wired like the example above and runs it as a subprocess. Contributing guide: [AGENTS.md](AGENTS.md) (`CONTRIBUTING.md` resolves to it).
