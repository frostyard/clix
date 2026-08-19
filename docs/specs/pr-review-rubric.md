# Spec: PR review rubric

One paragraph: the checklist every frostyard/clix pull-request review
applies, kept consistent, actionable, and focused on the risks the pull
request introduces. Consumers: human reviewers, the
[review runbook](../../.github/prompts/review.prompt.md), the
[PR template](../../.github/pull_request_template.md), whose sections mirror
these checks, and [`policies/agent-governance.json`](../../policies/agent-governance.json),
whose `review-required` decisions this rubric operationalizes.
`docs/review-rubric.md` is a conformance alias for this file
([ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)).

## Interface

Every review verifies each row; a PR merges only when all applicable rows
pass.

| Check | How to verify |
| --- | --- |
| Correctness and scope | The change solves the linked problem and handles the relevant error cases; the diff is focused — no unrelated refactors or generated artifacts (`coverage.out`, `coverage.html`, `dist/`). |
| API surface | clix is a library consumed by every Frostyard CLI: exported identifiers keep their doc comments; a breaking change to `App`, the flag variables, `OutputJSON`/`OutputJSONError`, `NewReporter`, or `BindViper` is intentional, called out in the PR, and reflected in `README.md` and [design/overview.md](../design/overview.md); the Silent > JSON > Text priority holds on every output path. |
| Build gate green | `make check` passes: `gofmt` leaves no diff, `golangci-lint run` (`.golangci.yml`) reports no issues, `go test -v ./...` passes; `go mod tidy` leaves `go.mod`/`go.sum` unchanged and `go vet ./...` is clean — the same checks as the `lint`, `test`, `race`, and `verify` jobs of [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml). |
| Tests | New or changed behavior has focused tests including failure paths (one `_test.go` per source file, fresh `cobra.Command` per test, package-level flag variables reset), or the PR explains why tests do not apply; behavior visible only in a finished binary (exit codes, stdout/stderr split, `--version`) is covered in [`tests/e2e/`](../../tests/e2e/README.md). |
| Docs housekeeping | User-facing (`README.md`) and agent-oriented docs (`docs/design/overview.md`, `docs/specs/*`, `AGENTS.md`) reflect the behavior change; new docs start from their category `TEMPLATE.md`, are indexed in [docs/README.md](../README.md), and cross-link both ways; a new significant decision ⇒ ADR first, in the same change. |
| Docs-integrity gate green | `node scripts/check-docs.mjs` passes: every doc indexed, every relative link resolving, every symlink alias intact, and the documented CI job inventory matches the workflow (thresholds in `.coverage-thresholds.json`). |
| Aliases untouched | Conformance aliases ([ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)) are not edited directly; canonical targets are. |
| Conventional title | The PR title (or lone commit subject) is `type(scope): summary`; the org squash-merges and svu derives the next version from it. |
| Release and workflow boundaries | Changes under `.github/workflows/**`, `.goreleaser.yaml`, or `.svu.yaml` are reviewed at high risk per [`policies/agent-governance.json`](../../policies/agent-governance.json): actions stay SHA-pinned, permissions least-privilege, the release flow stays library-shaped (no builds, no packages). |
| Agent limits respected | The PR was not merged, approved, or released by the agent that authored it; mechanically backed by `.claude/settings.json` and `policies/agent-governance.json`. |

## Rules

- Each check is independently verifiable from the PR diff plus the commands
  named in its row — a review MUST NOT rely on out-of-band context.
- Label findings by impact:
  - **Blocking:** a correctness, API-compatibility, security, or required
    test/documentation issue that must be resolved before approval.
  - **Non-blocking:** a worthwhile improvement that does not prevent merging.
  - **Question:** a request for context or clarification, not an assumed
    defect.
  - **Nit:** an optional minor style suggestion; avoid nits already enforced
    by `gofmt` or `golangci-lint`.
- Comments identify the affected behavior, explain its impact, and suggest a
  concrete resolution. Reviewers re-check resolved blocking findings and
  confirm required CI checks pass before approval.
- Rubric changes ride with the artifact that enforces them (the gate script,
  the workflow, or the template) in the same PR.
- The org squash-merges: the review covers the squashed result, not
  intermediate commits.

## References

- Rationale: [ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)
- Context: [design/quality-loop.md](../design/quality-loop.md)
- Related: [`policies/agent-governance.json`](../../policies/agent-governance.json)
