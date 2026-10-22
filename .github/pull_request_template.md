<!-- The org squash-merges: branch off main, never stack on another PR's
branch. Title and commits use Conventional Commits (`type(scope): summary`)
— svu derives the next release version from the squashed `main` commit.
Reviews apply docs/specs/pr-review-rubric.md. -->

## Summary

<!-- What changes and why, in a few sentences. Link the issue(s) this
closes. -->

## Checks

<!-- The build gate from AGENTS.md — run before opening the PR. -->

- [ ] `make check` — the `fmt`, `lint`, `test`, `test-coverage-check`, and
      `coverage-check` targets: `gofmt -w`, `golangci-lint run`
      (`.golangci.yml`) at the `mise.toml` pin, the coverage-instrumented
      `go test` over `./...` (unit tests plus `tests/e2e/`), the
      `check-coverage.sh` self-test, and the 95.0% statement-coverage floor
- [ ] `go mod tidy` leaves `go.mod`/`go.sum` unchanged; `go vet ./...` clean
      (the CI `verify` job)
- [ ] New or changed behavior has focused tests, including failure paths;
      binary-visible behavior (exit codes, stdout/stderr split) is covered in
      `tests/e2e/`
- [ ] Exported API changes are intentional and called out below

## Docs housekeeping

<!-- Delete rows that don't apply (no docs touched). -->

- [ ] `README.md` and `docs/design/overview.md` updated for behavior
      changes; `AGENTS.md` for convention/workflow changes
- [ ] New docs started from their category's `TEMPLATE.md` and indexed in
      `docs/README.md`
- [ ] New significant decision recorded as an ADR *first*, in this PR
- [ ] Conformance aliases (ADR-0001) untouched — canonical targets edited
      instead

## Verification

<!-- Paste evidence the gates ran locally. -->

- [ ] `node scripts/check-docs.mjs` green
- [ ] Checked against the
      [PR review rubric](https://github.com/frostyard/clix/blob/main/docs/specs/pr-review-rubric.md)
