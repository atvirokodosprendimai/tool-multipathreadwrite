package mcp

import (
	"strings"
	"testing"
)

// ADR-118 over MCP: a whole read, acknowledged, licenses occurrence=2, and the
// receipt names the match; a write to a file mrw just wrote still needs the
// matches read (Zy, "strict: always served").
func TestMrwWriteTakesAnOccurrence(t *testing.T) {
	root, path := checkout(t, "x.go", "package x\n\nfunc X() {}\n\nfunc X() {}\n")
	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}})))
	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ x.go /^func X/ replace occurrence=2\nfunc Y() {}\n", "ack": acks, "check": false})
	if res["isError"] == true || !strings.Contains(firstText(t, res), "occurrence=2") {
		t.Fatalf("occurrence=2 after an acknowledged read: %v", firstText(t, res))
	}
	res = call(t, root, "mrw_write", map[string]any{"plan": "@@ x.go /^func / replace occurrence=2\nfunc Z() {}\n", "check": false})
	if res["isError"] != true || !strings.Contains(firstText(t, res), "have not been read") {
		t.Errorf("right after a write, occurrence=2 with no match read: %v", firstText(t, res))
	}
}
