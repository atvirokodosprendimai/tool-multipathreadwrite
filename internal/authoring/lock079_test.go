package authoring

import (
	"sync"
	"testing"
)

// ADR-079. Record read the tally, added one and rewrote it, with no lock, so
// racing writers lost counts — and a reader that met the file emptied for its
// rewrite rebuilt from nothing. Every count survives racing records.
func TestConcurrentRecordsCountEveryOutcome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	const writers, each = 16, 20
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < each; j++ {
				_ = Record(root, Applied)
			}
		}()
	}
	wg.Wait()
	tally, _ := Load(root)
	if got := tally[Applied.name()]; got != writers*each {
		t.Errorf("applied = %d after %d racing records", got, writers*each)
	}
}

// ADR-079. A tally read while another process rewrote it could find the file
// emptied and report zero: the count went backwards. Read under the lock, the
// count a reader sees never falls.
func TestATallyReadNeverSeesAHalfSavedTally(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	_ = Record(root, Applied)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1500; i++ {
			_ = Record(root, Applied)
		}
	}()
	last := 1
	for {
		select {
		case <-done:
			return
		default:
		}
		tally, _ := Load(root)
		got := tally[Applied.name()]
		if got < last {
			t.Fatalf("a read saw applied = %d after %d: a half-saved tally", got, last)
		}
		last = got
	}
}
