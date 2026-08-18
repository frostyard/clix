package clix

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestOutputJSON_Active(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	JSONOutput = true
	defer func() { JSONOutput = false }()

	data := map[string]string{"key": "value"}
	ok, err := OutputJSON(data)

	_ = w.Close()
	os.Stdout = old

	if !ok {
		t.Error("OutputJSON() returned false when JSONOutput is true")
	}
	if err != nil {
		t.Errorf("OutputJSON() returned unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("got key=%q, want %q", got["key"], "value")
	}
}

func TestOutputJSON_Inactive(t *testing.T) {
	JSONOutput = false
	ok, err := OutputJSON("anything")
	if ok {
		t.Error("OutputJSON() returned true when JSONOutput is false")
	}
	if err != nil {
		t.Errorf("OutputJSON() returned unexpected error: %v", err)
	}
}

func TestOutputJSON_EncodeError(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	JSONOutput = true
	defer func() { JSONOutput = false }()

	// Channels cannot be JSON-encoded.
	ok, err := OutputJSON(make(chan int))

	_ = w.Close()
	os.Stdout = old

	if !ok {
		t.Error("OutputJSON() returned false on encode error; expected true (fallback written)")
	}
	if err == nil {
		t.Fatal("OutputJSON() returned nil error for unencodable type")
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var got map[string]any
	if jsonErr := json.Unmarshal(buf.Bytes(), &got); jsonErr != nil {
		t.Fatalf("fallback output is not valid JSON: %v\nraw: %s", jsonErr, buf.String())
	}
	if got["error"] != true {
		t.Errorf("error field = %v, want true", got["error"])
	}
	if got["message"] == "" {
		t.Error("fallback message is empty")
	}
}

func TestOutputJSONError(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	JSONOutput = true
	defer func() { JSONOutput = false }()

	err := OutputJSONError("deploy failed", errors.New("timeout"))

	_ = w.Close()
	os.Stdout = old

	if err == nil {
		t.Fatal("OutputJSONError() returned nil error")
	}
	if err.Error() != "deploy failed: timeout" {
		t.Errorf("error = %q, want %q", err.Error(), "deploy failed: timeout")
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["error"] != true {
		t.Errorf("error field = %v, want true", got["error"])
	}
	if got["message"] != "deploy failed" {
		t.Errorf("message = %v, want %q", got["message"], "deploy failed")
	}
}

func TestOutputJSONError_NilError(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	JSONOutput = true
	defer func() { JSONOutput = false }()

	err := OutputJSONError("something went wrong", nil)

	_ = w.Close()
	os.Stdout = old

	if err == nil {
		t.Fatal("OutputJSONError() returned nil error")
	}
	if err.Error() != "something went wrong" {
		t.Errorf("error = %q, want %q", err.Error(), "something went wrong")
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var got map[string]any
	if jsonErr := json.Unmarshal(buf.Bytes(), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}
	if got["error"] != true {
		t.Errorf("error field = %v, want true", got["error"])
	}
	if got["details"] != "something went wrong" {
		t.Errorf("details = %v, want %q", got["details"], "something went wrong")
	}
}

// The tests below exercise the Stdout writer seam: output is captured with a
// bytes.Buffer assigned to Stdout and os.Stdout is never touched. The
// os.Pipe tests above remain as the compatibility proof for consumers that
// still swap os.Stdout.

func TestOutputJSON_StdoutSeam(t *testing.T) {
	var buf bytes.Buffer
	Stdout = &buf
	defer func() { Stdout = nil }()

	JSONOutput = true
	defer func() { JSONOutput = false }()

	ok, err := OutputJSON(map[string]string{"key": "value"})
	if !ok {
		t.Error("OutputJSON() returned false when JSONOutput is true")
	}
	if err != nil {
		t.Errorf("OutputJSON() returned unexpected error: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}
	if got["key"] != "value" {
		t.Errorf("got key=%q, want %q", got["key"], "value")
	}
}

func TestOutputJSON_EncodeError_StdoutSeam(t *testing.T) {
	var buf bytes.Buffer
	Stdout = &buf
	defer func() { Stdout = nil }()

	JSONOutput = true
	defer func() { JSONOutput = false }()

	// Channels cannot be JSON-encoded; the fallback envelope must land in buf.
	ok, err := OutputJSON(make(chan int))
	if !ok {
		t.Error("OutputJSON() returned false on encode error; expected true (fallback written)")
	}
	if err == nil {
		t.Fatal("OutputJSON() returned nil error for unencodable type")
	}

	var got map[string]any
	if jsonErr := json.Unmarshal(buf.Bytes(), &got); jsonErr != nil {
		t.Fatalf("fallback output is not valid JSON: %v\nraw: %s", jsonErr, buf.String())
	}
	if got["error"] != true {
		t.Errorf("error field = %v, want true", got["error"])
	}
	if got["message"] == "" {
		t.Error("fallback message is empty")
	}
}

func TestOutputJSONError_StdoutSeam(t *testing.T) {
	var buf bytes.Buffer
	Stdout = &buf
	defer func() { Stdout = nil }()

	JSONOutput = true
	defer func() { JSONOutput = false }()

	err := OutputJSONError("deploy failed", errors.New("timeout"))
	if err == nil {
		t.Fatal("OutputJSONError() returned nil error")
	}
	if err.Error() != "deploy failed: timeout" {
		t.Errorf("error = %q, want %q", err.Error(), "deploy failed: timeout")
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if got["error"] != true {
		t.Errorf("error field = %v, want true", got["error"])
	}
	if got["message"] != "deploy failed" {
		t.Errorf("message = %v, want %q", got["message"], "deploy failed")
	}
	if got["details"] != "timeout" {
		t.Errorf("details = %v, want %q", got["details"], "timeout")
	}
}

func TestStdoutSeam_NilFallsBackToOSStdout(t *testing.T) {
	Stdout = nil
	if got := stdout(); got != os.Stdout {
		t.Errorf("stdout() with Stdout=nil = %v, want os.Stdout", got)
	}
	Stderr = nil
	if got := stderr(); got != os.Stderr {
		t.Errorf("stderr() with Stderr=nil = %v, want os.Stderr", got)
	}
}
