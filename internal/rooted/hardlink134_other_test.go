package rooted

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-134 compares a file with THIS checkout's state directory, the files that
// license a write here. Another checkout's ledger reached by a hard link is
// outside the comparison (the record's Out of Scope): the test pins that the
// boundary is a decision, not a gap nobody saw.
func TestALinkToAnotherCheckoutsLedgerIsNotJudged(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir, err := state.Dir(other)
	if err != nil {
		t.Fatal(err)
	}
	ledger := filepath.Join(dir, "seen")
	if err := os.WriteFile(ledger, []byte("#mrw-seen v4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(ledger, filepath.Join(root, "other.txt")); err != nil {
		t.Skipf("hard links are not available here: %v", err)
	}
	if _, err := Resolve(root, "other.txt"); err != nil {
		t.Errorf("Resolve(other.txt), a link to another checkout's ledger = %v, want it outside this checkout's comparison", err)
	}
}
