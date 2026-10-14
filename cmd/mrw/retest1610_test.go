package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The v1.61.0 Windows retest: a --create note says what was done to content that
// went into a file, so a create that then fails is told nothing.
func TestCreateNotesAreToldOnlyWhenTheCreateLanded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var code int
	note := stderrOf(t, func() {
		withStdin(t, "a\nb\n\r\n", func() { _, code = runIn(t, root, "write", "--no-check", "--create", "x.txt") })
	})
	if code == 0 || strings.Contains(note, "CRLF") {
		t.Errorf("a create of an existing path: exit %d, stderr %q, want a refusal and no note", code, note)
	}
	note = stderrOf(t, func() {
		withStdin(t, "a\nb\n\r\n", func() { _, code = runIn(t, root, "write", "--no-check", "--create", "y.txt") })
	})
	if code != 0 || !strings.Contains(note, "CRLF") {
		t.Errorf("a create that landed: exit %d, stderr %q, want the note", code, note)
	}
}
