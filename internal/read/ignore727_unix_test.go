//go:build !windows

package read

import (
	"os"
	"path/filepath"
	"testing"
)

// #727. A directory's .gitignore was opened by name after its link check: a
// directory swapped for a link out of the checkout in between let rules from
// outside it decide what the walk served. The inside .gitignore ignores
// nothing; the outside one ignores every .txt.
func TestAGitignoreUnderADirectorySwappedForALinkOutIsNotRead(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".git", "d"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for p, body := range map[string]string{filepath.Join(root, "d", ".gitignore"): "nomatch\n", filepath.Join(outside, ".gitignore"): "*.txt\n"} {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var done bool
	prev := afterIgnoreCheckFn
	afterIgnoreCheckFn = func() {
		afterIgnoreCheckFn = prev
		if done {
			return
		}
		done = true
		if err := os.Rename(filepath.Join(root, "d"), filepath.Join(root, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(root, "d")); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { afterIgnoreCheckFn = prev })
	ig := newIgnorer(root)
	if ig == nil {
		t.Fatal("no ignorer")
	}
	if ig.Ignored("d/f.txt", false, 0) {
		t.Error("rules from a .gitignore outside the checkout ignored a file")
	}
}

// The pair: a .gitignore of the checkout still applies.
func TestAGitignoreInsideTheCheckoutStillApplies(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".git", "d"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "d", ".gitignore"), []byte("*.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ig := newIgnorer(root)
	if ig == nil || !ig.Ignored("d/f.txt", false, 0) {
		t.Error("a .gitignore inside the checkout did not apply")
	}
}
