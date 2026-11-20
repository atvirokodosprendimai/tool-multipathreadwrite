//go:build darwin

package apply

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// Review of #744. A file that may not be deleted (an ACL denies it) is linked to
// its new name, and then neither name can be removed: the move fails, the file has
// two names, and the second is named in left_behind instead of the receipt saying
// only that nothing was written.
func TestAMoveThatCannotRemoveEitherNameNamesTheExtraOneAsLeftBehind(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	a := filepath.Join(root, "a.txt")
	if out, err := exec.Command("chmod", "+a", "everyone deny delete", a).CombinedOutput(); err != nil {
		t.Skipf("no ACLs here: %v %s", err, out)
	}
	t.Cleanup(func() {
		_ = exec.Command("chmod", "-N", a).Run()
		_ = exec.Command("chmod", "-N", filepath.Join(root, "b.txt")).Run()
	})
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1, Index: 0}}, Options{})
	if err == nil && res.Applied {
		t.Fatalf("a rename of an undeletable file was applied: %+v", res)
	}
	if !slices.Contains(res.LeftBehind, "b.txt") {
		t.Errorf("left_behind = %v, want the extra name b.txt", res.LeftBehind)
	}
	for _, n := range []string{"a.txt", "b.txt"} {
		if _, err := os.Lstat(filepath.Join(root, n)); err != nil {
			t.Errorf("%s: %v", n, err)
		}
	}
}
