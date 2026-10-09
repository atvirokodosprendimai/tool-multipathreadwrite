package read

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func linkTree(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "real", "x.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// ADR-138. A walk skips a link to a directory (ADR-096) and said nothing, while
// it counts ignored files, binaries, nested repositories and unkeepable names:
// a grep that missed a linked directory told the caller nothing. The real path
// is served once, the link is counted, and the note says what to do.
func TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow(t *testing.T) {
	root := linkTree(t)
	if err := os.Symlink("real", filepath.Join(root, "alias")); err != nil {
		t.Skipf("links are not available here: %v", err)
	}
	var sk WalkSkipped
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || filepath.ToSlash(specs[0].Path) != "real/x.txt" {
		t.Errorf("the walk served %v, want real/x.txt alone", specs)
	}
	if sk.LinkedDirs != 1 {
		t.Errorf("LinkedDirs = %d, want 1 (alias)", sk.LinkedDirs)
	}
	note := SkipNote(sk, "--no-ignore")
	if !strings.Contains(note, "1 link(s) to a directory, not followed") || !strings.Contains(note, "name one to be told why") || strings.Contains(note, "--no-ignore walks them") {
		t.Errorf("note = %q, want the count and no promise that --no-ignore follows it", note)
	}
	// Beside an ignore-class count the tail keeps the flag for the others.
	mixed := SkipNote(WalkSkipped{Ignored: 2, LinkedDirs: 1}, "--no-ignore")
	if !strings.Contains(mixed, "--no-ignore walks the others, and naming one tells why") {
		t.Errorf("mixed note = %q", mixed)
	}
	var none WalkSkipped
	if _, _, err := Walk(linkTree(t), nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &none}); err != nil || none != (WalkSkipped{}) {
		t.Errorf("a tree with no link skipped %+v (err %v), want nothing", none, err)
	}
}

// A link that leaves the root is refused by Resolve first and stays silent: the
// count must not say where a link leads.
func TestAnEscapingLinkToADirectoryIsNotCounted(t *testing.T) {
	root := linkTree(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Skipf("links are not available here: %v", err)
	}
	var sk WalkSkipped
	if _, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk}); err != nil {
		t.Fatal(err)
	}
	if sk.LinkedDirs != 0 {
		t.Errorf("a link out of the root was counted: %+v", sk)
	}
}

// A link to a FILE is a candidate and is served, not counted as a directory.
func TestALinkToAFileIsServedNotCounted(t *testing.T) {
	root := linkTree(t)
	if err := os.Symlink(filepath.Join("real", "x.txt"), filepath.Join(root, "filelink.txt")); err != nil {
		t.Skipf("links are not available here: %v", err)
	}
	var sk WalkSkipped
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	if sk.LinkedDirs != 0 || len(specs) != 2 {
		t.Errorf("a link to a file: served %d, counted %+v, want both files served and no count", len(specs), sk)
	}
}
