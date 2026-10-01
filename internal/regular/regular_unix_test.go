//go:build unix

package regular

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
)

// ADR-109 T1. A loader that judged a path by Stat and then opened it blocking
// hung on a file swapped for a FIFO in between. Open opens without blocking and
// asks the descriptor: a FIFO is refused at once, a regular file and a
// directory open, and a missing file keeps its own error.
func TestOpenRefusesANonRegularFileAtOnce(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		f, _, err := Open(pipe)
		if f != nil {
			_ = f.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, ErrNotRegular) {
			t.Errorf("a FIFO opened, or was refused with something else: %v", err)
		}
	case <-time.After(5 * time.Second):
		Release(pipe)
		t.Fatal("Open blocked on a FIFO")
	}
	if ErrNotRegular.Error() != lines.NotRegular {
		t.Errorf("the refusal text changed: %q", ErrNotRegular.Error())
	}

	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if f, fi, err := Open(file); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("a regular file did not open: %v", err)
	} else {
		_ = f.Close()
	}
	if f, fi, err := Open(dir); err != nil || !fi.IsDir() {
		t.Errorf("a directory was not handed back to its caller: %v", err)
	} else {
		_ = f.Close()
	}
	if _, _, err := Open(filepath.Join(dir, "missing")); !os.IsNotExist(err) {
		t.Errorf("a missing file lost its own error: %v", err)
	}

	// The Codex review of #306: a socket cannot be opened at all, so its open
	// fails before the descriptor is asked; it is still refused as not regular.
	short, err := os.MkdirTemp("/tmp", "mrw")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(short) })
	sock := filepath.Join(short, "s")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Skipf("no unix sockets here: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	if _, _, err := Open(sock); !errors.Is(err, ErrNotRegular) {
		t.Errorf("a socket was not refused as not regular: %v", err)
	}
}
