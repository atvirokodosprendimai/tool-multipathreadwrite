package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-142. `write --create` on what a PowerShell pipe sends: an LF file arrives
// with a CRLF appended and is created as it is, with a line on stderr naming
// what was dropped; a leading byte order mark is kept and named; a mixed shape
// that is not the pipe's terminator is still refused and makes no file.
func TestWriteCreateDropsAPipeTerminatorAndSaysSo(t *testing.T) {
	root := t.TempDir()
	var out string
	var code int
	for _, c := range []struct{ name, in, want, note string }{
		{"lf.txt", "a\nb\n\r\n", "a\nb\n", "CRLF"},
		{"bom.txt", "\xef\xbb\xbfa\n", "\xef\xbb\xbfa\n", "byte order mark"},
	} {
		note := stderrOf(t, func() {
			withStdin(t, c.in, func() { out, code = runIn(t, root, "write", "--no-check", "--create", c.name) })
		})
		if code != 0 {
			t.Fatalf("%s: exit %d\n%s", c.name, code, out)
		}
		if b, err := os.ReadFile(filepath.Join(root, c.name)); err != nil || string(b) != c.want {
			t.Errorf("%s = %q (err %v), want %q", c.name, b, err, c.want)
		}
		if !strings.Contains(note, "mrw: --create "+c.name+": ") || !strings.Contains(note, c.note) {
			t.Errorf("%s: stderr %q, want a line naming %q", c.name, note, c.note)
		}
	}
	note := stderrOf(t, func() {
		withStdin(t, "a\r\nb\n\r\n", func() { out, code = runIn(t, root, "write", "--no-check", "--create", "mixed.txt") })
	})
	if code == 0 || strings.Contains(note, "dropped") {
		t.Errorf("a mixed shape: exit %d, stderr %q, want a refusal and no drop", code, note)
	}
	if _, err := os.Stat(filepath.Join(root, "mixed.txt")); err == nil {
		t.Error("a refused --create made mixed.txt")
	}
	// The notes say what was done to content that goes on to the apply; content
	// CompileCreate then refuses is told nothing (the Codex review of #376).
	note = stderrOf(t, func() {
		withStdin(t, "\xef\xbb\xbfa\nb\r", func() { out, code = runIn(t, root, "write", "--no-check", "--create", "r.txt") })
	})
	if code == 0 || strings.Contains(note, "byte order mark") {
		t.Errorf("a refused create: exit %d, stderr %q, want a refusal and no note\n%s", code, note, out)
	}
}
