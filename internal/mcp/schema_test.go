package mcp

import (
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

func TestTheSchemaNamesTheFieldsOfTheResult(t *testing.T) {
	// Generated from the type, so it cannot drift from what the handler
	// returns. A hand-written schema beside the code is the form that rots.
	s, err := SchemaOf(apply.Result{})
	if err != nil {
		t.Fatalf("SchemaOf(apply.Result{}): %v", err)
	}
	props, ok := s["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema has no properties: %v", s)
	}
	// The json tags, not the Go field names — a caller sees the wire shape.
	for _, want := range []string{"root", "dry_run", "applied", "files", "hunks", "failed"} {
		if _, ok := props[want]; !ok {
			t.Errorf("schema has no property %q; got %v", want, keysOf(props))
		}
	}
	if props["applied"].(map[string]any)["type"] != "boolean" {
		t.Errorf("applied is %v, want boolean", props["applied"])
	}
	if props["failed"].(map[string]any)["type"] != "integer" {
		t.Errorf("failed is %v, want integer", props["failed"])
	}
	if props["hunks"].(map[string]any)["type"] != "array" {
		t.Errorf("hunks is %v, want array", props["hunks"])
	}
	if s["type"] != "object" {
		t.Errorf("type = %v, want object", s["type"])
	}
}

func TestAPropertylessObjectSchemaIsRefused(t *testing.T) {
	// The failure this generator exists to make impossible. A peer shipped a
	// schema of "object, no properties", which validates ANYTHING and tells a
	// caller nothing — and it reads fine. Refusing to emit one means the
	// permissive form cannot be produced by accident.
	type empty struct{}
	if _, err := SchemaOf(empty{}); err == nil {
		t.Error("SchemaOf(struct{}{}) returned a schema; a property-less object validates anything and must be refused")
	}

	// A struct whose fields are all json:"-" is the same thing wearing a
	// disguise, and must be refused for the same reason.
	type hidden struct {
		A int `json:"-"`
		B int `json:"-"`
	}
	if _, err := SchemaOf(hidden{}); err == nil {
		t.Error("a struct with no serialized fields produced a schema; it validates anything")
	}

	// And a non-struct is not an object at all.
	if _, err := SchemaOf(42); err == nil {
		t.Error("SchemaOf(42) returned a schema; only structs describe an object")
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
