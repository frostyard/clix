# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`github.com/frostyard/clix` is a CLI convenience module for Frostyard tools. It wraps charmbracelet/fang and spf13/cobra with standardized version injection, common flags, JSON output helpers, and reporter factory.

## Commands

```bash
make test            # run all tests
make lint            # run golangci-lint
make check           # fmt + lint + test (pre-commit gate)
make bump            # tag next semver with svu and push
go test -v -run TestName ./...  # run a single test
```

## Architecture

Single flat package `clix` with four source files:

- **clix.go** — `App` struct with `Run()` and `VersionString()`. Wires up fang.Execute with version string and signal handling.
- **flags.go** — Package-level flag variables (`JSONOutput`, `Verbose`, `DryRun`, `Silent`), registration on cobra commands, and optional `BindViper()`.
- **output.go** — `OutputJSON()` and `OutputJSONError()` helpers for standardized JSON output to stdout.
- **reporter.go** — `NewReporter()` factory that returns NoopReporter (`--silent`), TextReporter, or JSONReporter (`--json`) based on flags. Silent takes priority over JSON.

## Conventions

- Go 1.26; use modern Go syntax (range-over-int, omitzero, etc.)
- One test file per source file, standard `testing` package only
- Tests use fresh `cobra.Command` per test to avoid flag state leakage
- Tests capture output via `bytes.Buffer`; JSON tests unmarshal and validate fields
