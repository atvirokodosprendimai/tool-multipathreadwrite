package seen

import (
	"fmt"
	"testing"
	"time"
)

// ADR-075. A second holder of the write lock waits until the first releases,
// and a release may be called twice.
func TestAWriteLockExcludesASecondWriter(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	release, err := LockWrites(root)
	if err != nil {
		t.Fatal(err)
	}
	got := make(chan func(), 1)
	go func() {
		r, err := LockWrites(root)
		if err != nil {
			t.Error(err)
			r = func() {}
		}
		got <- r
	}()
	select {
	case r := <-got:
		r()
		release()
		t.Fatal("a second writer took the write lock while the first held it")
	case <-time.After(300 * time.Millisecond):
	}
	release()
	select {
	case r := <-got:
		r()
	case <-time.After(3 * time.Second):
		t.Fatal("the second writer never got the write lock after the first released it")
	}
	release()
}

// ADR-075. Record and Drop take seen.lock inside the write lock, so the two are
// different files: one process opening the same lock a second time waits on
// itself.
func TestTheLedgerCanBeWrittenUnderTheWriteLock(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	release, err := LockWrites(root)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	done := make(chan error, 1)
	go func() { done <- Record(root, map[string]Observation{"a.go": {SHA: "aaa"}}) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Record waited on the write lock: the two locks are one file")
	}
}

// Review of #233. save empties the ledger and rewrites it, so a Load taken
// while another writer saves can find it half-written or empty, and a Load that
// finds no header discards the ledger: that writer was refused "has not been
// read" for a file it had read. Snapshot loads under the ledger's own lock.
func TestASnapshotNeverSeesAHalfSavedLedger(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	base := map[string]Observation{}
	for i := 0; i < 200; i++ {
		base[fmt.Sprintf("f%03d.go", i)] = Observation{SHA: "aaa"}
	}
	if err := Record(root, base); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			_ = Record(root, map[string]Observation{"g.go": {SHA: fmt.Sprint(i)}})
		}
	}()
	for i := 0; i < 2000; i++ {
		l, err := Snapshot(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(l) < 200 {
			t.Fatalf("snapshot %d saw %d entries of at least 200: a ledger caught mid-save", i, len(l))
		}
	}
}
