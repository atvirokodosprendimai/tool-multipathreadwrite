package mcp

import (
	"math"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

const stepsHarness = `{"check":"exit 0","steps":{"ok":"echo fine","bad":"echo nope; exit 4"}}`

// ADR-115 T1. mrw_write ran the check and stopped: the steps a project
// declares stayed on the CLI. It takes `then`, names of declared steps, runs
// them after a passing check, and reports them as `mrw write --json` does. A
// step that ran and failed is data; one that could not start is isError.
func TestAnMCPWriteRunsItsSteps(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the steps are POSIX shell lines")
	}
	steps := func(t *testing.T, sc map[string]any) []map[string]any {
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
	t.Run("a declared step runs after a passing check", func(t *testing.T) {
		root := checkTree113(t, stepsHarness)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "then": []any{"ok"}})
		st := steps(t, structured(t, res))
		if len(st) != 1 || st[0]["status"] != "pass" || res["isError"] == true {
			t.Fatalf("then %v isError %v, want one passing step", st, res["isError"])
		}
		if got := tally113(t, root); got["applied"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want applied 1", got)
		}
	})
	t.Run("a failing step is data and counted failed_check", func(t *testing.T) {
		root := checkTree113(t, stepsHarness)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "then": []any{"bad", "ok"}})
		st := steps(t, structured(t, res))
		if len(st) != 2 || st[0]["status"] != "fail" || st[1]["status"] != "not_run" || res["isError"] == true {
			t.Fatalf("then %v isError %v, want fail then not_run, not isError", st, res["isError"])
		}
		if got := tally113(t, root); got["failed_check"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want failed_check 1", got)
		}
	})
	t.Run("an undeclared name writes nothing", func(t *testing.T) {
		root := checkTree113(t, stepsHarness)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "then": []any{"nosuch"}})
		if res["isError"] != true || !strings.Contains(firstText(t, res), "nothing was written") {
			t.Fatalf("want a refusal naming nothing written: %v", res)
		}
		// It names the argument this caller sends, not the CLI's flag.
		if txt := firstText(t, res); !strings.HasPrefix(txt, "then nosuch:") {
			t.Errorf("the refusal does not name `then`: %q", txt)
		}
		if got := tally113(t, root); got["refused_apply"] != 1 {
			t.Errorf("tally %v, want refused_apply 1", got)
		}
	})
	t.Run("a step that cannot start is isError with the receipt", func(t *testing.T) {
		root := checkTree113(t, stepsHarness)
		t.Setenv("PATH", t.TempDir())
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ notes.md 2 replace anchor=\"line two\"\nline 2\n", "then": []any{"ok"}})
		sc := structured(t, res)
		st := steps(t, sc)
		if res["isError"] != true || sc["applied"] != true || len(st) != 1 || st[0]["status"] != "could_not_start" {
			t.Fatalf("isError %v applied %v then %v, want isError with an applied receipt and could_not_start", res["isError"], sc["applied"], st)
		}
		// The reason a step could not start is in the text, as the CLI says it.
		if skipped, _ := st[0]["skipped"].(string); skipped == "" || !strings.Contains(firstText(t, res), skipped) {
			t.Errorf("the text does not say why the step could not start (%q): %q", skipped, firstText(t, res))
		}
	})
	t.Run("a failed check leaves the steps not run", func(t *testing.T) {
		root := checkTree113(t, `{"check":"exit 3","steps":{"ok":"echo fine"}}`)
		st := steps(t, structured(t, call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "then": []any{"ok"}})))
		if len(st) != 1 || st[0]["status"] != "not_run" {
			t.Fatalf("then %v, want the step not_run", st)
		}
	})
	t.Run("the depth limit refuses before anything is written", func(t *testing.T) {
		root := checkTree113(t, stepsHarness)
		t.Setenv("MRW_STEP_DEPTH", "8")
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "then": []any{"ok"}, "check": false})
		if res["isError"] != true || !strings.Contains(firstText(t, res), "then") {
			t.Fatalf("want a depth refusal naming then: %v", res)
		}
		if got := tally113(t, root); got.Plans() != 0 {
			t.Errorf("tally %v, want nothing counted", got)
		}
	})
	t.Run("every step phrase fits the write floor", func(t *testing.T) {
		for _, s := range []string{check.StepFail, check.StepCouldNotStart, check.StepTimedOut} {
			r := &check.StepsResult{Steps: []check.StepResult{{Status: s}}}
			msg := errorResult(appliedButUnreportable(math.MaxInt, math.MaxInt, math.MaxInt, true) + longestCheckPhrase + stepPhrase(r) + leftNote(math.MaxInt))
			if n, floor := encodedSize(msg), writeFloor(); n > floor || stepPhrase(r) == "" {
				t.Errorf("step %s: phrase %q, answer %d bytes over the floor %d", s, stepPhrase(r), n, floor)
			}
		}
	})
	t.Run("step tails go first under a small ceiling, each kept in its log", func(t *testing.T) {
		// quiet passes with output its tail holds whole, so internal/check
		// removes its log and the tail is the only copy: cut, it must be
		// kept in a log of its own. loudbad fails and keeps its log.
		line := `yes 0123456789abcdefghij0123456789abcdefghij0123456789`
		root := checkTree113(t, `{"check":"exit 0","steps":{"quiet":"`+line+` | head -n 8","loudbad":"`+line+` | head -n 300; exit 4"}}`)
		old := MaxResultChars
		MaxResultChars = 4000
		t.Cleanup(func() { MaxResultChars = old })
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "then": []any{"quiet", "loudbad"}})
		sc := structured(t, res)
		if el, _ := sc["elided"].(string); !strings.Contains(el, "step") {
			t.Errorf("elided does not say the step tails were cut: %q", el)
		}
		st := steps(t, sc)
		if len(st) != 2 || st[0]["status"] != "pass" || st[1]["status"] != "fail" {
			t.Fatalf("then %v, want pass then fail", st)
		}
		for _, s := range st {
			if tail, _ := s["tail"].([]any); len(tail) != 0 {
				t.Errorf("step %v kept its tail under the ceiling", s["name"])
			}
			f, _ := s["output_file"].(string)
			if f == "" {
				t.Errorf("step %v lost its tail: no output_file holds it", s["name"])
				continue
			}
			t.Cleanup(func() { _ = os.Remove(f) }) // the logs this run made are its own to remove
			// The log holds the tail it stands in for, not merely exists.
			if b, err := os.ReadFile(f); err != nil || !strings.Contains(string(b), "0123456789abcdefghij") {
				t.Errorf("step %v: output_file %s does not hold its tail (%v): %q", s["name"], f, err, b)
			}
		}
	})
}
