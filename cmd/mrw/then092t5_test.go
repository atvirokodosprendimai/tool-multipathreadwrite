package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

// ADR-092 T5. A malformed steps block in a tree whose writes ask for no step
// changes nothing — the write lands and check runs, as on v1.30.0. Asking for
// a step is what reads the block, and that is refused with nothing written.
func TestAPlainWriteIgnoresAMalformedStepsBlock(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0","steps":{"my step":"true"}}`,
	})
	if out, code := writeIn(t, root, primed(t, root)); code != 0 {
		t.Fatalf("a write asking for no step: exit %d:\n%s", code, out)
	}
	if out, code := runIn(t, root, "check", "--full"); code != 0 {
		t.Fatalf("a check asking for no step: exit %d:\n%s", code, out)
	}
	before, _ := os.ReadFile(filepath.Join(root, "a.go"))
	out, code := runIn(t, root, "write", "--then", "anything", planFile(t, "@@ a.go 2 replace anchor=\"func A\"\nfunc A() { _ = 3 }\n"))
	after, _ := os.ReadFile(filepath.Join(root, "a.go"))
	if code != exitUsage || string(before) != string(after) || !strings.Contains(out, `"my step"`) {
		t.Errorf("asking for a step: exit %d, changed %v, want 2 naming \"my step\":\n%s", code, string(before) != string(after), out)
	}
}

// ADR-092 T5. A step name or command holding control bytes is printed quoted,
// on the then line and in the declared list, so it cannot clear or forge the
// screen the caller reads.
func TestAStepNameWithControlBytesIsQuoted(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0","steps":{"x\u001b[2Jy":"true"}}`,
	})
	plan := primed(t, root)
	out, code := writeIn(t, root, "--no-check", "--then", "x\x1b[2Jy", "--then-sh", "true \x1b[31m", plan)
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if strings.Contains(out, "\x1b") || !strings.Contains(out, `"x\x1b[2Jy"`) {
		t.Errorf("the then lines are not quoted:\n%q", out)
	}
	out, code = runIn(t, root, "write", "--no-check", "--then", "zzz", plan)
	if code != exitUsage || strings.Contains(out, "\x1b") {
		t.Errorf("the declared list carries a raw control byte: exit %d:\n%q", code, out)
	}
}

// ADR-092 T5. A step that could not start says so once, on the then line and
// in the exit message alike.
func TestACouldNotStartStepSaysSoOnce(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	plan := primed(t, root)
	t.Setenv("PATH", t.TempDir())
	out, code := runIn(t, root, "write", "--no-check", "--then", "a", plan)
	if code != exitUsage || !strings.Contains(strings.ToLower(out), "could not start") {
		t.Fatalf("exit %d, want %d naming could not start:\n%s", code, exitUsage, out)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.Count(strings.ToLower(l), "could not start") > 1 {
			t.Errorf("said twice: %q", l)
		}
	}
}

// ADR-092 T5. mrw refuses --then and --then-sh at MRW_STEP_DEPTH 8, before
// anything is written, on write and on check; one level shallower runs. A step
// that re-runs mrw with steps therefore stops instead of recursing without end.
func TestTheStepDepthGuardRefusesADeepSequence(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	plan := primed(t, root)
	before, _ := os.ReadFile(filepath.Join(root, "a.go"))
	t.Setenv("MRW_STEP_DEPTH", "8")
	out, code := runIn(t, root, "write", "--no-check", "--then-sh", "echo x >> log", plan)
	after, _ := os.ReadFile(filepath.Join(root, "a.go"))
	if code != exitUsage || string(before) != string(after) || !strings.Contains(out, "MRW_STEP_DEPTH") || logOf(t, root) != "" {
		t.Errorf("at depth 8: exit %d, changed %v, log %q:\n%s", code, string(before) != string(after), logOf(t, root), out)
	}
	if out, code := runIn(t, root, "check", "--full", "--then", "a"); code != exitUsage {
		t.Errorf("check at depth 8: exit %d:\n%s", code, out)
	}
	t.Setenv("MRW_STEP_DEPTH", "7")
	if out, code := runIn(t, root, "write", "--no-check", "--then-sh", "echo x >> log", plan); code != 0 || logOf(t, root) != "x\n" {
		t.Errorf("at depth 7: exit %d, log %q:\n%s", code, logOf(t, root), out)
	}
	if !strings.Contains(guide.CLI(), "MRW_STEP_DEPTH") {
		t.Error("mrw instructions does not teach the depth guard")
	}
}
