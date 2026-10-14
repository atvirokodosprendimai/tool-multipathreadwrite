package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-150 Decisions 2 and 3, through the command: a read prints the newer
// line once and not the stale one, a write that needs a licence is refused
// naming the newer ledger, and the ledger keeps its bytes. The pair: the same
// entry under a v4 header licenses the write.
func TestTheCLINamesANewerLedgersWriter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	content := []byte("one\ntwo\n")
	if err := os.WriteFile(filepath.Join(root, "f.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	entry := hex.EncodeToString(sum[:]) + "  -  f.txt\n"
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	newer := "#mrw-seen v6 v9.9.9 (fffffff)\n" + entry
	if err := os.WriteFile(ledger, []byte(newer), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(plan, []byte("@@ f.txt 1 replace\nONE\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var code int
	errOut := stderrOf(t, func() { _, code = runIn(t, root, "read", "f.txt") })
	if code != 0 {
		t.Errorf("a read against a newer ledger exited %d", code)
	}
	if n := strings.Count(errOut, "written by a newer mrw (v9.9.9 (fffffff))"); n != 1 || strings.Contains(errOut, "written by an older mrw") {
		t.Errorf("read printed %q, want the newer line once and no stale notice", errOut)
	}

	var out string
	errOut = stderrOf(t, func() { out, code = runIn(t, root, "write", "--no-check", plan) })
	if code != 1 || !strings.Contains(out, "f.txt cannot be edited by this mrw") || !strings.Contains(out, "written by a newer mrw (v9.9.9 (fffffff))") {
		t.Errorf("a write the newer ledger would license exited %d, receipt %q stderr %q; want it refused naming the newer ledger", code, out, errOut)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "f.txt")); string(got) != string(content) {
		t.Errorf("the refused write changed the file: %q", got)
	}
	if got, _ := os.ReadFile(ledger); string(got) != newer {
		t.Errorf("the newer ledger was written over: %q", got)
	}

	if err := os.WriteFile(ledger, []byte("#mrw-seen v4\n"+entry), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, code = runIn(t, root, "write", "--no-check", plan); code != 0 {
		t.Errorf("the pair: a v4 ledger's entry did not license the write, exit %d: %s", code, out)
	}
	if got, _ := os.ReadFile(ledger); !strings.HasPrefix(string(got), "#mrw-seen v5 "+versionString()+"\n") {
		t.Errorf("the ledger the write saved does not name this mrw (%s): %q", versionString(), got)
	}
}
