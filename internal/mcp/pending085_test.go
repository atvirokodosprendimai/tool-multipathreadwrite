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
