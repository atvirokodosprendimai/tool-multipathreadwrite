//go:build unix

package apply

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// A FIFO named as a source is refused by name and never opened: opening one
// would block the write.
func TestAFromSourceThatIsAFIFOIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "b.txt", "x\ny\n")
	if err := syscall.Mkfifo(filepath.Join(root, "p"), 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	l := map[string]Seen{"b.txt": {SHA: shaOfFile(t, root, "b.txt")}, "p": {SHA: "0"}}
	res, err := Apply(root, []Input{copyInput167("b.txt", "insert-after", 1, "p", 1, 1)}, Options{Seen: l})
	if err != nil || res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "from=p is not a regular file") {
		t.Errorf("a FIFO source: err=%v %+v", err, res.Hunks)
	}
}
