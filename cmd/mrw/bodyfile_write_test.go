package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-060 T4: CLI create body=@src.txt writes those bytes.
func TestWriteBodyAtPathCreatesFromFile(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{"src.txt": "alpha\nbeta\n"})
	doc := "@@ dest.txt 0 create body=@src.txt\n"
	out, code := writeIn(t, root, "--no-check", planFile(t, doc))
	if code != 0 {
		t.Fatalf("create body=@src.txt exited %d:\n%s", code, out)
	}
	got, err := os.ReadFile(filepath.Join(root, "dest.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "alpha\nbeta\n" && string(got) != "alpha\nbeta" {
		t.Errorf("dest.txt = %q, want alpha/beta from src.txt", got)
	}
}
