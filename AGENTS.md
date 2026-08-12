# AGENTS.md

This file provides guidance to AI coding agents when working with code in this
repository. It is the canonical agent instructions file — `CLAUDE.md`,
`GEMINI.md`, and `.cursorrules` are symlinks to it (frostyard/core ADR-0018).

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

## Documentation

**update documentation** After any change to source code, update relevant documentation in AGENTS.md, README.md and the `docs/` tree. A task is not complete without reviewing and updating relevant documentation.

**docs/ tree** All repository documentation lives in the single `docs/` tree, in frostyard/core's four-category shape per [frostyard/core ADR-0025](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md) (formerly the separate `yeti/` AI-docs directory): `docs/adr/` (why — repo-local decisions), `docs/design/` (how it fits together), `docs/specs/` (exact contracts), `docs/plans/` (order of work), indexed in [docs/README.md](docs/README.md). [docs/design/overview.md](docs/design/overview.md) is the entry point — read it for codebase context before performing tasks. New repo-local decisions get an ADR in `docs/adr/` (start from its `TEMPLATE.md`); org-wide decisions belong in frostyard/core — see [docs/org-adrs.md](docs/org-adrs.md). Write these docs to be maximally useful to an AI agent understanding the codebase — detailed architecture, patterns, and decision rationale rather than user-facing guides.
