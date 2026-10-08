package read

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// ADR-131. A walk resolves the files it discovers through a cache that lives
// as long as the walk and no longer. What changed between two walks — a
// directory that became mrw's state base, a directory swapped for a link out
// of the root — is seen by the second.
func TestAWalkCacheDoesNotOutliveItsWalk(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	for name, body := range map[string]string{".st/mrw/k/seen": "needle\n", "a/f.txt": "needle\n", "b.txt": "needle\n"} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "f.txt"), []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("needle")
	walk := func() map[string]bool {
		t.Helper()
		specs, _, err := Walk(root, nil, WalkOptions{Pattern: re})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, s := range specs {
			got[filepath.ToSlash(s.Path)] = true
		}
		return got
	}

	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "elsewhere"))
	if got := walk(); !got[".st/mrw/k/seen"] || !got["a/f.txt"] {
		t.Fatalf("before the change the walk found %v, want .st/mrw/k/seen and a/f.txt", got)
	}

	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	if got := walk(); got[".st/mrw/k/seen"] {
		t.Errorf("a directory that became mrw's state base after the last walk was searched: %v", got)
	}

	if err := os.RemoveAll(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "a")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if got := walk(); got["a/f.txt"] {
		t.Errorf("a directory swapped for a link out of the root after the last walk was searched: %v", got)
	}
}
