package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// plantLedger writes header as the root's ledger, whose first line it is.
func plantLedger(t *testing.T, root, header string) {
	t.Helper()
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte(header+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ADR-157 Decision 1. Under a newer ledger a not-read refusal is led by its
// cause and names neither a read nor --force, neither of which can license the
// write; its kind stays NotRead, so stats still counts it. The pair: under a
// current ledger the reason is apply's own.
func TestANewerLedgerReplacesTheNotReadReason(t *testing.T) {
	const generic = "f.txt has not been read: mrw does not know what it currently holds, and a line address means nothing without that. Run mrw read f.txt first, or pass --force"
	refused := func() apply.Result {
		return apply.Result{Failed: 1, Hunks: []apply.HunkResult{{Path: "f.txt", Addr: "1", Op: "replace", Reason: generic, Kind: refusal.NotRead}}}
	}

	root := checkout(t)
	plantLedger(t, root, "#mrw-seen v7 v9.9.9")
	res := refused()
	newerLedger(root, &res)
	h := res.Hunks[0]
	if !strings.HasPrefix(h.Reason, "f.txt cannot be edited by this mrw: the read ledger was written by a newer mrw (v9.9.9)") {
		t.Errorf("the reason is not led by the newer ledger: %q", h.Reason)
	}
	if strings.Contains(h.Reason, "mrw read") || strings.Contains(h.Reason, "--force") || strings.Contains(h.Reason, "has not been read") {
		t.Errorf("the reason still offers advice that cannot help: %q", h.Reason)
	}
	if h.Kind != refusal.NotRead {
		t.Errorf("the replaced reason lost its kind: %v", h.Kind)
	}

	root = checkout(t)
	plantLedger(t, root, "#mrw-seen v5 v1.66.0 (05ec3f8)")
	res = refused()
	newerLedger(root, &res)
	if res.Hunks[0].Reason != generic {
		t.Errorf("the pair: under a current ledger the reason changed: %q", res.Hunks[0].Reason)
	}
}
