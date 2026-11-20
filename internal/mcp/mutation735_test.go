package mcp

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Issue #735, the MCP side: mrw_write's structured value says how much of the
// plan reached the tree on every outcome, as `mrw write --json` does.
func TestTheMCPWriteReceiptSaysHowMuchOfThePlanReachedTheTree(t *testing.T) {
	mutation := func(res map[string]any) string { return asString(structured(t, res)["mutation"]) }

	t.Run("a landed write is complete", func(t *testing.T) {
		root := licensed(t, map[string]string{"a.txt": "a\n"})
		if m := mutation(call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nA\n"})); m != "complete" {
			t.Errorf("mutation %q, want complete", m)
		}
	})
	t.Run("a dry run is none", func(t *testing.T) {
		root := licensed(t, map[string]string{"a.txt": "a\n"})
		if m := mutation(call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nA\n", "dry_run": true})); m != "none" {
			t.Errorf("mutation %q, want none", m)
		}
	})
	t.Run("a refused plan is none", func(t *testing.T) {
		root := licensed(t, map[string]string{"a.txt": "a\n"})
		if m := mutation(call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 9 replace\nA\n"})); m != "none" {
			t.Errorf("mutation %q, want none", m)
		}
	})
	t.Run("a ledger that could not record a landing is complete", func(t *testing.T) {
		skipUnlessPermissionsBind(t)
		root := licensed(t, map[string]string{"a.txt": "a\n"})
		ledger, err := seen.ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ledger, 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nA\n"})
		if m := mutation(res); res["isError"] != true || m != "complete" {
			t.Errorf("isError %v mutation %q, want an error receipt that says complete", res["isError"], m)
		}
	})
	t.Run("a commit that stopped after a file landed is partial", func(t *testing.T) {
		// The apply-level seam is unreachable from here; the receipt is driven
		// directly, as limit_test.go does for the same reason.
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		res := apply.Result{
			Root:  "/tmp/whatever",
			Files: []apply.FileResult{{Path: "a.go", Written: true}, {Path: "b.go"}},
		}
		out, rpcErr := boundedReceipt(t.TempDir(), res, errors.New("b.go: rename failed"))
		if rpcErr != nil {
			t.Fatal(rpcErr)
		}
		var sc struct{ Mutation string }
		raw, _ := out.StructuredContent.(json.RawMessage)
		if err := json.Unmarshal(raw, &sc); err != nil || sc.Mutation != "partial" {
			t.Errorf("mutation %q (%v), want partial", sc.Mutation, err)
		}
	})
}
