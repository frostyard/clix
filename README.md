# Clix

CLI convenience module for Frostyard tools.

Wraps [charm.land/fang/v2](https://pkg.go.dev/charm.land/fang/v2) and [spf13/cobra](https://github.com/spf13/cobra) to provide standardized version strings, common flags, JSON output helpers, and a reporter factory. Consuming CLIs only need to define their own commands.

## Install

```bash
go get github.com/frostyard/clix
```

## Usage

```go
package main

import (
	"fmt"
	"os"

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

			// OutputJSON writes to stdout when --json is set. An error means
			// the JSON could not be encoded (a complete fallback envelope may
			// have been written) or a complete document did not reach stdout;
			// either way, report it rather than falling through to text.
			written, err := clix.OutputJSON(result)
			if err != nil {
				return err
			}
			if written {
				return nil
			}

			// Fall through to text output
			fmt.Println("done")
			return nil
		},
	}

	if err := app.Run(rootCmd); err != nil {
		os.Exit(1)
	}
}
```

When a command fails, fang renders the error to stderr and `App.Run` returns
it, so the caller owns the process exit code — `os.Exit(1)` here is what makes
`mytool` exit non-zero (call `clix.OutputJSONError` inside a command's `RunE`
to emit the JSON error envelope under `--json`).

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

These four flags — `--json`, `--verbose`/`-v`, `--dry-run`/`-n`, and `--silent`/`-s` — are reserved by clix across the whole command tree: if your root command *or any subcommand at any depth* already defines one of those names or shorthands (persistent or local), `App.Run()` returns an error naming the command and the collision instead of panicking, before anything executes. (cobra merges each command's local flags with its ancestors' persistent flags at parse time, so a subcommand shorthand collision would otherwise panic only when that subcommand runs, and a name collision would silently shadow clix's flag.)

## RunContext

`App.Run(cmd)` is a thin wrapper around `App.RunContext(context.Background(), cmd)`. Call `RunContext` directly when you want to bound execution with a deadline/timeout or let a supervising process (a test, a parent command, an orchestrator) cancel the run instead of relying solely on the `SIGINT`/`SIGTERM` handling `Run` installs:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := app.RunContext(ctx, rootCmd); err != nil {
	os.Exit(1)
}
```

`RunContext` performs the same nil-command check, defaulting, and flag registration as `Run`, then runs `fang.Execute` with the supplied `ctx`; a command's `RunE` observes cancellation via `cmd.Context().Err()`.

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

This binds `--json`, `--verbose`, `--dry-run`, and `--silent` to viper keys so they can be set via config files or environment variables. cobra runs the root's `PersistentPreRunE` with `cmd` set to the command actually executing, and `BindViper` resolves the flags through that command's local, persistent, and inherited sets, so this one hook works for the root and every subcommand. Calling it before `App.Run` has registered the flags returns `clix: BindViper: --json is not registered on "<cmd>"; call App.Run on the root command first`.

`BindViper` also writes a viper-sourced value back onto the flag, so a config file or environment variable actually changes `clix.JSONOutput`, `clix.Verbose`, `clix.DryRun`, and `clix.Silent` — binding alone would leave the value inside viper, where nothing clix reads would see it.

Precedence is **command line > viper > default**. A flag the user passed is never overwritten, so an explicit `--json=false` wins over `json: true` in a config file. Call `BindViper` from `PersistentPreRunE` (not `RunE`) so the write-back happens before your `RunE` reads those variables.

## Development

```bash
make test            # run all tests
make lint            # run golangci-lint
make check           # format + lint + test + coverage-checker self-test + 95% floor (pre-commit gate)
make bump            # tag next semver with svu and push; the tag push publishes the GitHub release
```

The e2e suite in [`tests/e2e/`](tests/e2e/README.md) builds a consumer program wired like the example above and runs it as a subprocess. Contributing guide: [AGENTS.md](AGENTS.md) (`CONTRIBUTING.md` resolves to it).
