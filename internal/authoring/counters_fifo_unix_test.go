//go:build unix

package authoring

import (
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// A FIFO at the path of the tally, the recent ring or the pricing counters held
// every write that records there, under the tally's lock, after its files had
// reached disk: os.Open on a FIFO waits for a writer that never comes. ADR-200
// T1's review found it for the numbering counters; this is the rest of the
// class. Each is read through regular.Open, so a path that is not a regular
// file reads as empty and the write goes on.
func TestAFifoAtACountersPathDoesNotHoldAWrite(t *testing.T) {
	cases := []struct {
		name   string
		record func(root string)
	}{
		{file, func(root string) { Record(root, Applied); _, _ = Load(root) }},
		{recentFile, func(root string) { RecordRecent(root, 1); _ = Recent(root) }},
		{pricingFile, func(root string) { RecordPricing(root, true, true, PricedHeld); _ = LoadPricing(root) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			root := t.TempDir()
			path, err := state.Path(root, c.name)
			if err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Skipf("no FIFO here: %v", err)
			}
			done := make(chan struct{})
			go func() { c.record(root); close(done) }()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatalf("recording to %s is still waiting on the FIFO: the write that landed would print no receipt", c.name)
			}
		})
	}
}
