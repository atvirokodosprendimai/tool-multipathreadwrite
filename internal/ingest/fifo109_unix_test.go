//go:build unix

package ingest

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
)

// ADR-109 T3. targetBytes decided "regular" from a path Stat and opened the
// target blocking, so a file swapped for a FIFO between them hung the compile.
// The descriptor decides; the refusal text is the one it always was.
func TestForeignTargetFIFOIsRefusedByItsDescriptor(t *testing.T) {
	pipe := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := targetBytes(pipe)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, regular.ErrNotRegular) || err.Error() != lines.NotRegular {
			t.Errorf("a FIFO target was not refused by its descriptor with the usual text: %v", err)
		}
	case <-time.After(5 * time.Second):
		if w, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		t.Fatal("targetBytes blocked on a FIFO")
	}
}
