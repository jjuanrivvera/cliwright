// Template: internal/output/sanitize.go — copy into a generated CLI (GOAL.md §3d).
// Strips terminal-escape sequences from API-sourced text in the HUMAN output path only. Apply
// SanitizeTerminal to every free-text field (names, labels, error bodies) before it reaches a
// table cell or an error message; leave json/yaml/csv byte-faithful. Without it a value like a
// record named "\x1b]0;pwned\a" can rewrite the terminal title, move the cursor, or inject color
// — a real terminal-escape injection, distinct from the CSV-formula-injection guard.
package output

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeTerminal removes ANSI escape sequences and control characters (keeping tab/newline) from
// text that originated in the API. Fast-path returns the input unchanged when it's already clean.
func SanitizeTerminal(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool { return r == 0x1b || unicode.IsControl(r) }) {
		return s // fast path: nothing to strip (the common case)
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == 0x1b { // ESC — start of an ANSI sequence
			i = skipANSISequence(s, i)
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			i++ // drop invalid UTF-8
			continue
		}
		// Keep tab and newline (callers that want single-line cells strip separately);
		// drop every other control rune.
		if r == '\t' || r == '\n' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}

// CellOneLine collapses tabs/newlines to single spaces so a multi-line API value never breaks
// table layout. Use in the table renderer (after SanitizeTerminal).
func CellOneLine(s string) string {
	if !strings.ContainsAny(s, "\t\n\r") {
		return s
	}
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r == '\t' || r == '\n' || r == '\r' }), " ")
}

// skipANSISequence returns the index just past the ANSI escape sequence starting at i.
// Handles CSI (ESC [ … final-byte) and OSC (ESC ] … BEL or ESC \) plus lone/short escapes.
func skipANSISequence(s string, i int) int {
	n := len(s)
	i++ // past ESC
	if i >= n {
		return i
	}
	switch s[i] {
	case '[': // CSI: ESC [ params... final-byte (0x40–0x7E)
		i++
		for i < n && s[i] >= 0x20 && s[i] <= 0x3f {
			i++
		}
		if i < n && s[i] >= 0x40 && s[i] <= 0x7e {
			i++
		}
		return i
	case ']': // OSC: ESC ] ... terminated by BEL (0x07) or ST (ESC \)
		i++
		for i < n {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < n && s[i+1] == '\\' {
				return i + 2
			}
			i++
		}
		return i
	default: // lone ESC or ESC X — drop only the ESC, keep the following char
		return i
	}
}
