//go:build unix

package authoring

import (
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-200 T1, found by the Codex review of 1bfcaace: a FIFO at the counters'
// path held the landing being counted, under the tally's lock, and the write
// never printed its receipt. The counters are read through regular.Open, so a
// path that is not a regular file reads as zero and the count goes on.
func TestAFifoAtTheNumberingPathDoesNotHoldALanding(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	path, err := state.Path(root, numberingFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("no FIFO here: %v", err)
	}
	done := make(chan Numbering, 1)
	go func() {
		RecordNumbering(root, 1, 1)
		done <- LoadNumbering(root)
	}()
	select {
	case got := <-done:
		if got.Landed > 1 || got.Candidates > 1 || got.WouldRefuse > 1 {
			t.Errorf("counters after one landing over a FIFO: %+v", got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RecordNumbering is still waiting on the FIFO: a landing would have printed no receipt")
	}
}
