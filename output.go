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
//     instead and the encoding error is returned alongside true; if that
//     fallback write fails or is short, false is returned with both errors;
//   - if the write itself fails or reports a short write (closed pipe, full
//     disk, a failing Stdout seam), the complete document did not reach the
//     writer: no fallback is attempted and
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
	document, err := encodeIndented(data)
	if err != nil {
		fallback, fallbackErr := encodeIndented(map[string]any{
			"error":   true,
			"message": fmt.Sprintf("failed to encode JSON: %v", err),
		})
		if fallbackErr != nil {
			return false, errors.Join(err, fmt.Errorf("encode JSON fallback envelope: %w", fallbackErr))
		}
		if fallbackErr := writeAll(w, fallback, "write JSON fallback envelope"); fallbackErr != nil {
			return false, errors.Join(err, fallbackErr)
		}
		return true, err
	}
	if err := writeAll(w, document, "write JSON output"); err != nil {
		return false, err
	}
	return true, nil
}

// encodeIndented encodes v with two-space indentation and a trailing newline.
func encodeIndented(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeAll writes b once and reports a write error or short write with context.
func writeAll(w io.Writer, b []byte, what string) error {
	n, err := w.Write(b)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if n != len(b) {
		return fmt.Errorf("%s: wrote %d of %d bytes: %w", what, n, len(b), io.ErrShortWrite)
	}
	return nil
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
