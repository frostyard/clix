package clix

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// OutputJSON writes data as indented JSON to stdout if JSONOutput is true.
// Returns true if output was written, false if JSON mode is not active.
// If encoding fails, a fallback error envelope is written to stdout and the
// encoding error is returned alongside true (output was still written).
func OutputJSON(data any) (bool, error) {
	if !JSONOutput {
		return false, nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		// Write a fallback error envelope so the "JSON was written" contract holds.
		fallback := json.NewEncoder(os.Stdout)
		fallback.SetIndent("", "  ")
		_ = fallback.Encode(map[string]string{
			"error":   "true",
			"message": fmt.Sprintf("failed to encode JSON: %v", err),
		})
		return true, err
	}
	return true, nil
}

// OutputJSONError writes a structured error object as JSON to stdout and
// returns a wrapped error for the caller to propagate.
// If err is nil, the message alone is used as the error text.
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
	_, _ = OutputJSON(errOutput)
	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return errors.New(message)
}
