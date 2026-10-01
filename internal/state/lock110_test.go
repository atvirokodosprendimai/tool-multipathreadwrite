package state

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ADR-110 T1. Hold waited for a held lock without a bound, so a live holder
// that had stopped blocked every writer after it, for ever and in silence.
// HoldWithin gives up after its wait and names the holder, whose pid a taker
// writes beside the lock; a wait of 0 tries once.
func TestAWaitForAHeldLockIsBounded(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	release, err := HoldWithin(root, "w.lock", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		err     error
		elapsed time.Duration
	}
	got := make(chan outcome, 1)
	go func() {
		start := time.Now()
		r, err := HoldWithin(root, "w.lock", 300*time.Millisecond)
		if r != nil {
			r()
		}
		got <- outcome{err, time.Since(start)}
	}()
	var lt *LockTimeoutError
	select {
	case o := <-got:
		if !errors.As(o.err, &lt) || o.elapsed < 250*time.Millisecond {
			t.Errorf("a held lock was taken, or given up on before its wait: %v after %v", o.err, o.elapsed)
		}
		if o.err != nil && !strings.Contains(o.err.Error(), strconv.Itoa(os.Getpid())) {
			t.Errorf("the refusal does not name the holder's pid %d: %v", os.Getpid(), o.err)
		}
	case <-time.After(5 * time.Second):
		release()
		t.Fatal("HoldWithin waited past its bound for a held lock")
	}
	if r, err := HoldWithin(root, "w.lock", 0); !errors.As(err, &lt) {
		if r != nil {
			r()
		}
		t.Errorf("a wait of 0 did not refuse a held lock at once: %v", err)
	}
	release()
	if r, err := HoldWithin(root, "w.lock", time.Second); err != nil {
		t.Errorf("the released lock was not taken: %v", err)
	} else {
		r()
	}
}
