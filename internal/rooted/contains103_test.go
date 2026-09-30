package rooted

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-103 T1. Contains appended a separator to the root, so under `/` it asked
// for the prefix `//` and refused every child: `mrw --root / read etc/hosts`
// answered "outside the root /". A root that already ends in a separator — the
// filesystem root, or a Windows volume root — is its own prefix.
func TestContainsHoldsUnderAFilesystemRoot(t *testing.T) {
	sep := string(filepath.Separator)
	root := filepath.VolumeName(os.TempDir()) + sep
	for _, p := range []string{root, root + "a", filepath.Join(root, "a", "b")} {
		if !Contains(root, p) {
			t.Errorf("Contains(%q, %q) = false; a filesystem root contains everything beneath it", root, p)
		}
	}
	repo := filepath.Join(root, "repo")
	if !Contains(repo, filepath.Join(repo, "x")) {
		t.Errorf("Contains(%q, its child) = false", repo)
	}
	if Contains(repo, repo+"-backup") {
		t.Errorf("Contains(%q, %q) = true; a sibling that shares the prefix is not inside", repo, repo+"-backup")
	}
}
