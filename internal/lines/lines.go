// Package lines numbers a file's lines, once, for every surface that addresses
// them: the write engine, the read that licenses a write, --grep, MCP paging and
// the plan compilers. ADR-065: they used to split independently, so a CR-only
// file was one line to read and three to write, and a write to its line 2 applied
// though read had never served it.
package lines

import (
	"bytes"
	"fmt"
	"strings"
)

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

// nulScan is how many leading bytes Unsplittable searches for a NUL.
const nulScan = 8192

// Unsplittable names what makes b impossible to edit by line, or returns "":
// a UTF-32 or UTF-16 byte-order mark, or a NUL byte in the first 8 KiB
// (ADR-073). Split works on bytes and "\n", and UTF-16 puts a NUL beside every
// ASCII byte, so a "line" of it is half a character pair and an edit written in
// UTF-8 lands between the halves: the v1.25.1 round rewrote a UTF-16 file that
// way at exit 0. The marks are checked longest first, because FF FE is the
// start of UTF-32LE's FF FE 00 00.
func Unsplittable(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE, 0x00, 0x00}):
		return "is UTF-32 (BOM FF FE 00 00)"
	case bytes.HasPrefix(b, []byte{0x00, 0x00, 0xFE, 0xFF}):
		return "is UTF-32 (BOM 00 00 FE FF)"
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		return "is UTF-16 (BOM FF FE)"
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return "is UTF-16 (BOM FE FF)"
	}
	head := b
	if len(head) > nulScan {
		head = head[:nulScan]
	}
	if i := bytes.IndexByte(head, 0); i >= 0 {
		return fmt.Sprintf("holds a NUL byte at offset %d", i)
	}
	return ""
}
