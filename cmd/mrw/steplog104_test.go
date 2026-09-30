package main

import (
	"os"
	"strings"
	"testing"
)

// ADR-104 T1 (the review of #295). A passing step whose one output line was
// cut in the tail keeps its log, and the human receipt names it: the line under
// the head carries the cut marker, and the line after it names the log, which
// holds the whole line. Before, the pointer printed only when lines were
// dropped, so the marker counted bytes the receipt gave no place to find.
func TestAPassingStepNamesTheLogItKept(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": "{\"check\":\"exit 0\"}",
	})
	plan := primed(t, root)
	step := "head -c 10000 /dev/zero | tr '\\0' x; echo"
	out, code := writeIn(t, root, "--no-check", "--then-sh", step, plan)
	shown := afterHead(t, out, "— PASS")
	if code != 0 || !strings.Contains(shown, " … [") {
		t.Fatalf("exit %d, the line under the head is %.80q, want a cut line:\n%.400s", code, shown, out)
	}
	pointer := afterHead(t, out, shown)
	if !strings.HasPrefix(pointer, "full output: ") {
		t.Fatalf("the line after the cut line is %q, want it to name the kept log:\n%.400s", pointer, out)
	}
	log := strings.TrimSpace(strings.TrimPrefix(pointer, "full output: "))
	t.Cleanup(func() { _ = os.Remove(log) })
	b, err := os.ReadFile(log)
	if err != nil || !strings.Contains(string(b), strings.Repeat("x", 10000)) {
		t.Errorf("the named log does not hold the whole line (%v, %d bytes)", err, len(b))
	}
}
