//go:build unix

package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// ADR-108 T8. read refuses a FIFO, but under a ceiling smaller than that
// refusal the handler tried a first page, whose line count opened the FIFO and
// waited for a writer: the server, which answers one request at a time, hung.
// It answers at once.
func TestPagingARefusedFIFOReturnsAtOnce(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	pipe := filepath.Join(root, "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })
	MaxResultChars = 1

	done := make(chan struct{})
	go func() {
		_, _ = readTool(root, json.RawMessage(`{"specs":["pipe"]}`))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		// Release the blocked reader so the goroutine ends with the test.
		if w, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		t.Fatal("an mrw_read of a FIFO blocked under a small ceiling")
	}
}
