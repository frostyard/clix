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
  runs on every PR and push to `main`, one job per concern:
  - *Lint* (`golangci-lint-action`, configured by `.golangci.yml` — the same
    file `make lint` reads), *Unit Tests* (`go test -v ./...`, which includes
    the [e2e suite](../../tests/e2e/README.md)), *Race Detection*
    (`go test -race -short ./...`), *Verify* (`go mod tidy` drift, `go vet`,
    `gofmt`) — `make check` (fmt → lint → test) reproduces the local subset.
  - *Docs integrity* (`docs-gate`): `node scripts/check-docs.mjs` checks
    docs-index coverage, relative-link integrity, and symlink resolution
    against [.coverage-thresholds.json](../../.coverage-thresholds.json) —
    all 1.0, `never_relax: true` (the loop may tighten, never loosen).
  - There is no statement-coverage floor and no coverage service today;
    `make test-cover` produces a local `coverage.html` for inspection.
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
