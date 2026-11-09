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
	"fmt"
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
		{"pinned reusable workflow", "frostyard/core/.github/workflows/pr-triage.yml@" + hex40 + " # core#225", ""},
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

// riskCaller is the caller workflow file. The name stays risk-paths.yml; the
// workflow it calls is core's PR triage workflow since the list moved to
// version 2 (core ADR-0077, ADR-0078).
const riskCaller = ".github/workflows/risk-paths.yml"

// riskReusable is the reusable workflow that every caller pins to a full
// 40-character core commit SHA (core ADR-0021).
const riskReusable = "frostyard/core/.github/workflows/pr-triage.yml@"

// riskLevels are the 4 levels of a version 2 list (core ADR-0078).
var riskLevels = []string{"L1", "L2", "L3", "L4"}

// checkRiskPathsList reports every problem in a .github/risk-paths.json text:
// version 2, a default_level, a level on every entry, no unknown key at any
// level (deny by default, ADR-0019), and rule C with levels. The globs that
// cover the list and the caller must be L1 or L2. A glob is compared to each
// required path as it is written; this repository's list names both paths
// exactly. alsoRequired names further paths that the list must cover.
func checkRiskPathsList(text string, alsoRequired ...string) []string {
	var findings []string
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &top); err != nil {
		return []string{"not valid JSON: " + err.Error()}
	}
	for key := range top {
		if !contains([]string{"schema_version", "default_level", "paths"}, key) {
			findings = append(findings, "unknown top-level key "+strconv.Quote(key))
		}
	}
	var version int
	if raw, ok := top["schema_version"]; !ok {
		findings = append(findings, "no schema_version")
	} else if err := json.Unmarshal(raw, &version); err != nil {
		findings = append(findings, "schema_version is not an integer")
	} else if version != 2 {
		findings = append(findings, "schema_version = "+strconv.Itoa(version)+", want 2")
	}
	var defaultLevel string
	if raw, ok := top["default_level"]; !ok {
		findings = append(findings, "no default_level")
	} else if err := json.Unmarshal(raw, &defaultLevel); err != nil {
		findings = append(findings, "default_level is not a string")
	} else if !contains(riskLevels, defaultLevel) {
		findings = append(findings, "default_level = "+strconv.Quote(defaultLevel)+", want one of L1, L2, L3, L4")
	}
	raw, ok := top["paths"]
	if !ok {
		return append(findings, "no paths")
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return append(findings, "paths is not an array of objects")
	}
	if len(entries) == 0 {
		findings = append(findings, "paths is empty")
	}
	levels := map[string]string{} // glob -> level
	type entryLevel struct{ glob, level string }
	var ordered []entryLevel
	for i, entry := range entries {
		where := "entry " + strconv.Itoa(i+1)
		for key := range entry {
			if !contains([]string{"glob", "level", "why"}, key) {
				findings = append(findings, where+" has unknown key "+strconv.Quote(key))
			}
		}
		var glob, level, why string
		if raw, ok := entry["glob"]; !ok {
			findings = append(findings, where+" has no glob")
			continue
		} else if err := json.Unmarshal(raw, &glob); err != nil || strings.TrimSpace(glob) == "" {
			findings = append(findings, where+" has an empty glob")
			continue
		}
		if _, seen := levels[glob]; seen {
			findings = append(findings, "duplicate glob "+strconv.Quote(glob))
		}
		if raw, ok := entry["why"]; !ok {
			findings = append(findings, "glob "+strconv.Quote(glob)+" has no why")
		} else if err := json.Unmarshal(raw, &why); err != nil || strings.TrimSpace(why) == "" {
			findings = append(findings, "glob "+strconv.Quote(glob)+" has an empty why")
		}
		raw, ok := entry["level"]
		if !ok {
			findings = append(findings, "glob "+strconv.Quote(glob)+" has no level")
			continue
		}
		if err := json.Unmarshal(raw, &level); err != nil || !contains(riskLevels, level) {
			findings = append(findings, "glob "+strconv.Quote(glob)+" has level "+strconv.Quote(level)+", want one of L1, L2, L3, L4")
			continue
		}
		levels[glob] = level
		ordered = append(ordered, entryLevel{glob: glob, level: level})
	}
	// Rule C: every glob that matches the list or the caller is L1 or L2, and
	// the list covers both. The match uses the spec's glob semantics: "?" is 1
	// character inside a segment and "**/" matches 0 directories.
	for _, required := range append([]string{".github/risk-paths.json", riskCaller}, alsoRequired...) {
		covered := false
		for _, entry := range ordered {
			if !matchRiskGlob(entry.glob, required) {
				continue
			}
			covered = true
			if (required == ".github/risk-paths.json" || required == riskCaller) && entry.level != "L1" && entry.level != "L2" {
				findings = append(findings, "glob "+strconv.Quote(entry.glob)+" is "+entry.level+", want L1 or L2 (it matches "+required+")")
			}
		}
		if !covered {
			findings = append(findings, "the list does not cover "+required)
		}
	}
	return findings
}

// matchRiskGlob holds when glob matches path under the glob syntax of the
// risk-paths spec (docs/specs/risk-paths.md, rule A), as core's matchGlob
// reads it: "?" is exactly 1 character inside a segment, "*" is 0 or more
// characters inside a segment, "**" as a whole segment is 0 or more segments,
// and {a,b} is 1 of the alternatives. An empty glob or path never matches, and
// a glob whose brace syntax is invalid matches nothing, as in core.
func matchRiskGlob(glob, pathName string) bool {
	if glob == "" || pathName == "" {
		return false
	}
	expansions, ok := expandRiskBraces(glob)
	if !ok {
		return false
	}
	for _, expansion := range expansions {
		if matchRiskSegments(strings.Split(expansion, "/"), strings.Split(pathName, "/")) {
			return true
		}
	}
	return false
}

// expandRiskBraces expands each {a,b} group of a glob into 1 glob for each
// alternative, and holds a false second result when the brace syntax is
// invalid, as core's expandBraces reads it: an unmatched brace, nested braces,
// a group with fewer than 2 alternatives, an empty alternative, or more than
// 256 expansions.
func expandRiskBraces(glob string) ([]string, bool) {
	var parts [][]string
	literal := ""
	for i := 0; i < len(glob); {
		switch glob[i] {
		case '}':
			return nil, false
		case '{':
			close := strings.IndexByte(glob[i+1:], '}')
			if close < 0 {
				return nil, false
			}
			close += i + 1
			if next := strings.IndexByte(glob[i+1:], '{'); next >= 0 && i+1+next < close {
				return nil, false
			}
			alternatives := strings.Split(glob[i+1:close], ",")
			if len(alternatives) < 2 {
				return nil, false
			}
			for _, alternative := range alternatives {
				if alternative == "" {
					return nil, false
				}
			}
			parts = append(parts, []string{literal}, alternatives)
			literal = ""
			i = close + 1
		default:
			literal += glob[i : i+1]
			i++
		}
	}
	parts = append(parts, []string{literal})
	expansions := []string{""}
	for _, alternatives := range parts {
		next := make([]string, 0, len(expansions)*len(alternatives))
		for _, prefix := range expansions {
			for _, alternative := range alternatives {
				next = append(next, prefix+alternative)
			}
		}
		if len(next) > 256 {
			return nil, false
		}
		expansions = next
	}
	return expansions, true
}

func matchRiskSegments(globSegments, pathSegments []string) bool {
	memo := map[[2]int]bool{}
	seen := map[[2]int]bool{}
	var visit func(int, int) bool
	visit = func(g, p int) bool {
		key := [2]int{g, p}
		if seen[key] {
			return memo[key]
		}
		seen[key] = true
		var result bool
		switch {
		case g == len(globSegments):
			result = p == len(pathSegments)
		case globSegments[g] == "**":
			result = visit(g+1, p) || (p < len(pathSegments) && visit(g, p+1))
		case p < len(pathSegments):
			result = matchRiskSegment(globSegments[g], pathSegments[p]) && visit(g+1, p+1)
		}
		memo[key] = result
		return result
	}
	return visit(0, 0)
}

// matchRiskSegment matches 1 path segment against 1 glob segment with the
// rules of core's segmentPattern: "?" is exactly 1 character, "*" is 0 or more
// characters, and every other character is literal. "*" and "?" are recognized
// before literal equality, so "*a" matches "*ba" and "*" matches "**", as in
// core. A character class or a backslash is literal, as core reads it, not
// syntax.
func matchRiskSegment(glob, segment string) bool {
	pattern, value := []rune(glob), []rune(segment)
	i, j := 0, 0
	star, mark := -1, 0
	for j < len(value) {
		switch {
		case i < len(pattern) && pattern[i] == '*':
			star = i
			mark = j
			i++
		case i < len(pattern) && pattern[i] == '?':
			i++
			j++
		case i < len(pattern) && pattern[i] == value[j]:
			i++
			j++
		case star >= 0:
			i = star + 1
			mark++
			j = mark
		default:
			return false
		}
	}
	for i < len(pattern) && pattern[i] == '*' {
		i++
	}
	return i == len(pattern)
}

// prTriageJob is the job id of the caller: the reusable workflow is called
// from this job, never from another one.
const prTriageJob = "pr-triage"

// riskCallerUses is the 1 body line of the caller that may vary: the core pin,
// with 40 lowercase hex characters and an optional trailing " # comment".
const riskCallerUses = "    uses: " + riskReusable + "<SHA>[ # comment]"

// riskCallerBody is the caller after its leading comment block: every line must
// equal this body exactly (core ADR-0068, ADR-0077, ADR-0078; the risk-paths
// and PR triage specs). An exact text check cannot be fooled by a comment that
// holds a key, by a quoted scalar or by a second types: list.
var riskCallerBody = []string{
	"name: PR triage",
	"",
	"on:",
	"  pull_request_target:",
	"    types: [opened, synchronize, reopened, edited, labeled]",
	"",
	"permissions: {}",
	"",
	"jobs:",
	"  " + prTriageJob + ":",
	"    permissions:",
	"      contents: read",
	"      pull-requests: write",
	riskCallerUses,
}

// checkRiskPathsCaller compares the caller with its canonical body and reports
// the first difference.
func checkRiskPathsCaller(text string) []string {
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	start := 0
	for start < len(lines) && (strings.TrimSpace(lines[start]) == "" || strings.HasPrefix(strings.TrimSpace(lines[start]), "#")) {
		start++
	}
	body := lines[start:]
	for i, want := range riskCallerBody {
		if i >= len(body) {
			return []string{fmt.Sprintf("%s ends at line %d, want line %d: %q", riskCaller, len(body), i+1, want)}
		}
		if want == riskCallerUses {
			if problem := checkRiskCallerUses(body[i], i+1); problem != "" {
				return []string{problem}
			}
			continue
		}
		if body[i] != want {
			return []string{fmt.Sprintf("%s line %d is %q, want %q", riskCaller, i+1, body[i], want)}
		}
	}
	if len(body) > len(riskCallerBody) {
		return []string{fmt.Sprintf("%s has an extra line %d: %q", riskCaller, len(riskCallerBody)+1, body[len(riskCallerBody)])}
	}
	return nil
}

// checkRiskCallerUses holds when line calls core's PR triage workflow at a full
// 40-character lowercase commit SHA, with an optional " # comment" after it.
func checkRiskCallerUses(line string, number int) string {
	rest, ok := strings.CutPrefix(line, "    uses: "+riskReusable)
	if !ok {
		return fmt.Sprintf("%s line %d is %q, want %q", riskCaller, number, line, riskCallerUses)
	}
	sha, _, _ := strings.Cut(rest, " #")
	if len(sha) != 40 || strings.Trim(sha, "0123456789abcdef") != "" {
		return fmt.Sprintf("%s line %d holds the SHA %q, want 40 lowercase hex characters", riskCaller, number, sha)
	}
	return ""
}

func TestBaselineB6RiskPaths(t *testing.T) {
	r := loadRepo(t)
	r.require(t, ".github/risk-paths.json", riskCaller)
	noFindings(t, ".github/risk-paths.json", checkRiskPathsList(r.read(t, ".github/risk-paths.json"), ".github/workflows/release.yml"))
	noFindings(t, riskCaller, checkRiskPathsCaller(r.read(t, riskCaller)))
}

func TestCheckRiskPathsList(t *testing.T) {
	const valid = `{"schema_version":2,"default_level":"L3","paths":[
		{"glob":".github/risk-paths.json","level":"L2","why":"governance: the list"},
		{"glob":".github/workflows/risk-paths.yml","level":"L2","why":"governance: the caller"}]}`
	tests := []struct {
		name, text, want string
	}{
		{"version 2 list", valid, ""},
		{"version 1 list", `{"schema_version":1,"paths":[
			{"glob":".github/risk-paths.json","level":"L2","why":"governance"},
			{"glob":".github/workflows/risk-paths.yml","level":"L2","why":"governance"}]}`,
			"schema_version = 1, want 2"},
		{"missing level", `{"schema_version":2,"default_level":"L3","paths":[
			{"glob":".github/risk-paths.json","why":"governance"},
			{"glob":".github/workflows/risk-paths.yml","level":"L2","why":"governance"}]}`,
			`glob ".github/risk-paths.json" has no level`},
		{"bad default_level", strings.Replace(valid, `"default_level":"L3"`, `"default_level":"high"`, 1),
			`default_level = "high", want one of L1, L2, L3, L4`},
		{"missing default_level", strings.Replace(valid, `"default_level":"L3",`, "", 1), "no default_level"},
		{"caller glob at L3", strings.Replace(valid, `risk-paths.yml","level":"L2"`, `risk-paths.yml","level":"L3"`, 1),
			`glob ".github/workflows/risk-paths.yml" is L3, want L1 or L2`},
		{"list glob at L4", strings.Replace(valid, `risk-paths.json","level":"L2"`, `risk-paths.json","level":"L4"`, 1),
			`glob ".github/risk-paths.json" is L4, want L1 or L2`},
		{"level outside L1 to L4", strings.Replace(valid, `"level":"L2","why":"governance: the list"`, `"level":"L5","why":"governance: the list"`, 1),
			`want one of L1, L2, L3, L4`},
		{"unknown top-level key", strings.Replace(valid, `"paths":`, `"owner":"core","paths":`, 1),
			`unknown top-level key "owner"`},
		{"unknown entry key", strings.Replace(valid, `"why":"governance: the list"`, `"why":"governance: the list","reason":"x"`, 1),
			`entry 1 has unknown key "reason"`},
		{"no caller glob", `{"schema_version":2,"default_level":"L3","paths":[
			{"glob":".github/risk-paths.json","level":"L2","why":"governance"}]}`,
			"the list does not cover .github/workflows/risk-paths.yml"},
		// Rule C matches with the spec's glob syntax, so a wildcard glob that
		// matches the list or the caller is L1 or L2 too.
		{"caller glob with a wildcard at L3", `{"schema_version":2,"default_level":"L3","paths":[
			{"glob":".github/risk-paths.json","level":"L2","why":"governance: the list"},
			{"glob":".github/workflows/risk-paths.yml","level":"L2","why":"governance: the caller"},
			{"glob":".github/workflows/risk-paths.ym?","level":"L3","why":"governance: the caller"}]}`,
			`glob ".github/workflows/risk-paths.ym?" is L3, want L1 or L2`},
		{"list glob behind a double star at L3", `{"schema_version":2,"default_level":"L3","paths":[
			{"glob":".github/risk-paths.json","level":"L2","why":"governance: the list"},
			{"glob":".github/workflows/risk-paths.yml","level":"L2","why":"governance: the caller"},
			{"glob":"**/.github/risk-paths.json","level":"L3","why":"governance: the list"}]}`,
			`glob "**/.github/risk-paths.json" is L3, want L1 or L2`},
		{"not JSON", "{", "not valid JSON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.Join(checkRiskPathsList(tt.text), "; ")
			if tt.want == "" {
				if got != "" {
					t.Errorf("checkRiskPathsList(%s) = %q, want no finding", tt.name, got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("checkRiskPathsList(%s) = %q, want it to name %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestMatchRiskGlob pins the matcher to core's matchGlob (core ADR-0068, the
// risk-paths spec rule A), with 1 case for each way the 2 can disagree: an
// empty glob or path, a malformed brace group, a backslash and a character
// class, and the wildcard and "**" boundaries. Each case that once disagreed
// fails here if the matcher goes back to path.Match or to loose braces.
func TestMatchRiskGlob(t *testing.T) {
	tests := []struct {
		name  string
		glob  string
		path  string
		match bool
	}{
		{"empty glob", "", "a", false},
		{"empty path", "**", "", false},
		{"empty glob and path", "", "", false},
		{"brace group with 1 alternative", "{a}", "a", false},
		{"brace group with an empty alternative", "{a,}", "a", false},
		{"nested braces", "{a,{b,c}}", "b", false},
		{"nested braces leave no match", "{a,{b,c}}", "a}", false},
		{"backslash is literal", `a\b`, `a\b`, true},
		{"backslash does not escape", `a\b`, "ab", false},
		{"character class is literal", "[ab]", "[ab]", true},
		{"character class does not match its member", "[ab]", "a", false},
		{"question mark is 1 character", "a?b", "a😀b", true},
		{"2 question marks need 2 characters", "a??b", "a😀b", false},
		{"double star matches 0 directories", "**/x", "x", true},
		{"double star between 2 names matches no directory", "a/**/b", "a/b", true},
		{"double star between 2 names matches 1 directory", "a/**/b", "a/b/c", false},
		{"brace alternative with a question mark", "{a?,b}", "ax", true},
		{"brace first", ".github/{risk-paths.jso?,unused}", ".github/risk-paths.json", true},
		{"non-ASCII literal", "café/**", "café/file.go", true},
		{"non-ASCII alternative", "{café,thé}/x.go", "thé/x.go", true},
		{"star does not cross a directory", "*.go", "a/b.go", false},
		{"2 stars in 1 segment do not cross a directory", "a**b", "a/x/b", false},
		{"star before literal equality", "*a", "*ba", true},
		{"star matches a star", "*", "**", true},
		{"2 stars in 1 segment match 2 stars", "a*b", "a**b", true},
		{"question mark matches a star", "?", "*", true},
		{"question mark matches itself", "?", "?", true},
		{"question mark matches a star in the middle", "a?c", "a*c", true},
		{"star file pattern matches a star file", "*.go", "*.go", true},
		{"star then question mark", "*?", "**", true},
		{"trailing star does not match a leading star", "*b", "b*", false},
		{"literal question marks match", "a?c", "a?c", true},
		{"star segment matches a star segment", "a/*/c", "a/*/c", true},
		{"double star then star", "**/*a", "x/*ba", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchRiskGlob(tt.glob, tt.path); got != tt.match {
				t.Errorf("matchRiskGlob(%q, %q) = %v, want %v (core matchGlob)", tt.glob, tt.path, got, tt.match)
			}
		})
	}
}

func TestCheckRiskPathsCaller(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", riskCaller))
	if err != nil {
		t.Fatalf("read %s: %v", riskCaller, err)
	}
	text := string(data)
	usesPrefix := "    uses: " + riskReusable
	usesLine := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, usesPrefix) {
			usesLine = line
			break
		}
	}
	if usesLine == "" {
		t.Fatalf("%s has no uses: line for the core pin", riskCaller)
	}
	sha, _, _ := strings.Cut(strings.TrimPrefix(usesLine, usesPrefix), " #")
	if len(sha) != 40 {
		t.Fatalf("%s pins %q, want a 40-character SHA", riskCaller, sha)
	}
	types := "    types: [opened, synchronize, reopened, edited, labeled]"
	tests := []struct {
		name string
		text string
		want string
	}{
		{"the committed caller", text, ""},
		{"another SHA and another trailing comment",
			strings.Replace(text, usesLine, usesPrefix+strings.Repeat("a", 40)+" # pinned", 1), ""},
		{"pin under another job name",
			strings.Replace(text, "jobs:\n  "+prTriageJob+":", "jobs:\n  other:", 1), `want "  ` + prTriageJob + `:"`},
		{"caller that pins risk-paths.yml",
			strings.Replace(text, riskReusable, "frostyard/core/.github/workflows/risk-paths.yml@", 1), riskReusable + "<SHA>"},
		{"short SHA", strings.Replace(text, usesPrefix+sha, usesPrefix+sha[:7], 1), "40 lowercase hex"},
		{"no labeled type", strings.Replace(text, types, "    types: [opened, synchronize, reopened, edited]", 1), types},
		{"commented-out pull-requests scope",
			strings.Replace(text, "      pull-requests: write", "      # pull-requests: write", 1), "pull-requests: write"},
		{"an extra permission",
			strings.Replace(text, "      pull-requests: write\n", "      pull-requests: write\n      actions: read\n", 1), "actions: read"},
		{"a scalar job header with a comment",
			strings.Replace(text, "  "+prTriageJob+":", "  "+prTriageJob+": | # note: text", 1), prTriageJob + ": | # note: text"},
		{"the types list inside a comment",
			strings.Replace(text, types, "    types: # [opened, synchronize, reopened, edited, labeled]", 1), "types: #"},
		{"an added merge_group trigger",
			strings.Replace(text, "  "+prTriageJob+":", "  merge_group:\n  "+prTriageJob+":", 1), "merge_group"},
		{"an extra line inside the job",
			strings.Replace(text, "      contents: read\n", "      contents: read\n    timeout-minutes: 5\n", 1), "timeout-minutes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.Join(checkRiskPathsCaller(tt.text), "; ")
			if tt.want == "" {
				if got != "" {
					t.Fatalf("checkRiskPathsCaller(%s) = %q, want no finding", tt.name, got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("checkRiskPathsCaller(%s) = %q, want it to name %q", tt.name, got, tt.want)
			}
		})
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
