package ingest

import (
	"strings"
	"testing"
)

// The v1.64.0 Windows retest (five sessions, 2026-10-13). A create plan ends
// every line in LF, so content whose lines end in CRLF, or in CR alone, is
// written with LF; two sessions found that silent. It is named now, and LF
// content is not.
func TestContentNotEndingInLFIsNamedWhenWrittenWithLF(t *testing.T) {
	for _, c := range []struct{ in, ending string }{
		{"a\r\nb\r\n", "CRLF"},
		{"a\r\nb", "CRLF"},
		{"a\r\nb\r\n\r\n", "CRLF"},
		{"a\rb\r", "CR"},
	} {
		n := EndingNote([]byte(c.in))
		if !strings.Contains(n, "end in "+c.ending+" ") || !strings.Contains(n, "written with LF") {
			t.Errorf("%q: note %q, want one saying the %s lines are written with LF", c.in, n, c.ending)
		}
	}
	for _, in := range []string{"a\nb\n", "a", "", "a\nb\r\n"} {
		if n := EndingNote([]byte(in)); n != "" {
			t.Errorf("%q: named %q, want no note", in, n)
		}
	}
}

// The same retest: Windows PowerShell 5.1 prefixes a byte order mark when
// [Console]::InputEncoding has a preamble, whatever $OutputEncoding is, which a
// process started under code page 65001 gets. The note names both.
func TestTheByteOrderMarkNoteNamesTheConsoleInputEncoding(t *testing.T) {
	_, notes := CreateContent([]byte("\xef\xbb\xbfa\n"))
	if len(notes) != 1 || !strings.Contains(notes[0], "$OutputEncoding") || !strings.Contains(notes[0], "[Console]::InputEncoding") {
		t.Errorf("notes = %q, want one naming $OutputEncoding and [Console]::InputEncoding", notes)
	}
}
