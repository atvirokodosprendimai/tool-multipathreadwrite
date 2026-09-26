package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// stateUnderRoot points mrw's state inside root and returns the path of name
// in this checkout's state directory, root-relative. A read makes the ack
// store; the ledger is recorded the way an acknowledged read records it.
func stateUnderRoot(t *testing.T, root, name string) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}})
	if err := seen.Record(root, map[string]seen.Observation{"a.txt": {SHA: "0"}}); err != nil {
		t.Fatal(err)
	}
	hits, err := filepath.Glob(filepath.Join(root, ".st", "mrw", "*", name))
	if err != nil || len(hits) != 1 {
		t.Fatalf("no single %s under the state directory: %v %v", name, hits, err)
	}
	rel, err := filepath.Rel(root, hits[0])
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(rel)
}

// ADR-077. A read of pending.json served the checkpoint ids an ack names, so a
// caller could ack lines it never received (ADR-031). It is refused.
func TestTheAckStoreCannotBeReadOverMCP(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	rel := stateUnderRoot(t, root, "pending.json")
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{rel}})
	text := ""
	if c, ok := res["content"].([]any); ok && len(c) > 0 {
		text, _ = c[0].(map[string]any)["text"].(string)
	}
	if !strings.Contains(text, "own state") || strings.Contains(text, "| ") {
		t.Errorf("a read of %s was served, or not refused as mrw's own state:\n%s", rel, text)
	}
}

// ADR-077. A plan could edit the ledger that licenses every write. It is
// refused, and the ledger's bytes are unchanged.
func TestAPlanCannotEditTheLedger(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	rel := stateUnderRoot(t, root, "seen")
	before, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	sc := structured(t, call(t, root, "mrw_write", map[string]any{"plan": "@@ " + rel + " 1 replace\nX\n"}))
	if sc["applied"] != false || !strings.Contains(sc["hunks"].([]any)[0].(map[string]any)["reason"].(string), "own state") {
		t.Errorf("a plan editing the ledger was not refused as mrw's own state: %v", sc)
	}
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil || string(after) != string(before) {
		t.Errorf("the ledger changed: %v", err)
	}
}
