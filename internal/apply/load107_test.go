package apply

import (
	"path/filepath"
	"strings"
	"testing"
)

// ADR-107 T1. apply read a file a plan edits whole, with os.ReadFile, before
// its licence was checked, so a file of any size reached that allocation. A
// file over the limit is refused on its hunk at validation, naming its size
// and the limit, and writes nothing; readLines reads through a bound too, for
// a file that grows after its size was taken; a file under the limit edits as
// before.
func TestAFileOverTheEditLimitIsRefusedBeforeItIsRead(t *testing.T) {
	old := maxLoadBytes
	t.Cleanup(func() { maxLoadBytes = old })
	maxLoadBytes = 16
	root := t.TempDir()
	big := strings.Repeat("x\n", 20) // 40 bytes
	write(t, root, "big.txt", big)
	write(t, root, "small.txt", "a\n")

	res, err := Apply(root, []Input{{Path: "big.txt", Start: 1, End: 1, Op: "replace", Body: []string{"y"}, Lines: -1, Index: 0}}, Options{Force: true})
	h := hunkFor(t, res, "big.txt")
	if err != nil || res.Applied || h.Status != StatusFailed || !strings.Contains(h.Reason, "40 bytes") || !strings.Contains(h.Reason, "16") {
		t.Errorf("a file over the limit was not refused on its hunk naming size and limit: err %v, %+v", err, h)
	}
	if read(t, root, "big.txt") != big {
		t.Error("big.txt was written")
	}

	if _, _, err := readLines(filepath.Join(root, "big.txt")); err == nil || !strings.Contains(err.Error(), "16") {
		t.Errorf("readLines read a file over the limit whole: %v", err)
	}

	res, err = Apply(root, []Input{{Path: "small.txt", Start: 1, End: 1, Op: "replace", Body: []string{"b"}, Lines: -1, Index: 0}}, Options{Force: true})
	if err != nil || !res.Applied || read(t, root, "small.txt") != "b\n" {
		t.Errorf("a file under the limit did not edit as before: %v %+v", err, res)
	}
}
