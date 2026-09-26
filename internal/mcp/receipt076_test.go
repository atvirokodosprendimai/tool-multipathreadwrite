package mcp

import (
	"reflect"
	"testing"
)

// ADR-076 T4 over MCP: the receipt is the engine's Result, so the directories a
// plan made reach mrw_write's structured answer — the value a host hands the
// model (ADR-023) — and the schema describes them.
func TestTheWriteReceiptNamesTheDirectoriesItMade(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	sc := structured(t, call(t, root, "mrw_write", map[string]any{"plan": "@@ n/deep/c.txt 0 create\nc\n"}))
	if sc["applied"] != true {
		t.Fatalf("the create did not apply: %v", sc)
	}
	got, _ := sc["dirs_created"].([]any)
	if want := []any{"n", "n/deep"}; !reflect.DeepEqual(got, want) && !reflect.DeepEqual(got, []any{"n", `n\deep`}) {
		t.Errorf("dirs_created = %v, want %v", sc["dirs_created"], want)
	}
}
