package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-157 Decision 6. A read, a plan and a check that name mrw's own state are
// refused as before (ADR-077), and none of them sends the caller to a --root
// that cannot lift the refusal. The pair: a path that leaves the root through
// a link keeps its --root advice, which fits it.
func TestTheOwnStateRefusalOffersNoRootAdvice(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte(`{"check":"true"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, code := runIn(t, root, "read", "f.txt"); code != 0 {
		t.Fatalf("the read that makes the ledger exited %d", code)
	}
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, ledger)
	if err != nil {
		t.Fatal(err)
	}
	advice := []string{"pointed where you mean", "may only change files under"}
	refusedWithout := func(what, out string, code int) {
		t.Helper()
		if code == 0 || !strings.Contains(out, "own state directory") {
			t.Errorf("%s was not refused as mrw's own state: exit %d\n%s", what, code, out)
		}
		for _, a := range advice {
			if strings.Contains(out, a) {
				t.Errorf("%s still advises %q:\n%s", what, a, out)
			}
		}
	}

	out, code := runIn(t, root, "read", rel)
	refusedWithout("a read", out, code)

	plan := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(plan, []byte("@@ "+filepath.ToSlash(rel)+" 1 replace\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut string
	errOut = stderrOf(t, func() { out, code = runIn(t, root, "write", "--no-check", plan) })
	refusedWithout("a plan", out+errOut, code)

	errOut = stderrOf(t, func() { out, code = runIn(t, root, "check", rel) })
	refusedWithout("a check", out+errOut, code)

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "x.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Skipf("no symlink here for the pair: %v", err)
	}
	if out, _ := runIn(t, root, "read", "out/x.txt"); !strings.Contains(out, "pointed where you mean") {
		t.Errorf("the pair: a path that leaves the root lost its --root advice:\n%s", out)
	}
}
