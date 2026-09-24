package mcp

import (
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ADR-066 T3 guard. The MCP surface renders each hunk's own status, so the
// engine's commit-failure verdicts reach an MCP caller with no code change
// here. This pins that: a report of ok, failed and skipped hunks prints each
// row with its status, and names the error.
func TestTheMCPReceiptOfAPartialCommitNamesEachHunk(t *testing.T) {
	res := apply.Result{
		Hunks: []apply.HunkResult{
			{Path: "a.go", Addr: "1", Op: "replace", Status: apply.StatusOK},
			{Path: "b.go", Addr: "1", Op: "replace", Status: apply.StatusFailed, Reason: "injected"},
			{Path: "c.go", Addr: "1", Op: "replace", Status: apply.StatusSkipped},
		},
		Files:  []apply.FileResult{{Path: "a.go", Written: true}, {Path: "b.go"}, {Path: "c.go"}},
		Failed: 1,
	}
	out := writeReport(res, res.Hunks, errFor("b.go: injected (ALREADY WRITTEN: a.go)"), "")
	for _, want := range []string{"ok a.go", "failed b.go", "skipped c.go", "ALREADY WRITTEN: a.go"} {
		if !strings.Contains(out, want) {
			t.Errorf("the MCP report lacks %q:\n%s", want, out)
		}
	}
}

type errFor string

func (e errFor) Error() string { return string(e) }
