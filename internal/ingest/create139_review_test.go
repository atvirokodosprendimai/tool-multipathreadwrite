package ingest

import (
	"reflect"
	"strings"
	"testing"
)

// bom is a UTF-8 byte order mark, written as bytes so no source file holds one.
const bom = "\xef\xbb\xbf"

// The Codex review of #371, P1. plan.Parse strips a BOM before it decides what
// is a header, so a content line "<BOM>@@ x 0 create" read as another hunk and
// wrote a second file. --create declares body=N raw=true, so the parser takes
// exactly N lines as content; apply_patch's Add File shares the emitter, which
// now makes the same decision the parser does.
func TestABOMPrefixedHeaderInContentIsContentNotAHunk(t *testing.T) {
	content := "first\n" + bom + "@@ extra.txt 0 create\nsecond\n"
	out, err := CompileCreate("main.txt", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if h := parseOne(t, out); !reflect.DeepEqual(h.Body, []string{"first", bom + "@@ extra.txt 0 create", "second"}) {
		t.Errorf("--create carried %q, want the three lines whole", h.Body)
	}
	patch := "*** Begin Patch\n*** Add File: add.txt\n+first\n+" + bom + "@@ extra.txt 0 create\n+second\n*** End Patch\n"
	compiled, err := CompileApplyPatch(t.TempDir(), []byte(patch))
	if err != nil {
		t.Fatal(err)
	}
	if h := parseOne(t, compiled); !reflect.DeepEqual(h.Body, []string{"first", bom + "@@ extra.txt 0 create", "second"}) {
		t.Errorf("apply_patch Add File carried %q, want the three lines whole", h.Body)
	}
}

// The Codex review of #371, P2. A header that quotes the path always: two
// backslashes and single quotes after "=" are syntax to the header splitter and
// named another file.
func TestAPathIsQuotedSoNoNameIsReadAsSyntax(t *testing.T) {
	for _, p := range []string{`a\\b.txt`, `a='b'.txt`, `dir/with"quote.txt`, `back\slash.txt`, "plain.txt", "with space.txt", "-dash.txt", "~tilde.txt"} {
		out, err := CompileCreate(p, []byte("x\n"))
		if err != nil {
			t.Errorf("%q: %v", p, err)
			continue
		}
		if h := parseOne(t, out); h.Path != p {
			t.Errorf("path %q came back as %q", p, h.Path)
		}
	}
}

// The Codex review of #371, P2. A bare CR ending a line is lost by the plan
// parser's line scanner; the content is refused rather than made without it.
// Every-ending-CRLF and every-ending-LF content are fine.
func TestAMixedEndingContentIsRefusedNotQuietlyChanged(t *testing.T) {
	if _, err := CompileCreate("m.txt", []byte("a\nb\r")); err == nil || !strings.Contains(err.Error(), "bare CR") {
		t.Errorf("mixed endings: %v, want a refusal naming the bare CR", err)
	}
	if out, err := CompileCreate("crlf.txt", []byte("a\r\nb\r\n")); err != nil {
		t.Errorf("CRLF content refused: %v", err)
	} else if h := parseOne(t, out); !reflect.DeepEqual(h.Body, []string{"a", "b"}) {
		t.Errorf("CRLF content: %q", h.Body)
	}
}

// The Codex review of #371, P2. A path no header can carry is a usage error
// before standard input is read.
func TestCheckCreatePathRefusesWhatNoHeaderCanCarry(t *testing.T) {
	for _, bad := range []string{"", "/abs.txt", "a\nb.txt", "a\rb.txt", "a\x00b.txt"} {
		if CheckCreatePath(bad) == nil {
			t.Errorf("CheckCreatePath(%q) accepted", bad)
		}
	}
	if err := CheckCreatePath("ok/new.txt"); err != nil {
		t.Errorf("CheckCreatePath(ok/new.txt) = %v", err)
	}
}
