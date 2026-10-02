package mcp

import (
	"errors"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ADR-105 T1 over MCP. slashResult spells every left_behind entry with "/"
// and leaves the engine's slice alone, and the output schema describes the
// field — a property with no description is one a host shows as noise.
func TestMrwWriteSpellsLeftBehindWithSlashes(t *testing.T) {
	in := apply.Result{LeftBehind: []string{`d\.mrw-1`, `.mrw-aside-2`}}
	got := slashResult(in, '\\')
	if got.LeftBehind[0] != "d/.mrw-1" || got.LeftBehind[1] != ".mrw-aside-2" {
		t.Errorf("left_behind = %q, want d/.mrw-1 and .mrw-aside-2", got.LeftBehind)
	}
	if in.LeftBehind[0] != `d\.mrw-1` {
		t.Errorf("slashResult changed the engine's slice: %q", in.LeftBehind)
	}
	props, _ := writeSchema()["properties"].(map[string]any)
	p, _ := props["left_behind"].(map[string]any)
	if d, _ := p["description"].(string); d == "" {
		t.Errorf("the output schema does not describe left_behind: %v", p)
	}

	// The review of the record: a receipt too big for the ceiling ends in a
	// sentence with no structured value, and left_behind went with it. Both
	// terminal sentences — nothing written, and written but unreportable —
	// carry the count, and the write floor bounds the longer one.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withCeiling(t, 64)
	failed := []apply.HunkResult{{Path: "a.go", Addr: "1", Op: "replace", Status: apply.StatusFailed, Reason: "r"}}
	for _, res := range []apply.Result{
		{Failed: 1, Hunks: failed, LeftBehind: []string{".mrw-1", ".mrw-2"}},
		{Files: []apply.FileResult{{Path: "a.go", Written: true}}, Hunks: failed[:0], LeftBehind: []string{".mrw-1", ".mrw-2"}},
	} {
		out, rpcErr := boundedReceipt(t.TempDir(), res, errors.New("stopped"))
		if rpcErr != nil || len(out.Content) == 0 {
			t.Fatalf("no terminal answer: %v %+v", rpcErr, out)
		}
		if text := out.Content[0].Text; !strings.Contains(text, "left 2 path(s)") || !strings.Contains(text, "create and rename targets") || !strings.Contains(text, "only to a path that is free") {
			t.Errorf("the terminal sentence does not carry the leftover count, the probe targets and the aside warning:\n%s", text)
		}
	}
	if n := encodedSize(errorResult(appliedButUnreportable(1<<40, 1<<40, 1<<40, true) + leftNote(1<<40))); n > floorAt(MaxResultChars) {
		t.Errorf("the write floor %d does not bound the sentence with its leftover note (%d)", floorAt(MaxResultChars), n)
	}
}
