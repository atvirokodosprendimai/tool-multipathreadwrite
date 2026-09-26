package authoring

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
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

// ADR-079, the reviews of #240. A writer that could not take the tally's lock
// wrote anyway, and racing unlocked writers wiped the tally — and a lock mrw
// cannot open says nothing about whether the tally beside it is writable. A
// writer that cannot lock writes nothing; a reader still reads; a reset says it
// could not.
func TestAWriterThatCannotLockWritesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	_ = Record(root, Applied)
	p, err := state.Path(root, lockName)
	if err != nil {
		t.Fatal(err)
	}
	// A directory where the lock file belongs: it cannot be opened as one, and
	// the tally beside it stays writable.
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = Record(root, Applied)
	_ = RecordRecent(root, 1)
	_ = RecordPricing(root, true, true, PricedBroke)
	if tally, _ := Load(root); tally[Applied.name()] != 1 {
		t.Errorf("applied = %d: a writer that could not lock wrote", tally[Applied.name()])
	}
	if n := len(Recent(root)); n != 0 {
		t.Errorf("the ring holds %d: a writer that could not lock wrote", n)
	}
	if _, err := Reset(root); err == nil {
		t.Error("a reset that could not lock reported success")
	}
	if tally, _ := Load(root); tally[Applied.name()] != 1 {
		t.Error("a reset that could not lock emptied the tally")
	}
}

// ADR-079, the review of #240: only Record and Load were pinned, so any other
// call could drop the lock unseen. Each exported call waits while another holder
// has the tally's lock, and finishes once it is released.
func TestEveryTallyCallWaitsForTheLock(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for name, call := range map[string]func(){
		"Record":        func() { _ = Record(root, Applied) },
		"Reclassify":    func() { _ = Reclassify(root, Applied, FailedCheck) },
		"Load":          func() { _, _ = Load(root) },
		"Reset":         func() { _, _ = Reset(root) },
		"RecordRecent":  func() { _ = RecordRecent(root, 0) },
		"Recent":        func() { _ = Recent(root) },
		"RecordPricing": func() { _ = RecordPricing(root, false, false, PricedHeld) },
		"LoadPricing":   func() { _ = LoadPricing(root) },
	} {
		release, err := state.Hold(root, lockName)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			call()
			close(done)
		}()
		select {
		case <-done:
			t.Errorf("%s finished while another holder had the tally's lock", name)
		case <-time.After(200 * time.Millisecond):
		}
		release()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s never finished after the lock was released", name)
		}
	}
}
