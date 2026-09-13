package mcp

import (
	"strings"
	"testing"
)

// ADR-055 T3: strict_balance lives on the existing mrw_write tool (ADR-044:
// two tools, flags not cargo), defaults to off, and when set refuses the
// wrap-tail signature through this surface exactly as the CLI does.
func TestWriteToolDeclaresStrictBalance(t *testing.T) {
	var schema map[string]any
	for _, tl := range tools() {
		if tl.Name == "mrw_write" {
			schema, _ = tl.InputSchema.(map[string]any)
		}
	}
	props, _ := schema["properties"].(map[string]any)
	p, ok := props["strict_balance"].(map[string]any)
	if !ok {
		t.Fatal("mrw_write does not declare strict_balance")
	}
	if p["type"] != "boolean" {
		t.Errorf("strict_balance type = %v, want boolean", p["type"])
	}
	desc := strings.ToLower(stringOf(p["description"]))
	for _, must := range []string{"refuse", "single-line", "off"} {
		if !strings.Contains(desc, must) {
			t.Errorf("strict_balance description lacks %q: %v", must, p["description"])
		}
	}
}

func TestAnMCPWriteWithStrictBalanceRefusesTheSignature(t *testing.T) {
	root, path := checkout(t, "f.go", "func A() {\n\treturn\n}\n")
	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path + ":1"}})))
	got := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan":           "@@ f.go 1 replace anchor=\"func A\"\nfunc A() { return }\n",
		"ack":            acks,
		"strict_balance": true,
	}))
	if applied, _ := got["applied"].(bool); applied {
		t.Fatalf("strict_balance over MCP applied the signature: %v", got)
	}
	hunks, _ := got["hunks"].([]any)
	h, _ := hunks[0].(map[string]any)
	if h["status"] != "failed" || !strings.Contains(stringOf(h["reason"]), "strict-balance") {
		t.Errorf("hunk = %v, want failed with a strict-balance reason", h)
	}
}
