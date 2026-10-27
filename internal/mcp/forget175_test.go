package mcp

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-175 Decision 1: a span served by mrw_read and not yet acknowledged is
// promoted into the ledger by a later ack, with no earlier observation needed,
// so ForgetPending must empty the store or a pre-compaction serve would license
// a post-compaction write. The pair: without it the same ack licenses the write.
func TestForgetDropsPendingAcknowledgements(t *testing.T) {
	for _, forget := range []bool{false, true} {
		root, path := checkout(t, "a.txt", "one\ntwo\nthree\n")
		acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}})))
		if len(acks) == 0 {
			t.Fatal("the read carried no checkpoints")
		}
		if forget {
			if err := ForgetPending(root); err != nil {
				t.Fatal(err)
			}
		}
		got := structured(t, call(t, root, "mrw_write", map[string]any{
			"plan": "@@ a.txt 2 replace\nTWO\n", "ack": acks}))
		failed, _ := got["failed"].(float64)
		body, err := os.ReadFile(filepath.Join(root, "a.txt"))
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case forget && (failed == 0 || string(body) != "one\ntwo\nthree\n"):
			t.Errorf("an ack served before ForgetPending licensed a write: failed=%v body=%q", failed, body)
		case !forget && failed != 0:
			t.Errorf("the pair: the ack did not license the write without ForgetPending: %v", got["hunks"])
		}
	}
}

// ADR-175 Decision 1: ForgetPending takes the pending store's own lock, as
// every change to it does (ADR-085), so a server mid-promotion is not emptied
// under. Held by the test, the lock keeps it waiting with the store intact;
// released, it finishes and the store is empty.
func TestForgetPendingWaitsForThePendingLock(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\ntwo\nthree\n")
	if len(checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}})))) == 0 {
		t.Fatal("the read carried no checkpoints")
	}
	if store, err := loadPending(root); err != nil || len(store) == 0 {
		t.Fatalf("the read left no pending span to forget: %v (%v)", store, err)
	}
	release, err := state.Hold(root, pendingLock)
	if err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	letGo := func() { once.Do(func() { release() }) }
	done := make(chan error, 1)
	started := make(chan struct{})
	exited := make(chan struct{})
	// The lock is let go and the worker joined, bounded, on every path out.
	defer func() {
		letGo()
		select {
		case <-exited:
		case <-time.After(15 * time.Second):
			t.Error("the ForgetPending goroutine did not finish after the lock was released")
		}
	}()
	go func() {
		defer close(exited)
		close(started)
		done <- ForgetPending(root)
	}()
	// The worker has begun before the window opens, or the window proves nothing.
	select {
	case <-started:
	case <-time.After(15 * time.Second):
		t.Fatal("the ForgetPending goroutine did not start")
	}
	select {
	case err := <-done:
		t.Fatalf("ForgetPending returned %v while the pending lock was held", err)
	case <-time.After(300 * time.Millisecond):
	}
	if store, err := loadPending(root); err != nil || len(store) == 0 {
		t.Errorf("the pending store changed while the lock was held: %v (%v)", store, err)
	}
	letGo()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ForgetPending = %v; want nil", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("ForgetPending did not return after the lock was released")
	}
	if store, err := loadPending(root); err != nil || len(store) != 0 {
		t.Errorf("the pending store still holds %v after ForgetPending (%v)", store, err)
	}
}
