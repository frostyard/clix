package clix

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/frostyard/std/reporter"
)

func TestNewReporter_JSON(t *testing.T) {
	JSONOutput = true
	defer func() { JSONOutput = false }()

	r := NewReporter()
	if !r.IsJSON() {
		t.Error("NewReporter() with JSONOutput=true should return JSON reporter")
	}
	if _, ok := r.(*reporter.JSONReporter); !ok {
		t.Errorf("NewReporter() type = %T, want *reporter.JSONReporter", r)
	}
}

func TestNewReporter_Silent(t *testing.T) {
	Silent = true
	defer func() { Silent = false }()

	r := NewReporter()
	if _, ok := r.(reporter.NoopReporter); !ok {
		t.Errorf("NewReporter() type = %T, want reporter.NoopReporter", r)
	}
}

func TestNewReporter_SilentOverridesJSON(t *testing.T) {
	Silent = true
	JSONOutput = true
	defer func() { Silent = false; JSONOutput = false }()

	r := NewReporter()
	if _, ok := r.(reporter.NoopReporter); !ok {
		t.Errorf("NewReporter() with Silent+JSON type = %T, want reporter.NoopReporter", r)
	}
}

func TestNewReporter_Text(t *testing.T) {
	JSONOutput = false

	r := NewReporter()
	if r.IsJSON() {
		t.Error("NewReporter() with JSONOutput=false should return text reporter")
	}
	if _, ok := r.(*reporter.TextReporter); !ok {
		t.Errorf("NewReporter() type = %T, want *reporter.TextReporter", r)
	}
}

// The tests below exercise the Stdout / Stderr writer seam: reporter output
// is captured with a bytes.Buffer and os.Stdout / os.Stderr are never touched.

func TestNewReporter_JSON_StdoutSeam(t *testing.T) {
	var out, errBuf bytes.Buffer
	Stdout = &out
	Stderr = &errBuf
	defer func() { Stdout = nil; Stderr = nil }()

	JSONOutput = true
	defer func() { JSONOutput = false }()

	r := NewReporter()
	r.Message("hello %s", "world")

	if errBuf.Len() != 0 {
		t.Errorf("JSON reporter wrote to stderr: %q", errBuf.String())
	}
	var event map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &event); err != nil {
		t.Fatalf("JSON reporter output is not valid JSON: %v\nraw: %s", err, out.String())
	}
	if event["message"] != "hello world" {
		t.Errorf("message = %v, want %q", event["message"], "hello world")
	}
}

func TestNewReporter_Text_StderrSeam(t *testing.T) {
	var out, errBuf bytes.Buffer
	Stdout = &out
	Stderr = &errBuf
	defer func() { Stdout = nil; Stderr = nil }()

	JSONOutput = false
	Silent = false

	r := NewReporter()
	r.Message("hello %s", "world")

	if out.Len() != 0 {
		t.Errorf("text reporter wrote to stdout: %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "hello world") {
		t.Errorf("text reporter stderr = %q, want it to contain %q", errBuf.String(), "hello world")
	}
}
