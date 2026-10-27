package main

import (
	"os"
	"os/exec"
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

// The review of #527: over a ledger a newer mrw wrote, `seen --forget` is
// refused, exit 2, and says so once (ADR-150), not once from the root's notice
// and again from its own refusal, and without the "mrw: " main prints itself.
// The flag's spellings the parser accepts are one flag.
func TestSeenForgetOverANewerLedgerSaysSoOnce(t *testing.T) {
	for _, flag := range []string{"--forget", "--forget=true", "-forget"} {
		root := newerRoot(t)
		var out string
		var code int
		errOut := stderrOf(t, func() { out, code = runIn(t, root, "seen", flag) })
		if code != 2 {
			t.Fatalf("seen %s over a newer ledger exited %d, want 2: %s%s", flag, code, out, errOut)
		}
		if n := strings.Count(out+errOut, "written by a newer mrw"); n != 1 {
			t.Errorf("seen %s: the newer-ledger notice appears %d times, want once:\n%s%s", flag, n, out, errOut)
		}
		if strings.HasPrefix(out, "mrw: ") {
			t.Errorf("seen %s: the refusal carries its own mrw: prefix, main adds one: %q", flag, out)
		}
	}
}

// The review of #527: the ledger is the checkout's, so a forget run from a
// subdirectory (a hook's cwd) empties what a read at the top recorded.
func TestSeenForgetFromASubdirectoryForgetsTheCheckout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if b, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init: %v: %s", err, b)
	}
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
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
	if out, code := runIn(t, sub, "seen", "--forget"); code != 0 || !strings.Contains(out, "forgot 1 file(s)") {
		t.Fatalf("seen --forget from sub/ exited %d: %q; want the checkout's one file forgotten", code, out)
	}
	if out, code := runIn(t, root, "write", "--no-check", plan); code != 1 {
		t.Errorf("a write at the top after a forget from sub/ exited %d, want 1: %s", code, out)
	}
	if got, _ := os.ReadFile(f); string(got) != "one\ntwo\nthree\n" {
		t.Errorf("the refused write changed the file: %q", got)
	}
}
