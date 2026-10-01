//go:build unix

package read

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
)

// ADR-109 T1. read judges a named spec, and the --grep walk a candidate, by a
// path Stat, then readCapped opened it blocking: a file swapped for a FIFO in
// between hung the read. readCapped asks the descriptor and refuses at once.
func TestReadCappedRefusesAFIFOAtOnce(t *testing.T) {
	pipe := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := readCapped(pipe)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), lines.NotRegular) {
			t.Errorf("readCapped read a FIFO, or refused it with something else: %v", err)
		}
	case <-time.After(5 * time.Second):
		if w, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		t.Fatal("readCapped blocked on a FIFO")
	}
}
