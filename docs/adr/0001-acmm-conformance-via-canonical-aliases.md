# 0001 — ACMM conformance via canonical aliases

- **Status:** Accepted
- **Date:** 2026-08-18

## Context

The Hive ACMM evaluation grades repositories by checking that fixed paths
exist — test suites, templates, style configs, rubrics, metrics, agent-safety
settings. Each criterion lists acceptable paths and states "the content can
follow your project's conventions." Hive itself is retired and clix never
received the per-criterion issues other fleet repos did, but the same
criteria are the prerequisites for agentic fleet management, and Fluent's
enrollment of this repository additionally requires the canonical surfaces
named by frostyard/core's repository-surfaces contract v1
([core ADR-0035](https://github.com/frostyard/core/blob/main/docs/adr/0035-author-organization-authority-as-strict-json.md)):
`AGENTS.md`, `policies/agent-governance.json`, `.agents/skills/`, and
`docs/README.md`, each as real content.

clix already held canonical equivalents for part of the list at paths fixed
by earlier decisions: `AGENTS.md` with `CLAUDE.md`, `GEMINI.md`, and
`.cursorrules` symlinks
([core ADR-0018](https://github.com/frostyard/core/blob/main/docs/adr/0018-org-wide-agent-instruction-and-knowledge-surfaces.md)),
`.agents/skills/` synced from core
([core ADR-0026](https://github.com/frostyard/core/blob/main/docs/adr/0026-distribute-core-skills-via-sync-prs.md)),
the four-category `docs/` tree
([core ADR-0025](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md)),
and `.svu.yaml` for `make bump`
([core ADR-0012](https://github.com/frostyard/core/blob/main/docs/adr/0012-svu-versioning-and-rolling-dev-prerelease.md))
— but nothing published the tags `make bump` pushed. frostyard/core solved
the identical criteria set with
[core ADR-0029](https://github.com/frostyard/core/blob/main/docs/adr/0029-acmm-conformance-via-canonical-aliases.md):
committed relative symlinks to canonical content wherever a canonical
equivalent exists, genuinely new artifacts only where none does, all guarded
by a docs-integrity gate; repogen and updex followed
([repogen ADR-0012](https://github.com/frostyard/repogen/blob/main/docs/adr/0012-acmm-conformance-via-canonical-aliases.md),
[updex ADR-0012](https://github.com/frostyard/updex/blob/main/docs/adr/0012-acmm-conformance-via-canonical-aliases.md)).
Duplicating content into ACMM's paths would guarantee drift — exactly what
core ADR-0002 rejected.

## Decision

`AGENTS.md` is the single canonical instruction file **and** the contributing
guide (core ADR-0002/0018 pattern, already binding via
[org-adrs.md](../org-adrs.md)). ACMM's required paths are satisfied by
**committed relative symlinks to canonical content** wherever a canonical
equivalent exists, and by genuinely new artifacts only where none does.
Canonical content lives where org conventions put it — the four-category
`docs/` tree and `AGENTS.md` — never at the ACMM path.

The alias table (edit the targets, never the aliases):

| Alias | Target | Criterion |
| --- | --- | --- |
| `CLAUDE.md` | `AGENTS.md` | agent surface (core ADR-0002/0018) |
| `GEMINI.md` | `AGENTS.md` | agent surface (core ADR-0002/0018) |
| `.cursorrules` | `AGENTS.md` | cursor rules |
| `CONTRIBUTING.md` | `AGENTS.md` | contributing guide |
| `.github/copilot-instructions.md` | `../AGENTS.md` | agent surface (core ADR-0002) |
| `.claude/skills` | `../.agents/skills` | simple skills |
| `docs/metrics.md` | `specs/pr-acceptance-metric.md` | PR acceptance metric |
| `docs/review-rubric.md` | `specs/pr-review-rubric.md` | PR review rubric |
| `docs/quality.md` | `design/quality-loop.md` | quality dashboard |

Rules:

- **Directory criteria always get real git trees** (`tests/e2e/`,
  `.github/ISSUE_TEMPLATE/`, `.github/prompts/`, `.memory/`, `policies/`) —
  an evaluator reading the git tree via API sees a symlink as a blob, not a
  tree.
- **Aliases are not docs**: they get no `docs/README.md` index entries and
  carry no cross-link obligations; the canonical target does.
- **The Fluent enrollment surfaces are real content, never aliases**:
  `AGENTS.md`, `policies/agent-governance.json` (validated against core's
  `organization/schemas/v1/repository-agent-governance.schema.json`: deny by
  default; read/write/run-tests allowed; open-issue/open-pr/create-followup
  review-required; `workflow-and-permissions` and `release-and-publication`
  boundaries review-required at high risk), `.agents/skills/`, and
  `docs/README.md`.
- Genuinely new artifacts, each doing real work: the merged `AGENTS.md`
  (contributing guide, release flow, repository boundary);
  [`tests/e2e/`](../../tests/e2e/README.md) — a black-box suite that builds
  `tests/e2e/exampletool` (a consumer wired the way `README.md` prescribes)
  and asserts `--version`, the stdout/stderr split, Silent > JSON > Text, the
  JSON error envelope, and exit codes as a real binary shows them;
  `.github/pull_request_template.md` mirroring `make check`, the `verify`
  job, docs housekeeping, and the aliases rule; `.github/ISSUE_TEMPLATE/`
  (`config.yml` with `blank_issues_enabled: true`, bug and feature
  templates); `.golangci.yml` (v2 schema, the standard set — pinning what
  `make lint` and the CI Lint job already ran, so local and CI agree);
  `.editorconfig`; `.coverage-thresholds.json` enforced by
  `scripts/check-docs.mjs` in the new `docs-gate` CI job (docs-index
  coverage, link integrity, symlink resolution); `.github/prompts/README.md`,
  `review.prompt.md`, and `issue-triage.prompt.md`; the `.memory/` inbox
  with core ADR-0018's append-only five-field schema;
  [specs/pr-acceptance-metric.md](../specs/pr-acceptance-metric.md),
  [specs/pr-review-rubric.md](../specs/pr-review-rubric.md), and
  [design/quality-loop.md](../design/quality-loop.md); `.claude/settings.json`
  denying merge-own-PR, approve-own-work, release-publishing, and pushes to
  `main` at the tool layer; `.claude/session-summary.md`;
  `policies/agent-governance.json`; and the release flow that `make bump`
  was missing — `.goreleaser.yaml` (`version: 2`, `pro: true`, `builds:
  skip: true`, a conventional-commit-grouped changelog, GitHub release
  `prerelease: auto`) and `.github/workflows/release.yml` (tag push →
  GoReleaser Pro with `GITHUB_TOKEN` and the org `GORELEASER_KEY`;
  `permissions: contents: write`; SHA-pinned actions per core ADR-0021).
  clix is a library: the release publishes notes only, no binaries,
  packages, or rolling `dev` prerelease.

## Consequences

- One canonical body of content per criterion; conformance paths cannot
  drift from it.
- GitHub's web renderer shows a symlinked `.md` as its target path rather
  than its content; checkouts on Windows need `core.symlinks=true` or WSL.
- The alias table above is the registry; adding or removing an alias means
  amending it here (a new ADR if the mechanism itself changes).
- `scripts/check-docs.mjs` fails CI on any broken alias, unindexed doc, or
  dead relative link, making the lattice self-guarding.
- `make bump` now has a consumer: the tag it pushes produces a GitHub release
  with grouped notes. `.goreleaser.yaml`, `.svu.yaml`, and
  `.github/workflows/release.yml` are a review-required high-risk boundary
  in `policies/agent-governance.json`.
- The e2e suite builds a binary with the `go` toolchain on `PATH` inside
  `go test ./...`; it exits early when none is present.
- Contingency: if the ACMM evaluator rejects a symlink for one of the file
  criteria (contributing guide, cursor rules, acceptance metric, review
  rubric, quality dashboard), that alias is replaced by a real stub file
  pointing at the canonical doc — a one-commit change that does not reverse
  this decision.

## Alternatives considered

- **Real duplicate files at the ACMM paths:** guaranteed drift; rejected for
  the same reason core ADR-0002 rejected per-tool instruction copies.
- **Content-free stub files:** a second class of "doc" that the index and
  cross-link rules would nominally govern; symlinks are aliases, not docs.
- **A README-only `tests/e2e/`:** would satisfy the path check but do no
  work; the consumer-program suite covers what unit tests structurally
  cannot (fang's `--version`, exit codes, the process-level stdout/stderr
  split) and doubles as a compiling copy of the README usage.
- **Skip the release flow (tags only):** `make bump` would keep pushing tags
  nobody publishes; a library still benefits from release notes consumers
  can read before `go get`.

## References

- Shapes: [design/quality-loop.md](../design/quality-loop.md),
  [specs/pr-review-rubric.md](../specs/pr-review-rubric.md),
  [specs/pr-acceptance-metric.md](../specs/pr-acceptance-metric.md),
  [design/overview.md](../design/overview.md) (CI and release sections)
- Pattern source:
  [core ADR-0029](https://github.com/frostyard/core/blob/main/docs/adr/0029-acmm-conformance-via-canonical-aliases.md),
  [repogen ADR-0012](https://github.com/frostyard/repogen/blob/main/docs/adr/0012-acmm-conformance-via-canonical-aliases.md),
  and
  [updex ADR-0012](https://github.com/frostyard/updex/blob/main/docs/adr/0012-acmm-conformance-via-canonical-aliases.md),
  building on core ADR-0002/0012/0018/0019/0021/0025/0035 (see
  [org-adrs.md](../org-adrs.md))
