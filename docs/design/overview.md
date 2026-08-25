# clix — CLI Convenience Module

## Purpose

`github.com/frostyard/clix` is a shared Go module that provides standardized CLI scaffolding for Frostyard command-line tools. It wraps `charm.land/fang/v2` and `spf13/cobra` to give every tool consistent version strings, common flags (`--json`, `--verbose`, `--dry-run`, `--silent`), JSON output helpers, and a reporter factory — so individual CLIs only need to define their own commands and business logic.

## Architecture

Single flat package (`package clix`) with four source files and matching test files:

```
clix.go          — App struct, Run(), RunContext(), VersionString()
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
| `charm.land/fang/v2` v2.0.1 | Command execution with version injection and signal handling |
| `spf13/cobra` v1.10.2 | Command tree and flag parsing |
| `spf13/viper` v1.21.0 | Optional config binding via `BindViper()` |
| `frostyard/std` v0.2.2 (`reporter` subpackage) | `Reporter` interface and concrete implementations (`TextReporter`, `JSONReporter`, `NoopReporter`) |

### Data Flow

```
main() creates App{} with build-time metadata
  └─ App.Run(rootCmd)
       ├─ reject a nil root command
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
1. Rejects a nil root command with `clix: Run: root command is nil`
2. Fills zero-value fields with defaults
3. Calls `registerFlags(cmd)` to add common persistent flags; if that returns an error, `Run` returns it before executing anything
4. Delegates to `fang.Execute()` with version string and `SIGINT`/`SIGTERM` signal handling

**`App.RunContext(ctx, cmd)`** is `Run` with a caller-supplied `context.Context`: `Run(cmd)` is implemented as `RunContext(context.Background(), cmd)`, and `RunContext` performs the same nil-check, `defaults()`, and `registerFlags(cmd)` steps before calling `fang.Execute(ctx, cmd, ...)` with the given `ctx` instead of a hardcoded background context (the `SIGINT`/`SIGTERM` handling `fang.WithNotifySignal` installs is unchanged). Use it to bound a run with a deadline/timeout or let a supervising process (a test, a parent command, an orchestrator) cancel execution; `fang.Execute` threads `ctx` onto `cmd.Context()`, so a `RunE` observes cancellation via `cmd.Context().Err()`. Pinned by `TestRunContextPropagatesCancellation` in `clix_test.go`.

### flags.go — Common Flags

Four package-level boolean variables are populated by cobra flag parsing:

| Variable | Flag | Short | Default | Description |
|---|---|---|---|---|
| `JSONOutput` | `--json` | — | `false` | Enable JSON output mode |
| `Verbose` | `--verbose` | `-v` | `false` | Enable verbose output |
| `DryRun` | `--dry-run` | `-n` | `false` | Dry run (no side effects) |
| `Silent` | `--silent` | `-s` | `false` | Suppress all progress output |

**`registerFlags(cmd)`** (unexported) adds these as persistent flags on the root command. Called automatically by `App.Run()`. The four names and three shorthands (`--json`, `--verbose`/`-v`, `--dry-run`/`-n`, `--silent`/`-s`) are reserved across the whole command tree: before registering, `checkCommandTree` walks the root and every descendant from `(*cobra.Command).Commands()` recursively and checks each command's persistent *and* local flag sets (cobra merges a command's local flags with every ancestor's persistent flags at parse time — a colliding shorthand panics in pflag when that command runs, a colliding name silently shadows clix's flag so its package variable is never set for that command). It returns `clix: root command already defines flag --<name>` / `... shorthand -<c> (used by --<other>)` for the root and `clix: command "<command path>" already defines flag --<name>` / `... shorthand -<c> (used by --<other>)` for a subcommand, so `App.Run()` returns that error before anything executes instead of pflag panicking on redefinition. Pinned by `TestRegisterFlags_CommandTree` and `TestRunSubcommandCollisionReturnsBeforeExecuting` in `flags_test.go`.

`registerFlags` is safe to call again on the same root, so `App.Run`/`App.RunContext` can execute the same root command more than once. Each flag it registers carries a `clix:owned` pflag annotation; a later call whose lookup finds a flag with that annotation under the reserved name and shorthand (`ownedByClix`) reuses it instead of re-registering (which would panic in pflag as "flag redefined") or reporting it as a collision against itself. The reuse check applies only when `cmd == root`, the one command `registerFlags` ever writes to, so a subcommand that happens to define a same-named/shorthand flag is still a genuine collision. This matters because cobra's `mergePersistentFlags` copies a root's persistent flags into its own local flag set the first time it executes, so after one run the same `*pflag.Flag` — annotation included, since both flag sets hold the identical pointer — turns up under both `cmd.PersistentFlags()` and `cmd.Flags()`. Pinned by `TestRegisterFlags_SecondCallOnSameRootReusesFlags` in `flags_test.go` and `TestRunTwiceOnSameRootIsRepeatable` in `clix_test.go`.

**`BindViper(cmd)`** binds all four flags to viper keys (`json`, `verbose`, `dry-run`, `silent`). Optional — call in `PersistentPreRunE` if the consuming CLI uses viper for config. It resolves each flag through the unexported `lookupFlag(cmd, name)`: `cmd.Flags()` (holds merged persistent flags once parsing has run), then `cmd.PersistentFlags()`, then `cmd.InheritedFlags()` — so it works when cobra invokes the root's `PersistentPreRunE` with `cmd` set to an executing subcommand, whose only view of the clix flags is inherited (pinned by `TestBindViper_FromSubcommandPreRun`). A flag that resolves nowhere returns `clix: BindViper: --<name> is not registered on "<cmd>"; call App.Run on the root command first` instead of viper's bare `flag for "json" is nil` (pinned by `TestBindViper_UnregisteredFlagError`).

### output.go — JSON Output Helpers

**`var Stdout io.Writer`, `var Stderr io.Writer`** — package-level writer seams, `nil` by default. The unexported resolvers `stdout()` / `stderr()` return the override when set and otherwise the *current* `os.Stdout` / `os.Stderr` at call time; they are the only places in the package that read those globals. Every output path (`OutputJSON` including its fallback envelope, `OutputJSONError`, and the reporters built by `NewReporter`) goes through them, so tests in clix and in consuming CLIs capture output with a `bytes.Buffer` instead of swapping the process-global `os.Stdout` (which is process-wide and unsafe under parallel/race tests). The nil default is deliberate: consumers that already swap `os.Stdout` in tests keep working unchanged, byte for byte.

**`OutputJSON(data any) (bool, error)`** — If `JSONOutput` is true, writes `data` as indented JSON to stdout (via the `Stdout` seam) and returns `(true, nil)`. Returns `(false, nil)` when JSON mode is off. The boolean means a complete document reached the writer: `data` is encoded into a buffer *before* the writer is touched, so the failure kinds are told apart — if JSON encoding fails, a buffered fallback error envelope is written to stdout (preserving the "JSON was written" contract) and the encoding error is returned as `(true, err)`; if that fallback write fails or reports fewer bytes than the envelope length, the complete fallback did not reach the writer and the result is `(false, err)` with both the original encoding error and contextualized fallback-write error discoverable (`io.ErrShortWrite` identifies a short count); if the primary document write fails or reports fewer bytes than the document length (a closed pipe, full disk, or a failing `Stdout` seam), the complete document did not reach the writer, no fallback envelope is attempted, and `(false, "write JSON output: …")` is returned with `io.ErrShortWrite` discoverable for a short count (pinned by `TestOutputJSON_WriteError`, `TestOutputJSON_ShortWrite`, and `TestOutputJSON_EncodeErrorFallbackShortWrite`; `TestOutputJSON_SuccessBytesUnchanged` pins that the success bytes are exactly an indented `json.Encoder`'s: two-space indent, trailing newline). Typical usage:
```go
written, err := clix.OutputJSON(result)
if err != nil {
    return err
}
if written {
    return nil
}
// fall through to text output
```

**`OutputJSONError(message string, err error) error`** — Builds a structured error envelope (`error: true`, `message`, `details`) and writes it via `OutputJSON`, then returns an error for the caller to propagate. If `err` is non-nil, `details` contains `err.Error()` and the returned error wraps it via `fmt.Errorf`; if `err` is nil, `details` falls back to `message` and a plain `errors.New` is returned. If writing the envelope fails, the returned error joins the caller-facing command error with the `OutputJSON` failure so `errors.Is` can discover both causes.

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

### Concurrency
`concurrency_test.go` is the one test that drives clix's output paths from several goroutines at once: `TestOutputPathsUnderConcurrentUse` runs 8 goroutines × 4 iterations over `OutputJSON`, `OutputJSONError`, and `NewReporter` against a single writer. It pins the property that clix's output paths introduce no shared mutable state of their own beyond the documented `Stdout` / `Stderr` seams and the flag globals, so a consumer that serializes its own writer may call them from multiple goroutines and every JSON record still arrives whole. The test's only synchronization is a mutex-guarded `io.Writer` declared in the test file — replacing it with a bare `bytes.Buffer` makes `go test -race` report a data race, which is what keeps the `race` CI job from passing vacuously. Like every test that touches `Stdout` / `Stderr`, it neither calls `t.Parallel()` nor swaps `os.Stdout`.

### Silent > JSON > Text priority
The reporter factory and output helpers follow a consistent priority: `--silent` suppresses reporter/progress output (`NewReporter` returns `NoopReporter`), `--json` switches the reporter to structured stdout, and the default is human-readable text on stderr. Silent applies to the reporter, not to application data: `OutputJSON` ignores `Silent`, so `--json --silent` still emits the result document on stdout while omitting the JSON reporter record. This convention should be maintained in any new output paths.

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

Binding is not enough on its own: viper would hold the value while `JSONOutput`, `Verbose`, `DryRun`, and `Silent` — the variables clix and its consumers actually read — stayed at their defaults. So after binding each flag, `BindViper` writes a viper-sourced `true` back through the owning flag set (`FlagSet.Set`, which records that the value was set, unlike assigning the flag's `Value` directly).

Precedence is **command line > viper > default**. The write-back is skipped when `flag.Changed` is set, so a flag the user typed is never overwritten — an explicit `--json=false` beats `json: true` in a config file, and beats a `viper.Set` override too. (`viper.BindPFlag` already ranks a changed flag above the config layer; the `Changed` guard is what also keeps viper's explicit override layer from winning.)

It must be called from `PersistentPreRunE` rather than `RunE`, so the write-back happens before `RunE` reads those variables.

### Build variables

Set via ldflags in the consuming CLI's build:
```bash
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -I) -X main.builtBy=ci"
```

## CI

GitHub Actions CI (`.github/workflows/ci.yml`) runs on pushes to `main`, all PRs targeting `main`, and merge-queue (`merge_group`) branches, so the required checks report on the queue's own temporary branches (core ADR-0042). Seven independent jobs:

| Job | Purpose |
|---|---|
| **lint** | `golangci-lint` via `jdx/mise-action`, installing the release pinned in `mise.toml` and checksum-verified against `mise.lock` (core ADR-0043; bump the pin in a dedicated commit), configured by `.golangci.yml` (the same file `make lint` reads) |
| **security** | `govulncheck ./...` with `golang.org/x/vuln/cmd/govulncheck@v1.7.0` pinned (job-level `permissions: contents: read`, 15-minute timeout) — fails on a reachable vulnerability in the module graph |
| **test** | `go test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=github.com/frostyard/clix ./...` — unit tests plus the `tests/e2e/` suite, then `make test-coverage-check` and `make coverage-check` (95.0% statement-coverage floor on the `clix` package) |
| **race** | `go test -race ./...` — the whole suite under the race detector; `concurrency_test.go` supplies the concurrent workload that gives the job something to detect |
| **verify** | `go mod tidy` drift check, `go vet`, `gofmt` formatting check |
| **docs-gate** | `node scripts/check-docs.mjs` — docs-index coverage, relative-link integrity, symlink (alias) resolution, and CI job inventory agreement against `.coverage-thresholds.json` |
| **release-config** | One stable `Release config` context validates `.goreleaser.yaml` on the `~> v2` line: fork pull requests receive no org secrets, so they remove only the top-level `pro: true` marker into a temporary file and run the OSS checker over the shared config body; same-repository pull requests, pushes, scheduled/manual runs, and `merge_group` run the authoritative GoReleaser Pro check with `GORELEASER_KEY` |

Each concern runs as a separate job for clear failure signals in the GitHub UI. The Makefile `check` target remains for local pre-commit use. No build job — this is a library with no binary artifacts. The whole declare → review → gate → learn → observe loop is described in [quality-loop.md](quality-loop.md).

The Makefile lint target fails when `mise.toml` pins no `golangci-lint` or the binary is not installed (with a `mise install` hint), warns when the installed release differs from the `mise.toml` pin (so a local run is not silently linted by a different version than CI), and fails on lint errors when the binary is present.

## Release

`make bump` runs `make check`, refuses a dirty tree, tags the next semantic version with `svu next` (`.svu.yaml`: `v` prefix, conventional-commit derived), and pushes the tag. The tag push runs `.github/workflows/release.yml`, which runs GoReleaser Pro against `.goreleaser.yaml` with `builds: skip: true` — no binaries, archives, or packages, only a GitHub release whose notes are grouped by conventional-commit type. Consumers upgrade with `go get github.com/frostyard/clix@<tag>`. Rationale and the protected-boundary status of these files: [ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md), `policies/agent-governance.json`.

## Development

```bash
make test            # run all tests
make lint            # golangci-lint (fails with an install hint if missing)
make check           # fmt + lint + test + coverage floor (pre-commit gate)
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
