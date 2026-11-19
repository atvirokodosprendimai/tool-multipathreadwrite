package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	notesPlan205  = "@@ notes.md 2 replace anchor=\"line two\"\nline 2\n"
	notesBytes205 = "# notes\nline two\n"
	// markHarness205 appends a line to a marker per run of the check; the step
	// `after` fails unless the marker exists, which only a check before it can
	// have made.
	markHarness205 = `{"check":"echo ran >> marker","steps":{"after":"test -f marker"}}`
	// failHarness205's step has a side effect of its own, so a step that ran
	// leaves stepran and one that did not leaves none; the two below do the
	// same for a passing check with a dry run and for a tree with no check.
	failHarness205  = `{"check":"echo ran >> marker; exit 1","steps":{"after":"echo ran >> stepran"}}`
	sideHarness205  = `{"check":"echo ran >> marker","steps":{"after":"echo ran >> stepran"}}`
	noCmdHarness205 = `{"steps":{"after":"echo ran >> stepran"}}`
)

// markers205 is how many times the check ran in root.
func markers205(t *testing.T, root string) int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "marker"))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(b), "\n")
}

// stepRan205 reports whether a step that appends to stepran ran in root.
func stepRan205(root string) bool {
	_, err := os.Stat(filepath.Join(root, "stepran"))
	return err == nil
}

func disk205(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ADR-205. check: true asked for a check and got ADR-054's rule: a write that
// touches only prose, or a tree with no command, ran nothing and the receipt
// carried no check, so "passed" and "never ran" read alike. check: true now
// means a check is due after any write that lands, as the CLI's --check does.
// Each row starts from its own tree, so one row's marker cannot satisfy another.
func TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the checks are POSIX shell lines")
	}
	t.Run("a prose plan with check true runs the check", func(t *testing.T) {
		root := checkTree113(t, markHarness205)
		res := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205, "check": true})
		sc := structured(t, res)
		chk, _ := sc["check"].(map[string]any)
		if disk205(t, root, "notes.md") != "# notes\nline 2\n" || markers205(t, root) != 1 {
			t.Fatalf("notes %q, markers %d, want the edit landed and one check run", disk205(t, root, "notes.md"), markers205(t, root))
		}
		if chk == nil || chk["ran"] != true || chk["exit_code"] != float64(0) || res["isError"] == true {
			t.Fatalf("check %v isError %v, want a passing check and no error", chk, res["isError"])
		}
	})
	t.Run("the same plan without check runs none", func(t *testing.T) {
		root := checkTree113(t, markHarness205)
		sc := structured(t, call(t, root, "mrw_write", map[string]any{"plan": notesPlan205}))
		if _, has := sc["check"]; has || sc["applied"] != true || markers205(t, root) != 0 || disk205(t, root, "notes.md") != "# notes\nline 2\n" {
			t.Fatalf("a write that asked for no check ran one, or did not land: %v", sc)
		}
	})
	t.Run("grep with sub over prose runs the check", func(t *testing.T) {
		root := checkTree113(t, markHarness205)
		res := call(t, root, "mrw_write", map[string]any{"grep": "line two", "sub": "line 2", "check": true})
		if res["isError"] == true || markers205(t, root) != 1 || disk205(t, root, "notes.md") != "# notes\nline 2\n" {
			t.Fatalf("markers %d isError %v notes %q, want one check after the edit landed:\n%s", markers205(t, root), res["isError"], disk205(t, root, "notes.md"), firstText(t, res))
		}
	})
	t.Run("a failing check on a code plan is data, not isError", func(t *testing.T) {
		root := checkTree113(t, failHarness205)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "check": true})
		chk, _ := structured(t, res)["check"].(map[string]any)
		if chk == nil || chk["ran"] != true || chk["exit_code"] != float64(1) || res["isError"] == true {
			t.Fatalf("check %v isError %v, want a check that ran and exited 1, not an error", chk, res["isError"])
		}
		if !strings.Contains(disk205(t, root, "a.go"), "_ = 1") {
			t.Errorf("the edit did not land")
		}
	})
	t.Run("a dry run runs no check and no step", func(t *testing.T) {
		root := checkTree113(t, sideHarness205)
		res := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205, "check": true, "dry_run": true, "then": []any{"after"}})
		sc := structured(t, res)
		if _, has := sc["check"]; has || res["isError"] == true || markers205(t, root) != 0 || stepRan205(root) || disk205(t, root, "notes.md") != notesBytes205 {
			t.Fatalf("a dry run checked, ran a step, wrote, or was refused: %v", sc)
		}
		// Still validates: a step the project did not declare refuses the dry run.
		bad := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205, "check": true, "dry_run": true, "then": []any{"nosuch"}})
		if bad["isError"] != true || disk205(t, root, "notes.md") != notesBytes205 {
			t.Fatalf("a dry run with an undeclared step was not refused: %v", bad)
		}
	})
	t.Run("a step waits for the check", func(t *testing.T) {
		root := checkTree113(t, markHarness205)
		res := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205, "check": true, "then": []any{"after"}})
		st := steps205(t, structured(t, res))
		if len(st) != 1 || st[0]["status"] != "pass" || markers205(t, root) != 1 {
			t.Fatalf("then %v markers %d, want the step passed after one check", st, markers205(t, root))
		}
	})
	t.Run("a step is not run after a check that failed", func(t *testing.T) {
		root := checkTree113(t, failHarness205)
		res := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205, "check": true, "then": []any{"after"}})
		st := steps205(t, structured(t, res))
		if disk205(t, root, "notes.md") != "# notes\nline 2\n" || len(st) != 1 || st[0]["status"] != "not_run" {
			t.Fatalf("then %v, want the edit landed and the step not_run", st)
		}
		if markers205(t, root) != 1 || stepRan205(root) {
			t.Fatalf("markers %d, step ran %v, want the check run once and the step's side effect absent", markers205(t, root), stepRan205(root))
		}
	})
	t.Run("a due check is refused at the depth limit before the write", func(t *testing.T) {
		root := checkTree113(t, markHarness205)
		t.Setenv("MRW_STEP_DEPTH", "8")
		res := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205, "check": true})
		if res["isError"] != true || !strings.Contains(firstText(t, res), "MRW_STEP_DEPTH") || disk205(t, root, "notes.md") != notesBytes205 {
			t.Fatalf("want a depth refusal with the bytes unchanged:\n%v", res)
		}
	})
	t.Run("the same at the depth limit without check lands", func(t *testing.T) {
		root := checkTree113(t, markHarness205)
		t.Setenv("MRW_STEP_DEPTH", "8")
		res := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205})
		if res["isError"] == true || disk205(t, root, "notes.md") != "# notes\nline 2\n" {
			t.Fatalf("a write that asked for no check was refused at depth 8: %v", res)
		}
	})
	t.Run("a tree with no check answers as a check that could not run", func(t *testing.T) {
		root := checkTree113(t, "")
		res := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205, "check": true})
		sc := structured(t, res)
		chk, _ := sc["check"].(map[string]any)
		if res["isError"] != true || sc["applied"] != true || chk == nil || chk["ran"] != false || chk["exit_code"] != float64(-1) {
			t.Fatalf("isError %v receipt %v, want isError, applied, check.ran false, exit_code -1", res["isError"], sc)
		}
		if skipped, _ := chk["skipped"].(string); skipped == "" {
			t.Errorf("check %v carries no skipped reason", chk)
		}
		if disk205(t, root, "notes.md") != "# notes\nline 2\n" {
			t.Errorf("the edit did not land")
		}
	})
	t.Run("a tree with no check leaves a named step not run", func(t *testing.T) {
		root := checkTree113(t, noCmdHarness205)
		res := call(t, root, "mrw_write", map[string]any{"plan": notesPlan205, "check": true, "then": []any{"after"}})
		st := steps205(t, structured(t, res))
		if res["isError"] != true || len(st) != 1 || st[0]["status"] != "not_run" || stepRan205(root) || markers205(t, root) != 0 {
			t.Fatalf("isError %v then %v step ran %v, want an error, the step not_run and no side effect", res["isError"], st, stepRan205(root))
		}
	})
}

// steps205 is the receipt's then.steps.
func steps205(t *testing.T, sc map[string]any) []map[string]any {
	t.Helper()
	then, _ := sc["then"].(map[string]any)
	raw, _ := then["steps"].([]any)
	var out []map[string]any
	for _, s := range raw {
		m, _ := s.(map[string]any)
		out = append(out, m)
	}
	return out
}
