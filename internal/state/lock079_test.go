package state

import (
	"testing"
	"time"
)

// ADR-079. Hold is the one lock primitive for every state file: a second
// holder of the same name waits until the first releases, and a release twice
// is harmless.
func TestAStateLockExcludesASecondHolder(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	release, err := Hold(root, "x.lock")
	if err != nil {
		t.Fatal(err)
	}
	got := make(chan func(), 1)
	go func() {
		r, err := Hold(root, "x.lock")
		if err != nil {
			t.Error(err)
		}
		got <- r
	}()
	select {
	case <-got:
		t.Fatal("a second holder took the lock while the first held it")
	case <-time.After(300 * time.Millisecond):
	}
	release()
	release()
	select {
	case r := <-got:
		r()
	case <-time.After(5 * time.Second):
		t.Fatal("the second holder never got the lock after the first released it")
	}
}
