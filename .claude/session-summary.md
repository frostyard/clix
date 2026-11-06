# Session summary

Ephemeral session state — agents replace the block below at session end
(session state lives in `.claude/`). Durable learnings go to
[.memory/](../.memory/README.md), never here; ongoing work is tracked in
GitHub issues (`gh --repo frostyard/clix`). Never include credentials,
tokens, private user data, or raw command output; link issues, PRs, and
commits instead of copying logs.

## Current state

- clix meets frostyard/core's repository baseline (core ADR-0071, rows B1
  to B17), held by the guard test in `tests/baseline/`: golangci-lint 2.15.0
  in `mise.toml`, `ci: verify` in the Makefile, least-privilege workflow
  permissions with `persist-credentials: false`, `.github/risk-paths.json`
  and the `risk-paths.yml` caller, the PR template's risk tier,
  `SECURITY.md`, and the library README (core ADR-0072) with
  [docs/specs/consumer-api.md](../docs/specs/consumer-api.md) holding the
  consumer rules. Releasing stays a maintainer act: `make bump` → tag push →
  `.github/workflows/release.yml`.

## Last landed

- #109 mise-action 5.2.4; #108 multi-line and off-terminal error rendering
  (in `v0.5.0`).

## Next

- Baseline row B11: move `go.mod` to the fleet Go toolchain (Go 1.27) in its
  own pull request.
- After each release, update the supported tag in `SECURITY.md`.
