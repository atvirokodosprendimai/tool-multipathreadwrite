//go:build unix

package plan

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
)

// ADR-109 T3. LoadBodyFiles decided "regular" from a path Stat and readBounded
// opened the body file blocking, so a file swapped for a FIFO between them hung
// the write. The descriptor decides, and the refusal says what it always said.
func TestABodyFileFIFOIsRefusedByItsDescriptor(t *testing.T) {
	root := t.TempDir()
	pipe := filepath.Join(root, "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- LoadBodyFiles(root, []Hunk{{BodyFile: "pipe", SrcLine: 1}})
	}()
	select {
	case err := <-done:
		if !errors.Is(err, regular.ErrNotRegular) || err == nil || !strings.Contains(err.Error(), "body=@pipe is not a regular file") {
			t.Errorf("a FIFO body file was not refused by its descriptor with the usual text: %v", err)
		}
	case <-time.After(5 * time.Second):
		if w, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		t.Fatal("LoadBodyFiles blocked on a FIFO")
	}
}
