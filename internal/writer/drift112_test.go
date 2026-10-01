package writer

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ADR-112 T1. A write's check runs after the write lock is released, so a file
// the write landed can change while it runs — another writer, or the check
// itself — and the verdict is then about a tree that is not the write's. Drift
// names each written file whose bytes no longer hash to its sha_after.
func TestDriftNamesAFileChangedAfterTheWrite(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var in []apply.Input
	for i, name := range []string{"a.txt", "b.txt", "c.txt"} {
		in = append(in, apply.Input{Path: name, Start: 1, End: 1, Op: "replace", Body: []string{"new " + name}, Lines: -1, Index: i})
	}
	res, err := Apply(root, in, apply.Options{Force: true})
	if err != nil || !res.Applied {
		t.Fatalf("the write did not land: %v %+v", err, res)
	}
	before := Before(root, res)
	if d := Drift(root, before); len(d) != 0 {
		t.Errorf("nothing changed after the write, and Drift named %v", d)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "b.txt")); err != nil {
		t.Fatal(err)
	}
	if d := Drift(root, before); !slices.Equal(d, []string{"a.txt", "b.txt"}) {
		t.Errorf("a changed and a removed file: Drift named %v, want [a.txt b.txt]", d)
	}

	// The Codex review of #309: a renamed relative symlink resolves to another
	// file at its destination, so the baseline is what the destination holds
	// after the write, not the sha the link carried before it. An idle check
	// names nothing; a change to the new referent is named.
	root = t.TempDir()
	for name, body := range map[string]string{"a/payload.txt": "A\n", "b/payload.txt": "B\n"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("payload.txt", filepath.Join(root, "a", "link.txt")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	res, err = Apply(root, []apply.Input{{Path: "a/link.txt", Op: "rename", Body: []string{"b/link.txt"}, SrcLine: 1, Lines: -1, Index: 0}}, apply.Options{Force: true})
	if err != nil || !res.Applied {
		t.Fatalf("the rename did not land: %v %+v", err, res)
	}
	before = Before(root, res)
	if d := Drift(root, before); len(d) != 0 {
		t.Errorf("a renamed relative symlink with an idle check: Drift named %v", d)
	}
	if err := os.WriteFile(filepath.Join(root, "b", "payload.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if d := Drift(root, before); !slices.Equal(d, []string{"b/link.txt"}) {
		t.Errorf("the renamed link's new referent changed: Drift named %v, want [b/link.txt]", d)
	}
}
