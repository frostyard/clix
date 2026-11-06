# Security policy

clix is a Go library that a CLI imports: it wraps charm.land/fang/v2 and
spf13/cobra and adds the common flags, the JSON output helpers, the error
rendering and the reporter factory. There is no command of its own. Please
report security problems privately, as described below.

## Supported versions

Only the latest release tag receives security fixes. Today that is `v0.5.0`;
see the [releases page](https://github.com/frostyard/clix/releases) for the
current one. clix is a library, so a fix reaches a tool when that tool
upgrades: `go get github.com/frostyard/clix@latest`. Older tags do not
receive fixes — update to the latest tag.

## Report a vulnerability

Do not open a public issue.

Use GitHub private vulnerability reporting:
https://github.com/frostyard/clix/security. Click "Report a vulnerability",
or use the direct form
https://github.com/frostyard/clix/security/advisories/new.

If you cannot use GitHub, email the maintainer at
[bketelsen@gmail.com](mailto:bketelsen@gmail.com).

## What happens next

The maintainer confirms receipt of the report. The maintainer then fixes the
problem on `main`, tags a patch release, and publishes a security advisory.
Each tool that imports clix upgrades with `go get`.

## Scope

clix runs inside the CLI that imports it, with that CLI's rights and its
arguments and environment. These areas are the most sensitive:

- **Flag parsing and the reserved-flag checks** (`flags.go`): the 4 common
  flags are registered as persistent flags on the root command, and the
  reserved-flag check walks the whole command tree before anything executes.
  A collision with a consumer-defined flag is an error that names the command,
  never a pflag panic or a silently shadowed flag.
- **The JSON output and its error envelope** (`output.go`): `OutputJSON`
  writes the result document that scripts parse, and `OutputJSONError` writes
  the `error`, `message`, `details` envelope. Both encode into a buffer before
  the writer is touched, and a short write is reported instead of being
  treated as success, so a caller can tell an incomplete document from a
  complete one. The data in the document is the consumer's; clix adds no field
  of its own beyond the envelope.
- **Error rendering** (`errors.go`): a command error is written to the
  terminal, or to a pipe, journal or redirected stderr. On a terminal, every
  line of the error is printed; off a terminal the text is `err.Error()` as is,
  with no style that clix adds. The error text is the
  consumer's, so an error that carries a secret is the consumer's to redact.
- **`BindViper`** (`flags.go`): values from a config file or the environment
  reach the 4 flag variables through this binding. Precedence is: command line
  first, then viper, then the default, so a flag the user typed is never overwritten by a
  config value. clix reads only the 4 keys `json`, `verbose`, `dry-run` and
  `silent`.
