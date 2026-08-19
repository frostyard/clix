package clix

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestREADMEUsageExampleExitsNonZero pins the README "Usage" example to the
// exit-code contract tests/e2e/exampletool follows: App.Run returns the error
// and the caller must exit non-zero.
func TestREADMEUsageExampleExitsNonZero(t *testing.T) {
	t.Parallel()
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	m := regexp.MustCompile("(?s)```go\n(.*?)\n```").FindSubmatch(readme)
	if m == nil {
		t.Fatal("README.md has no ```go fenced block")
	}
	block := string(m[1])
	for _, want := range []string{"app.Run(rootCmd)", "os.Exit(1)"} {
		if !strings.Contains(block, want) {
			t.Errorf("README.md first Go block lacks %q:\n%s", want, block)
		}
	}
}
