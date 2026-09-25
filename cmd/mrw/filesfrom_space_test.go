package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-069 T2. A --files-from line was TrimSpaced, so a list naming "x "
// served x. The trim is needed only to recognise a blank line or a comment.
func TestFilesFromKeepsAPathsTrailingSpace(t *testing.T) {
	root := t.TempDir()
	for n, body := range map[string]string{"x": "plain\n", "x ": "padded\n"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(body), 0o644); err != nil {
			t.Skipf("this filesystem cannot hold %q: %v", n, err)
		}
	}
	if fi, err := os.ReadDir(root); err != nil || len(fi) != 2 {
		t.Skipf("this filesystem folds names that differ by a trailing space")
	}
	list := filepath.Join(t.TempDir(), "list")
	if err := os.WriteFile(list, []byte("# a comment\n\nx \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := readIn(t, root, "--files-from", list)
	if err != nil {
		t.Fatalf("--files-from failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "padded") || strings.Contains(out, "plain") {
		t.Errorf("a --files-from line naming %q did not serve that file:\n%s", "x ", out)
	}
}
