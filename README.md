<!-- readme-kind: library -->
# clix

[![CI](https://github.com/frostyard/clix/actions/workflows/ci.yml/badge.svg)](https://github.com/frostyard/clix/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/frostyard/clix.svg)](https://pkg.go.dev/github.com/frostyard/clix)

clix is a Go module of CLI scaffolding for frostyard command-line tools: it
wraps [charm.land/fang/v2](https://pkg.go.dev/charm.land/fang/v2) and
[spf13/cobra](https://github.com/spf13/cobra), so a tool defines only its own
commands and business logic and gets standardized version strings, 4 common
flags, JSON output helpers and a reporter factory.

**Status:** pre-release. Releases are `v0` tags, and the API can change between minor releases.

**Documentation:** [pkg.go.dev/github.com/frostyard/clix](https://pkg.go.dev/github.com/frostyard/clix)

## What it does

| Piece | What it gives you |
| --- | --- |
| [`App`, `Run`, `RunContext`](docs/specs/consumer-api.md#entry-points) | Build-time version, commit, date and builder metadata, and the entry point that registers the flags and executes the command tree. |
| [The 4 common flags](docs/specs/consumer-api.md#common-flags) | `--json`, `--verbose`/`-v`, `--dry-run`/`-n` and `--silent`/`-s`, reserved across the whole command tree, with a package-level variable each. |
| [`OutputJSON`, `OutputJSONError`](docs/specs/consumer-api.md#json-output) | The result document on stdout under `--json`, and the `error`, `message`, `details` envelope for a failure. |
| [`NewReporter`](docs/specs/consumer-api.md#reporter-priority) | Progress output as text (stderr), JSON (stdout) or nothing, by flag priority. |
| [`BindViper`](docs/specs/consumer-api.md#viper-integration) | The 4 flags bound to viper keys, with command line > viper > default precedence. |
| [Error rendering](docs/specs/consumer-api.md#error-rendering) | A multi-line error on its own lines, a flag kept in lower case, and plain text off a terminal. |
| [`Stdout`, `Stderr`](docs/specs/consumer-api.md#writer-seams) | Writer seams so a test captures output in a `bytes.Buffer` instead of swapping `os.Stdout`. |

## Install

```sh
go get github.com/frostyard/clix
```

## Use

```go
package main

import (
	"os"

	"github.com/frostyard/clix"
	"github.com/spf13/cobra"
)

var version = "dev" // set with -ldflags "-X main.version=1.2.3"

func main() {
	rootCmd := &cobra.Command{Use: "mytool", RunE: func(cmd *cobra.Command, args []string) error {
		clix.NewReporter().Message("doing work")
		_, err := clix.OutputJSON(map[string]string{"status": "ok"})
		return err
	}}
	app := clix.App{Version: version}
	if err := app.Run(rootCmd); err != nil {
		os.Exit(1)
	}
}
```

`App.Run` returns the command error, so the caller owns the process exit
code. Build with the version injected (`App` also takes `Commit`, `Date`
and `BuiltBy`):

```sh
go build -ldflags "-X main.version=1.2.3"
```

The flags, the reserved-flag rule, reporter priority, error rendering and the
Viper binding are in [spec: consumer API](docs/specs/consumer-api.md).

## Develop

```sh
mise install   # golangci-lint
make ci        # the canonical gate (what CI's Unit Tests job runs)
```

Design and invariants: [docs/design/overview.md](docs/design/overview.md).

## Where to go next

- [AGENTS.md](AGENTS.md): the rules for agents and contributors.
- [docs/README.md](docs/README.md): the decisions, designs, specs and plans.
- [SECURITY.md](SECURITY.md): report a vulnerability privately.
- [LICENSE](LICENSE): MIT.
