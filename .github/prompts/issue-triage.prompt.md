# Issue Triage Prompt

Use this prompt when triaging a new GitHub issue or bug report for clix.

## Instructions

1. Read `docs/design/overview.md` and any relevant notes under `.memory/` for
   architecture and prior gotchas before proposing a fix.
2. Classify the issue:
   - **Library bug** — wrong behavior in `clix.go` (`App.Run`,
     `VersionString`), `flags.go` (flag registration, `BindViper`),
     `output.go` (`OutputJSON`, `OutputJSONError`), or `reporter.go`
     (`NewReporter` priority).
   - **Consumer-integration bug** — only reproducible in a built CLI (exit
     codes, stdout/stderr split, `--version`); reproduce it in
     `tests/e2e/exampletool` first.
   - **Docs gap** — missing or stale guidance in `AGENTS.md`, `README.md`, or
     `docs/`.
   - **ACMM/process gap** — repository hygiene or agent-maturity criterion
     (see `AGENTS.md` and
     [ADR-0001](../../docs/adr/0001-acmm-conformance-via-canonical-aliases.md)).
3. For a reproducible bug, write a minimal failing test first (fresh
   `cobra.Command`, package-level flags reset with `defer`, output captured
   via `bytes.Buffer` or `os.Pipe()`), confirm it fails, then implement the
   smallest fix that makes it pass.
4. Keep the API surface stable: every Frostyard CLI depends on it, so a
   breaking change needs an explicit rationale and a `feat!`/`fix!` commit.
5. Run `make check` (and `node scripts/check-docs.mjs` if docs changed)
   before proposing the change.
6. If the fix reveals a durable architectural fact or pitfall, fold it into
   the right `docs/` page (or drop a note in `.memory/`) so future agents
   avoid repeating the mistake.
