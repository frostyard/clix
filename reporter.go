package clix

import (
	"os"

	"github.com/frostyard/std/reporter"
)

// NewReporter returns the appropriate reporter based on the JSONOutput flag.
// JSON mode: JSONReporter writing to os.Stdout (for piping/parsing).
// Text mode: TextReporter writing to os.Stderr (keeps stdout clean for data).
func NewReporter() reporter.Reporter {
	if JSONOutput {
		return reporter.NewJSONReporter(os.Stdout)
	}
	return reporter.NewTextReporter(os.Stderr)
}
