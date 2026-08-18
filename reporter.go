package clix

import (
	"github.com/frostyard/std/reporter"
)

// NewReporter returns the appropriate reporter based on the Silent and JSONOutput flags.
// Silent mode: NoopReporter (suppresses all output). Silent takes priority over JSON.
// JSON mode: JSONReporter writing to stdout (for piping/parsing).
// Text mode: TextReporter writing to stderr (keeps stdout clean for data).
// The writers are resolved through Stdout / Stderr when set, otherwise the
// current os.Stdout / os.Stderr.
func NewReporter() reporter.Reporter {
	if Silent {
		return reporter.NoopReporter{}
	}
	if JSONOutput {
		return reporter.NewJSONReporter(stdout())
	}
	return reporter.NewTextReporter(stderr())
}
