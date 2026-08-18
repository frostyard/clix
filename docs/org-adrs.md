# Org-wide decisions (frostyard/core ADRs)

Conventions this repository follows that are decided at the org level are
recorded as ADRs in
[frostyard/core](https://github.com/frostyard/core/tree/main/docs/adr).
The ones that bind clix:

- [ADR-0002 — Agent-portable instruction surface](https://github.com/frostyard/core/blob/main/docs/adr/0002-agent-portable-instruction-surface.md) — one canonical `AGENTS.md`; every per-tool instruction path is a symlink to it (registered in [ADR-0001](adr/0001-acmm-conformance-via-canonical-aliases.md))
- [ADR-0012 — svu-derived versions, make bump, and the rolling dev prerelease](https://github.com/frostyard/core/blob/main/docs/adr/0012-svu-versioning-and-rolling-dev-prerelease.md) — `.svu.yaml` and `make bump` tag releases; the tag push runs GoReleaser (`.goreleaser.yaml`, `.github/workflows/release.yml`) which publishes release notes only — as a library with no binary, clix has no rolling `dev` prerelease
- [ADR-0018 — Org-wide agent instruction and knowledge surfaces](https://github.com/frostyard/core/blob/main/docs/adr/0018-org-wide-agent-instruction-and-knowledge-surfaces.md) — `AGENTS.md` is canonical; `CLAUDE.md`, `GEMINI.md`, `CONTRIBUTING.md`, `.cursorrules`, and `.github/copilot-instructions.md` are symlinks to it; `.memory/corrections.jsonl` schema; `.github/prompts/*.prompt.md`
- [ADR-0019 — Repository governance as machine-readable policy with risk tiers](https://github.com/frostyard/core/blob/main/docs/adr/0019-governance-as-code-and-risk-tiers.md) — deny-by-default agent limits (`.claude/settings.json`), `never_relax` thresholds (`.coverage-thresholds.json`)
- [ADR-0021 — SHA-pinned actions and least-privilege CI workflows](https://github.com/frostyard/core/blob/main/docs/adr/0021-sha-pinned-actions-and-least-privilege-ci.md) — binds `.github/workflows/ci.yml`: actions pinned, permissions least-privilege
- [ADR-0022 — make ci is the canonical gate; TestI* is reserved](https://github.com/frostyard/core/blob/main/docs/adr/0022-make-ci-gate-and-test-naming-filter.md) — the local gate here is `make check` (fmt → lint → test); the `TestI` prefix stays reserved for environment-requiring integration tests
- [ADR-0025 — One docs/ tree per repository, in core's four-category shape](https://github.com/frostyard/core/blob/main/docs/adr/0025-consolidate-repository-docs-into-docs.md) — this `docs/` tree (formerly `yeti/`); indexed in [README.md](README.md)
- [ADR-0026 — Distribute core agent skills to repos via sync PRs from core](https://github.com/frostyard/core/blob/main/docs/adr/0026-distribute-core-skills-via-sync-prs.md) — clix receives `.agents/skills/` via sync PRs (listed in core's `.github/skills-sync.json`); edit skills in core, not here; `.claude/skills` is a symlink to it
- [ADR-0029 — ACMM conformance via canonical aliases](https://github.com/frostyard/core/blob/main/docs/adr/0029-acmm-conformance-via-canonical-aliases.md) — the alias-not-copy pattern this repo adopts in [ADR-0001](adr/0001-acmm-conformance-via-canonical-aliases.md); `scripts/check-docs.mjs` is ported from core
- [ADR-0035 — Author organization authority as strict JSON](https://github.com/frostyard/core/blob/main/docs/adr/0035-author-organization-authority-as-strict-json.md) — the repository-surfaces contract v1 that names `AGENTS.md`, `policies/agent-governance.json`, `.agents/skills/`, and `docs/README.md` as the canonical surfaces Fluent reads on enrollment

When changing behavior covered by one of these, update or supersede the ADR
in frostyard/core first, then change this repo in the same effort.
