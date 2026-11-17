//go:build unix

package apply

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

// ADR-199. A FIFO is not a directory, and not a regular file either: the
// obstruction is "does not lead to a directory", never "is not regular".
func TestAPathBelowAFifoIsRefusedOnItsHunk(t *testing.T) {
	root := t.TempDir()
	write(t, root, "d/g.txt", "a\n")
	if err := syscall.Mkfifo(filepath.Join(root, "d", "pipe"), 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	noStaging(t)
	before := snapshot(t, root)
	res, err := Apply(root, []Input{{Path: filepath.FromSlash("d/pipe/x"), Op: "create", Body: []string{"x"}, Lines: -1, SrcLine: 1}}, Options{})
	if err != nil {
		t.Fatalf("Apply returned an error, want a failed hunk: %v", err)
	}
	const want = "d/pipe/x cannot be written: d/pipe is not a directory"
	if r := failedReason(res); r != want || res.Failed != 1 || res.Applied {
		t.Errorf("reason %q Failed=%d Applied=%v, want %q", r, res.Failed, res.Applied, want)
	}
	if after := snapshot(t, root); !reflect.DeepEqual(after, before) {
		t.Errorf("the tree changed: %v -> %v", before, after)
	}
	if _, err := os.Lstat(filepath.Join(root, "d", "pipe", "x")); err == nil {
		t.Error("d/pipe/x exists")
	}
}
