package mcp

import (
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
)

// ADR-079. Every MCP dry run was tallied as refused_apply, whether it would
// have applied or not. A clean one records nothing; a refused one is a refusal.
func TestACleanMCPDryRunRecordsNothing(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	sc := structured(t, call(t, root, "mrw_write", map[string]any{"plan": "@@ n.txt 0 create\nx\n", "dry_run": true}))
	if sc["failed"] != float64(0) {
		t.Fatalf("the dry run failed: %v", sc)
	}
	if tally, _ := authoring.Load(root); tally.Plans() != 0 {
		t.Errorf("a clean MCP dry run was tallied: %v", tally)
	}
	call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 9 replace\nb\n", "dry_run": true})
	if tally, _ := authoring.Load(root); tally["refused_apply"] != 1 || tally.Plans() != 1 {
		t.Errorf("a refused MCP dry run is not one refusal: %v", tally)
	}
}
