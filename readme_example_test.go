package clix

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The README's Go examples are part of the contract clix ships: renaming or
// removing an exported symbol they use must fail the Unit Tests job, not wait
// for a reader to paste a broken snippet. Every fenced ```go block is
// extracted; the complete `package main` block is compiled against this
// checkout with `go build`, and every `clix.<Ident>` selector in any block is
// resolved against the package's exported names read from source.

var goFence = regexp.MustCompile("(?s)```go\n(.*?)\n```")
var clixSelector = regexp.MustCompile(`\bclix\.([A-Za-z_][A-Za-z0-9_]*)`)

// goBlocks returns the body of every ```go fenced block in markdown, in order.
func goBlocks(markdown string) []string {
	var blocks []string
	for _, m := range goFence.FindAllSubmatch([]byte(markdown), -1) {
		blocks = append(blocks, string(m[1]))
	}
	return blocks
}

// unknownIdentifiers returns, sorted and unique, every `clix.<Ident>` used in
// any Go block of markdown that is not in exported.
func unknownIdentifiers(markdown string, exported map[string]bool) []string {
	seen := map[string]bool{}
	var unknown []string
	for _, block := range goBlocks(markdown) {
		for _, m := range clixSelector.FindAllStringSubmatch(block, -1) {
			name := m[1]
			if exported[name] || seen[name] {
				continue
			}
			seen[name] = true
			unknown = append(unknown, name)
		}
	}
	slices.Sort(unknown)
	return unknown
}

// exportedNames reads the exported top-level identifiers of the non-test Go
// files in dir (funcs without receivers, types, vars, consts) from source, so
// the check tracks the API without a hand-maintained list.
func exportedNames(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	names := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && ast.IsExported(d.Name.Name) {
					names[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if ast.IsExported(s.Name.Name) {
							names[s.Name.Name] = true
						}
					case *ast.ValueSpec:
						for _, ident := range s.Names {
							if ast.IsExported(ident.Name) {
								names[ident.Name] = true
							}
						}
					}
				}
			}
		}
	}
	return names, nil
}

// errNoGoToolchain reports that `go` is not on PATH; callers skip, never pass.
var errNoGoToolchain = errors.New("go toolchain not found on PATH")

// buildProgram compiles source, a complete `package main` file that imports
// github.com/frostyard/clix, against the checkout at repoRoot: it writes the
// file into a temp module whose go.mod replaces clix with repoRoot, copies the
// checkout's go.sum so the build resolves offline from the module cache, and
// runs `go build`. The returned error carries the compiler output.
func buildProgram(source, repoRoot string) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return errNoGoToolchain
	}
	dir, err := os.MkdirTemp("", "clix-readme-example-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return err
	}
	goMod := "module readmeexample\n\ngo 1.26\n\nrequire github.com/frostyard/clix v0.0.0\n\nreplace github.com/frostyard/clix => " + absRoot + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return err
	}
	goSum, err := os.ReadFile(filepath.Join(absRoot, "go.sum"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0o644); err != nil {
		return err
	}
	build := exec.Command(goBin, "build", "-o", filepath.Join(dir, "readmeexample"), ".")
	build.Dir = dir
	// -mod=mod lets the temp module take clix's requirements through the
	// replace without a tidy step; the copied go.sum vouches for them.
	build.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOWORK=off")
	out, err := build.CombinedOutput()
	if err != nil {
		return errors.New("go build: " + err.Error() + "\n" + string(out))
	}
	return nil
}

func TestREADMEGoExamples(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	blocks := goBlocks(string(readme))
	if len(blocks) < 3 {
		t.Fatalf("README.md has %d ```go fenced blocks, want at least 3 (usage program, flags fragment, viper fragment)", len(blocks))
	}

	// The exit-code contract from PR #40 lives in the first (usage) block.
	usage := blocks[0]
	if !strings.HasPrefix(usage, "package main") {
		t.Fatalf("README.md first Go block is not a complete package main program:\n%s", usage)
	}
	for _, want := range []string{"app.Run(rootCmd)", "os.Exit(1)"} {
		if !strings.Contains(usage, want) {
			t.Errorf("README.md first Go block lacks %q:\n%s", want, usage)
		}
	}

	// Every clix.<Ident> in every block is exported by the package today.
	exported, err := exportedNames(".")
	if err != nil {
		t.Fatalf("read exported names: %v", err)
	}
	if unknown := unknownIdentifiers(string(readme), exported); len(unknown) != 0 {
		t.Errorf("README.md Go blocks use identifiers clix does not export: %v", unknown)
	}

	// The complete program compiles against this checkout.
	if err := buildProgram(usage, "."); err != nil {
		if errors.Is(err, errNoGoToolchain) {
			t.Skip("go toolchain not on PATH; cannot compile the README usage example")
		}
		t.Fatalf("README.md usage example does not build:\n%v", err)
	}
}

// TestREADMEExampleHelpers proves both signals fire: an unknown identifier is
// reported, and a program with a type error is a build failure.
func TestREADMEExampleHelpers(t *testing.T) {
	fixture := "text\n\n```go\napp := clix.App{}\n_ = clix.NoSuchSymbolForProof\n```\n\n```go\nvar _ = clix.OutputJSON\n```\n"
	exported, err := exportedNames(".")
	if err != nil {
		t.Fatalf("read exported names: %v", err)
	}
	if got := unknownIdentifiers(fixture, exported); !slices.Equal(got, []string{"NoSuchSymbolForProof"}) {
		t.Errorf("unknownIdentifiers() = %v, want [NoSuchSymbolForProof]", got)
	}
	if got := len(goBlocks(fixture)); got != 2 {
		t.Errorf("goBlocks() found %d blocks, want 2", got)
	}

	broken := "package main\n\nimport \"github.com/frostyard/clix\"\n\nfunc main() {\n\tvar app clix.App\n\tvar wrong int = \"not an int\"\n\t_, _ = app, wrong\n}\n"
	err = buildProgram(broken, ".")
	if errors.Is(err, errNoGoToolchain) {
		t.Skip("go toolchain not on PATH; cannot prove the compile path")
	}
	if err == nil {
		t.Fatal("buildProgram() = nil for a program with a type error, want a build failure")
	}
	if !strings.Contains(err.Error(), "cannot use") {
		t.Errorf("buildProgram() error = %v, want the compiler's type error in it", err)
	}
}
