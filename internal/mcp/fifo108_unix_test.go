//go:build unix

package mcp

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// ADR-108 T6. currentSHA opened and streamed the path without asking whether
// it was still a regular file, so a file swapped for a FIFO before its
// acknowledgement blocked the open — under the pending-store lock, in a server
// that answers one request at a time. It returns at once, refusing, and a
// regular file still gets its digest.
func TestAcknowledgingANonRegularFileReturnsAtOnce(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := currentSHA(root, "pipe")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a FIFO was hashed as if it were a file")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("currentSHA blocked on a FIFO")
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if sha, err := currentSHA(root, "a.txt"); err != nil || len(sha) != 64 {
		t.Errorf("a regular file lost its digest: %q %v", sha, err)
	}
}
