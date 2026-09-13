package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnlinkOfGoRunsTheCheckByDefault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, planFile(t, "@@ a.go - unlink\n"))
	if code != exitCheckFailed {
		t.Fatalf("unlink of a.go exited %d, want %d:\n%s", code, exitCheckFailed, out)
	}
	if !strings.Contains(out, "removed a.go") {
		t.Errorf("receipt missing after unlink+check:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "a.go")); !os.IsNotExist(err) {
		t.Fatalf("a.go still exists after unlink: %v", err)
	}
}

func TestRenameOfGoRunsTheCheckOnTheDest(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, planFile(t, "@@ a.go - rename\nb.go\n"))
	if code != exitCheckFailed {
		t.Fatalf("rename of a.go exited %d, want %d:\n%s", code, exitCheckFailed, out)
	}
	if _, err := os.Stat(filepath.Join(root, "b.go")); err != nil {
		t.Fatalf("dest missing after rename+check: %v\n%s", err, out)
	}
}
