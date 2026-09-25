//go:build unix

package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ADR-073, review of #230. The foreign formats read the target to locate the
// old side, and they read it before apply's regular-file check: a FIFO named in
// an apply_patch or search_replace document blocked the write until something
// wrote to the pipe. It is refused by name, as one named in a native plan is.
func TestAFIFOTargetIsRefusedNotWaitedOn(t *testing.T) {
	root := t.TempDir()
	fifo := filepath.Join(root, "p")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skipf("no FIFO here: %v", err)
	}
	formats := []struct {
		name    string
		compile func() error
	}{
		{"apply_patch", func() error {
			_, err := CompileApplyPatch(root, []byte("*** Begin Patch\n*** Update File: p\n@@\n-x\n+y\n*** End Patch\n"))
			return err
		}},
		{"search_replace", func() error {
			_, err := CompileSearchReplace(root, []byte("p\n<<<<<<< SEARCH\nx\n=======\ny\n>>>>>>> REPLACE\n"))
			return err
		}},
	}
	for _, f := range formats {
		done := make(chan error, 1)
		go func() { done <- f.compile() }()
		select {
		case <-time.After(3 * time.Second):
			// Let the stuck open return, so the goroutine does not outlive the test.
			if w, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
				w.Close()
			}
			t.Fatalf("%s is still waiting on the FIFO at 3 s", f.name)
		case err := <-done:
			if err == nil || !strings.Contains(err.Error(), "not a regular file") {
				t.Errorf("%s: want a refusal naming a file that is not regular, got %v", f.name, err)
			}
		}
	}
}
