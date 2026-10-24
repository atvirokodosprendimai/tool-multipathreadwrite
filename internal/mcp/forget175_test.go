package mcp

import (
	"os"
	"path/filepath"
	"testing"
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
