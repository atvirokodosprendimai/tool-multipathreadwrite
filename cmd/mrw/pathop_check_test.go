package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

func TestUnlinkOfGoRunsTheCheckByDefault(t *testing.T) {
	needShell(t)
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
	needShell(t)
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

func TestRenameGoToTxtRunsTheCheckByDefault(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, planFile(t, "@@ a.go - rename\nout.txt\n"))
	if code != exitCheckFailed {
		t.Fatalf("rename of a.go to .txt exited %d, want %d:\n%s", code, exitCheckFailed, out)
	}
	if _, err := os.Stat(filepath.Join(root, "out.txt")); err != nil {
		t.Fatalf("dest missing after rename+check: %v\n%s", err, out)
	}
}

func TestWriteCheckPathsKeepsRenameSourcePackage(t *testing.T) {
	paths, code := writeCheckPaths([]apply.FileResult{
		{Path: "pkg/a.go", Written: true, Removed: true, RenamedTo: "other/out.txt"},
		{Path: "other/out.txt", Written: true, Created: true},
	})
	if !code {
		t.Fatal("renaming a .go to .txt did not count as code")
	}
	want := map[string]bool{"pkg": false, "other/out.txt": false}
	for _, p := range paths {
		if _, ok := want[p]; !ok {
			t.Errorf("unexpected check path %q", p)
		}
		want[p] = true
	}
	for p, seen := range want {
		if !seen {
			t.Errorf("missing check path %q in %v", p, paths)
		}
	}
}
