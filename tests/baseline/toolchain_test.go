package baseline_test

// Row B11 of the repository baseline (core ADR-0071): go.mod's
// `go 1.<minor>.<patch>` line is the only Go pin (core ADR-0043), mise.toml
// pins golangci-lint, and no other file pins another golangci-lint release.
// The row's fleet versions (the Go minor and the golangci-lint release) live
// in the frostyard-go-repo skill's "Fleet toolchain" table; the weekly sweep
// compares them with core's origin/main. These tests hold the shape that
// makes that comparison meaningful, and the Makefile guard that reads it.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	// goModGoLine matches go.mod's go directive in the form B11 requires.
	goModGoLine = regexp.MustCompile(`(?m)^go (1\.[0-9]+\.[0-9]+)[ \t]*$`)
	// goModAnyGoLine matches any go directive, to count them.
	goModAnyGoLine = regexp.MustCompile(`(?m)^go\s`)
	// goModToolchainLine matches a toolchain directive: a second Go pin.
	goModToolchainLine = regexp.MustCompile(`(?m)^toolchain\s`)
	// miseGoPin matches a go entry in mise.toml's [tools] table.
	miseGoPin = regexp.MustCompile(`(?m)^go\s*=`)
	// miseLintPin extracts the golangci-lint release from mise.toml.
	miseLintPin = regexp.MustCompile(`(?m)^golangci-lint = "([^"]*)"[ \t]*$`)
	// miseLockGo extracts the Go release that mise.lock verifies.
	miseLockGo = regexp.MustCompile(`(?m)^\[\[tools\.go\]\]\nversion = "([^"]*)"`)
	// lintVersionPin matches a golangci-lint release written into a workflow
	// or the Makefile instead of read from mise.toml.
	lintVersionPin = regexp.MustCompile(`golangci-lint[^\n]*\bv?2\.[0-9]+\.[0-9]+|golangci/golangci-lint-action`)
)

// b11Findings returns 1 line for each way the go.mod, mise.toml and mise.lock
// texts break row B11 or core ADR-0043.
func b11Findings(goMod, miseToml, miseLock string) []string {
	var findings []string
	goLines := goModGoLine.FindAllStringSubmatch(goMod, -1)
	if len(goLines) != 1 {
		findings = append(findings, fmt.Sprintf("go.mod has %d `go 1.<minor>.<patch>` line(s), want exactly 1", len(goLines)))
	}
	if n := len(goModAnyGoLine.FindAllString(goMod, -1)); n != 1 {
		findings = append(findings, fmt.Sprintf("go.mod has %d go directive(s), want exactly 1", n))
	}
	if goModToolchainLine.MatchString(goMod) {
		findings = append(findings, "go.mod has a toolchain line; the go line is the only Go pin (run `go mod edit -toolchain=none`)")
	}
	if miseGoPin.MatchString(miseToml) {
		findings = append(findings, "mise.toml pins go; Go comes from go.mod only")
	}
	if m := miseLintPin.FindStringSubmatch(miseToml); m == nil || m[1] == "" {
		findings = append(findings, "mise.toml pins no golangci-lint")
	}
	lock := miseLockGo.FindStringSubmatch(miseLock)
	switch {
	case lock == nil:
		findings = append(findings, "mise.lock verifies no Go release (run `mise lock`)")
	case len(goLines) == 1 && lock[1] != goLines[0][1]:
		findings = append(findings, fmt.Sprintf("mise.lock verifies go %s, go.mod says go %s (run `mise lock`)", lock[1], goLines[0][1]))
	}
	return findings
}

func TestB11Findings(t *testing.T) {
	const (
		mod  = "module example.org/m\n\ngo 1.27.4\n"
		toml = "[tools]\ngolangci-lint = \"2.15.0\"\n"
		lock = "lockfile_version = 1\n\n[[tools.go]]\nversion = \"1.27.4\"\n"
	)
	tests := []struct {
		name                  string
		goMod, miseToml, lock string
		want                  string
	}{
		{name: "conforming", goMod: mod, miseToml: toml, lock: lock},
		{name: "toolchain line", goMod: mod + "\ntoolchain go1.27.4\n", miseToml: toml, lock: lock, want: "toolchain line"},
		{name: "language-only go line", goMod: "module m\n\ngo 1.27\n", miseToml: toml, lock: lock, want: "want exactly 1"},
		{name: "two go lines", goMod: mod + "go 1.27.5\n", miseToml: toml, lock: lock, want: "2 `go 1.<minor>.<patch>` line(s)"},
		{name: "mise pins go", goMod: mod, miseToml: toml + "go = \"1.27.4\"\n", lock: lock, want: "mise.toml pins go"},
		{name: "no golangci-lint pin", goMod: mod, miseToml: "[tools]\n", lock: lock, want: "pins no golangci-lint"},
		{name: "lock without go", goMod: mod, miseToml: toml, lock: "lockfile_version = 1\n", want: "verifies no Go release"},
		{name: "lock behind go.mod", goMod: mod, miseToml: toml, lock: strings.Replace(lock, "1.27.4", "1.26.11", 1), want: "mise.lock verifies go 1.26.11, go.mod says go 1.27.4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.Join(b11Findings(tt.goMod, tt.miseToml, tt.lock), "\n")
			if tt.want == "" && got != "" {
				t.Fatalf("want no finding, got:\n%s", got)
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("want a finding with %q, got:\n%s", tt.want, got)
			}
		})
	}
}

func TestBaselineB11GoToolchain(t *testing.T) {
	r := loadRepo(t)
	r.require(t, "go.mod", "mise.toml", "mise.lock", "Makefile")
	noFindings(t, "B11", b11Findings(r.read(t, "go.mod"), r.read(t, "mise.toml"), r.read(t, "mise.lock")))

	// No other file pins a golangci-lint release: the Makefile and CI read
	// mise.toml (CI installs it with jdx/mise-action).
	paths := []string{"Makefile"}
	for path := range r.files {
		if strings.HasPrefix(path, ".github/workflows/") {
			paths = append(paths, path)
		}
	}
	for _, path := range paths {
		for i, line := range strings.Split(r.read(t, path), "\n") {
			if lintVersionPin.MatchString(line) {
				t.Errorf("%s:%d pins golangci-lint outside mise.toml: %s", path, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// makeTargetRecipe returns the rule line of target and its recipe lines.
func makeTargetRecipe(t *testing.T, makefile, target string) string {
	t.Helper()
	var recipe []string
	in := false
	for _, line := range strings.Split(makefile, "\n") {
		if !in {
			if strings.HasPrefix(line, target+":") {
				in = true
				recipe = append(recipe, line)
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "\t") {
			recipe = append(recipe, line)
			continue
		}
		break
	}
	if !in {
		t.Fatalf("the Makefile has no %s rule", target)
	}
	return strings.Join(recipe, "\n")
}

// TestMakefileLintVersionCheckReadsGoLine pins where the Makefile reads the Go
// version: go.mod's go line. go.mod has no toolchain line (B11), so a Makefile
// that read one would get an empty version, and its "built with" guard would
// pass for every Go.
func TestMakefileLintVersionCheckReadsGoLine(t *testing.T) {
	r := loadRepo(t)
	makefile := r.read(t, "Makefile")
	if strings.Contains(makefile, "^toolchain") || strings.Contains(makefile, "GO_TOOLCHAIN") {
		t.Error("the Makefile reads a toolchain line from go.mod; read the go line (the only Go pin)")
	}
	if !strings.Contains(makefile, "GO_VERSION := ") || !strings.Contains(makefile, "s/^go ") {
		t.Error("the Makefile does not read GO_VERSION from go.mod's go line")
	}
	if recipe := makeTargetRecipe(t, makefile, "lint-version-check"); !strings.Contains(recipe, `test -n "$(GO_LANGUAGE)"`) {
		t.Errorf("lint-version-check does not fail when go.mod has no readable go line; got:\n%s", recipe)
	}
}

// TestLintVersionCheckBehaviour runs the real lint-version-check recipe in a
// temporary folder with a fake golangci-lint on PATH: the pinned release
// built with the same Go minor passes whatever the patch, an older minor
// fails, another release fails, and a go.mod without a readable go line fails
// closed.
func TestLintVersionCheckBehaviour(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the make subprocess test in -short mode")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not on PATH")
	}
	r := loadRepo(t)
	makefile := r.read(t, "Makefile")
	miseToml := r.read(t, "mise.toml")
	m := miseLintPin.FindStringSubmatch(miseToml)
	if m == nil {
		t.Fatal("mise.toml pins no golangci-lint")
	}
	pin := m[1]

	const module = "module example.org/lintcheck\n\n"
	tests := []struct {
		name      string
		goMod     string
		short     string
		builtWith string
		want      string // "" means exit 0
	}{
		{name: "same minor, older patch", goMod: module + "go 1.27.4\n", short: pin, builtWith: "1.27.0"},
		{name: "newer minor", goMod: module + "go 1.27.4\n", short: pin, builtWith: "1.28.1"},
		{name: "older minor", goMod: module + "go 1.27.4\n", short: pin, builtWith: "1.26.11", want: "was built with go1.26.11, older than go.mod's go1.27"},
		{name: "another release", goMod: module + "go 1.27.4\n", short: "2.13.1", builtWith: "1.27.0", want: "expected golangci-lint " + pin + ", found 2.13.1"},
		{name: "no go line", goMod: module + "toolchain go1.27.4\n", short: pin, builtWith: "1.27.0", want: "go.mod has no readable go line"},
		{name: "unreadable go line", goMod: module + "go latest\n", short: pin, builtWith: "1.27.0", want: "go.mod has no readable go line"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range map[string]string{"Makefile": makefile, "mise.toml": miseToml, "go.mod": tt.goMod} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatalf("write %s: %v", name, err)
				}
			}
			bin := t.TempDir()
			script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = version ] && [ \"$2\" = --short ]; then echo %q; exit 0; fi\n"+
				"if [ \"$1\" = version ]; then echo \"golangci-lint has version %s built with go%s from 0000000 on 2026-01-01T00:00:00Z\"; exit 0; fi\nexit 1\n",
				tt.short, tt.short, tt.builtWith)
			if err := os.WriteFile(filepath.Join(bin, "golangci-lint"), []byte(script), 0o755); err != nil {
				t.Fatalf("write fake golangci-lint: %v", err)
			}
			cmd := exec.Command("make", "-s", "lint-version-check")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			var out bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &out
			err := cmd.Run()
			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("make lint-version-check: %v; output:\n%s", err, out.String())
			case tt.want != "" && err == nil:
				t.Fatalf("make lint-version-check exited 0, want a failure; output:\n%s", out.String())
			case tt.want != "" && !strings.Contains(out.String(), tt.want):
				t.Fatalf("output does not contain %q; got:\n%s", tt.want, out.String())
			}
		})
	}
}
