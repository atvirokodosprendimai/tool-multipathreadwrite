package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// staleWords is what ADR-157 Decision 5 has the stale notice say instead of a
// discard it may not make.
const staleWords = "it is not used, and the next recorded read or write replaces it"

// ledgerRoot is a checkout holding f.txt, with the state home pointed at a
// temporary directory, and the path its ledger lives at.
func ledgerRoot(t *testing.T) (root, ledger string) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, ledger
}

// ADR-157 Decision 5. help, h and a bare mrw use no ledger, so they say nothing
// of a stale one and leave it as it was; a refused write records nothing and
// says the notice, which claims no discard; a read says it once and replaces
// the ledger.
func TestOnlyCommandsThatUseTheLedgerTellItIsStale(t *testing.T) {
	root, ledger := ledgerRoot(t)
	const old = "#mrw-seen v3\n"
	if err := os.WriteFile(ledger, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, argv := range [][]string{{"help"}, {"h"}, {}} {
		errOut := stderrOf(t, func() { runIn(t, root, argv...) })
		if strings.Contains(errOut, "read ledger") {
			t.Errorf("mrw %v spoke of the ledger: %q", argv, errOut)
		}
		if got, _ := os.ReadFile(ledger); string(got) != old {
			t.Errorf("mrw %v changed the ledger: %q", argv, got)
		}
	}

	plan := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(plan, []byte("@@ f.txt 1 replace\nONE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var code int
	errOut := stderrOf(t, func() { _, code = runIn(t, root, "write", "--no-check", plan) })
	if code != 1 || strings.Count(errOut, staleWords) != 1 || strings.Contains(errOut, "has been discarded") {
		t.Errorf("a refused write: exit %d, stderr %q; want the notice once, claiming no discard", code, errOut)
	}
	if got, _ := os.ReadFile(ledger); string(got) != old {
		t.Errorf("a refused write, which records nothing, changed the ledger: %q", got)
	}

	errOut = stderrOf(t, func() { _, code = runIn(t, root, "read", "f.txt") })
	if code != 0 || strings.Count(errOut, staleWords) != 1 {
		t.Errorf("a read: exit %d, stderr %q; want the notice once", code, errOut)
	}
	if got, _ := os.ReadFile(ledger); !strings.HasPrefix(string(got), "#mrw-seen v5 ") {
		t.Errorf("the read did not replace the ledger: %q", got)
	}
}

// ADR-157 Decision 4. mrw seen names the ledger's format and writer on one
// comment line after the state directory, which stays first; nothing when
// there is no ledger.
func TestSeenNamesTheLedgersFormatAndWriter(t *testing.T) {
	root, ledger := ledgerRoot(t)
	seenOut := func(t *testing.T) string {
		t.Helper()
		out, code := runIn(t, root, "seen")
		if code != 0 || strings.HasPrefix(out, "#") {
			t.Fatalf("mrw seen: exit %d, or its first line is not the state directory:\n%s", code, out)
		}
		return out
	}
	if out := seenOut(t); strings.Contains(out, "# ledger") {
		t.Errorf("with no ledger, mrw seen names one:\n%s", out)
	}
	if _, code := runIn(t, root, "read", "f.txt"); code != 0 {
		t.Fatalf("read exited %d", code)
	}
	if out := seenOut(t); !strings.Contains(out, "# ledger format v5, written by "+versionString()+"\n") {
		t.Errorf("after a read, mrw seen does not name this mrw:\n%s", out)
	}
	for header, want := range map[string]string{
		"#mrw-seen v4":        "# ledger format v4, written by unknown\n",
		"#mrw-seen v5":        "# ledger format v5, written by unknown\n",
		"#mrw-seen v6 v9.9.9": "# ledger written by a newer mrw (v9.9.9), format v6: left untouched, licenses nothing\n",
		"#mrw-seen v3":        "# ledger in an older format: not used; mrw replaces it the next time it records a read or a write\n",
	} {
		if err := os.WriteFile(ledger, []byte(header+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if out := seenOut(t); !strings.Contains(out, want) {
			t.Errorf("%s: mrw seen does not say %q:\n%s", header, want, out)
		}
	}
}
