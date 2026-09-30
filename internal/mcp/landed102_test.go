package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// licensed makes a checkout holding files, every one wholly known to the ledger.
func licensed(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	obs := map[string]seen.Observation{}
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		obs[name] = seen.Observation{SHA: seen.SHA([]byte(body))}
	}
	if err := seen.Record(root, obs); err != nil {
		t.Fatal(err)
	}
	return root
}

func skipUnlessPermissionsBind(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file or directory does not stop this user from writing it")
	}
}

// ADR-102 T1. The MCP twin of the CLI test: a commit that failed after a file
// landed is one partially_applied, not a refusal.
func TestAnMCPPartialCommitIsCountedAsPartiallyApplied(t *testing.T) {
	skipUnlessPermissionsBind(t)
	root := licensed(t, map[string]string{"a.go": "package a\n", "d/x.txt": "x\n"})
	d := filepath.Join(root, "d")
	if err := os.Chmod(d, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	call(t, root, "mrw_write", map[string]any{"plan": "@@ a.go 1 replace\npackage b\n@@ d/x.txt - unlink\n"})
	tally, err := authoring.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if tally["partially_applied"] != 1 || tally["refused_apply"] != 0 || tally.Landed() != 1 {
		t.Errorf("tally %v landed %d, want partially_applied 1, refused_apply 0, landed 1", tally, tally.Landed())
	}
	// A landed write joins the ring and the pricing (the review of #293).
	if n, p := len(authoring.Recent(root)), authoring.LoadPricing(root); n != 1 || p.Candidates != 1 {
		t.Errorf("recent ring %d, strict_candidates %d; want 1 and 1", n, p.Candidates)
	}
}

// ADR-102 T3. A write that landed and whose ledger could not be saved was
// answered with a bare JSON-RPC error and no receipt, so a client could not tell
// it from a write that did nothing. It is answered with the receipt: isError,
// applied, the written file, and the error.
func TestALedgerFailureStillSendsTheMCPReceipt(t *testing.T) {
	skipUnlessPermissionsBind(t)
	root := licensed(t, map[string]string{"a.txt": "a\n"})
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ledger, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nA\n"})
	sc := structured(t, res)
	files, _ := sc["files"].([]any)
	var wrote bool
	if len(files) == 1 {
		f, _ := files[0].(map[string]any)
		wrote = f["written"] == true
	}
	if res["isError"] != true || sc["applied"] != true || !wrote || strings.TrimSpace(asString(sc["error"])) == "" {
		t.Errorf("want isError, applied, a.txt written and an error; got isError %v, structured %v", res["isError"], sc)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.txt")); string(b) != "A\n" {
		t.Errorf("a.txt did not land: %q", b)
	}
	if tally, _ := authoring.Load(root); tally["applied"] != 1 {
		t.Errorf("tally %v, want applied 1", tally)
	}
	// The landed write joins the ring, and the receipt's pattern says so (the review of #293).
	pat, _ := sc["pattern"].(map[string]any)
	if n := len(authoring.Recent(root)); n != 1 || pat["window"] != float64(1) {
		t.Errorf("recent ring %d, receipt pattern %v; want 1 and window 1", n, pat)
	}
}

func asString(v any) string { s, _ := v.(string); return s }

// ADR-102 T4. The message for a write whose receipt does not fit said the
// write happened and then advised re-running — which applies the plan again.
func TestAnUnreportableWriteSaysNotToRerun(t *testing.T) {
	for _, partial := range []bool{false, true} {
		m := unreportableAt(200000, 3, 5, 0, partial)
		if !strings.Contains(m, "HAPPENED") || !strings.Contains(m, "do not re-send") || strings.Contains(m, "re-run with") {
			t.Errorf("partial %v: %q", partial, m)
		}
	}
}
