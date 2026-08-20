# Quality loop

Living document. Rationale:
[ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md).
Contracts: [specs/pr-review-rubric.md](../specs/pr-review-rubric.md),
[specs/pr-acceptance-metric.md](../specs/pr-acceptance-metric.md).
`docs/quality.md` is a conformance alias for this file (ADR-0001); this page
is also the quality dashboard.

[![CI](https://github.com/frostyard/clix/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/frostyard/clix/actions/workflows/ci.yml?query=branch%3Amain)

## Overview

How change quality is proposed, gated, observed, and learned from in this
repo. Changes produced with AI assistance follow the same loop as every
other contribution: they are never merged solely on the basis of generated
output, and [`policies/agent-governance.json`](../../policies/agent-governance.json)
bounds what an agent may do without review. One loop, five stations:

```
PR template ──► review rubric ──► CI gates ──► corrections ──► promotion
(checklist)     (spec)            (ci.yml,     (.memory/)      (AGENTS.md,
     ▲                             docs-gate)                    docs, skills)
     └────────────── acceptance metric (spec) observes the stream ─────────┘
```

## Design

- **Declare** — [.github/pull_request_template.md](../../.github/pull_request_template.md)
  makes every PR walk the build-gate and docs-housekeeping checklists and
  asks for a Conventional Commits title (the org squash-merges; svu derives
  the next version from the resulting `main` commit).
- **Review** — the [PR review rubric](../specs/pr-review-rubric.md) is the
  contract a review applies; the
  [review runbook](../../.github/prompts/review.prompt.md) is its
  task-shaped form for agents. Maintainers remain accountable for the merge
  decision.
- **Gate** — [.github/workflows/ci.yml](../../.github/workflows/ci.yml)
  runs on every PR, every push to `main`, and every merge-queue
  (`merge_group`) branch, one job per concern:
  - *Lint* (`golangci-lint-action`, installing the release pinned as
    `GOLANGCI_LINT_VERSION` in the `Makefile` — the single place to bump it;
    `make lint` warns when the local binary differs — configured by
    `.golangci.yml`, the same file `make lint` reads), *Security Scan*
    (`govulncheck ./...` with `golang.org/x/vuln/cmd/govulncheck@v1.7.0`
    pinned — fails on a reachable vulnerability anywhere in the module
    graph, including indirect modules Dependabot never proposes), *Unit Tests*
    (`go test -v -coverprofile=coverage.out -covermode=atomic
    -coverpkg=github.com/frostyard/clix ./...`, which includes
    the [e2e suite](../../tests/e2e/README.md) and the README example check
    (`readme_example_test.go` compiles README.md's `package main` block
    against the checkout and resolves every `clix.<Ident>` in every Go block
    against the package's exported names, so a renamed or removed exported
    symbol the documentation uses fails this required check), then `make
    test-coverage-check` and `make coverage-check` — the 95.0% total
    statement-coverage floor on the `clix` package measured across every
    test package, `scripts/check-coverage.sh`, ported from updex), *Race Detection*
    (`go test -race ./...`), *Verify* (`go mod tidy` drift, `go vet`,
    `gofmt`) — `make check` (fmt → lint → test → test-coverage-check →
    coverage-check) reproduces the local subset.
  - *Docs integrity* (`docs-gate`): `node scripts/check-docs.mjs` checks
    docs-index coverage, relative-link integrity, symlink resolution, and the
    documented CI job inventory against
    [.coverage-thresholds.json](../../.coverage-thresholds.json) — all 1.0,
    `never_relax: true` (the loop may tighten, never loosen).
  - *Release config* (`release-config`): one stable required context validates
    [`.goreleaser.yaml`](../../.goreleaser.yaml) before merge. GitHub withholds
    `GORELEASER_KEY` from fork pull requests, so that matrix branch removes only
    the top-level `pro: true` marker into a temporary config and runs the
    secret-independent OSS `goreleaser check` over the shared body.
    Same-repository pull requests, pushes to `main`, scheduled/manual runs, and
    `merge_group` retain the authoritative GoReleaser Pro check with the org
    key. A broken config therefore fails before an immutable tag exists without
    making the required job context disappear for contributors. `make
    release-check` runs the installed checker locally, skipping with a message
    when goreleaser is absent; it is deliberately not part of `make check`, so
    contributors need not install goreleaser.
  - The statement-coverage floor is 95.0% (`COVERAGE_MIN` and the script's
    default; `make coverage-check` after `make test`); there is no coverage
    service, and `make test-cover` still produces a local `coverage.html` for
    inspection. The floor may tighten, never loosen.
- **Learn** — corrections land in
  [.memory/corrections.jsonl](../../.memory/README.md) (append-only,
  five-field schema) and are promoted into `AGENTS.md`, docs, or skills;
  promotion is the only sanctioned duplication. Session continuity lives in
  [.claude/session-summary.md](../../.claude/session-summary.md).
- **Enforce mechanically** — [.claude/settings.json](../../.claude/settings.json)
  denies the forbidden acts at the tool layer: merging PRs (`gh pr merge`),
  approving own work (`gh pr review --approve`), publishing releases
  (`gh release`), and pushing to `main`;
  [`policies/agent-governance.json`](../../policies/agent-governance.json)
  states the same controls as machine-readable policy under frostyard/core's
  repository-surfaces contract v1 (deny by default; read/write/run-tests
  allowed; issues, PRs, and follow-ups review-required; workflows and the
  release configuration review-required at high risk).
- **Observe** — the [PR acceptance metric](../specs/pr-acceptance-metric.md)
  summarizes the stream; it informs, never gates.

## Release flow

`make bump` runs `make check`, refuses a dirty tree, tags the next semantic
version with `svu next` (`.svu.yaml`: `v` prefix, conventional-commit
derived, `v0` allowed), and pushes the tag. The tag push runs
[.github/workflows/release.yml](../../.github/workflows/release.yml), which
runs GoReleaser Pro against [.goreleaser.yaml](../../.goreleaser.yaml):
builds are skipped (clix is a library), so the only output is the GitHub
release with a changelog grouped by conventional-commit type. Consumers pick
the release up with `go get github.com/frostyard/clix@<tag>`.

## Human oversight

AI assistance is disclosed when the contribution process requires it.
Maintainers remain responsible for reviewing behavior, API compatibility,
test evidence, and documentation, and may request revisions or reject a
change regardless of automated results.

## Operational notes

Re-run every gate locally before pushing:

```
make check
node scripts/check-docs.mjs
```

Failure modes: a broken alias or missing index line fails docs-gate (fix the
canonical target or the index, never the alias); a lint finding after
pinning `.golangci.yml` means the gate was already red — fix the finding,
never loosen the config; an e2e failure that only reproduces in CI usually
means the `go` toolchain differs — the suite builds `tests/e2e/exampletool`
with the toolchain on `PATH`.

## References

- Rationale: [ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)
- Contracts: [specs/pr-review-rubric.md](../specs/pr-review-rubric.md),
  [specs/pr-acceptance-metric.md](../specs/pr-acceptance-metric.md)
- Policy: [`policies/agent-governance.json`](../../policies/agent-governance.json)
- Entry point: [design/overview.md](overview.md)
