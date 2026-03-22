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
	ok := OutputJSON(data)

	_ = w.Close()
	os.Stdout = old

	if !ok {
		t.Error("OutputJSON() returned false when JSONOutput is true")
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
	ok := OutputJSON("anything")
	if ok {
		t.Error("OutputJSON() returned true when JSONOutput is false")
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
