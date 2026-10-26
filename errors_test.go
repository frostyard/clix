package clix

import (
	"bytes"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"charm.land/fang/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
)

// ansiEscape matches the SGR sequences lipgloss uses for the error styles.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// fangStyles returns the styles fang v2.0.1 builds for a CLI, which is the
// value it passes to an error handler. fang builds them in the unexported
// makeStyles, so they are captured from a real fang.Execute call.
func fangStyles(t *testing.T) fang.Styles {
	t.Helper()
	var captured fang.Styles
	root := &cobra.Command{
		Use: "styles",
		RunE: func(*cobra.Command, []string) error {
			return errors.New("capture")
		},
	}
	// A handler that only records the styles prints nothing.
	err := fang.Execute(t.Context(), root, fang.WithErrorHandler(
		func(_ io.Writer, styles fang.Styles, _ error) {
			captured = styles
		},
	))
	if err == nil {
		t.Fatal("fang.Execute() error = nil, want the command's error")
	}
	if captured.ErrorHeader.String() == "" {
		t.Fatal("captured fang styles are empty, want the styles fang builds")
	}
	return captured
}

// errorLinesOf returns the rendered output as visible text lines: ANSI
// sequences removed, trailing width padding trimmed, and blank lines dropped.
func errorLinesOf(t *testing.T, out string) []string {
	t.Helper()
	var lines []string
	for line := range strings.SplitSeq(ansiEscape.ReplaceAllString(out, ""), "\n") {
		if line = strings.TrimRight(line, " \t"); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// TestErrorHandlerJoinedErrorKeepsLines pins issue #107: an errors.Join error
// rendered on a terminal keeps each error on its own line.
func TestErrorHandlerJoinedErrorKeepsLines(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	errorHandler(&buf, fangStyles(t), errors.Join(
		errors.New("a failed"),
		errors.New("b failed"),
	))

	lines := errorLinesOf(t, buf.String())
	first, second := -1, -1
	for i, line := range lines {
		switch {
		case strings.Contains(strings.ToLower(line), "a failed"):
			first = i
		case strings.Contains(strings.ToLower(line), "b failed"):
			second = i
		}
	}
	if first < 0 || second < 0 {
		t.Fatalf("rendered lines = %q, want one line for each joined error", lines)
	}
	if first == second {
		t.Fatalf("both joined errors are on line %d of %q, want separate lines", first, lines)
	}
	if first > second {
		t.Errorf("joined errors rendered in order %d then %d, want the (a, b) order", first, second)
	}
	// fang capitalizes the first word of the error and closes the last line;
	// the second error line is not a sentence start, so it stays as it is.
	if got := lines[second]; !strings.HasSuffix(got, "b failed.") {
		t.Errorf("last error line = %q, want it to end with %q", got, "b failed.")
	}
}

// TestErrorHandlerKeepsLowercaseFlag pins that a leading flag keeps its lower
// case: fang's titleFirstWord would render "--Clear".
func TestErrorHandlerKeepsLowercaseFlag(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	errorHandler(&buf, fangStyles(t), errors.New("--clear cannot be combined with --other"))

	got := buf.String()
	if !strings.Contains(got, "--clear cannot be combined with --other") {
		t.Errorf("rendered error = %q, want the lowercase flags kept", got)
	}
	if strings.Contains(got, "--Clear") {
		t.Errorf("rendered error = %q, want no capitalized flag", got)
	}
}

// TestErrorHandlerStyledOneLineMatchesFang pins that a one-line error that
// needs the closing period renders exactly as fang renders it.
func TestErrorHandlerStyledOneLineMatchesFang(t *testing.T) {
	t.Parallel()

	styles := fangStyles(t)
	err := errors.New("a thing failed")
	var got, want bytes.Buffer
	errorHandler(&got, styles, err)
	fang.DefaultErrorHandler(&want, styles, err)

	if got.String() != want.String() {
		t.Errorf("one-line error = %q, want fang's rendering %q", got.String(), want.String())
	}
	if !strings.Contains(got.String(), "A thing failed.") {
		t.Errorf("one-line error = %q, want it capitalized and closed with a period", got.String())
	}
}

// TestErrorHandlerClosingPeriod pins that the closing period is added only
// when the error text needs one: fang v2.0.1 appends one unconditionally and
// printed "failed.." for an error that already ended in a period.
func TestErrorHandlerClosingPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "no punctuation", err: errors.New("failed"), want: "Failed."},
		{name: "period", err: errors.New("failed."), want: "Failed."},
		{name: "question mark", err: errors.New("failed?"), want: "Failed?"},
		{name: "exclamation mark", err: errors.New("failed!"), want: "Failed!"},
		{name: "colon", err: errors.New("failed:"), want: "Failed:"},
		{name: "trailing newline", err: errors.New("failed\n"), want: "Failed."},
		{name: "empty", err: errors.New(""), want: "."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			errorHandler(&buf, fangStyles(t), tt.err)
			lines := errorLinesOf(t, buf.String())
			if len(lines) != 2 {
				t.Fatalf("rendered lines = %q, want the header and one error line", lines)
			}
			if got := strings.TrimSpace(lines[1]); got != tt.want {
				t.Errorf("error line = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestErrorHandlerNonTerminalPrintsPlainError pins the path a journal or a
// pipe takes: the error text and one newline, with no header, style, closing
// period or usage hint. fang's own plain branch never runs, because it wraps
// the writer in a *colorprofile.Writer, so clix prints the plain error itself.
func TestErrorHandlerNonTerminalPrintsPlainError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "joined error",
			err:  errors.Join(errors.New("a failed"), errors.New("b failed")),
			want: "a failed\nb failed\n",
		},
		{
			name: "leading flag",
			err:  errors.New("--clear cannot be combined with --other"),
			want: "--clear cannot be combined with --other\n",
		},
		{
			name: "closing period",
			err:  errors.New("a thing failed."),
			want: "a thing failed.\n",
		},
		{
			name: "usage error",
			err:  errors.New("unknown flag: --bogus"),
			want: "unknown flag: --bogus\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A pipe write end is a *os.File that is not a terminal. fang
			// wraps the writer in a *colorprofile.Writer before it calls a
			// handler, so the test does the same.
			got := writeToPipe(t, func(w io.Writer) {
				errorHandler(colorprofile.NewWriter(w, nil), fangStyles(t), tt.err)
			})
			if got != tt.want {
				t.Errorf("non-terminal error = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestErrorHandlerUsageErrorKeepsHint pins that a usage error still ends with
// fang's "Try --help for usage." hint.
func TestErrorHandlerUsageErrorKeepsHint(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	errorHandler(&buf, fangStyles(t), errors.New("unknown flag: --bogus"))

	lines := errorLinesOf(t, buf.String())
	if got := lines[len(lines)-1]; !strings.Contains(got, "Try --help for usage.") {
		t.Errorf("last rendered line = %q, want fang's help hint", got)
	}
}

// TestErrorHandlerNoHintForOtherErrors pins that the hint stays limited to
// usage errors.
func TestErrorHandlerNoHintForOtherErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	errorHandler(&buf, fangStyles(t), errors.New("a thing failed"))

	if strings.Contains(buf.String(), "--help") {
		t.Errorf("rendered error = %q, want no help hint for a run error", buf.String())
	}
}

// writeToPipe runs write against the write end of an os.Pipe, which is a file
// and not a terminal, and returns everything written to it.
func writeToPipe(t *testing.T, write func(w io.Writer)) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	write(writer)
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the pipe writer: %v", err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading the pipe: %v", err)
	}
	return string(out)
}
