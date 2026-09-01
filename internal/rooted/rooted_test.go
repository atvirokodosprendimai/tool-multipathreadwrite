package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realRoot is the root as Resolve sees it. On macOS t.TempDir() hands back a
// path under /var, which is itself a symlink to /private/var — so a test that
// compares against the unevaluated string is asserting the platform, not the
// boundary.
func realRoot(t *testing.T, root string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func TestResolveAcceptsWhatIsInside(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"a.go", "sub/a.go", "sub/../sub/a.go", "./a.go", "."} {
		got, err := Resolve(root, path)
		if err != nil {
			t.Errorf("Resolve(%q) refused a path inside the root: %v", path, err)
			continue
		}
		if !strings.HasPrefix(got, realRoot(t, root)) {
			t.Errorf("Resolve(%q) = %q, which is not under %q", path, got, realRoot(t, root))
		}
	}
}

func TestResolveRefusesWhatIsOutside(t *testing.T) {
	root := t.TempDir()

	for _, path := range []string{"../escape.txt", "sub/../../escape.txt", "/etc/hosts"} {
		if _, err := Resolve(root, path); err == nil && path != "/etc/hosts" {
			t.Errorf("Resolve(%q) was accepted", path)
		}
	}
	// An absolute path is joined onto the root rather than honoured, so it
	// lands inside; the check is that it cannot ESCAPE, not that it is
	// rejected. Stated here because the opposite is easy to assume.
	if got, err := Resolve(root, "/etc/hosts"); err != nil || !strings.HasPrefix(got, realRoot(t, root)) {
		t.Errorf("an absolute path resolved to %q (err %v); want it joined under the root", got, err)
	}
}

// A symlink is the other way out, and the reason the check resolves before it
// compares: /etc/hosts was read through one.
func TestResolveRefusesASymlinkOutOfTheRoot(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(outer, "secret.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := Resolve(root, "link.txt"); err == nil {
		t.Error("a symlink pointing out of the root was accepted")
	}
	if _, err := Resolve(root, "nothere.txt"); err != nil {
		t.Errorf("a file that does not exist yet was refused: %v — create needs it", err)
	}
}

// The separator in the prefix check is load-bearing: without it a sibling whose
// name merely STARTS with the root's counts as inside.
func TestASiblingWithASharedPrefixIsOutside(t *testing.T) {
	if Contains("/repo", "/repo-backup/secret") {
		t.Error(`"/repo-backup/secret" was judged to be inside "/repo"`)
	}
	if !Contains("/repo", "/repo/a.go") || !Contains("/repo", "/repo") {
		t.Error("a real child, or the root itself, was judged to be outside")
	}
}
