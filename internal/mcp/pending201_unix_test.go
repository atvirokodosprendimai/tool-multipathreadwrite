//go:build unix

package mcp

import (
	"io/fs"
	"os"
	"syscall"
	"testing"
	"time"
)

// ADR-201 T1. A FIFO at the pending store is an empty store, with no error, and
// ForgetPending returns none; a later save leaves a regular file. A read that
// failed otherwise (a busy file) is still an error, and the store is untouched.
func TestAFifoAtThePendingStoreIsAnEmptyStore(t *testing.T) {
	// within bounds fn and releases a reader a red run left blocked on the FIFO.
	within := func(t *testing.T, p, what string, fn func()) {
		t.Helper()
		t.Cleanup(func() {
			if w, err := os.OpenFile(p, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
				_ = w.Close()
			}
		})
		done := make(chan struct{})
		go func() { defer close(done); fn() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s blocked on a FIFO", what)
		}
	}
	fixture := func(t *testing.T) (root, p string) {
		t.Helper()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root = t.TempDir()
		p, err := pendingPath(root)
		if err != nil {
			t.Fatal(err)
		}
		return root, p
	}

	t.Run("loadPending", func(t *testing.T) {
		root, p := fixture(t)
		if err := syscall.Mkfifo(p, 0o600); err != nil {
			t.Skipf("no FIFOs here: %v", err)
		}
		var store map[string]pending
		var err error
		within(t, p, "loadPending", func() { store, err = loadPending(root) })
		if err != nil || store == nil || len(store) != 0 {
			t.Errorf("a FIFO store loaded as %v, %v; want an empty store and no error", store, err)
		}
	})

	t.Run("ForgetPending and a later save", func(t *testing.T) {
		root, p := fixture(t)
		if err := syscall.Mkfifo(p, 0o600); err != nil {
			t.Skipf("no FIFOs here: %v", err)
		}
		var err error
		within(t, p, "ForgetPending", func() { err = ForgetPending(root) })
		if err != nil {
			t.Errorf("ForgetPending over a FIFO store: %v, want none", err)
		}
		// hold reads the store and then saves it: the read sees the FIFO as an
		// empty store, and the save replaces it with a regular file.
		within(t, p, "hold", func() {
			err = hold(root, "f.txt", "s", map[string][2]int{"k": {1, 2}})
		})
		fi, serr := os.Lstat(p)
		if err != nil || serr != nil || !fi.Mode().IsRegular() {
			t.Errorf("a hold over a FIFO store: %v, %v, %v; want a regular file", err, serr, fi)
		}
	})

	t.Run("a busy store is still an error", func(t *testing.T) {
		root, p := fixture(t)
		const body = `{"k":{"path":"f.txt","sha":"s","start":1,"end":2,"seq":1}}` + "\n"
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		real := readPendingFile
		t.Cleanup(func() { readPendingFile = real })
		readPendingFile = func(path string) ([]byte, error) {
			return nil, &fs.PathError{Op: "open", Path: path, Err: syscall.EWOULDBLOCK}
		}
		if _, err := loadPending(root); err == nil {
			t.Error("loadPending over a busy store returned no error")
		}
		if err := ForgetPending(root); err == nil {
			t.Error("ForgetPending over a busy store returned no error")
		}
		if b, err := os.ReadFile(p); err != nil || string(b) != body {
			t.Errorf("the store holds %q, %v; want it untouched", b, err)
		}
	})
}
