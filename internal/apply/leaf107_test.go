package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-107 T3. After a.txt staged, another process moved it to b.txt and made
// a.txt a link to b.txt. The identity rechecks stat'ed through the link, saw
// the original file and passed, and the commit replaced the link with a
// regular file, leaving b.txt as it was — against the preserved-link rule
// staging keeps. The recheck lstats the leaf through the root, so a link where
// validation saw a regular file stops the plan.
func TestTheRecheckRefusesALeafSwappedForALink(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", abcde)
	duringStage(t, "a.txt", func() {
		if err := os.Rename(filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("b.txt", filepath.Join(root, "a.txt")); err != nil {
			t.Skipf("symlinks cannot be made here: %v", err)
		}
	})
	res, err := Apply(root, []Input{{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0}}, Options{Force: true})
	if err == nil || res.Applied || !strings.Contains(err.Error(), "a.txt") {
		t.Errorf("a leaf swapped for a link was not refused by name: %v %+v", err, res)
	}
	if fi, lerr := os.Lstat(filepath.Join(root, "a.txt")); lerr != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("a.txt is no longer the link another process made (%v)", lerr)
	}
	if got := read(t, root, "b.txt"); got != abcde {
		t.Errorf("b.txt holds %q, want it untouched", got)
	}
}
