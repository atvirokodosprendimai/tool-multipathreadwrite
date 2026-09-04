package mcp

import (
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
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
	// A nil slice marshals as null, so the schema admits both. Declaring only
	// "array" made a strict validator reject an ordinary response, since
	// seen.Observation.Spans is nil for a whole-file read — the common case.
	if got := props["hunks"].(map[string]any)["type"]; !sameStrings(got, []string{"array", "null"}) {
		t.Errorf(`hunks type is %v, want ["array","null"]`, got)
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

// sameStrings compares a schema "type" value against an expected list.
func sameStrings(got any, want []string) bool {
	gs, ok := got.([]string)
	if !ok || len(gs) != len(want) {
		return false
	}
	for i := range gs {
		if gs[i] != want[i] {
			return false
		}
	}
	return true
}

func TestACustomMarshallerMeansNoRequiredPromise(t *testing.T) {
	// apply.HunkResult marshals itself and omits removed_first/removed_last
	// for anything but a successful delete. A schema built from its struct
	// tags therefore required two fields an ordinary replace response does not
	// carry — measured 2026-09-03, and missed by a conformance test that only
	// compared top-level keys.
	s, err := SchemaOf(apply.Result{})
	if err != nil {
		t.Fatal(err)
	}
	hunks := s["properties"].(map[string]any)["hunks"].(map[string]any)
	item := hunks["items"].(map[string]any)
	if _, ok := item["required"]; ok {
		t.Errorf("hunks items declare required fields, but apply.HunkResult marshals itself: %v", item["required"])
	}
	// The property NAMES are still useful and must survive.
	if _, ok := item["properties"].(map[string]any)["removed_first"]; !ok {
		t.Error("dropping required also dropped the property names")
	}
}

func TestANilSliceIsAdmittedByTheSchema(t *testing.T) {
	// seen.Observation.Spans is nil for a whole-file read and marshals as null.
	s, err := SchemaOf(seen.Observation{})
	if err != nil {
		t.Fatal(err)
	}
	spans := s["properties"].(map[string]any)["Spans"].(map[string]any)
	if !sameStrings(spans["type"], []string{"array", "null"}) {
		t.Errorf(`Spans type is %v, want ["array","null"] — a whole-file read sends null`, spans["type"])
	}
}

// TestADescribedPropertyThatNoLongerExistsIsRefused covers the quieter half of
// the drift.
//
// An UNdescribed property is loud: the coverage test names it. A description
// for a property that has been renamed or removed is silent — the schema still
// validates every response, and the table just describes a field nobody sends,
// until someone reads it and believes it. So a table entry that matches nothing
// is an error at construction, not a no-op.
func TestADescribedPropertyThatNoLongerExistsIsRefused(t *testing.T) {
	schema, err := SchemaOf(apply.Result{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := describeResult(schema, map[string]string{"failed": "how many hunks did not apply"}); err != nil {
		t.Fatalf("describing a property the schema declares failed: %v", err)
	}
	_, err = describeResult(schema, map[string]string{"hunks.was_renamed": "a field that no longer exists"})
	if err == nil {
		t.Fatal("a description for a property the schema does not declare was accepted; it would sit there describing a field nobody sends")
	}
	if !strings.Contains(err.Error(), "hunks.was_renamed") {
		t.Errorf("the refusal does not name the stale entry: %v", err)
	}
}
