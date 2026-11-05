package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// forgetCheckout makes a git checkout holding f.txt and a plan that replaces its
// second line, and returns the checkout's link-resolved path, which is what
// state.Checkout names, and the plan.
func forgetCheckout(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	if b, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init: %v: %s", err, b)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(plan, []byte("@@ f.txt 2 replace\nTWO\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return real, plan
}

// forgetReceipt is the receipt ADR-180 Decision 1 prints for n files in top.
func forgetReceipt(n int, top string) string {
	if n == 0 {
		return "forgot 0 file(s) in " + top
	}
	return fmt.Sprintf("forgot %d file(s) in %s: a write now needs a fresh read of the lines it touches", n, top)
}

// ADR-180 Decision 1: run from a subdirectory of checkout A, the receipt names
// A's top, for a nonzero and a zero count; A's ledger is emptied and B's is not.
// From a git worktree of A, whose .git is a file, it names the worktree's own
// top and leaves A's ledger alone.
func TestSeenForgetNamesTheCheckoutItEmptied(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a, planA := forgetCheckout(t)
	b, planB := forgetCheckout(t)
	sub := filepath.Join(a, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{a, b} {
		if out, code := runIn(t, r, "read", "f.txt"); code != 0 {
			t.Fatalf("read in %s exited %d: %s", r, code, out)
		}
	}

	out, code := runIn(t, sub, "seen", "--forget")
	if want := forgetReceipt(1, a); code != 0 || strings.TrimSpace(out) != want {
		t.Fatalf("seen --forget from A/sub: exit %d %q; want %q", code, out, want)
	}
	if out, code = runIn(t, a, "write", "--no-check", planA); code != 1 {
		t.Errorf("a write in A after the forget exited %d, want 1: %s", code, out)
	}
	if got, _ := os.ReadFile(filepath.Join(a, "f.txt")); string(got) != "one\ntwo\nthree\n" {
		t.Errorf("the refused write changed A's file: %q", got)
	}
	if out, code = runIn(t, b, "write", "--no-check", planB); code != 0 {
		t.Errorf("the pair: B's read no longer licenses B's write, exit %d: %s", code, out)
	}
	out, code = runIn(t, sub, "seen", "--forget")
	if want := forgetReceipt(0, a); code != 0 || strings.TrimSpace(out) != want {
		t.Errorf("an empty ledger: exit %d %q; want %q", code, out, want)
	}

	git := func(args ...string) {
		t.Helper()
		full := append([]string{"-C", a, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)
		if b, err := exec.Command("git", full...).CombinedOutput(); err != nil {
			t.Skipf("git %v: %v: %s", args, err, b)
		}
	}
	git("add", "f.txt")
	git("commit", "-q", "-m", "f")
	w := filepath.Join(t.TempDir(), "w")
	git("worktree", "add", "-q", w)
	w, err := filepath.EvalSymlinks(w)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{a, w} {
		if out, code := runIn(t, r, "read", "f.txt"); code != 0 {
			t.Fatalf("read in %s exited %d: %s", r, code, out)
		}
	}
	out, code = runIn(t, w, "seen", "--forget")
	if want := forgetReceipt(1, w); code != 0 || strings.TrimSpace(out) != want {
		t.Errorf("seen --forget in a worktree of A: exit %d %q; want %q", code, out, want)
	}
	if out, code = runIn(t, a, "write", "--no-check", planA); code != 0 {
		t.Errorf("the pair: a forget in A's worktree emptied A's ledger, exit %d: %s", code, out)
	}
}

// ADR-180 Decision 1 with --root: from A's directory, `mrw --root B seen
// --forget` names B and empties B's ledger, and A's read still licenses A's write.
func TestSeenForgetWithRootNamesThatCheckoutAndLeavesTheOtherAlone(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a, planA := forgetCheckout(t)
	b, planB := forgetCheckout(t)
	for _, r := range []string{a, b} {
		if out, code := runIn(t, r, "read", "f.txt"); code != 0 {
			t.Fatalf("read in %s exited %d: %s", r, code, out)
		}
	}
	t.Chdir(a)
	out, code := runIn(t, b, "seen", "--forget")
	if want := forgetReceipt(1, b); code != 0 || strings.TrimSpace(out) != want {
		t.Fatalf("mrw --root B seen --forget from A: exit %d %q; want %q", code, out, want)
	}
	if out, code = runIn(t, b, "write", "--no-check", planB); code != 1 {
		t.Errorf("a write in B after its forget exited %d, want 1: %s", code, out)
	}
	if out, code = runIn(t, a, "write", "--no-check", planA); code != 0 {
		t.Errorf("the pair: A's read no longer licenses A's write, exit %d: %s", code, out)
	}
}
