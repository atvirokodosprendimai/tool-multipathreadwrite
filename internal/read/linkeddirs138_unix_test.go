//go:build unix

package read

import (
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
)

// ADR-138 counts a link to a DIRECTORY. A FIFO met by a walk is not a candidate
// and is not a link to a directory either; it stays uncounted (BACKLOG), and the
// walk does not open it.
func TestAFifoIsNotCountedAsALinkToADirectory(t *testing.T) {
	root := linkTree(t)
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0o600); err != nil {
		t.Skipf("a FIFO cannot be made here: %v", err)
	}
	var sk WalkSkipped
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	if sk.LinkedDirs != 0 || len(specs) != 1 {
		t.Errorf("a FIFO: served %d, counted %+v, want real/x.txt served and nothing counted", len(specs), sk)
	}
}
