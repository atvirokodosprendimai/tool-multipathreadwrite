package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-108 T10. T1 stops a sparse read's checkpoint from spanning its gaps, but
// what an older binary already issued survived the upgrade: span 1-100 in a
// #mrw-seen v2 ledger, or held in pending.json, still licensed line 50. Both
// are discarded: the ledger's version is bumped and the store renamed.
func TestPermissionsIssuedUnderTheOldCheckpointRulesAreDiscarded(t *testing.T) {
	body := strings.Repeat("x\n", 100)

	root, _ := checkout(t, "a.txt", body)
	sha, err := currentSHA(root, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	lp, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(lp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lp, []byte("#mrw-seen v2\n"+sha+"  1-100  a.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if l, err := seen.Load(root); err != nil || licenses50(l) {
		t.Errorf("a v2 ledger's span 1-100 still licenses line 50 (%v)", err)
	}
	if stale, err := seen.IsStale(root); err != nil || !stale {
		t.Errorf("a v2 ledger was not reported stale: %v %v", stale, err)
	}

	root, _ = checkout(t, "a.txt", body)
	old, err := state.Path(root, "pending.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(old), 0o755); err != nil {
		t.Fatal(err)
	}
	store, _ := json.Marshal(map[string]pending{"ck1": {Path: "a.txt", SHA: sha, Start: 1, End: 100, Seq: 1}})
	if err := os.WriteFile(old, store, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := promote(root, []string{"ck1"}); err != nil {
		t.Fatal(err)
	}
	if l, err := seen.Load(root); err != nil || licenses50(l) {
		t.Errorf("a hold of 1-100 from pending.json was acknowledged into a licence (%v)", err)
	}
}

// licenses50 reports whether the ledger holds a.txt at all and covers its line
// 50; a missing entry's zero Observation would read as whole-file.
func licenses50(l seen.Ledger) bool {
	o, ok := l["a.txt"]
	return ok && o.Covers(50, 50)
}
