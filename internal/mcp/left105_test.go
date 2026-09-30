package mcp

import (
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
}
