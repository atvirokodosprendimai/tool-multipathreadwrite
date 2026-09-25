package seen

import (
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
