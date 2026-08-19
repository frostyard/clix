# Review a pull request

Review the given frostyard/clix PR against the repo rubric. You are
reviewing, not merging: never approve-and-merge in one act, and never merge
a PR you authored (mechanically backed by `.claude/settings.json`,
[ADR-0001](../../docs/adr/0001-acmm-conformance-via-canonical-aliases.md)).
Automated feedback is advisory: do not approve changes, weaken required
checks, or claim verification passed without evidence from the pull request.
Apply the machine-readable controls in
[policies/agent-governance.json](../../policies/agent-governance.json).

1. Read [AGENTS.md](../../AGENTS.md) — the architecture, conventions, and
   documentation rules the diff must satisfy. In particular:
   - **One flat package** — `clix.go`, `flags.go`, `output.go`,
     `reporter.go`, each with its own `_test.go`; no subpackages beyond the
     e2e suite under `tests/e2e/`.
   - **Library API** — every Frostyard CLI depends on `App`, the four flag
     variables, `OutputJSON`/`OutputJSONError`, `NewReporter`, and
     `BindViper`; breaking changes must be deliberate, documented in
     `README.md` and `docs/design/overview.md`, and reflected in the
     Conventional Commit type (`feat!`/`fix!`).
   - **Silent > JSON > Text** — `--silent` suppresses everything, `--json`
     switches to structured stdout, default is text on stderr; data stays on
     stdout.
   - **Test isolation** — fresh `cobra.Command` per test; package-level flag
     variables reset with `defer`; output captured by assigning a
     `bytes.Buffer` to `clix.Stdout` / `clix.Stderr` with a `defer` reset to
     nil — never by swapping `os.Stdout` (process-wide, unsafe under
     `t.Parallel` / `-race`). The `os.Pipe` tests in `output_test.go` stay
     only as the compatibility proof that the nil default still honors a
     swapped `os.Stdout`; they are not the pattern for new tests.
   - **Go 1.26 idioms** — range-over-int, `omitzero`, `any`, `slices`/`maps`.
2. Apply every row of the
   [PR review rubric](../../docs/specs/pr-review-rubric.md)
   (`docs/review-rubric.md` resolves to the same file). Check each row
   independently; cite file and line for every failure.
3. Run the gates the rubric names:
   - `make check` (gofmt, `golangci-lint run` with `.golangci.yml`,
     `go test -v ./...` including `tests/e2e/`)
   - `go mod tidy && git diff --exit-code go.mod go.sum && go vet ./...`
   - `node scripts/check-docs.mjs`
4. If the diff changes exported API, flags, or output behavior, verify
   `README.md` and [docs/design/overview.md](../../docs/design/overview.md)
   changed alongside the code, and that the repo-local ADRs still hold. If
   it touches `.github/workflows/**`, `.goreleaser.yaml`, or `.svu.yaml`,
   treat it as high risk per `policies/agent-governance.json`: actions stay
   SHA-pinned, permissions least-privilege, the release stays library-shaped
   (no builds, no packages).
5. Report findings as review comments ordered by severity, labelled
   blocking / non-blocking / question / nit per the rubric; state plainly
   when a row passes. A PR with any failing rubric row gets "request
   changes", not silence.
