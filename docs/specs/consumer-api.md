# Spec: consumer API

The contract a CLI that imports `github.com/frostyard/clix` writes against:
the build-time metadata, the entry points, the 4 common flags, the JSON
output helpers, the reporter factory, the writer seams, and the optional
Viper binding. Every frostyard CLI that imports clix is a consumer; its own
`main` and `RunE` functions are the code that depends on this page. The
package tests pin each rule below ([`clix_test.go`](../../clix_test.go),
[`flags_test.go`](../../flags_test.go), [`errors_test.go`](../../errors_test.go),
[`output_test.go`](../../output_test.go), [`reporter_test.go`](../../reporter_test.go),
and the black-box suite in [`tests/e2e/`](../../tests/e2e/README.md)).

## Interface

### Entry points

| Symbol | Shape | Contract |
| --- | --- | --- |
| `clix.App` | struct | Build-time metadata: `Version`, `Commit`, `Date`, `BuiltBy`, all `string`. A zero field takes the default `dev`, `none`, `unknown`, or `local`. |
| `(*clix.App).VersionString()` | `string` | `1.2.3 (Commit: abc) (Date: 2026-01-01) (Built by: ci)`. |
| `(*clix.App).Run(cmd)` | `error` | Registers the common flags on `cmd`, then executes the command tree through `fang.Execute` with the version string and `SIGINT`/`SIGTERM` handling. `Run` is a wrapper around `RunContext(context.Background(), cmd)`. Returns `clix: Run: root command is nil` for a nil `cmd`, and the reserved-flag error before anything executes. |
| `(*clix.App).RunContext(ctx, cmd)` | `error` | `Run` with a caller-supplied `context.Context` (see [RunContext](#runcontext)). |
| `clix.OutputJSON(data)` | `(bool, error)` | Writes `data` as indented JSON to stdout when `clix.JSONOutput` is true; `(false, nil)` when it is false. |
| `clix.OutputJSONError(message, err)` | `error` | Writes the JSON error envelope (`error: true`, `message`, `details`) through `OutputJSON`, then returns an error for the caller to propagate. |
| `clix.NewReporter()` | `reporter.Reporter` | Chooses the reporter from the flags (see [Reporter priority](#reporter-priority)). |
| `clix.BindViper(cmd)` | `error` | Binds the 4 flags to viper keys of the same names (see [Viper integration](#viper-integration)). |

### Common flags

`App.Run` registers these 4 flags as persistent flags on the root command.
The package-level variable is the flag's target: read it directly, as in
`if clix.Verbose { ... }`.

| Flag | Shorthand | Variable | Default | Usage string |
| --- | --- | --- | --- | --- |
| `--json` | — | `clix.JSONOutput` | `false` | output in JSON format |
| `--verbose` | `-v` | `clix.Verbose` | `false` | verbose output |
| `--dry-run` | `-n` | `clix.DryRun` | `false` | dry run mode (no actual changes) |
| `--silent` | `-s` | `clix.Silent` | `false` | suppress all progress output |

### Writer seams

| Variable | Type | Default | Replaces |
| --- | --- | --- | --- |
| `clix.Stdout` | `io.Writer` | `nil` | the stdout of `OutputJSON` (and of its fallback envelope) and of the JSON reporter |
| `clix.Stderr` | `io.Writer` | `nil` | the stderr of the text reporter |

`nil` means "the current `os.Stdout` / `os.Stderr` at call time".

## Rules

### The 4 flags are reserved across the whole command tree

- `--json`, `--verbose`/`-v`, `--dry-run`/`-n` and `--silent`/`-s` are
  reserved by clix across the whole tree.
- The check covers the root command and every subcommand at any depth, on
  both the persistent and the local flag set.
- A root command or subcommand that already defines one of those names or
  shorthands makes `App.Run`/`App.RunContext` return an error that names the
  command and the collision, instead of letting pflag panic. Nothing
  executes.
- The error text is `clix: root command already defines flag --<name>` or
  `clix: root command already defines shorthand -<c> (used by --<other>)`
  for the root, and `clix: command "<command path>" already defines flag
  --<name>` (or the shorthand form) for a subcommand.
- The check exists because cobra merges a command's local flags with its
  ancestors' persistent flags at parse time: a colliding shorthand would
  panic in pflag only when that subcommand runs, and a colliding name would
  silently shadow clix's flag, so its variable would never be set.

### Repeated Run calls on the same root

Calling `App.Run` or `App.RunContext` more than once against the same root
command is safe. The flags clix registered on an earlier call are recognized
(a `clix:owned` pflag annotation) and reused rather than re-registered, so a
test or a REPL-style caller can execute the same root repeatedly without
tripping its own reserved-flag collision check. A reused flag is reset to its
declared default and its `Changed` state cleared before that invocation's
arguments are parsed, so each invocation starts isolated: a `--json` passed on
one run cannot leak into a later run that omits it, and `BindViper`'s
command-line-over-viper precedence sees only the current invocation's
explicit flags. A consumer-defined flag that collides with a reserved name or
shorthand is still rejected exactly as on the first call.

### RunContext

`App.Run(cmd)` is a thin wrapper around
`App.RunContext(context.Background(), cmd)`. Call `RunContext` directly when
you want to bound execution with a deadline or timeout, or let a supervising
process (a test, a parent command, an orchestrator) cancel the run instead of
relying solely on the `SIGINT`/`SIGTERM` handling `Run` installs.
`RunContext` performs the same nil-command check, defaulting and flag
registration as `Run`, then runs `fang.Execute` with the supplied `ctx`; a
command's `RunE` observes cancellation through `cmd.Context().Err()`.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := app.RunContext(ctx, rootCmd); err != nil {
	os.Exit(1)
}
```

### Reporter priority

`clix.NewReporter()` returns a reporter based on the active flags:

| Priority | Condition | Reporter | Output |
| --- | --- | --- | --- |
| 1 (highest) | `--silent` | `NoopReporter` | none |
| 2 | `--json` | `JSONReporter` | stdout |
| 3 (default) | neither | `TextReporter` | stderr |

Silent always takes priority over JSON. Text output goes to stderr to keep
stdout clean for data. Both destinations are resolved through the
`clix.Stdout` / `clix.Stderr` seams when `NewReporter` is called. Silent
applies to the reporter, not to application data: `clix.OutputJSON` ignores
`Silent`, so `--json --silent` still emits the result document on stdout
while omitting the JSON reporter record.

### JSON output

`clix.OutputJSON` returns a boolean that means "a complete document reached
the writer", so a caller can tell these outcomes apart:

| Outcome | Return | What was written |
| --- | --- | --- |
| JSON mode off | `(false, nil)` | nothing |
| success | `(true, nil)` | the document, exactly as an indented `json.Encoder` emits it (2-space indent, trailing newline) |
| encode failure | `(true, err)` | a fallback envelope `{"error": true, "message": "failed to encode JSON: …"}` |
| encode failure, fallback write fails or is short | `(false, err)` | an incomplete or no fallback; both causes are in the error |
| write fails or is short | `(false, err)` | an incomplete document; no fallback is attempted |

```go
written, err := clix.OutputJSON(result)
if err != nil {
	return err
}
if written {
	return nil
}
// Fall through to text output.
```

`clix.OutputJSONError` builds the envelope `{"error": true, "message": …,
"details": …}`, writes it through `OutputJSON`, and returns an error: it wraps
`err` with `message` when `err` is not nil, and is a plain `errors.New(message)`
otherwise. If the envelope write fails, the returned error joins the
command error and the output failure, so `errors.Is` finds both.

### Error rendering

`RunContext` passes its own handler to `fang.Execute`, so clix renders a
command error. On a terminal:

- every line of a multi-line error (for example an `errors.Join` of 2
  failures) keeps its own line, because the width style is applied to each
  line;
- a first word that is a flag keeps its lower case: `--clear cannot be
  combined with --other`, not `--Clear cannot be combined with --other`;
- the closing period is added only when the text does not already end in
  `.`, `!`, `?` or `:`, so `failed.` is not printed as `failed..`;
- a usage error keeps fang's `Try --help for usage.` hint.

Output that does not go to a terminal (a pipe, a journal, a redirected
stderr) is the plain `err.Error()` and 1 newline: no `ERROR` header, no
style, no closing period and no usage hint. clix v0.4.0 and earlier printed fang's
styled block there, with only the ANSI sequences stripped; the design doc explains why
([design: error rendering](../design/overview.md#errorsgo--error-rendering)).

### Testing output

Every clix output path — `OutputJSON`, `OutputJSONError`, and the reporters
from `NewReporter` — writes through the 2 package-level seams `clix.Stdout`
and `clix.Stderr`. Both default to `nil`, meaning "the current `os.Stdout` /
`os.Stderr` at call time", so production behavior is unchanged and consumers
that already swap `os.Stdout` in tests keep working. The seams are shared
package-level variables, so tests that set them must not run in parallel
unless they coordinate access. In tests, assign a `bytes.Buffer` instead of
touching the process globals:

```go
func TestStatusJSON(t *testing.T) {
	var out bytes.Buffer
	clix.Stdout = &out
	defer func() { clix.Stdout = nil }()
	clix.JSONOutput = true
	defer func() { clix.JSONOutput = false }()

	// ... run the command ...

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
}
```

### Viper integration

If your CLI uses [spf13/viper](https://github.com/spf13/viper) for config,
bind the common flags in `PersistentPreRunE`:

```go
rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
	return clix.BindViper(cmd)
}
```

- `clix.BindViper` binds `--json`, `--verbose`, `--dry-run` and `--silent` to
  the viper keys `json`, `verbose`, `dry-run` and `silent`, so they can be set
  in a config file or an environment variable.
- Call it from `PersistentPreRunE`, not `RunE`, so the write-back below
  happens before your `RunE` reads the flag variables. cobra runs the root's
  `PersistentPreRunE` with `cmd` set to the command actually executing, and
  `BindViper` resolves the flags through that command's local, persistent and
  inherited sets, so 1 hook serves the root and every subcommand.
- Called before `App.Run` registered the flags, it returns
  `clix: BindViper: --json is not registered on "<cmd>"; call App.Run on the
  root command first`. A nil `cmd` returns
  `clix: BindViper: command is nil` instead of panicking.
- `BindViper` also writes a viper-sourced value back onto the flag, so a
  config file or an environment variable really changes `clix.JSONOutput`,
  `clix.Verbose`, `clix.DryRun` and `clix.Silent`; binding alone would leave
  the value inside viper, where nothing clix reads would see it.
- Precedence is **command line > viper > default**. A flag the user passed is
  never overwritten, so an explicit `--json=false` wins over `json: true` in a
  config file.

## References

- Rationale: [ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)
- Context: [design: overview](../design/overview.md) — the internals of
  `errorHandler`, `registerFlags` and `OutputJSON`
- README: [README.md](../../README.md) — the short program a consumer starts
  from
