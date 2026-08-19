package clix

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// Stdout, when non-nil, replaces the process's standard output for every
// clix output path (OutputJSON, OutputJSONError, and the JSON reporter from
// NewReporter). It exists so tests — in clix and in consuming CLIs — can
// capture output with a bytes.Buffer instead of swapping os.Stdout.
// It defaults to nil, which means "use the current os.Stdout at call time",
// so consumers that already swap os.Stdout keep working unchanged.
var Stdout io.Writer

// Stderr, when non-nil, replaces the process's standard error for the text
// reporter returned by NewReporter. Same contract as Stdout: nil means "use
// the current os.Stderr at call time".
var Stderr io.Writer

// stdout resolves the writer for stdout output: the Stdout override when
// set, otherwise the current os.Stdout. This is the only place clix reads
// os.Stdout for output.
func stdout() io.Writer {
	if Stdout != nil {
		return Stdout
	}
	return os.Stdout
}

// stderr resolves the writer for stderr output: the Stderr override when
// set, otherwise the current os.Stderr. This is the only place clix reads
// os.Stderr for output.
func stderr() io.Writer {
	if Stderr != nil {
		return Stderr
	}
	return os.Stderr
}

// OutputJSON writes data as indented JSON to stdout if JSONOutput is true.
// Returns (false, nil) when JSON mode is not active. The boolean means "a
// document reached the writer": data is encoded before the writer is
// touched, and
//   - if encoding fails, a fallback error envelope
//     {"error": true, "message": "failed to encode JSON: …"} is written
//     instead and the encoding error is returned alongside true;
//   - if the write itself fails (closed pipe, full disk, a failing Stdout
//     seam), nothing reached the writer: no fallback is attempted and
//     (false, "write JSON output: …") is returned.
//
// Output goes to Stdout when set, otherwise to os.Stdout.
func OutputJSON(data any) (bool, error) {
	if !JSONOutput {
		return false, nil
	}
	w := stdout()
	// Encode first so an encoding failure and a write failure are told apart:
	// the bytes below are exactly what an indented Encoder would emit
	// (two-space indent, trailing newline).
	var document bytes.Buffer
	enc := json.NewEncoder(&document)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		// Write a fallback error envelope so the "JSON was written" contract holds.
		fallback := json.NewEncoder(w)
		fallback.SetIndent("", "  ")
		_ = fallback.Encode(map[string]any{
			"error":   true,
			"message": fmt.Sprintf("failed to encode JSON: %v", err),
		})
		return true, err
	}
	if _, err := w.Write(document.Bytes()); err != nil {
		return false, fmt.Errorf("write JSON output: %w", err)
	}
	return true, nil
}

// OutputJSONError writes a structured error object as JSON to stdout and
// returns a wrapped error for the caller to propagate.
// If err is nil, the message alone is used as the error text.
// If writing the JSON envelope fails, the returned error preserves both the
// caller's error and the output failure.
func OutputJSONError(message string, err error) error {
	details := message
	if err != nil {
		details = err.Error()
	}
	errOutput := map[string]any{
		"error":   true,
		"message": message,
		"details": details,
	}

	var commandErr error
	if err != nil {
		commandErr = fmt.Errorf("%s: %w", message, err)
	} else {
		commandErr = errors.New(message)
	}

	_, outputErr := OutputJSON(errOutput)
	return errors.Join(commandErr, outputErr)
}
