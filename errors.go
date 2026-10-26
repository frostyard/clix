package clix

import (
	"fmt"
	"io"
	"strings"

	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
)

// errorHandler is the error handler RunContext installs with
// fang.WithErrorHandler.
//
// fang v2.0.1 renders the whole error text as one styled paragraph (help.go,
// DefaultErrorHandler; theme.go, Styles.ErrorText with
// Transform(titleFirstWord) and Width(width()-4)). clix renders one line per
// line of the error instead, so a multi-line error such as the errors.Join of
// two failures never depends on how the width style wraps (fang v1.0.0
// reflowed the text into one sentence; issue #107). That per-line rendering
// also corrects two effects of the one-paragraph style:
//
//   - the first word is capitalized even when it is a flag, so
//     "--clear cannot be combined with --other" printed as
//     "--Clear cannot be combined with --other";
//   - the closing period is appended even when the text already ends in one,
//     so "failed." printed as "failed..".
//
// On a terminal the handler prints fang's header, then the error line by
// line. A writer that is not a terminal gets the plain error text: fang's own
// comment says a non-tty gets the error "without any styling", but the
// *colorprofile.Writer fang passes hides the file, so that plain branch never
// ran and a journal or a pipe got the styled block with only the ANSI
// sequences stripped. Non-terminal output therefore differs from clix v0.4.0
// and main.
func errorHandler(w io.Writer, styles fang.Styles, err error) {
	if !styledErrorWriter(w) {
		// fang v2.0.1 (help.go, DefaultErrorHandler) intends the plain error
		// for a writer that is not a terminal. Print it here, through w, so
		// the color profile still strips any escape sequence.
		_, _ = fmt.Fprintln(w, err.Error())
		return
	}

	lines := errorLines(err.Error())
	// The closing period goes on the text, before the style pads the line to
	// its width.
	lines[len(lines)-1] = withClosingPeriod(lines[len(lines)-1])
	for i, line := range lines {
		lines[i] = styledErrorLine(styles, line, i == 0)
	}

	_, _ = fmt.Fprintln(w, styles.ErrorHeader.String())
	for _, line := range lines {
		_, _ = fmt.Fprintln(w, line)
	}
	_, _ = fmt.Fprintln(w)

	if isUsageError(err) {
		_, _ = fmt.Fprintln(w, lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.ErrorText.UnsetWidth().Render("Try"),
			styles.Program.Flag.Render(" --help "),
			styles.ErrorText.UnsetWidth().UnsetMargins().UnsetTransform().Render("for usage."),
		))
		_, _ = fmt.Fprintln(w)
	}
}

// styledErrorWriter reports whether the error gets the styled rendering.
//
// It mirrors the rule fang v2.0.1 uses (help.go, DefaultErrorHandler): only a
// term.File that is not a terminal gets the plain rendering, and a writer that
// is no file at all takes the styled path. fang also wraps the writer in a
// *colorprofile.Writer before it calls the handler, which no term.File check
// can look through, so that wrapper is unwrapped first: a redirected or piped
// stderr is recognized as such and gets the plain error.
func styledErrorWriter(w io.Writer) bool {
	if cw, ok := w.(*colorprofile.Writer); ok {
		w = cw.Forward
	}
	f, ok := w.(term.File)
	return !ok || term.IsTerminal(f.Fd())
}

// errorLines splits text into its non-empty lines, so that an error ending in
// a newline does not print an empty line. An empty text yields one empty
// line, which withClosingPeriod turns into fang's bare period.
func errorLines(text string) []string {
	var lines []string
	for line := range strings.SplitSeq(text, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// styledErrorLine renders one line of the error with fang's error text style.
//
// Only the first line keeps fang's first-word capitalization (theme.go,
// titleFirstWord), which title-cases the first word of the error. A first word
// that starts with "-" is a flag, and the style would turn "--clear" into
// "--Clear", so such a line keeps the text as it is.
func styledErrorLine(styles fang.Styles, line string, firstLine bool) string {
	if firstLine && !strings.HasPrefix(strings.TrimLeft(line, " \t"), "-") {
		return styles.ErrorText.Render(line)
	}
	return styles.ErrorText.UnsetTransform().Render(line)
}

// withClosingPeriod appends fang's closing period unless the line already ends
// in sentence punctuation. fang v2.0.1 always appends one, which printed
// "failed.." for an error that already ended in a period.
func withClosingPeriod(line string) string {
	if line == "" {
		return "."
	}
	switch line[len(line)-1] {
	case '.', '!', '?', ':':
		return line
	}
	return line + "."
}

// isUsageError reports whether err is a cobra usage error, for which fang
// prints the "Try --help for usage." hint. It is a copy of fang v2.0.1's
// unexported isUsageError (charm.land/fang/v2@v2.0.1/help.go), which marks
// every message cobra itself produces for a bad invocation. fang exports no
// predicate for it, so keep this list in step when fang is upgraded.
func isUsageError(err error) bool {
	message := err.Error()
	for _, prefix := range []string{
		"flag needs an argument:",
		"unknown flag:",
		"unknown shorthand flag:",
		"unknown command",
		"invalid argument",
	} {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}
	return false
}
