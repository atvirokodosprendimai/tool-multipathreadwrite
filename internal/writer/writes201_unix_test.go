//go:build unix

package writer

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-201 T1. A FIFO at the write counter reads as zero, within the deadline,
// and the bump that follows replaces it with a regular file holding 1.
func TestAFifoAtTheWriteCounterReadsAsZero(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	top, _ := state.Checkout(root)
	p, err := state.Path(top, writesName)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(p, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	t.Cleanup(func() {
		if w, err := os.OpenFile(p, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
	})
	within := func(what string, fn func() int64) int64 {
		t.Helper()
		done := make(chan int64, 1)
		go func() { done <- fn() }()
		select {
		case n := <-done:
			return n
		case <-time.After(5 * time.Second):
			t.Fatalf("%s blocked on a FIFO", what)
			return -1
		}
	}
	if n := within("Writes", func() int64 { return Writes(root) }); n != 0 {
		t.Errorf("Writes over a FIFO is %d, want 0", n)
	}
	if n := within("bumpWrites", func() int64 { return bumpWrites(root) }); n != 1 {
		t.Errorf("bumpWrites over a FIFO returned %d, want 1", n)
	}
	fi, err := os.Lstat(p)
	if err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("the counter is %v, %v; want a regular file", fi, err)
	}
	if b, _ := os.ReadFile(p); string(b) != "1\n" {
		t.Errorf("the counter holds %q, want \"1\\n\"", b)
	}
}
