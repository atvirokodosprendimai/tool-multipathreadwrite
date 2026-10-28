package seen

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// ADR-175 Decision 1: Forget removes every observation of the checkout, read
// and written alike, keeps the header, and says how many files it held. A
// newer mrw's ledger is not touched and the refusal is ADR-150's notice.
func TestForgetDropsEveryLicenceInTheCheckout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	sha := strings.Repeat("c", 64)
	if err := Record(root, map[string]Observation{
		"read.go":  {SHA: sha},
		"wrote.go": {SHA: sha, Written: true},
	}); err != nil {
		t.Fatal(err)
	}
	n, err := Forget(root)
	if err != nil || n != 2 {
		t.Fatalf("Forget = %d, %v; want 2 files forgotten", n, err)
	}
	if l, err := Load(root); err != nil || len(l) != 0 {
		t.Errorf("the ledger still licenses %v after Forget (%v)", l, err)
	}
	p, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "#mrw-seen v6 ") {
		t.Errorf("the emptied ledger lost its header: %q", b)
	}
	if n, err := Forget(root); err != nil || n != 0 {
		t.Errorf("Forget on an empty ledger = %d, %v; want 0, nil", n, err)
	}

	body := "#mrw-seen v7 v9.9.9 (fffffff)\n" + sha + "  -  a.go\nwhatever v7 means\n"
	p = plant(t, root, body)
	n, err = Forget(root)
	if err == nil || n != 0 || !strings.Contains(err.Error(), "written by a newer mrw (v9.9.9 (fffffff))") {
		t.Errorf("Forget over a newer ledger = %d, %v; want 0 and ADR-150's notice", n, err)
	}
	if got, _ := os.ReadFile(p); string(got) != body {
		t.Errorf("a newer ledger was written over:\n%q\nwant\n%q", got, body)
	}
}

// ADR-175 Decision 1: Forget runs under the checkout's seen.lock, as Record
// does, so a writer mid-update is never emptied under. Held by the test, the
// lock keeps Forget waiting with the ledger intact; released, Forget finishes
// and empties it.
func TestForgetWaitsForTheLedgerLock(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := Record(root, map[string]Observation{"a.go": {SHA: strings.Repeat("c", 64)}}); err != nil {
		t.Fatal(err)
	}
	type result struct {
		n   int
		err error
	}
	var (
		wg        sync.WaitGroup
		once      sync.Once
		holderErr error
		holderEnd = make(chan struct{})
		done      = make(chan result, 1)
		held      = make(chan struct{})
		started   = make(chan struct{})
		release   = make(chan struct{})
	)
	letGo := func() { once.Do(func() { close(release) }) }
	// Every goroutine is joined, bounded, on every path out of the test.
	defer func() {
		letGo()
		joined := make(chan struct{})
		go func() { wg.Wait(); close(joined) }()
		select {
		case <-joined:
		case <-time.After(15 * time.Second):
			t.Error("a goroutine of the test did not finish after the lock was released")
			return
		}
		if holderErr != nil {
			t.Errorf("the test could not hold the ledger lock: %v", holderErr)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		holderErr = withLock(root, func() error {
			close(held)
			<-release
			return nil
		})
		close(holderEnd)
	}()
	select {
	case <-held:
	case <-holderEnd:
		t.Fatal("the test did not get the ledger lock (see the error reported on exit)")
	case <-time.After(15 * time.Second):
		t.Fatal("the test did not get the ledger lock")
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(started)
		n, err := Forget(root)
		done <- result{n, err}
	}()
	// The worker has begun before the window opens, or the window proves nothing.
	select {
	case <-started:
	case <-time.After(15 * time.Second):
		t.Fatal("the Forget goroutine did not start")
	}
	select {
	case r := <-done:
		t.Fatalf("Forget returned %d, %v while the ledger lock was held", r.n, r.err)
	case <-time.After(300 * time.Millisecond):
	}
	if l, err := Load(root); err != nil || len(l) != 1 {
		t.Errorf("the ledger changed while the lock was held: %v (%v)", l, err)
	}
	letGo()
	select {
	case r := <-done:
		if r.err != nil || r.n != 1 {
			t.Errorf("Forget = %d, %v; want 1, nil", r.n, r.err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Forget did not return after the lock was released")
	}
	if l, err := Load(root); err != nil || len(l) != 0 {
		t.Errorf("the ledger still licenses %v after Forget (%v)", l, err)
	}
}
