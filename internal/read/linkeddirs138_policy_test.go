package read

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// The Codex review of #369. The caller's exclusion and the ignore rules apply to
// a link as to a file: an excluded link name is not reported, and a link a
// .gitignore rule names is counted as the ignored directory it is, not as a
// link.
func TestALinkTheCallerExcludedOrIgnoredIsNotCountedAsALink(t *testing.T) {
	root := linkTree(t)
	if err := os.Symlink("real", filepath.Join(root, "alias")); err != nil {
		t.Skipf("links are not available here: %v", err)
	}
	var ex WalkSkipped
	if _, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Exclude: []string{"alias"}, Skipped: &ex}); err != nil {
		t.Fatal(err)
	}
	if ex.LinkedDirs != 0 {
		t.Errorf("--exclude alias still reported the link: %+v", ex)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("alias\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var ig WalkSkipped
	if _, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &ig}); err != nil {
		t.Fatal(err)
	}
	if ig.LinkedDirs != 0 || ig.IgnoredDirs != 1 {
		t.Errorf(".gitignore naming the link: %+v, want it counted as 1 ignored directory and no link", ig)
	}
}
