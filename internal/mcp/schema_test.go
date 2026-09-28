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

// TestTheReadReceiptMatchesItsSchema keeps readSchema honest now that it lives
// in a test file (ADR-088). mrw_read publishes no schema (ADR-023), so the
// table is checked against real receipts — a served read, a page, a grep that
// serves, an index and a page of one — in both directions: every key a receipt
// carries is declared, and every declared key is carried by some receipt.
func TestTheReadReceiptMatchesItsSchema(t *testing.T) {
	schema := readSchema()
	props, _ := schema["properties"].(map[string]any)
	observed, _ := props["observed"].(map[string]any)
	entry, _ := observed["additionalProperties"].(map[string]any)
	entryProps, _ := entry["properties"].(map[string]any)
	if len(props) == 0 || len(entryProps) == 0 {
		t.Fatalf("readSchema declares no properties to compare: %v", schema)
	}

	root, path := checkout(t, "a.txt", "one\ntwo\n")
	bigRoot, bigPath := bigCheckout(t, 12000)
	grepRoot := grepTree(t, 3, 2)
	idxRoot := grepTree(t, 60, 400)
	pageRoot := grepTree(t, 400, 400)
	answers := map[string]map[string]any{
		"served": call(t, root, "mrw_read", map[string]any{"specs": []any{path}}),
		"paged":  call(t, bigRoot, "mrw_read", map[string]any{"specs": []any{bigPath}}),
		"grep":   call(t, grepRoot, "mrw_read", map[string]any{"grep": "NEEDLE"}),
		"index":  call(t, idxRoot, "mrw_read", map[string]any{"grep": "NEEDLE"}),
	}
	old := MaxResultChars
	MaxResultChars = 4000
	t.Cleanup(func() { MaxResultChars = old })
	answers["index page"] = call(t, pageRoot, "mrw_read", map[string]any{"grep": "NEEDLE"})

	carried := map[string]bool{}
	for name, res := range answers {
		rc := receipt(t, res)
		for k := range rc {
			if _, ok := props[k]; !ok {
				t.Errorf("%s: the receipt carries %q, which readSchema does not declare", name, k)
			}
			carried[k] = true
		}
		required, _ := schema["required"].([]string)
		for _, k := range required {
			if _, ok := rc[k]; !ok {
				t.Errorf("%s: the receipt lacks %q, which readSchema requires", name, k)
			}
		}
		files, _ := rc["observed"].(map[string]any)
		for file, raw := range files {
			o, ok := raw.(map[string]any)
			if !ok {
				t.Errorf("%s: observed[%q] is %T, not an object", name, file, raw)
				continue
			}
			for k := range o {
				if _, ok := entryProps[k]; !ok {
					t.Errorf("%s: observed[%q] carries %q, which readSchema does not declare", name, file, k)
				}
				carried["observed."+k] = true
			}
		}
	}
	for k := range props {
		if !carried[k] {
			t.Errorf("readSchema declares %q, and none of the %d receipts carries it", k, len(answers))
		}
	}
	for k := range entryProps {
		if !carried["observed."+k] {
			t.Errorf("readSchema declares observed.%s, and none of the %d receipts carries it", k, len(answers))
		}
	}
}

// readSchema describes the mrw_read receipt at content[1]. mrw_read declares no
// schema (ADR-023), so this table lives beside the test that holds it to real
// receipts, TestTheReadReceiptMatchesItsSchema, and not in production (ADR-088).
func readSchema() map[string]any {
	return mustDescribe(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"observed": map[string]any{
				"type":                 "object",
				"additionalProperties": mustSchema(seen.Observation{}),
			},
			"problems": map[string]any{"type": "integer"},
			// Present only on a paged answer, so it is NOT in `required` — a
			// caller's exit condition is precisely its absence.
			"next_read": map[string]any{"type": "string"},
			// Present only on a grep's INDEX answer, and likewise not
			// required. ⚠ ADR-017-T1's first cut claimed this file needed no
			// change because matchIndex builds its own map — which is exactly
			// how a response comes to violate the schema its own tool
			// advertises. A schema-validating host would have rejected it.
			// Found by review of #80.
			"matches":    map[string]any{"type": "integer"},
			"index":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"next_index": map[string]any{"type": "string"},
		},
		"required": []string{"observed", "problems"},
	}, readDescriptions)
}

// readDescriptions says what each receipt property MEANS; mustDescribe refuses
// an entry naming a property readSchema no longer declares.
var readDescriptions = map[string]string{
	"observed":       "What THIS call observed of each served file, keyed by its root-relative path spelled with `/` on every platform, the way a plan names it. It is merged into the per-checkout ledger rather than replacing it, so a later write is authorised by the accumulated spans for the same sha — not by this response alone.",
	"observed.SHA":   "The sha256 of the whole file as it was when served. A later write is refused if the file no longer hashes to this.",
	"observed.Spans": "The line spans this call rendered, as [start, end] pairs; null means the whole file. Authorisation is per LINE: a write to a line no read has served is refused, though a line served by an EARLIER read of the same sha is still licensed.",
	"problems":       "How many requested ranges could not be served. Non-zero means part of what you asked for is missing from `observed` — the call itself still answered.",
	"next_read":      "The spec to send next when this answer is only a PAGE of what you asked for. Absent when nothing remains, which is how you know you have the whole thing. A paged answer is NOT an error and carries no `isError`; it says so in its served text, with a `-- PARTIAL:` line naming the range and what remains. Stopping there leaves you holding part of a file, not the file.",
	"matches":        "How many files matched a `grep`, counting the whole match set and not just this page. Present on an INDEX answer and on a grep that matched nothing (0); a grep whose matches fit is served instead, and its `observed` names the files.",
	"index":          "The matching FILE PATHS, served instead of content when the matches are too large to return. No content came with them and nothing was recorded, so this licenses no write. Send one back as a spec WITH the same grep to read its matches.",
	"next_index":     "The last path on this page of an INDEX. Send the same grep again with `after` set to this for the next page, and repeat until it is empty — an empty value is how you know you have the whole match set.",
}
