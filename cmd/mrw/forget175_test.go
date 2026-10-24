package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-175 Decision 1 through the command: a read licenses a write, `seen
// --forget` says how many files it held and the same plan is then refused, a
// fresh read licenses it again. The usage errors: --forget beside --prune or
// --dry-run is exit 2.
func TestSeenForgetMakesTheNextWriteNeedAFreshRead(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	f := filepath.Join(root, "f.txt")
	if err := os.WriteFile(f, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(plan, []byte("@@ f.txt 2 replace\nTWO\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, code := runIn(t, root, "read", "f.txt"); code != 0 {
		t.Fatalf("read exited %d: %s", code, out)
	}
	out, code := runIn(t, root, "seen", "--forget")
	if code != 0 || !strings.Contains(out, "forgot 1 file(s): a write now needs a fresh read of the lines it touches") {
		t.Fatalf("seen --forget exited %d: %q", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", plan); code != 1 {
		t.Fatalf("a write after --forget exited %d, want 1: %s", code, out)
	}
	if got, _ := os.ReadFile(f); string(got) != "one\ntwo\nthree\n" {
		t.Errorf("the refused write changed the file: %q", got)
	}
	if out, code = runIn(t, root, "read", "f.txt"); code != 0 {
		t.Fatalf("fresh read exited %d: %s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", plan); code != 0 {
		t.Errorf("the pair: a fresh read did not license the write, exit %d: %s", code, out)
	}
	if out, code = runIn(t, root, "seen", "--forget"); code != 0 || !strings.Contains(out, "forgot 1 file(s)") {
		t.Errorf("after a write the ledger holds the file again: exit %d %q", code, out)
	}
	if out, code = runIn(t, root, "seen", "--forget"); code != 0 || !strings.Contains(out, "forgot 0 file(s)") {
		t.Errorf("an empty ledger: exit %d %q; want forgot 0 file(s)", code, out)
	}

	for _, extra := range []string{"--prune", "--dry-run"} {
		if out, code = runIn(t, root, "seen", "--forget", extra); code != 2 {
			t.Errorf("--forget %s exited %d, want 2: %s", extra, code, out)
		}
	}
}
