//go:build unix

package mcp

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestAFilesFromFIFOIsRefusedAtOnce: a files_from that is a FIFO would hold
// the server until something wrote to it (ADR-109); it is refused at once.
func TestAFilesFromFIFOIsRefusedAtOnce(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\n")
	fifo := filepath.Join(root, "list.fifo")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan map[string]any, 1)
	go func() { done <- call(t, root, "mrw_read", map[string]any{"files_from": "list.fifo"}) }()
	select {
	case res := <-done:
		if res["isError"] != true {
			t.Errorf("a FIFO files_from was not refused: %v", res)
		}
	case <-time.After(5 * time.Second):
		// Release the reader before failing: a blocked call holds the
		// in-process server, and every test after this one would queue
		// behind it until the package's timeout.
		if w, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		t.Fatal("files_from waited on a FIFO")
	}
}
