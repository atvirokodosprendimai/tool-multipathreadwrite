//go:build unix

package check

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

// ADR-201 T1. lastLines opened the log of a check or a step with os.Open, so a
// FIFO put there after the create held the command. It reads as no lines; a
// regular log reads as before.
func TestAFifoAtTheCheckLogReadsAsNoLines(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "pipe.log")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	t.Cleanup(func() {
		if w, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
	})
	type out struct {
		tail    []string
		earlier int
		cut     bool
	}
	done := make(chan out, 1)
	go func() {
		var o out
		o.tail, o.earlier, o.cut = lastLines(pipe, 5)
		done <- o
	}()
	select {
	case o := <-done:
		if len(o.tail) != 0 || o.earlier != 0 || o.cut {
			t.Errorf("a FIFO log read as %+v, want no lines, none earlier, not cut", o)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("lastLines blocked on a FIFO")
	}

	log := filepath.Join(dir, "a.log")
	if err := os.WriteFile(log, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if tail, earlier, cut := lastLines(log, 2); !reflect.DeepEqual(tail, []string{"two", "three"}) || earlier != 1 || cut {
		t.Errorf("a regular log read as %q, %d, %v; want [two three], 1, false", tail, earlier, cut)
	}
}
