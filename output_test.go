package clix

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
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

var errBrokenPipe = errors.New("broken pipe")

// failingWriter fails every Write and counts the attempts.
type failingWriter struct{ calls int }

func (f *failingWriter) Write([]byte) (int, error) {
	f.calls++
	return 0, errBrokenPipe
}

// TestOutputJSON_WriteError pins the write-failure contract: a writer that
// fails is reported as not written, the error names the write (not the
// encoder), and no fallback envelope is attempted on the broken writer.
func TestOutputJSON_WriteError(t *testing.T) {
	fw := &failingWriter{}
	Stdout = fw
	defer func() { Stdout = nil }()
	JSONOutput = true
	defer func() { JSONOutput = false }()

	ok, err := OutputJSON(map[string]int{"a": 1})
	if ok {
		t.Error("OutputJSON() = true on a failed write, want false (nothing reached the writer)")
	}
	if err == nil || !strings.HasPrefix(err.Error(), "write JSON output: ") {
		t.Fatalf("OutputJSON() error = %v, want it to start with %q", err, "write JSON output: ")
	}
	if fw.calls != 1 {
		t.Errorf("writer saw %d Write calls, want exactly 1 (no fallback envelope on a broken writer)", fw.calls)
	}
}

func TestOutputJSONError_WriteError(t *testing.T) {
	fw := &failingWriter{}
	Stdout = fw
	defer func() { Stdout = nil }()
	JSONOutput = true
	defer func() { JSONOutput = false }()

	commandErr := errors.New("timeout")
	err := OutputJSONError("deploy failed", commandErr)
	if !errors.Is(err, commandErr) {
		t.Errorf("OutputJSONError() error = %v, want it to preserve command error %v", err, commandErr)
	}
	if !errors.Is(err, errBrokenPipe) {
		t.Errorf("OutputJSONError() error = %v, want it to preserve write error %v", err, errBrokenPipe)
	}
	if !strings.Contains(err.Error(), "deploy failed: timeout") ||
		!strings.Contains(err.Error(), "write JSON output: broken pipe") {
		t.Errorf("OutputJSONError() error = %q, want command and write failure details", err)
	}
	if fw.calls != 1 {
		t.Errorf("writer saw %d Write calls, want exactly 1", fw.calls)
	}
}

// TestOutputJSON_SuccessBytesUnchanged pins that consumers see exactly the
// bytes an indented json.Encoder produced before: two-space indent and a
// trailing newline.
func TestOutputJSON_SuccessBytesUnchanged(t *testing.T) {
	var out bytes.Buffer
	Stdout = &out
	defer func() { Stdout = nil }()
	JSONOutput = true
	defer func() { JSONOutput = false }()

	value := map[string]any{"name": "x", "list": []int{1, 2}, "nested": map[string]bool{"ok": true}}
	ok, err := OutputJSON(value)
	if !ok || err != nil {
		t.Fatalf("OutputJSON() = (%v, %v), want (true, nil)", ok, err)
	}
	var want bytes.Buffer
	enc := json.NewEncoder(&want)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		t.Fatal(err)
	}
	if out.String() != want.String() {
		t.Errorf("OutputJSON() bytes changed:\ngot:  %q\nwant: %q", out.String(), want.String())
	}
}
