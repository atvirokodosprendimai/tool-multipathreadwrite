package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-214 Decision 10. A mark on a document is transport and stays so: the
// plan's leading marks are dropped, a counted raw body line that begins with
// one is content, `read -` drops marks and `write --create` keeps them; and a
// CRLF target keeps its terminators under a native edit that keeps its mark.
// M is the UTF-8 byte order mark, written as bytes.
func TestTransportMarksStayTransport(t *testing.T) {
	const m = "\xEF\xBB\xBF"
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	disk := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	// Two marks before the first header are the document's and are dropped; the
	// counted body line M@@ x is content and is written.
	var out string
	var code int
	withStdin(t, m+m+"@@ t.txt 0 create body=2\n"+m+"@@ x\nplain\n", func() { out, code = runIn(t, root, "write", "--no-check", "-") })
	if code != 0 {
		t.Fatalf("plan with leading marks: exit %d\n%s", code, out)
	}
	if got := disk("t.txt"); got != m+"@@ x\nplain\n" {
		t.Errorf("t.txt = % x, want the body line with its own mark and no document mark", got)
	}

	// read - drops every leading mark and licenses nothing.
	withStdin(t, m+m+"one\n", func() { out, code = runIn(t, root, "read", "-") })
	if code != 0 || strings.Contains(out, m) || strings.Contains(out, "byte order mark") {
		t.Errorf("read -: exit %d, output %q: want the marks dropped and no note", code, out)
	}

	// write --create keeps both marks and names them.
	note := stderrOf(t, func() {
		withStdin(t, m+m+"one\n", func() { out, code = runIn(t, root, "write", "--no-check", "--create", "c.txt") })
	})
	if code != 0 || disk("c.txt") != m+m+"one\n" || !strings.Contains(note, "byte order mark") {
		t.Errorf("--create: exit %d, file % x, stderr %q: want both marks kept and named", code, disk("c.txt"), note)
	}

	// A CRLF target keeps its terminators, and its mark, under a native edit.
	if err := os.WriteFile(filepath.Join(root, "w.txt"), []byte(m+"one\r\ntwo\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, code = runIn(t, root, "read", "w.txt"); code != 0 {
		t.Fatalf("read w.txt: exit %d", code)
	}
	withStdin(t, "@@ w.txt 1 replace\nONE\n", func() { out, code = runIn(t, root, "write", "--no-check", "-") })
	if code != 0 {
		t.Fatalf("native edit: exit %d\n%s", code, out)
	}
	if got := disk("w.txt"); got != m+"ONE\r\ntwo\r\n" {
		t.Errorf("w.txt = % x, want the mark at byte zero and CRLF kept", got)
	}
}
