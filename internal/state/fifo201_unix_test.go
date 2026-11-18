//go:build unix

package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fifoAt replaces path with a FIFO, and arranges for a reader a red run left
// blocked on its open to be let go, so no goroutine strands holding a lock.
func fifoAt(t *testing.T, path string) {
	t.Helper()
	_ = os.Remove(path)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	t.Cleanup(func() {
		if w, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
	})
}

// within runs fn and fails the test when it has not returned by the deadline.
func within(t *testing.T, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); fn() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s blocked on a FIFO", what)
	}
}

// ADR-201 T1. A FIFO where mrw expects one of its state files no longer holds
// a command: the marker is replaced, a lock's holder is unknown, and a lock
// file that cannot be locked names itself.
func TestAFifoAtAStateFileDoesNotHoldMrw(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	t.Run("the root marker", func(t *testing.T) {
		root := t.TempDir()
		dir, err := Dir(root)
		if err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(dir, "root")
		fifoAt(t, marker)
		var got string
		var derr error
		within(t, "Dir", func() { got, derr = Dir(root) })
		if derr != nil || got != dir {
			t.Fatalf("Dir over a FIFO marker: %q, %v; want %q", got, derr, dir)
		}
		fi, err := os.Lstat(marker)
		if err != nil || !fi.Mode().IsRegular() {
			t.Fatalf("the marker is %v, %v; want a regular file", fi, err)
		}
		want, _ := absReal(root)
		if b, _ := os.ReadFile(marker); strings.TrimSuffix(string(b), "\n") != want {
			t.Errorf("the marker holds %q, want the root %q", b, want)
		}
	})

	t.Run("the holder of a lock", func(t *testing.T) {
		root := t.TempDir()
		release, err := HoldWithin(root, "h.lock", time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		path, _ := Path(root, "h.lock")
		fifoAt(t, path+".holder")
		var herr error
		within(t, "HoldWithin", func() {
			r, err := HoldWithin(root, "h.lock", 0)
			if r != nil {
				r()
			}
			herr = err
		})
		var lt *LockTimeoutError
		if !errors.As(herr, &lt) || lt.Holder != "" || !strings.Contains(herr.Error(), "another process") {
			t.Errorf("a held lock with a FIFO holder: %v, want the timeout error naming another process", herr)
		}
	})

	t.Run("the lock file", func(t *testing.T) {
		root := t.TempDir()
		path, err := Path(root, "f.lock")
		if err != nil {
			t.Fatal(err)
		}
		fifoAt(t, path)
		var release func()
		var herr error
		within(t, "HoldWithin", func() { release, herr = HoldWithin(root, "f.lock", 0) })
		if herr == nil {
			defer release()
			t.Log("this platform took a lock on a FIFO")
			var lt *LockTimeoutError
			r2, err := HoldWithin(root, "f.lock", 0)
			if r2 != nil {
				r2()
			}
			if !errors.As(err, &lt) {
				t.Errorf("a second taker with no wait got %v, want the timeout: the first was not a lock", err)
			}
			return
		}
		var pe *fs.PathError
		if !errors.As(herr, &pe) || pe.Path != path {
			t.Errorf("a lock on a FIFO failed with %v, want a *fs.PathError naming %s", herr, path)
		}
		t.Logf("this platform refused a lock on a FIFO: %v", herr)
	})

	t.Run("a lock that fails names its file", func(t *testing.T) {
		// On Linux a lock on a FIFO is taken, so the failure is made here.
		root := t.TempDir()
		path, err := Path(root, "g.lock")
		if err != nil {
			t.Fatal(err)
		}
		real := tryLockFile
		tryLockFile = func(*os.File) (bool, error) { return false, syscall.EIO }
		release, herr := HoldWithin(root, "g.lock", 0)
		tryLockFile = real
		if release != nil {
			release()
		}
		var pe *fs.PathError
		if !errors.As(herr, &pe) || pe.Path != path || pe.Op != "lock" || !errors.Is(herr, syscall.EIO) {
			t.Errorf("a lock that failed returned %v, want a *fs.PathError with Op lock naming %s and wrapping the cause", herr, path)
		}
	})
}

// ADR-201 T1. Another directory's FIFO marker leaves it unidentified and kept,
// and a prune, dry or real, returns: a dead entry beside it is still found.
func TestAFifoMarkerInAnotherEntryDoesNotHoldPrune(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	base, err := Base()
	if err != nil {
		t.Fatal(err)
	}
	odd := filepath.Join(base, "odd")
	dead := filepath.Join(base, "dead")
	for _, d := range []string{odd, dead} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	fifoAt(t, filepath.Join(odd, "root"))
	gone := filepath.Join(t.TempDir(), "gone")
	if err := os.WriteFile(filepath.Join(dead, "root"), []byte(gone+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var entries []Entry
	var eerr error
	within(t, "Entries", func() { entries, eerr = Entries() })
	if eerr != nil {
		t.Fatal(eerr)
	}
	byName := map[string]Entry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if e, ok := byName["odd"]; !ok || e.Identified || e.Dead {
		t.Errorf("the FIFO-marker entry is %+v (listed %v), want listed, unidentified and not dead", e, ok)
	}
	if e := byName["dead"]; !e.Identified || !e.Dead {
		t.Errorf("the dead entry beside it is %+v, want identified and dead", e)
	}

	for _, dry := range []bool{true, false} {
		var removed []Entry
		var perr error
		within(t, "Prune", func() { removed, perr = Prune(t.TempDir(), entries, dry) })
		if perr != nil || len(removed) != 1 || removed[0].Name != "dead" {
			t.Errorf("Prune(dry=%v): %v, %v; want only the dead entry", dry, removed, perr)
		}
		if _, err := os.Lstat(odd); err != nil {
			t.Errorf("Prune(dry=%v) lost the entry with the FIFO marker: %v", dry, err)
		}
	}
}
