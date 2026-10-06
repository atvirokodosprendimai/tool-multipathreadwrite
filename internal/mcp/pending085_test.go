package mcp

import (
	"fmt"
	"sync"
	"testing"
)

// ADR-085. hold and promote loaded pending.json, changed it and saved it with
// no lock, so two servers on one checkout could each save over the other and
// drop its spans: a page served with a checkpoint id whose acknowledgement then
// matched nothing. Goroutines here stand in for processes: each state.Hold
// opens its own file description, so they exclude each other as processes do.
func TestConcurrentHoldsLoseNoPendingSpan(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	const n = 64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := hold(root, "a.txt", "sha", map[string][2]int{fmt.Sprintf("ck%d", i): {1, 1}}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	store, err := loadPending(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(store) != n {
		t.Errorf("the pending store holds %d of the %d spans held concurrently", len(store), n)
	}
}

// ADR-085 (Codex review of #253). promote loads the store, deletes the
// acknowledged ids and saves; unlocked, a promotion saving an old snapshot
// erased a concurrent hold's new spans or brought back ids another promotion
// had consumed. Held and promoted at once, every new span survives and every
// consumed id stays gone. The spans carry a sha no file has, so promote drops
// them from the store without recording anything in the ledger.
func TestConcurrentPromotesNeitherDropNorResurrect(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	const n = 32
	for i := 0; i < n; i++ {
		if err := hold(root, "a.txt", "old", map[string][2]int{fmt.Sprintf("old%d", i): {1, 1}}); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			if _, err := promote(root, []string{fmt.Sprintf("old%d", i)}); err != nil {
				t.Error(err)
			}
		}(i)
		go func(i int) {
			defer wg.Done()
			if err := hold(root, "a.txt", "new", map[string][2]int{fmt.Sprintf("new%d", i): {1, 1}}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	store, err := loadPending(root)
	if err != nil {
		t.Fatal(err)
	}
	dropped, resurrected := 0, 0
	for i := 0; i < n; i++ {
		if _, ok := store[fmt.Sprintf("new%d", i)]; !ok {
			dropped++
		}
		if _, ok := store[fmt.Sprintf("old%d", i)]; ok {
			resurrected++
		}
	}
	if dropped != 0 || resurrected != 0 {
		t.Errorf("held and promoted at once: %d new span(s) dropped, %d consumed id(s) back in the store", dropped, resurrected)
	}
}
