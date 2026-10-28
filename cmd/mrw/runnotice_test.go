package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// mrw run records nothing in the ledger and serves nothing from it (ADR-177
// Decision 3), so a ledger notice under it names a ledger the call never used:
// three runs printed the stale notice three times. A read on the same ledger
// still says so, which is where the notice helps.
func TestRunPrintsNoLedgerNoticeWhileReadStillDoes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ledger := func(header string) {
		t.Helper()
		if err := seen.Record(root, map[string]seen.Observation{"f.txt": {SHA: strings.Repeat("0", 64)}}); err != nil {
			t.Fatal(err)
		}
		p, err := seen.ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(header+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct{ name, header, notice string }{
		{"older", "#not-a-ledger", "written by an older mrw"},
		{"newer", "#mrw-seen v999 v9.9.9", "written by a newer mrw"},
	} {
		ledger(c.header)
		if got := stderrOf(t, func() {
			if out, code := runIn(t, root, "run", "true"); code != 0 {
				t.Errorf("%s ledger: mrw run true exited %d: %s", c.name, code, out)
			}
		}); strings.Contains(got, "read ledger") {
			t.Errorf("%s ledger: mrw run printed a ledger notice, though it never uses the ledger:\n%s", c.name, got)
		}
		ledger(c.header)
		if got := stderrOf(t, func() { runIn(t, root, "read", "f.txt") }); !strings.Contains(got, c.notice) {
			t.Errorf("%s ledger: read on the same ledger did not say %q: %q", c.name, c.notice, got)
		}
	}
}
