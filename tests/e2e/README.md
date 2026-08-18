# End-to-end tests

clix's e2e suite is [`e2e_test.go`](e2e_test.go) in this directory: a
black-box suite that builds [`exampletool/`](exampletool/main.go) — a
consumer program wired exactly the way [README.md](../../README.md) tells a
Frostyard CLI to use clix (ldflags-injected build metadata, `App.Run` on a
cobra root, `NewReporter` for progress, `OutputJSON`/`OutputJSONError` for
data) — and runs it as a subprocess. It asserts what only a finished binary
shows: fang's `--version` output carrying the injected metadata, the
stdout/stderr split (data on stdout, text progress on stderr), the
Silent > JSON > Text priority, the JSON error envelope, and process exit
codes. The unit tests next to each source file (`clix_test.go`,
`flags_test.go`, `output_test.go`, `reporter_test.go`) cover the same
behavior in-process. `TestMain` exits early when no `go` toolchain is on
`PATH`.

Run it with:

```bash
go test -v ./tests/e2e/...   # the e2e suite alone
make test                    # everything, e2e included
```

CI runs it inside the `Unit Tests` and `Race Detection` jobs of
[`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) — both run
`go test ./...`, which includes this package. This README is the
discoverable e2e entry point named by
[ADR-0001](../../docs/adr/0001-acmm-conformance-via-canonical-aliases.md).
