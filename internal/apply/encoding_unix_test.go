//go:build unix

package apply

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ADR-073. A FIFO named in a plan blocked the write in os.ReadFile. A path
// that is not a regular file is refused before it is opened.
func TestAFIFONamedInAPlanIsRefusedNotWaitedOn(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "p"), 0o644); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	done := make(chan Result, 1)
	go func() {
		res, _ := Apply(root, []Input{
			{Path: "p", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1},
		}, Options{Force: true})
		done <- res
	}()
	select {
	case res := <-done:
		if res.Applied || res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "not a regular file") {
			t.Fatalf("a FIFO in a plan: %+v, want one refusal saying it is not a regular file", res.Hunks)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("a write naming a FIFO blocked")
	}
}
