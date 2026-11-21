package writer

import (
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-209 Decision 6. The ledger records a file only when its path holds the bytes
// mrw wrote: a file an undo left missing, or holding another process's file
// (NotHeld), is not recorded though its record stays written; and with every file
// put back there is nothing to record.
func TestTheLedgerSkipsAFileWhosePathDoesNotHoldWhatMrwWrote(t *testing.T) {
	root := checkout(t)
	held := apply.FileResult{Path: "held.txt", Written: true, SHAAfter: seen.SHA([]byte("H\n"))}
	notHeld := apply.FileResult{Path: "gone.txt", Written: true, SHAAfter: seen.SHA([]byte("G\n")), NotHeld: true}

	wrote, _ := ledgerChanges(root, apply.Result{Files: []apply.FileResult{held, notHeld}}, apply.Options{})
	if o, ok := wrote["held.txt"]; !ok || o.SHA != held.SHAAfter || !o.Whole() {
		t.Errorf("the file that holds mrw's bytes is not recorded wholly: %+v (present %v)", o, ok)
	}
	if o, ok := wrote["gone.txt"]; ok {
		t.Errorf("a file whose path does not hold mrw's bytes is recorded: %+v", o)
	}

	if wrote, gone := ledgerChanges(root, apply.Result{}, apply.Options{}); len(wrote) != 0 || len(gone) != 0 {
		t.Errorf("a commit that put every file back records %v and drops %v", wrote, gone)
	}
}
