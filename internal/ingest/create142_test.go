package ingest

import (
	"strings"
	"testing"
)

// ADR-142. PowerShell appends CRLF to what it pipes, so an LF file arrives as
// "a\nb\n\r\n". Only that shape loses its final CRLF; every other mixed shape
// is returned untouched for CompileCreate to refuse, and a note says what
// happened.
func TestAPowerShellPipeTerminatorAfterLFContentIsDropped(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"a\nb\n\r\n", "a\nb\n"},
		{"a\nb\r\n", "a\nb"},
		{"a\n\r\n", "a\n"},
	} {
		got, notes := CreateContent([]byte(c.in))
		if string(got) != c.want || len(notes) != 1 || !strings.Contains(notes[0], "CRLF") {
			t.Errorf("%q: got %q, notes %q; want %q and one note naming the CRLF", c.in, got, notes, c.want)
		}
		if _, err := CompileCreate("p.txt", got); err != nil {
			t.Errorf("%q: the cleaned content is refused: %v", c.in, err)
		}
	}
	for _, in := range []string{"a\r\nb\r\n", "a\r\n", "a", "a\nb\r", "a\r\nb\n\r\n", "a\rb\nc\r\n", "a\nb\n"} {
		got, notes := CreateContent([]byte(in))
		if string(got) != in || len(notes) != 0 {
			t.Errorf("%q: changed to %q with notes %q, want it untouched", in, got, notes)
		}
	}
	if _, err := CompileCreate("p.txt", []byte("a\r\nb\n\r\n")); err == nil {
		t.Error("a mixed shape that is not the pipe terminator was accepted")
	}
}

// PowerShell 5.1 prefixes a byte order mark. It is content and is kept, and
// the caller is told.
func TestALeadingByteOrderMarkIsKeptAndNamed(t *testing.T) {
	in := "\xef\xbb\xbfa\nb\n"
	got, notes := CreateContent([]byte(in))
	if string(got) != in {
		t.Errorf("the BOM content changed: %q", got)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "byte order mark") {
		t.Errorf("notes = %q, want one naming the byte order mark", notes)
	}
	if _, notes := CreateContent([]byte("a\n")); len(notes) != 0 {
		t.Errorf("content without a BOM got notes %q", notes)
	}
}

// An all-CRLF content that ends in an empty line cannot be told from a file that
// really ends in one: it is kept as it came and named (the Windows retest of
// v1.60.0, four sessions). Content that is not all CRLF is not named.
func TestAnEmptyLastLineAfterCRLFLinesIsKeptAndNamed(t *testing.T) {
	for _, in := range []string{"a\r\nb\r\n\r\n", "a\r\n\r\n"} {
		got, notes := CreateContent([]byte(in))
		if string(got) != in || len(notes) != 1 || !strings.Contains(notes[0], "empty line") || !strings.Contains(notes[0], "may be") {
			t.Errorf("%q: got %q, notes %q; want it kept and one note naming the empty line", in, got, notes)
		}
	}
	for _, in := range []string{"a\r\nb\r\n", "a\r\n", "a\nb\r\n\r\n", "a\n\n"} {
		if _, notes := CreateContent([]byte(in)); len(notes) != 0 {
			t.Errorf("%q: named as an empty last line: %q", in, notes)
		}
	}
}
