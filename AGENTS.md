# frostyard/clix

`github.com/frostyard/clix` is a CLI convenience module for Frostyard tools.
It wraps charm.land/fang/v2 and spf13/cobra with standardized version
injection, common flags, JSON output helpers, and a reporter factory, so
individual CLIs only define their own commands and business logic. Start at
[docs/README.md](docs/README.md); read
[docs/design/overview.md](docs/design/overview.md) for codebase context before
performing tasks.

This file (`AGENTS.md`) is the CANONICAL agent instructions **and** the
contributing guide — `CLAUDE.md`, `GEMINI.md`, `CONTRIBUTING.md`,
`.cursorrules`, and `.github/copilot-instructions.md` are symlinks to it, and
`.claude/skills` symlinks to `.agents/skills/`
([ADR-0001](docs/adr/0001-acmm-conformance-via-canonical-aliases.md); pattern
from
[frostyard/core ADR-0002](https://github.com/frostyard/core/blob/main/docs/adr/0002-agent-portable-instruction-surface.md),
[ADR-0018](https://github.com/frostyard/core/blob/main/docs/adr/0018-org-wide-agent-instruction-and-knowledge-surfaces.md),
and
[ADR-0029](https://github.com/frostyard/core/blob/main/docs/adr/0029-acmm-conformance-via-canonical-aliases.md)).
Edit only the canonical paths; keep content tool-agnostic. Conformance alias
symlinks are listed in ADR-0001 — edit their canonical targets, never the
aliases.

## Skills (follow these for common tasks)

Step-by-step procedures live in [.agents/skills/](.agents/skills/) (synced
from frostyard/core — edit them there, not here); follow them rather than
improvising, whichever agent you are:

- **Structuring, building, testing, or releasing this repo the frostyard Go
  way** → [.agents/skills/frostyard-go-repo/SKILL.md](.agents/skills/frostyard-go-repo/SKILL.md)
- **Maintaining the `docs/` tree (four-category shape, index, migrations)**
  → [.agents/skills/frostyard-repo-docs/SKILL.md](.agents/skills/frostyard-repo-docs/SKILL.md)
- **Hive ACMM conformance / agentic-fleet prerequisites** →
  [.agents/skills/frostyard-acmm-conformance/SKILL.md](.agents/skills/frostyard-acmm-conformance/SKILL.md)
  — canonical aliases per ADR-0001, never duplicated content

## Getting started

### Prerequisites

- **Go 1.26.7** (`go.mod`'s `toolchain go1.26.7` line is the only Go pin;
  CI, `mise`, and Snowcat workers all read it)
- `make`
- [`mise`](https://mise.jdx.dev/) to provision every other pinned tool:
  `mise install` reads [`mise.toml`](mise.toml) and verifies each download
  against [`mise.lock`](mise.lock) (core ADR-0043). Today that is
  [`golangci-lint`](https://golangci-lint.run/) v2 for `make lint`
  (configured by [`.golangci.yml`](.golangci.yml)); the Makefile reads the
  pin from `mise.toml`, fails with `mise install` when the binary is absent,
  warns when the installed version differs, and fails on findings; CI
  installs the same lock through `jdx/mise-action`
- [`svu`](https://github.com/caarlos0/svu) for `make bump`; Node ≥ 20 for
  the docs-integrity gate

clix is a library: there is no binary to build or install. Building and
testing needs nothing beyond the Go toolchain (the e2e suite builds its own
throwaway consumer program with `go build`).

## Commands

```bash
make test            # run all tests (unit tests + tests/e2e/); writes coverage.out for the clix package
make lint            # run golangci-lint (.golangci.yml; fails if not installed,
                     # warns if not the golangci-lint pin in mise.toml)
make check           # fmt + lint + test + coverage floor (pre-commit gate)
make coverage-check  # enforce the 95.0% statement-coverage floor on coverage.out (scripts/check-coverage.sh)
make test-coverage-check   # self-test scripts/check-coverage.sh with fixture profiles
make test-cover      # tests with coverage report (coverage.html)
make fmt             # format Go source files
make tidy            # go mod tidy
make clean           # remove generated artifacts
make bump            # tag next semver with svu and push (the tag push publishes the release)
make help            # list all targets
go test -v -run TestName ./...  # run a single test
go test -v ./tests/e2e/...      # end-to-end suite alone
node scripts/check-docs.mjs     # docs-integrity gate (index, links, aliases, CI job inventory)
```

Run `make check` before opening a pull request; CI additionally checks
`go mod tidy` drift and `go vet` (the `verify` job) and the docs gate.

## Architecture

Single flat package `clix` with four source files:

- **clix.go** — `App` struct with `Run()` and `VersionString()`. Wires up fang.Execute with version string and signal handling; `Run()` returns an error (never panics) if the root command or any subcommand (any depth) already defines one of clix's reserved flags (`--json`, `--verbose`/`-v`, `--dry-run`/`-n`, `--silent`/`-s`).
- **flags.go** — Package-level flag variables (`JSONOutput`, `Verbose`, `DryRun`, `Silent`), registration on cobra commands, and optional `BindViper()`.
- **output.go** — `OutputJSON()` and `OutputJSONError()` helpers for standardized JSON output to stdout, plus the `Stdout`/`Stderr` writer seams (nil = current `os.Stdout`/`os.Stderr`) that every output path resolves through.
- **reporter.go** — `NewReporter()` factory that returns NoopReporter (`--silent`), TextReporter, or JSONReporter (`--json`) based on flags. Silent takes priority over JSON.

Plus the end-to-end suite: **tests/e2e/** — `e2e_test.go` builds
`tests/e2e/exampletool` (a consumer program wired the way `README.md`
prescribes) and runs it as a subprocess, asserting `--version`, the
stdout/stderr split, Silent > JSON > Text, the JSON error envelope, and exit
codes ([tests/e2e/README.md](tests/e2e/README.md)).

## Conventions

- Go 1.26; use modern Go syntax (range-over-int, omitzero, etc.)
- One test file per source file, standard `testing` package only; the one
  cross-cutting exception is `concurrency_test.go`, which drives the output
  paths from several goroutines so the `race` CI job has a real workload
- Tests use fresh `cobra.Command` per test to avoid flag state leakage
- Tests capture output via `bytes.Buffer` assigned to `clix.Stdout` / `clix.Stderr` (with `defer` reset to nil), never by swapping `os.Stdout`; JSON tests unmarshal and validate fields. The `os.Pipe` tests in `output_test.go` stay as the compatibility proof that the nil default still honors a swapped `os.Stdout`.
- clix is a library every Frostyard CLI depends on: exported identifiers
  keep doc comments; a breaking change to `App`, the flag variables,
  `OutputJSON`/`OutputJSONError`, `NewReporter`, or `BindViper` is
  deliberate, called out in the PR, marked `feat!`/`fix!`, and reflected in
  `README.md` and `docs/design/overview.md`
- Silent > JSON > Text: `--silent` suppresses reporter/progress output
  (`NewReporter` returns `NoopReporter`) but application data stays on stdout —
  `--json --silent` omits the JSON reporter record yet `OutputJSON` still emits
  the result document; `--json` switches the reporter to structured stdout,
  default is text on stderr — keep this on any new output path

## Commits & Pull Requests

Commit messages **and pull request titles** use
[Conventional Commits](https://www.conventionalcommits.org/):
`type(scope): summary`, e.g. `feat(flags): add --quiet`,
`fix(output): keep envelope on encode error`, `docs(agents): …`. Allowed
types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`,
`perf`, `style`, `revert` (optional `(scope)`) — `svu` derives the next
version from them. The repository squash-merges, so the PR title — or, for a
single-commit PR, that commit's subject — becomes the `main` commit that svu
versions and the release changelog groups by. See
[Pull requests](#pull-requests) below for the full process.

## Release

`make bump` runs `make check`, refuses a dirty tree, tags the next semantic
version with `svu next` (`.svu.yaml`: `v` prefix, conventional-commit
derived, `v0` allowed), and pushes the tag. The tag push runs
`.github/workflows/release.yml`, which runs GoReleaser Pro against
`.goreleaser.yaml`: builds are skipped (clix is a library — no binaries,
archives, or packages), so the workflow publishes the GitHub release with a
changelog grouped by conventional-commit type. Consumers upgrade with
`go get github.com/frostyard/clix@<tag>`. Never run `make bump`, `gh release`,
or push a tag as an agent — releasing is a maintainer act
(`.claude/settings.json` denies `gh release`; `policies/agent-governance.json`
marks `.goreleaser.yaml`, `.svu.yaml`, and `.github/workflows/release.yml`
review-required at high risk).

## Documentation

**update documentation** After any change to source code, update relevant documentation in AGENTS.md, README.md and the `docs/` tree. A task is not complete without reviewing and updating relevant documentation.

**docs/ tree** All repository documentation lives in the single `docs/` tree, in frostyard/core's four-category shape per [frostyard/core ADR-0025](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md) (formerly the separate `yeti/` AI-docs directory): `docs/adr/` (why — repo-local decisions), `docs/design/` (how it fits together), `docs/specs/` (exact contracts), `docs/plans/` (order of work), indexed in [docs/README.md](docs/README.md). [docs/design/overview.md](docs/design/overview.md) is the entry point — read it for codebase context before performing tasks. New repo-local decisions get an ADR in `docs/adr/` (start from its `TEMPLATE.md`); org-wide decisions belong in frostyard/core — see [docs/org-adrs.md](docs/org-adrs.md). Write these docs to be maximally useful to an AI agent understanding the codebase — detailed architecture, patterns, and decision rationale rather than user-facing guides.

**docs-integrity gate** `node scripts/check-docs.mjs` (the `docs-gate` CI
job) fails on any unindexed doc in the four categories, any dead relative
link in `AGENTS.md`/`README.md`/`docs/`/`.github/prompts/`/`.memory/`/
`tests/e2e/`, any broken/repo-escaping symlink, or drift between the CI job
inventory in `.github/workflows/ci.yml` and `docs/design/overview.md`; thresholds in
`.coverage-thresholds.json` are all 1.0 with `never_relax: true`.

**conformance aliases** Conformance alias symlinks are listed in
[ADR-0001](docs/adr/0001-acmm-conformance-via-canonical-aliases.md) — edit
their canonical targets, never the aliases. `docs/review-rubric.md`,
`docs/metrics.md`, and `docs/quality.md` resolve to
[docs/specs/pr-review-rubric.md](docs/specs/pr-review-rubric.md),
[docs/specs/pr-acceptance-metric.md](docs/specs/pr-acceptance-metric.md), and
[docs/design/quality-loop.md](docs/design/quality-loop.md).

**session handoffs** Use `.claude/session-summary.md` for concise context
needed to continue unfinished work in a later session. Fold durable
architecture decisions and non-obvious lessons into the right `docs/` page,
or drop them in the `.memory/` inbox (the single learnings inbox, append-only
`corrections.jsonl`, drained into `docs/`).

## Pull requests

1. Branch off `main`; the org squash-merges, so never stack on another PR's
   branch.
2. Keep changes focused; unrelated fixes belong in a separate PR.
3. Use Conventional Commits for commit messages **and the pull request
   title** (see [Commits & Pull Requests](#commits--pull-requests)).
4. Run `make check` and `node scripts/check-docs.mjs` and make sure they
   pass; `go mod tidy` must leave `go.mod`/`go.sum` unchanged.
5. Update the documentation and add tests for your change (a `_test.go`
   next to the source file; `tests/e2e/` for behavior only a built binary
   shows).
6. Open the PR with a description of what changed and why, and link any
   related issue ([`.github/pull_request_template.md`](.github/pull_request_template.md)
   walks through it).

Before requesting review, check the change against the
[pull request review rubric](docs/specs/pr-review-rubric.md). Reviewers apply
its rows to every pull request and label findings blocking / non-blocking /
question / nit, explaining the impact of each finding and suggesting a
concrete resolution; the task-shaped form is
[`.github/prompts/review.prompt.md`](.github/prompts/review.prompt.md).
Automated feedback is advisory: never approve changes, weaken required
checks, or claim verification passed without evidence from the pull request,
and never merge, approve, or release your own work (`.claude/settings.json`
denies these at the tool layer).

CI runs on every pull request, on pushes to `main`, and on merge-queue
(`merge_group`) branches (`.github/workflows/ci.yml`) and must pass:
lint (the golangci-lint release pinned in
[`mise.toml`](mise.toml) with its checksums in [`mise.lock`](mise.lock),
installed by `jdx/mise-action`, with `.golangci.yml`), security scan (`govulncheck ./...`,
pinned at `golang.org/x/vuln/cmd/govulncheck@v1.7.0`, failing on any
reachable vulnerability in the module graph), unit tests (`go test -v ./...`,
including `tests/e2e/`), race-detector tests, verification (`go mod tidy`
cleanliness, `go vet`, `gofmt`), docs integrity
(`scripts/check-docs.mjs`), and release config (`goreleaser check` over
`.goreleaser.yaml`, so a broken release configuration fails before merge
rather than on a tag push; `make release-check` runs the same check locally
and skips when goreleaser is not installed). The whole loop — declare, review, gate, learn,
observe — is described in [docs/design/quality-loop.md](docs/design/quality-loop.md).

## Repository boundary

[`policies/agent-governance.json`](policies/agent-governance.json) is this
repository's canonical agent-governance surface under the frostyard/core
repository-surfaces contract v1; Fluent reads it (from GitHub, at the
observed default-branch head) when enrolling this repository in its fleet,
alongside `AGENTS.md`, `.agents/skills/`, and `docs/README.md` — all four
are real content, never aliases. Deny by default; read, write, and run-tests
allowed; issues, pull requests, and follow-ups review-required; workflows
(`.github/workflows/**`) and the release configuration (`.goreleaser.yaml`,
`.svu.yaml`, `.github/workflows/release.yml`) are review-required at high
risk. Change it only alongside the matching ADR or design change.

## Org-wide decisions

Org-level conventions this repo follows are recorded as ADRs in
frostyard/core — see [docs/org-adrs.md](docs/org-adrs.md) for the list that
binds this repo. Change the ADR (in core) before changing behavior it covers.

## License

By contributing, you agree that your contributions are licensed under the
[MIT License](LICENSE) that covers this project.
