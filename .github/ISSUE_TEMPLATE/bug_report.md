---
name: Bug report
about: Report a problem with clix
title: ''
labels: bug
assignees: ''
---

## Description

A clear description of the bug.

## Steps to reproduce

1.
2.
3.

Minimal consumer program if possible (a cobra root run through
`clix.App{}.Run`, the flags passed, and the `OutputJSON` / `NewReporter`
calls involved).

## Expected behavior

What you expected to happen (stdout / stderr / exit code).

## Actual behavior

What actually happened, including any output.

## Environment

- clix version (`go list -m github.com/frostyard/clix`):
- Go version (`go version`):
- OS / architecture:

## Additional context

Anything else that helps — related `--json` / `--silent` combinations,
viper binding, etc.
