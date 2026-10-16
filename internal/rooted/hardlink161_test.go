package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-161 Decision 14. The ledger below a checkout's top is the top's, so a
// hard link under a subdirectory root to that ledger is mrw's state too (ADR-134),
// and a hard link to an ordinary file is served as before.
func TestAHardLinkInASubdirectoryRootToTheCheckoutsLedgerIsRefused(t *testing.T) {
	st := t.TempDir()
	t.Setenv("XDG_STATE_HOME", st)
	top := Real(t.TempDir())
	sub := filepath.Join(top, "sub")
	for _, d := range []string{filepath.Join(top, ".git"), sub} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dir, err := state.Dir(top)
	if err != nil {
		t.Fatal(err)
	}
	ledger := filepath.Join(dir, "seen")
	ord := filepath.Join(sub, "ord.txt")
	for p, body := range map[string]string{ledger: "#mrw-seen v6\n", ord: "ordinary\n"} {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for from, to := range map[string]string{ledger: "hl.txt", ord: "ord2.txt"} {
		if err := os.Link(from, filepath.Join(sub, to)); err != nil {
			t.Skipf("hard links are not available here: %v", err)
		}
	}
	if _, err := Resolve(sub, "hl.txt"); err == nil || !strings.Contains(err.Error(), "own state") {
		t.Errorf("Resolve(sub, hl.txt), a hard link to the top's ledger = %v, want it refused as mrw's own state", err)
	}
	for _, p := range []string{"ord.txt", "ord2.txt"} {
		if _, err := Resolve(sub, p); err != nil {
			t.Errorf("Resolve(sub, %s) refused a file that is not mrw's state: %v", p, err)
		}
	}
}
