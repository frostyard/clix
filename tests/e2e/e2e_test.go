// Package e2e is clix's end-to-end suite: it builds the consumer program in
// ./exampletool the way a Frostyard CLI is built (ldflags-injected metadata,
// App.Run on a cobra root) and runs it as a subprocess, asserting on the
// stdout/stderr split, exit codes, and the Silent > JSON > Text priority as a
// user of the finished binary sees them. The unit tests next to each source
// file cover the same behavior in-process; this suite covers what only a real
// binary can — fang's version flag, error rendering, and process exit codes.
package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	e2eVersion = "1.2.3"
	e2eCommit  = "abc123"
	e2eDate    = "2026-03-04"
	e2eBuiltBy = "e2e"
)

// binary is the path of the built exampletool for the whole package run.
var binary string

func TestMain(m *testing.M) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		// No Go toolchain on PATH: nothing to build, nothing to run.
		os.Exit(0)
	}
	dir, err := os.MkdirTemp("", "clix-e2e-")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "exampletool")
	ldflags := strings.Join([]string{
		"-X main.version=" + e2eVersion,
		"-X main.commit=" + e2eCommit,
		"-X main.date=" + e2eDate,
		"-X main.builtBy=" + e2eBuiltBy,
	}, " ")
	build := exec.Command(goBin, "build", "-o", binary, "-ldflags", ldflags, "./exampletool")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		_ = os.RemoveAll(dir)
		panic("building exampletool: " + err.Error())
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// run executes the built binary and returns stdout, stderr, and the exit code.
func run(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		exitCode = 0
	case errors.As(err, &exitErr):
		exitCode = exitErr.ExitCode()
	default:
		t.Fatalf("running %s %v: %v", binary, args, err)
	}
	return out.String(), errOut.String(), exitCode
}

func TestVersionFlagPrintsInjectedMetadata(t *testing.T) {
	stdout, _, code := run(t, "--version")
	if code != 0 {
		t.Fatalf("--version exit code = %d, want 0", code)
	}
	want := "1.2.3 (Commit: abc123) (Date: 2026-03-04) (Built by: e2e)"
	if !strings.Contains(stdout, want) {
		t.Errorf("--version stdout = %q, want it to contain %q", stdout, want)
	}
}

func TestTextModeKeepsStdoutForData(t *testing.T) {
	stdout, stderr, code := run(t, "greet")
	if code != 0 {
		t.Fatalf("greet exit code = %d, want 0", code)
	}
	if stdout != "hello world\n" {
		t.Errorf("greet stdout = %q, want %q", stdout, "hello world\n")
	}
	if !strings.Contains(stderr, "greeting world") {
		t.Errorf("greet stderr = %q, want the reporter message on stderr", stderr)
	}
}

func TestJSONModeWritesEnvelopeToStdout(t *testing.T) {
	stdout, stderr, code := run(t, "greet", "bob", "--json", "--verbose", "--dry-run")
	if code != 0 {
		t.Fatalf("greet --json exit code = %d, want 0", code)
	}
	if stderr != "" {
		t.Errorf("greet --json stderr = %q, want empty (JSON reporter writes to stdout)", stderr)
	}
	// The JSON reporter emits one JSON-lines record, then OutputJSON emits the
	// indented result; both are on stdout and every top-level value decodes.
	dec := json.NewDecoder(strings.NewReader(stdout))
	var records []map[string]any
	for dec.More() {
		var v map[string]any
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("greet --json stdout is not a sequence of JSON values: %v\n%s", err, stdout)
		}
		records = append(records, v)
	}
	if len(records) != 2 {
		t.Fatalf("greet --json wrote %d JSON values, want 2 (reporter line + result):\n%s", len(records), stdout)
	}
	if got := records[0]["message"]; got != "greeting bob" {
		t.Errorf("reporter record message = %v, want %q", got, "greeting bob")
	}
	result := records[1]
	if got := result["greeting"]; got != "hello bob" {
		t.Errorf("result greeting = %v, want %q", got, "hello bob")
	}
	if got := result["verbose"]; got != true {
		t.Errorf("result verbose = %v, want true (--verbose was passed)", got)
	}
	if got := result["dry_run"]; got != true {
		t.Errorf("result dry_run = %v, want true (--dry-run was passed)", got)
	}
}

func TestSilentWinsOverJSON(t *testing.T) {
	stdout, stderr, code := run(t, "greet", "--json", "--silent")
	if code != 0 {
		t.Fatalf("greet --json --silent exit code = %d, want 0", code)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty under --silent", stderr)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("stdout under --json --silent must be exactly one JSON value (no reporter line): %v\n%s", err, stdout)
	}
	if got := result["greeting"]; got != "hello world" {
		t.Errorf("result greeting = %v, want %q", got, "hello world")
	}
}

func TestSilentSuppressesTextProgress(t *testing.T) {
	stdout, stderr, code := run(t, "greet", "--silent")
	if code != 0 {
		t.Fatalf("greet --silent exit code = %d, want 0", code)
	}
	if stdout != "hello world\n" {
		t.Errorf("stdout = %q, want the data line to survive --silent", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty under --silent", stderr)
	}
}

func TestErrorEnvelopeAndExitCode(t *testing.T) {
	stdout, stderr, code := run(t, "fail", "--json")
	if code != 1 {
		t.Fatalf("fail --json exit code = %d, want 1", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("fail --json stdout is not a JSON error envelope: %v\n%s", err, stdout)
	}
	if got := envelope["error"]; got != true {
		t.Errorf("envelope error = %v, want true", got)
	}
	if got := envelope["message"]; got != "fail command failed" {
		t.Errorf("envelope message = %v, want %q", got, "fail command failed")
	}
	if got := envelope["details"]; got != "boom" {
		t.Errorf("envelope details = %v, want %q", got, "boom")
	}
	if !strings.Contains(stderr, "boom") {
		t.Errorf("stderr = %q, want fang's rendered error mentioning the cause", stderr)
	}

	stdout, _, code = run(t, "fail")
	if code != 1 {
		t.Fatalf("fail exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("fail stdout = %q, want empty without --json", stdout)
	}
}

func TestUnknownCommandExitsNonZero(t *testing.T) {
	stdout, stderr, code := run(t, "nope")
	if code == 0 {
		t.Fatal("unknown command exit code = 0, want non-zero")
	}
	if stdout != "" {
		t.Errorf("unknown command stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "nope") {
		t.Errorf("unknown command stderr = %q, want it to name the unknown command", stderr)
	}
}
