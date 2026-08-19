# Documentation

Docs are split by the question they answer (frostyard/core's four-category
shape, [core ADR-0025](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md)):

| Directory | Question | Contents |
| --- | --- | --- |
| [adr/](adr/) | **Why** did we choose this? | Repo-local Architecture Decision Records — immutable once accepted; superseded, never edited. Org-wide decisions live in frostyard/core — see [org-adrs.md](org-adrs.md) |
| [design/](design/) | **How** does it fit together? | Living documents describing the current architecture |
| [specs/](specs/) | **What exactly** is the contract? | Precise, testable interface definitions |
| [plans/](plans/) | **When/in what order** do we build? | Roadmaps and phase plans; updated as work lands |

## Index

### Decisions (ADRs)

- [0001 — ACMM conformance via canonical aliases](adr/0001-acmm-conformance-via-canonical-aliases.md)
  — `AGENTS.md` canonical, committed relative symlinks at the ACMM paths,
  real trees for directory criteria, the docs-integrity gate, the Fluent
  enrollment surface (`policies/agent-governance.json`), and the GoReleaser
  release flow behind `make bump`

Org-wide decisions binding this repo are listed in [org-adrs.md](org-adrs.md).

### Design

- [Overview](design/overview.md) — purpose, architecture, source file
  details, key patterns, configuration, CI (the entry-point doc)
- [Quality loop](design/quality-loop.md) — declare → review → gate → learn →
  observe, wired to `ci.yml`, `docs-gate`, `.memory/`, and the release flow
  (`docs/quality.md` resolves here)

### Specs

- [PR acceptance metric](specs/pr-acceptance-metric.md) — the monthly
  acceptance-rate definition and rules (`docs/metrics.md` resolves here)
- [PR review rubric](specs/pr-review-rubric.md) — the rows every review
  applies (`docs/review-rubric.md` resolves here)

### Plans

*(none yet)*

## Conventions

- **New docs start from their category's `TEMPLATE.md`** (in each directory).
- New decision → new ADR with the next number; if it reverses an old one, mark
  the old one `Superseded by NNNN` rather than editing it. Decisions that bind
  more than this repo become ADRs in
  [frostyard/core](https://github.com/frostyard/core/tree/main/docs/adr) plus
  a line in [org-adrs.md](org-adrs.md).
- Design docs are updated in place to always reflect reality.
- Specs change only alongside the code that implements them.
- Cross-links between categories are mandatory in both directions.
- Adding a doc means adding it to the index above; `node scripts/check-docs.mjs`
  (the `docs-gate` CI job) fails on an unindexed doc, a dead relative link,
  a broken alias, or drift between the workflow and documented CI job inventories.
- Conformance aliases (`docs/metrics.md`, `docs/review-rubric.md`,
  `docs/quality.md`) are symlinks registered in
  [ADR-0001](adr/0001-acmm-conformance-via-canonical-aliases.md); they are
  not docs, are not indexed, and are never edited — their targets are.
