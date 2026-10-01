package seen

import (
	"strings"
	"testing"
	"time"
)

// ADR-110 T2. LockWrites waited for the write lock without a bound. It waits
// MRW_WRITE_LOCK_TIMEOUT seconds, 120 when unset, then refuses saying nothing
// was applied; a value that is not a whole number of seconds is refused too.
func TestAWriterWaitsABoundedTimeForTheWriteLock(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	release, err := LockWrites(root)
	if err != nil {
		t.Fatal(err)
	}
	try := func(value string) error {
		t.Helper()
		t.Setenv("MRW_WRITE_LOCK_TIMEOUT", value)
		got := make(chan error, 1)
		go func() {
			r, err := LockWrites(root)
			if r != nil {
				r()
			}
			got <- err
		}()
		select {
		case err := <-got:
			return err
		case <-time.After(5 * time.Second):
			release()
			t.Fatalf("LockWrites under MRW_WRITE_LOCK_TIMEOUT=%q waited past 5 s for a held lock", value)
			return nil
		}
	}
	if err := try("0"); err == nil || !strings.Contains(err.Error(), "write lock") || !strings.Contains(err.Error(), "nothing was applied") || !strings.Contains(err.Error(), "MRW_WRITE_LOCK_TIMEOUT") {
		t.Errorf("a held write lock under a wait of 0 was not refused saying what to do: %v", err)
	}
	if err := try("soon"); err == nil || !strings.Contains(err.Error(), `"soon"`) || !strings.Contains(err.Error(), "whole number of seconds") {
		t.Errorf("a wait that is not a number was not refused naming it: %v", err)
	}
	t.Setenv("MRW_WRITE_LOCK_TIMEOUT", "")
	if w, err := writeLockWait(); err != nil || w != 120*time.Second {
		t.Errorf("an unset wait is %v, %v; want 120s", w, err)
	}
	// The Codex review of #307: seconds that overflow time.Duration wrapped to
	// a negative or tiny wait; they are refused, and the largest that fits is not.
	t.Setenv("MRW_WRITE_LOCK_TIMEOUT", "9223372037")
	if w, err := writeLockWait(); err == nil {
		t.Errorf("a wait that overflows time.Duration was accepted as %v", w)
	}
	t.Setenv("MRW_WRITE_LOCK_TIMEOUT", "9223372036")
	if w, err := writeLockWait(); err != nil || w != 9223372036*time.Second {
		t.Errorf("the largest wait that fits is %v, %v", w, err)
	}
	release()
	if err := try("0"); err != nil {
		t.Errorf("the released write lock was not taken: %v", err)
	}
}
