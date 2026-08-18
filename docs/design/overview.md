# clix — CLI Convenience Module

## Purpose

`github.com/frostyard/clix` is a shared Go module that provides standardized CLI scaffolding for Frostyard command-line tools. It wraps `charmbracelet/fang` and `spf13/cobra` to give every tool consistent version strings, common flags (`--json`, `--verbose`, `--dry-run`, `--silent`), JSON output helpers, and a reporter factory — so individual CLIs only need to define their own commands and business logic.

## Architecture

Single flat package (`package clix`) with four source files and matching test files:

```
clix.go          — App struct, Run(), VersionString()
clix_test.go
flags.go         — Package-level flag variables, registerFlags(), BindViper()
flags_test.go
output.go        — OutputJSON(), OutputJSONError(), Stdout/Stderr writer seams
output_test.go
reporter.go      — NewReporter() factory
reporter_test.go
```

There are no subpackages or internal directories apart from the end-to-end suite in `tests/e2e/` (a black-box test that builds `tests/e2e/exampletool`, a consumer program, and runs it as a subprocess — see [tests/e2e/README.md](../../tests/e2e/README.md)). CI is defined in `.github/workflows/ci.yml`; releases in `.github/workflows/release.yml`.

### Dependencies

| Direct dependency | Role |
|---|---|
| `charmbracelet/fang` v1.0.0 | Command execution with version injection and signal handling |
| `spf13/cobra` v1.10.2 | Command tree and flag parsing |
| `spf13/viper` v1.21.0 | Optional config binding via `BindViper()` |
| `frostyard/std` v0.2.0 (`reporter` subpackage) | `Reporter` interface and concrete implementations (`TextReporter`, `JSONReporter`, `NoopReporter`) |

### Data Flow

```
main() creates App{} with build-time metadata
  └─ App.Run(rootCmd)
       ├─ registerFlags(cmd)          ← adds --json, --verbose, --dry-run, --silent as persistent flags
       └─ fang.Execute(cmd, ...)      ← runs the cobra command tree with version + signal handling
            └─ Command handlers use:
                 ├─ clix.OutputJSON() / clix.OutputJSONError()   ← structured JSON to stdout
                 └─ clix.NewReporter()                            ← progress reporting
```

## Source File Details

### clix.go — App and Execution

**`App` struct** holds build-time metadata injected via ldflags:

| Field | Default | Description |
|---|---|---|
| `Version` | `"dev"` | Semantic version |
| `Commit` | `"none"` | Git commit SHA |
| `Date` | `"unknown"` | Build date |
| `BuiltBy` | `"local"` | Build system identifier |

**`App.VersionString()`** returns a formatted string:
`"1.2.3 (Commit: abc123) (Date: 2026-03-04) (Built by: ci)"`

**`App.Run(cmd)`** is the main entry point:
1. Fills zero-value fields with defaults
2. Calls `registerFlags(cmd)` to add common persistent flags
3. Delegates to `fang.Execute()` with version string and `SIGINT`/`SIGTERM` signal handling

### flags.go — Common Flags

Four package-level boolean variables are populated by cobra flag parsing:

| Variable | Flag | Short | Default | Description |
|---|---|---|---|---|
| `JSONOutput` | `--json` | — | `false` | Enable JSON output mode |
| `Verbose` | `--verbose` | `-v` | `false` | Enable verbose output |
| `DryRun` | `--dry-run` | `-n` | `false` | Dry run (no side effects) |
| `Silent` | `--silent` | `-s` | `false` | Suppress all progress output |

**`registerFlags(cmd)`** (unexported) adds these as persistent flags on the root command. Called automatically by `App.Run()`.

**`BindViper(cmd)`** binds all four flags to viper keys (`json`, `verbose`, `dry-run`, `silent`). Optional — call in `PersistentPreRunE` if the consuming CLI uses viper for config.

### output.go — JSON Output Helpers

**`var Stdout io.Writer`, `var Stderr io.Writer`** — package-level writer seams, `nil` by default. The unexported resolvers `stdout()` / `stderr()` return the override when set and otherwise the *current* `os.Stdout` / `os.Stderr` at call time; they are the only places in the package that read those globals. Every output path (`OutputJSON` including its fallback envelope, `OutputJSONError`, and the reporters built by `NewReporter`) goes through them, so tests in clix and in consuming CLIs capture output with a `bytes.Buffer` instead of swapping the process-global `os.Stdout` (which is process-wide and unsafe under parallel/race tests). The nil default is deliberate: consumers that already swap `os.Stdout` in tests keep working unchanged, byte for byte.

**`OutputJSON(data any) (bool, error)`** — If `JSONOutput` is true, writes `data` as indented JSON to stdout (via the `Stdout` seam) and returns `(true, nil)`. Returns `(false, nil)` when JSON mode is off. If JSON encoding fails, a fallback error envelope is written to stdout (preserving the "JSON was written" contract) and the encoding error is returned as `(true, err)`. Typical usage:
```go
if written, err := clix.OutputJSON(result); written {
    return err
}
// fall through to text output
```

**`OutputJSONError(message string, err error) error`** — Builds a structured error envelope (`error: true`, `message`, `details`) and writes it via `OutputJSON`, then returns an error for the caller to propagate. If `err` is non-nil, `details` contains `err.Error()` and the returned error wraps it via `fmt.Errorf`; if `err` is nil, `details` falls back to `message` and a plain `errors.New` is returned. Any encoding error from `OutputJSON` is silently discarded (the caller's error takes priority).

### reporter.go — Reporter Factory

**`NewReporter() reporter.Reporter`** selects the reporter implementation based on flag state:

| Priority | Condition | Reporter | Output destination |
|---|---|---|---|
| 1 (highest) | `Silent == true` | `NoopReporter` | none |
| 2 | `JSONOutput == true` | `JSONReporter` | stdout |
| 3 (default) | neither | `TextReporter` | stderr |

Silent always wins over JSON — this is explicitly tested.

Text reporter writes to stderr to keep stdout clean for data/JSON output. Both destinations are resolved through the `Stdout` / `Stderr` seams in `output.go` at the time `NewReporter()` is called.

## Key Patterns

### Package-level flag state
Flags are stored as package-level `var` globals (`JSONOutput`, `Verbose`, `DryRun`, `Silent`). This means consuming code can read flag values directly (e.g., `if clix.Verbose { ... }`) without passing config structs around. The tradeoff is that tests must reset these variables and use fresh `cobra.Command` instances to avoid state leakage.

### Test isolation
Every test creates a new `cobra.Command` and explicitly resets package-level flag variables with `defer` cleanup. Tests that capture output assign a `bytes.Buffer` to `Stdout` / `Stderr` and `defer` a reset to nil — never swap `os.Stdout`, which is process-wide and unsafe under `t.Parallel` / `-race`. Note that `Stdout` / `Stderr` are still shared package-level state, so tests that set them should not run in parallel unless they coordinate access. The older `os.Pipe()` tests in `output_test.go` are kept deliberately: they prove the nil default still honors a swapped `os.Stdout`, which downstream CLIs rely on. JSON output tests unmarshal and validate individual fields.

### Silent > JSON > Text priority
The reporter factory and output helpers follow a consistent priority: `--silent` suppresses everything, `--json` switches to structured output, and the default is human-readable text on stderr. This convention should be maintained in any new output paths.

### Build-time injection
`App` fields are designed to be set via Go ldflags (`-X main.version=...`). The `defaults()` method ensures the tool runs gracefully during development without ldflags.

## Configuration

### Flags (registered automatically by `App.Run()`)

| Flag | Type | Description |
|---|---|---|
| `--json` | bool | JSON output mode |
| `--verbose` / `-v` | bool | Verbose output |
| `--dry-run` / `-n` | bool | No-op mode |
| `--silent` / `-s` | bool | Suppress progress output |

### Viper integration (optional)

Call `clix.BindViper(cmd)` in a `PersistentPreRunE` to bind the four flags to viper keys. This allows them to be set via config files or environment variables through viper's standard mechanisms.

### Build variables

Set via ldflags in the consuming CLI's build:
```bash
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -I) -X main.builtBy=ci"
```

## CI

GitHub Actions CI (`.github/workflows/ci.yml`) runs on pushes to `main` and all PRs targeting `main`. Five independent jobs:

| Job | Purpose |
|---|---|
| **lint** | `golangci-lint` via `golangci-lint-action@v9`, configured by `.golangci.yml` (the same file `make lint` reads) |
| **test** | `go test -v ./...` — unit tests plus the `tests/e2e/` suite |
| **race** | `go test -race -short ./...` |
| **verify** | `go mod tidy` drift check, `go vet`, `gofmt` formatting check |
| **docs-gate** | `node scripts/check-docs.mjs` — docs-index coverage, relative-link integrity, symlink (alias) resolution against `.coverage-thresholds.json` |

Each concern runs as a separate job for clear failure signals in the GitHub UI. The Makefile `check` target remains for local pre-commit use. No build job — this is a library with no binary artifacts. The whole declare → review → gate → learn → observe loop is described in [quality-loop.md](quality-loop.md).

The Makefile lint target gracefully skips when `golangci-lint` is not installed (for local dev) but properly fails on lint errors when the binary is present (for CI).

## Release

`make bump` runs `make check`, refuses a dirty tree, tags the next semantic version with `svu next` (`.svu.yaml`: `v` prefix, conventional-commit derived), and pushes the tag. The tag push runs `.github/workflows/release.yml`, which runs GoReleaser Pro against `.goreleaser.yaml` with `builds: skip: true` — no binaries, archives, or packages, only a GitHub release whose notes are grouped by conventional-commit type. Consumers upgrade with `go get github.com/frostyard/clix@<tag>`. Rationale and the protected-boundary status of these files: [ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md), `policies/agent-governance.json`.

## Development

```bash
make test            # run all tests
make lint            # golangci-lint (skips if not installed)
make check           # fmt + lint + test (pre-commit gate)
make test-cover      # tests with coverage report (coverage.html)
make fmt             # format Go source files
make tidy            # go mod tidy
make clean           # remove generated artifacts
make bump            # tag next semver with svu and push (the tag push publishes the release)
make help            # list all targets
go test -v -run TestName ./...  # single test
go test -v ./tests/e2e/...      # end-to-end suite alone
node scripts/check-docs.mjs     # docs-integrity gate
```
