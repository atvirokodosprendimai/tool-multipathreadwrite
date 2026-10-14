package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ADR-132. A refusal found while staging, whose cause is the target's — here a
// directory the write may not create in — answers over MCP as any refused plan
// does: isError with its receipt, nothing applied, one failed hunk, no `error`
// (which names a write's own failure), and the tally counts it refused.
func TestATargetRefusalOverMCPIsARefusedPlan(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a read-only directory is a unix permission")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores the directory's mode")
	}
	root := checkTree113(t, `{"steps":{"x":"echo ran"}}`)
	ro := filepath.Join(root, "ro")
	if err := os.Mkdir(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	before := tally113(t, root)["refused_apply"]
	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ ro/n.txt 0 create\nx\n", "check": false, "then": []any{"x"}})
	if res["isError"] != true {
		t.Fatalf("a refused plan was not isError: %v", res)
	}
	sc := structured(t, res)
	if sc["applied"] != false || sc["failed"] != float64(1) {
		t.Fatalf("want applied false and one failed hunk: %v", sc)
	}
	if e, ok := sc["error"]; ok {
		t.Fatalf("a target-caused refusal carries a write error %q, as a failure of the write itself would", e)
	}
	// The refused plan reaches the steps' verdict as every refused plan does:
	// not run. A write that failed with an error never got that far.
	then, _ := sc["then"].(map[string]any)
	steps, _ := then["steps"].([]any)
	if len(steps) != 1 || steps[0].(map[string]any)["status"] != "not_run" {
		t.Fatalf("the requested step is not reported not_run: %v", sc["then"])
	}
	if got := tally113(t, root)["refused_apply"]; got != before+1 {
		t.Fatalf("refused_apply went from %d to %d, want one more", before, got)
	}
	if _, err := os.Stat(filepath.Join(ro, "n.txt")); !os.IsNotExist(err) {
		t.Fatal("the refused create left a file")
	}
}
