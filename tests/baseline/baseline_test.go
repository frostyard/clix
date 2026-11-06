// Package baseline_test holds clix to the frostyard repository baseline: core
// ADR-0071 and its checklist, docs/specs/repository-baseline.md (rows B1 to
// B17). Each row has 1 test; the checks that need no repository state (an
// action pin, a checkout's credentials, a top-level permissions block) are
// pure helpers with their own table tests, so a failure path is covered too.
//
// B9 (the tool documentation site) is exempt: clix is a library and
// pkg.go.dev documents it. B11 (the Go toolchain aligned with the fleet) is a
// follow-up.
//
// The tests read the index with `git ls-files -s -z` (modes and paths) and the
// contents from disk. Without git, every test skips.
package baseline_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const hex40 = "0123456789abcdef0123456789abcdef01234567"

// repo is the tracked tree: path to git file mode.
type repo struct {
	root  string
	files map[string]string
}

func loadRepo(t *testing.T) *repo {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skipf("git rev-parse failed, no repository to check: %v", err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		t.Skip("git rev-parse printed no top level folder")
	}
	raw, err := exec.Command("git", "-C", root, "ls-files", "-s", "-z").Output()
	if err != nil {
		t.Skipf("git ls-files failed: %v", err)
	}
	r := &repo{root: root, files: map[string]string{}}
	for _, entry := range strings.Split(string(raw), "\x00") {
		if meta, path, ok := strings.Cut(entry, "\t"); ok {
			if mode, _, ok := strings.Cut(meta, " "); ok {
				r.files[path] = mode
			}
		}
	}
	return r
}

func (r *repo) has(path string) bool { _, ok := r.files[path]; return ok }

// require fails the test for each tracked path that is absent.
func (r *repo) require(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if !r.has(path) {
			t.Errorf("%s is not tracked", path)
		}
	}
}

func (r *repo) read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.root, path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// noFindings fails the test with 1 line for each finding.
func noFindings(t *testing.T, what string, findings []string) {
	t.Helper()
	for _, finding := range findings {
		t.Errorf("%s: %s", what, finding)
	}
}

func contains(values []string, want string) bool { return slices.Contains(values, want) }

func indentOf(line string) int { return len(line) - len(strings.TrimLeft(line, " ")) }

// checkUsesValue reports why 1 `uses:` value is not pinned, or "" when it is.
func checkUsesValue(value string) string {
	ref, comment := value, ""
	if i := strings.Index(value, " #"); i >= 0 {
		ref = strings.TrimSpace(value[:i])
		comment = strings.TrimSpace(strings.TrimPrefix(value[i:], " #"))
	}
	if strings.HasPrefix(ref, "./") {
		return "" // a local action lives in this repository
	}
	i := strings.LastIndex(ref, "@")
	if i < 0 {
		return "no @ref"
	}
	if sha := ref[i+1:]; len(sha) != 40 || strings.Trim(sha, "0123456789abcdefABCDEF") != "" {
		return "ref is not a full 40-character commit SHA"
	}
	if comment == "" {
		return "no version comment"
	}
	return ""
}

// findUsesPins reports every `uses:` line that is not SHA-pinned with a
// version comment.
func findUsesPins(text string) []string {
	var findings []string
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		value, ok := strings.CutPrefix(trimmed, "uses:")
		if !ok {
			value, ok = strings.CutPrefix(trimmed, "- uses:")
		}
		if !ok {
			continue
		}
		if why := checkUsesValue(strings.TrimSpace(value)); why != "" {
			findings = append(findings, "line "+strconv.Itoa(i+1)+": "+trimmed+": "+why)
		}
	}
	return findings
}

// findCheckoutCredentials reports every actions/checkout step whose step block
// does not set `persist-credentials: false`.
func findCheckoutCredentials(text string) []string {
	lines := strings.Split(text, "\n")
	var findings []string
	for i, line := range lines {
		if !strings.Contains(line, "uses: actions/checkout@") {
			continue
		}
		indent, found := indentOf(line), false
		for _, next := range lines[i+1:] {
			if strings.TrimSpace(next) == "" {
				continue
			}
			if indentOf(next) < indent || (indentOf(next) == indent && strings.HasPrefix(strings.TrimSpace(next), "- ")) {
				break
			}
			if strings.TrimSpace(next) == "persist-credentials: false" {
				found = true
				break
			}
		}
		if !found {
			findings = append(findings, "line "+strconv.Itoa(i+1)+": actions/checkout without persist-credentials: false")
		}
	}
	return findings
}

// findTopLevelPermissions reports a workflow whose top-level `permissions:` key
// is absent, is not `{}`, or grants a scope other than `read`.
func findTopLevelPermissions(text string) []string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if indentOf(line) != 0 || !strings.HasPrefix(line, "permissions:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "permissions:"))
		if value == "{}" {
			return nil
		}
		if value != "" {
			return []string{"line " + strconv.Itoa(i+1) + ": top-level permissions is " + value}
		}
		var findings []string
		for j, next := range lines[i+1:] {
			if strings.TrimSpace(next) == "" {
				continue
			}
			if indentOf(next) == 0 {
				break
			}
			if parts := strings.SplitN(strings.TrimSpace(next), ":", 2); len(parts) != 2 || strings.TrimSpace(parts[1]) != "read" {
				findings = append(findings, "line "+strconv.Itoa(i+2+j)+": "+strings.TrimSpace(next)+" is not a read scope")
			}
		}
		return findings
	}
	return []string{"no top-level permissions key"}
}

func TestCheckUsesValue(t *testing.T) {
	tests := []struct {
		name, value, want string
	}{
		{"pinned with comment", "actions/checkout@" + hex40 + " # v7.0.1", ""},
		{"pinned reusable workflow", "frostyard/core/.github/workflows/risk-paths.yml@" + hex40 + " # core#194", ""},
		{"local action", "./.github/workflows/risk-paths.yml", ""},
		{"tag ref", "actions/checkout@v4 # v4.1.0", "ref is not a full 40-character commit SHA"},
		{"short sha", "actions/checkout@3d3c42e # v7.0.1", "ref is not a full 40-character commit SHA"},
		{"no ref", "actions/checkout # v7.0.1", "no @ref"},
		{"no comment", "actions/checkout@" + hex40, "no version comment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkUsesValue(tt.value); got != tt.want {
				t.Errorf("checkUsesValue(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestFindUsesPins(t *testing.T) {
	good := "steps:\n  - name: Checkout\n    uses: actions/checkout@" + hex40 + " # v7.0.1\n"
	bad := "steps:\n  - name: Checkout\n    uses: actions/checkout@v4\n"
	if got := findUsesPins(good); len(got) != 0 {
		t.Errorf("findUsesPins(good) = %v, want none", got)
	}
	if got := findUsesPins(bad); len(got) != 1 {
		t.Errorf("findUsesPins(bad) = %v, want 1 finding", got)
	}
}

func TestFindCheckoutCredentials(t *testing.T) {
	good := "    steps:\n      - name: Checkout\n        uses: actions/checkout@" + hex40 + " # v7.0.1\n        with:\n          persist-credentials: false\n"
	bad := "    steps:\n      - name: Checkout\n        uses: actions/checkout@" + hex40 + " # v7.0.1\n\n      - name: Next\n        run: true\n"
	if got := findCheckoutCredentials(good); len(got) != 0 {
		t.Errorf("findCheckoutCredentials(good) = %v, want none", got)
	}
	if got := findCheckoutCredentials(bad); len(got) != 1 {
		t.Errorf("findCheckoutCredentials(bad) = %v, want 1 finding", got)
	}
}

func TestFindTopLevelPermissions(t *testing.T) {
	tests := []struct {
		name, text string
		want       int
	}{
		{"empty object", "on: push\npermissions: {}\njobs: {}\n", 0},
		{"read scopes", "permissions:\n  contents: read\njobs: {}\n", 0},
		{"job scopes are not top level", "permissions: {}\njobs:\n  x:\n    permissions:\n      contents: write\n", 0},
		{"missing key", "on: push\njobs: {}\n", 1},
		{"write scope", "permissions:\n  contents: write\njobs: {}\n", 1},
		{"inline write-all", "permissions: write-all\n", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findTopLevelPermissions(tt.text); len(got) != tt.want {
				t.Errorf("findTopLevelPermissions(%q) = %v, want %d findings", tt.text, got, tt.want)
			}
		})
	}
}

func TestBaselineB1InstructionSurface(t *testing.T) {
	r := loadRepo(t)
	aliases := []string{"CLAUDE.md", "GEMINI.md", ".cursorrules", "CONTRIBUTING.md", ".github/copilot-instructions.md"}
	r.require(t, append(aliases, "AGENTS.md")...)
	if mode := r.files["AGENTS.md"]; mode != "100644" {
		t.Errorf("AGENTS.md mode = %s, want 100644", mode)
	}
	for _, alias := range aliases {
		if mode := r.files[alias]; mode != "120000" {
			t.Errorf("%s mode = %s, want a symlink (120000)", alias, mode)
		}
		target, err := filepath.EvalSymlinks(filepath.Join(r.root, alias))
		if err != nil {
			t.Errorf("%s does not resolve: %v", alias, err)
			continue
		}
		if want := filepath.Join(r.root, "AGENTS.md"); target != want {
			t.Errorf("%s resolves to %s, want %s", alias, target, want)
		}
	}
}

func TestBaselineB2CorrectionsInbox(t *testing.T) {
	r := loadRepo(t)
	r.require(t, ".memory/corrections.jsonl")
	want := "correction,date,evidence,promoted_to,scope"
	for i, line := range strings.Split(r.read(t, ".memory/corrections.jsonl"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var record map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Errorf("line %d is not a JSON object: %v", i+1, err)
			continue
		}
		keys := make([]string, 0, len(record))
		for key := range record {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		if got := strings.Join(keys, ","); got != want {
			t.Errorf("line %d has keys %s, want %s", i+1, got, want)
		}
	}
}

func TestBaselineB3DocsShape(t *testing.T) {
	r := loadRepo(t)
	r.require(t, "docs/README.md", "docs/org-adrs.md", "docs/adr/TEMPLATE.md",
		"docs/design/TEMPLATE.md", "docs/specs/TEMPLATE.md", "docs/plans/TEMPLATE.md")
	for _, folder := range []string{"docs/adr/", "docs/design/", "docs/specs/", "docs/plans/"} {
		found := false
		for path := range r.files {
			if strings.HasPrefix(path, folder) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s holds no tracked file", folder)
		}
	}
	if !strings.Contains(r.read(t, "docs/README.md"), "org-adrs.md") {
		t.Error("docs/README.md does not link org-adrs.md")
	}
	if !strings.Contains(r.read(t, "docs/org-adrs.md"), "ADR-0071") {
		t.Error("docs/org-adrs.md does not name ADR-0071")
	}
}

// makeRulePrereqs returns the prerequisites of one Makefile rule, or nil.
func makeRulePrereqs(makefile, target string) []string {
	for _, line := range strings.Split(makefile, "\n") {
		if rest, ok := strings.CutPrefix(line, target+":"); ok {
			return strings.Fields(rest)
		}
	}
	return nil
}

func TestBaselineB4MakeGate(t *testing.T) {
	r := loadRepo(t)
	r.require(t, "Makefile", "mise.toml", "mise.lock")
	makefile := r.read(t, "Makefile")
	for _, target := range []string{"verify", "check", "ci"} {
		if !strings.Contains(makefile, "\n"+target+":") {
			t.Errorf("the Makefile has no %s rule", target)
		}
	}
	if prereqs := makeRulePrereqs(makefile, "ci"); !contains(prereqs, "verify") {
		t.Errorf("the ci rule runs %v, want verify as a prerequisite", prereqs)
	}
}

func TestBaselineB5ActionsPinned(t *testing.T) {
	r := loadRepo(t)
	workflows := 0
	for path := range r.files {
		if ok, _ := filepath.Match(".github/workflows/*.yml", path); !ok {
			continue
		}
		workflows++
		text := r.read(t, path)
		noFindings(t, path+" (uses)", findUsesPins(text))
		noFindings(t, path+" (permissions)", findTopLevelPermissions(text))
		noFindings(t, path+" (checkout)", findCheckoutCredentials(text))
	}
	if workflows == 0 {
		t.Fatal("no workflow is tracked")
	}
}

func TestBaselineB6RiskPaths(t *testing.T) {
	r := loadRepo(t)
	r.require(t, ".github/risk-paths.json", ".github/workflows/risk-paths.yml")
	var list struct {
		SchemaVersion int `json:"schema_version"`
		Paths         []struct {
			Glob string `json:"glob"`
			Why  string `json:"why"`
		} `json:"paths"`
	}
	if err := json.Unmarshal([]byte(r.read(t, ".github/risk-paths.json")), &list); err != nil {
		t.Fatalf(".github/risk-paths.json is not valid JSON: %v", err)
	}
	if list.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", list.SchemaVersion)
	}
	var globs []string
	for _, entry := range list.Paths {
		globs = append(globs, entry.Glob)
		if strings.TrimSpace(entry.Why) == "" {
			t.Errorf("glob %q has no why", entry.Glob)
		}
	}
	for _, want := range []string{".github/risk-paths.json", ".github/workflows/risk-paths.yml", ".github/workflows/release.yml"} {
		if !contains(globs, want) {
			t.Errorf("the list does not cover %s", want)
		}
	}
	caller := "frostyard/core/.github/workflows/risk-paths.yml@"
	text := r.read(t, ".github/workflows/risk-paths.yml")
	i := strings.Index(text, caller)
	if i < 0 {
		t.Fatalf("risk-paths.yml does not call %s", caller)
	}
	ref := text[i+len(caller):]
	if len(ref) < 40 || strings.Trim(ref[:40], "0123456789abcdef") != "" {
		t.Errorf("the reusable workflow is not pinned to a 40-character commit SHA: %q", ref[:min(len(ref), 40)])
	}
}

func TestBaselineB7PRTemplate(t *testing.T) {
	r := loadRepo(t)
	r.require(t, ".github/pull_request_template.md")
	for _, line := range strings.Split(r.read(t, ".github/pull_request_template.md"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") && strings.Contains(line, "Risk tier") {
			return
		}
	}
	t.Error("the PR template has no Risk tier heading")
}

func TestBaselineB8SvuConfig(t *testing.T) {
	r := loadRepo(t)
	r.require(t, ".svu.yml")
	if r.has(".svu.yaml") {
		t.Error(".svu.yaml is tracked; svu reads .svu.yml only (ADR-0053)")
	}
	if !strings.Contains(r.read(t, "Makefile"), "svu next") {
		t.Error("the Makefile never calls `svu next`")
	}
}

func TestBaselineB10License(t *testing.T) {
	r := loadRepo(t)
	r.require(t, "LICENSE")
	if text := r.read(t, "LICENSE"); !strings.HasPrefix(text, "MIT License") {
		t.Errorf("LICENSE does not start with \"MIT License\"")
	}
}

func TestBaselineB12CIEntry(t *testing.T) {
	r := loadRepo(t)
	r.require(t, ".github/workflows/ci.yml")
	text := r.read(t, ".github/workflows/ci.yml")
	if !strings.Contains(text, "pull_request:") {
		t.Error("ci.yml has no pull_request trigger")
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "run: make ci" {
			return
		}
	}
	t.Error("no job of ci.yml runs `make ci`")
}

// dependabotEntries maps each package-ecosystem to its directory.
func dependabotEntries(text string) map[string]string {
	entries := map[string]string{}
	ecosystem := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(trimmed, "- package-ecosystem:"); ok {
			ecosystem = strings.TrimSpace(value)
			continue
		}
		if value, ok := strings.CutPrefix(trimmed, "package-ecosystem:"); ok {
			ecosystem = strings.TrimSpace(value)
			continue
		}
		if value, ok := strings.CutPrefix(trimmed, "directory:"); ok && ecosystem != "" {
			entries[ecosystem] = strings.TrimSpace(value)
		}
	}
	return entries
}

func TestDependabotEntries(t *testing.T) {
	text := "updates:\n  - package-ecosystem: gomod\n    directory: /\n    schedule:\n      interval: weekly\n" +
		"  - package-ecosystem: github-actions\n    directory: /\n"
	got := dependabotEntries(text)
	if got["gomod"] != "/" || got["github-actions"] != "/" {
		t.Errorf("dependabotEntries = %v, want gomod / and github-actions /", got)
	}
	if got := dependabotEntries("updates:\n  - package-ecosystem: gomod\n"); len(got) != 0 {
		t.Errorf("dependabotEntries without a directory = %v, want none", got)
	}
}

func TestBaselineB13Dependabot(t *testing.T) {
	r := loadRepo(t)
	r.require(t, ".github/dependabot.yml")
	entries := dependabotEntries(r.read(t, ".github/dependabot.yml"))
	// clix is a Go module with no other ecosystem, so the 2 that keep the
	// Go module and the action pins current are the whole list.
	for _, want := range []string{"github-actions", "gomod"} {
		directory, ok := entries[want]
		if !ok {
			t.Errorf("dependabot.yml has no %s entry", want)
			continue
		}
		if directory != "/" {
			t.Errorf("the %s entry watches %q, want /", want, directory)
		}
	}
}

func TestBaselineB14SecurityPolicy(t *testing.T) {
	r := loadRepo(t)
	r.require(t, "SECURITY.md")
	if !strings.Contains(r.read(t, "SECURITY.md"), "https://github.com/frostyard/clix/security") {
		t.Error("SECURITY.md does not link the private vulnerability reporting page")
	}
}

func TestBaselineB15NoCodeowners(t *testing.T) {
	r := loadRepo(t)
	for _, path := range []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"} {
		if r.has(path) {
			t.Errorf("%s is tracked; baseline row B15 allows no CODEOWNERS file", path)
		}
	}
}

func TestBaselineB16NoAgentFrameworkTrees(t *testing.T) {
	r := loadRepo(t)
	for path := range r.files {
		for _, prefix := range []string{"docs/superpowers/", ".planning/", ".cursor/rules/", ".gaai/", "yeti/", "cairn/"} {
			if strings.HasPrefix(path, prefix) {
				t.Errorf("%s is a leftover agent-framework path", path)
			}
		}
		if !strings.HasPrefix(path, ".claude/") || path == ".claude/skills" || strings.HasPrefix(path, ".claude/skills/") {
			continue
		}
		if path != ".claude/settings.json" && path != ".claude/session-summary.md" {
			t.Errorf("%s is not one of settings.json, skills, session-summary.md", path)
		}
	}
}

func TestBaselineB17Readme(t *testing.T) {
	r := loadRepo(t)
	r.require(t, "README.md")
	text := r.read(t, "README.md")
	if !strings.Contains(text, "<!-- readme-kind: library -->") {
		t.Error("README.md has no `<!-- readme-kind: library -->` line (ADR-0072)")
	}
	if !strings.Contains(text, "https://pkg.go.dev/github.com/frostyard/clix") {
		t.Error("README.md does not link https://pkg.go.dev/github.com/frostyard/clix")
	}
}
