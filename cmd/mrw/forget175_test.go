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

// stateEntries counts the state directories under the test's state home.
func stateEntries(t *testing.T) int {
	t.Helper()
	es, err := os.ReadDir(filepath.Join(os.Getenv("XDG_STATE_HOME"), "mrw"))
	if err != nil {
		return 0
	}
	return len(es)
}

// The review of #533, P2: the missing-root gate is decided by the flag the
// parser read, not by argv. `--forget=` is true to the parser, so a gate on raw
// arguments let it past; `--forget --forget=false` and `-- --forget` are not a
// forget, so the same gate refused what was a plain `seen`. The pair: every
// spelling that is a forget is refused with no state directory, and the others
// answer as a plain `seen` over the same root does.
func TestSeenForgetGateFollowsTheParsedFlag(t *testing.T) {
	for _, argv := range [][]string{
		{"seen", "--forget="},
		{"seen", "--forget=true"},
		{"seen", "-forget"},
		{"seen", "--forget", "--forget=false", "--forget"},
	} {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		gone := filepath.Join(t.TempDir(), "gone")
		out, code := runIn(t, gone, argv...)
		if code != 2 || !strings.Contains(out, "does not exist") {
			t.Errorf("%v over a missing root exited %d, want 2 naming the root: %q", argv, code, out)
		}
		if n := stateEntries(t); n != 0 {
			t.Errorf("%v over a missing root made %d state director(ies)", argv, n)
		}
	}
	for _, argv := range [][]string{
		{"seen", "--forget", "--forget=false"},
		{"seen", "--", "--forget"},
	} {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		gone := filepath.Join(t.TempDir(), "gone")
		plainOut, plain := runIn(t, gone, "seen")
		out, code := runIn(t, gone, argv...)
		if code != plain || strings.Contains(out, "seen --forget:") {
			t.Errorf("%v is not a forget: exited %d with %q; a plain seen exited %d with %q", argv, code, out, plain, plainOut)
		}
	}
}

// The review of #533, P2: a root below a regular file passes rooted.Abs (the
// symlink walk answers ENOTDIR, not "does not exist"), so the gate stats the
// resolved root itself. The pair: an existing directory still forgets.
func TestSeenForgetOverARootBelowAFileIsRefused(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(file, "child")
	out, code := runIn(t, root, "seen", "--forget")
	if code != 2 || !strings.Contains(out, root) {
		t.Errorf("seen --forget over %s exited %d, want 2 naming the root: %q", root, code, out)
	}
	if n := stateEntries(t); n != 0 {
		t.Errorf("a root below a file made %d state director(ies)", n)
	}
	if out, code = runIn(t, dir, "seen", "--forget"); code != 0 {
		t.Errorf("the pair: an existing root exited %d: %q", code, out)
	}
}
