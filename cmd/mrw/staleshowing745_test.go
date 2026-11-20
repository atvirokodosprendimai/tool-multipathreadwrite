package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Final pass of #745. A record of another version is no showing: read real.txt:1,
// change line 1, read link.txt:3, and occurrence=2 through link.txt counted the
// old read of line 1 for the new line.
func TestAStaleShowingUnderAnAliasDoesNotCountForAnOccurrence(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	real := filepath.Join(root, "real.txt")
	if err := os.WriteFile(real, []byte("start\none\nstart\ntwo\nrest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("no symlinks: %v", err)
	}
	if out, code := runIn(t, root, "read", "real.txt:1"); code != 0 {
		t.Fatalf("read: %d\n%s", code, out)
	}
	if err := os.WriteFile(real, []byte("start first\none\nstart\ntwo\nrest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "read", "link.txt:3"); code != 0 {
		t.Fatalf("read: %d\n%s", code, out)
	}
	plan := planFile(t, "@@ link.txt /^start/ replace anchor=\"start\" occurrence=2\nnew\n")
	out, code := writeIn(t, root, "--no-check", plan)
	if code != 1 || !strings.Contains(out, "have not been read") {
		t.Fatalf("the old showing of line 1 counted for the new line: exit %d\n%s", code, out)
	}
	if out, code := runIn(t, root, "read", "link.txt:1"); code != 0 {
		t.Fatalf("read: %d\n%s", code, out)
	}
	if out, code := writeIn(t, root, "--no-check", plan); code != 0 {
		t.Errorf("after reading line 1 of the current file: exit %d\n%s", code, out)
	}
}
