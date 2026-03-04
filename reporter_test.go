package clix

import (
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
