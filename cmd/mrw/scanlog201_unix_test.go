//go:build unix

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// ADR-201 T1. scanLog opened a run's log with os.Open, so a FIFO put there
// after the create held `mrw run` after its command had ended. It reads as
// empty; a regular log reads as before.
func TestAFifoAtARunLogReadsAsEmpty(t *testing.T) {
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
		lines int
		size  int64
		last  string
	}
	done := make(chan out, 1)
	go func() {
		var o out
		o.lines, o.size, o.last = scanLog(pipe)
		done <- o
	}()
	select {
	case o := <-done:
		if o != (out{}) {
			t.Errorf("a FIFO log scanned as %+v, want zero lines, size 0, no last line", o)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("scanLog blocked on a FIFO")
	}

	log := filepath.Join(dir, "a.log")
	if err := os.WriteFile(log, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n, size, last := scanLog(log); n != 2 || size != 8 || last != "two" {
		t.Errorf("a regular log scanned as %d, %d, %q; want 2, 8, \"two\"", n, size, last)
	}
}
