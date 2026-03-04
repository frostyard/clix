package clix

import "testing"

func TestVersionString(t *testing.T) {
	app := App{
		Version: "1.2.3",
		Commit:  "abc123",
		Date:    "2026-03-04",
		BuiltBy: "ci",
	}
	got := app.VersionString()
	want := "1.2.3 (Commit: abc123) (Date: 2026-03-04) (Built by: ci)"
	if got != want {
		t.Errorf("VersionString() = %q, want %q", got, want)
	}
}

func TestVersionStringDefaults(t *testing.T) {
	app := App{}
	got := app.VersionString()
	want := "dev (Commit: none) (Date: unknown) (Built by: local)"
	if got != want {
		t.Errorf("VersionString() = %q, want %q", got, want)
	}
}
