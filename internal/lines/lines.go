// Package lines numbers a file's lines, once, for every surface that addresses
// them: the write engine, the read that licenses a write, --grep, MCP paging and
// the plan compilers. ADR-065: they used to split independently, so a CR-only
// file was one line to read and three to write, and a write to its line 2 applied
// though read had never served it.
package lines

import "strings"

// Split splits s into lines by the terminator s itself uses, and reports that
// terminator and whether s ended with it. Three are recognised (ADR-005 §3), in
// the order that makes each unambiguous: text whose every "\n" is "\r\n" is
// CRLF; text with no "\n" at all but a "\r" is CR-only (old Mac), whose interior
// would otherwise be one unaddressable line; anything else is LF, including text
// that MIXES them — there a stray "\r" stays part of its line, which is what
// keeps untouched lines byte-identical. Empty text has no lines.
func Split(s string) (ls []string, eol string, final bool) {
	if s == "" {
		return nil, "\n", false
	}
	eol = eolOf(s)
	final = strings.HasSuffix(s, eol)
	if final {
		s = s[:len(s)-len(eol)]
	}
	return strings.Split(s, eol), eol, final
}

func eolOf(s string) string {
	if !strings.Contains(s, "\n") {
		if strings.Contains(s, "\r") {
			return "\r"
		}
		return "\n"
	}
	// CRLF only when EVERY newline is one: a mixed file is left alone.
	if strings.Contains(s, "\r\n") &&
		strings.Count(s, "\r\n") == strings.Count(s, "\n") {
		return "\r\n"
	}
	return "\n"
}
