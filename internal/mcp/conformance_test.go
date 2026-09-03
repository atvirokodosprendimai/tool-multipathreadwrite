package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestEveryDeclaredOutputSchemaValidatesARealResponse is ADR-011's Enforced-by.
//
// It CALLS each tool and validates what actually came back. That is the whole
// point: a schema checked against the code that produced it is a mirror, and a
// peer measured two ways a hand-checked schema still lies — one attached to the
// wrong tool, one that declared "object, no properties" and validated anything.
// Both read fine.
func TestEveryDeclaredOutputSchemaValidatesARealResponse(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\ntwo\n")

	responses := map[string]map[string]any{}
	responses["mrw_read"] = structured(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}}))
	responses["mrw_write"] = structured(t, call(t, root, "mrw_write",
		map[string]any{"plan": "@@ a.txt 1 replace\nONE\n"}))

	declared := 0
	for _, tool := range tools() {
		if tool.OutputSchema == nil {
			t.Errorf("tool %q declares no outputSchema; a caller cannot validate what it gets", tool.Name)
			continue
		}
		declared++
		got, ok := responses[tool.Name]
		if !ok {
			t.Fatalf("no real response captured for %q — this test must call every declared tool", tool.Name)
		}
		schema, ok := tool.OutputSchema.(map[string]any)
		if !ok {
			t.Fatalf("tool %q outputSchema is not an object: %T", tool.Name, tool.OutputSchema)
		}
		if err := validate(got, schema, tool.Name); err != nil {
			t.Errorf("tool %q: %v", tool.Name, err)
		}
	}
	if declared != len(tools()) {
		t.Errorf("%d of %d tools declare an outputSchema", declared, len(tools()))
	}
}

func TestTheFirstContentBlockIsTheSerializedStructuredContent(t *testing.T) {
	// The spec's SHOULD: "a tool that returns structured content SHOULD also
	// return the serialized JSON in a TextContent block" — A block, not the
	// first. The report stays at content[0] because for mrw_read that is where
	// the file content lives, and a caller already reading content[0] must not
	// silently start receiving a receipt instead. The JSON rides at content[1].
	root, path := checkout(t, "a.txt", "one\ntwo\n")
	for name, args := range map[string]map[string]any{
		"mrw_read":  {"specs": []any{path}},
		"mrw_write": {"plan": "@@ a.txt 1 replace\nONE\n"},
	} {
		res := call(t, root, name, args)
		content, ok := res["content"].([]any)
		if !ok || len(content) < 2 {
			t.Fatalf("%s: want at least two content blocks (json, then the report), got %v", name, res["content"])
		}
		if txt, _ := content[0].(map[string]any)["text"].(string); txt == "" {
			t.Errorf("%s: content[0] is empty; the human-readable report belongs there", name)
		}
		second, _ := content[1].(map[string]any)
		text, _ := second["text"].(string)
		var decoded map[string]any
		if err := json.Unmarshal([]byte(text), &decoded); err != nil {
			t.Fatalf("%s: content[1] is not the serialized JSON: %v\n%q", name, err, text)
		}
		// One marshal used twice, not two marshals of one value — two is how
		// the halves start to disagree.
		want, _ := json.Marshal(res["structuredContent"])
		var wantMap map[string]any
		_ = json.Unmarshal(want, &wantMap)
		gotJSON, _ := json.Marshal(decoded)
		wantJSON, _ := json.Marshal(wantMap)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("%s: content[1] and structuredContent disagree.\n got %s\nwant %s", name, gotJSON, wantJSON)
		}

	}
}

func TestTheAnnotationsMatchWhatTheToolDoes(t *testing.T) {
	// A host shows annotations to a user before asking them to approve a call.
	// An annotation that flatters the tool is a lie the user acts on, so this
	// asserts read-only-ness by OBSERVATION rather than by reading the field.
	root, path := checkout(t, "a.txt", "one\ntwo\n")
	full := filepath.Join(root, "a.txt")
	before, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]tool{}
	for _, tl := range tools() {
		byName[tl.Name] = tl
		if tl.Annotations == nil {
			t.Errorf("tool %q carries no annotations; a host cannot tell whether it writes", tl.Name)
		}
		if tl.Title == "" {
			t.Errorf("tool %q has no title", tl.Name)
		}
	}

	if ro := readOnly(t, byName["mrw_read"]); !ro {
		t.Error("mrw_read is not annotated readOnlyHint:true")
	}
	call(t, root, "mrw_read", map[string]any{"specs": []any{path}})
	after, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("mrw_read claims readOnlyHint:true and changed the file")
	}

	if ro := readOnly(t, byName["mrw_write"]); ro {
		t.Error("mrw_write is annotated readOnlyHint:true, and it writes")
	}
	if d, _ := byName["mrw_write"].Annotations.(map[string]any)["destructiveHint"].(bool); !d {
		t.Error("mrw_write is not annotated destructiveHint:true")
	}
}

func readOnly(t *testing.T, tl tool) bool {
	t.Helper()
	a, ok := tl.Annotations.(map[string]any)
	if !ok {
		return false
	}
	v, _ := a["readOnlyHint"].(bool)
	return v
}

// validate checks a real payload against a declared schema: every property it
// carries must be declared, every required property must be present, and every
// declared type must match the JSON kind that actually arrived.
//
// It RECURSES. The first version compared only top-level names and required
// keys, and missed two live violations for that reason — apply.HunkResult
// omits removed_first for anything but a delete while the schema required it,
// and seen.Observation.Spans arrives as null against a schema that allowed
// only an array. Both live one level down. A conformance test that stops at
// the top level is a conformance test for the top level.
func validate(got map[string]any, schema map[string]any, where string) error {
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return fmt.Errorf("%s: schema declares no properties, so it validates anything", where)
	}
	for k, v := range got {
		ps, ok := props[k].(map[string]any)
		if !ok {
			return fmt.Errorf("%s: response carries %q, which the schema does not declare (declares %v)", where, k, keysOf(props))
		}
		if err := kindOK(v, ps, where+"."+k); err != nil {
			return err
		}
	}
	if req, ok := schema["required"].([]string); ok {
		for _, k := range req {
			if _, ok := got[k]; !ok {
				return fmt.Errorf("%s: schema requires %q, which the response does not carry", where, k)
			}
		}
	}
	return nil
}

// kindOK compares one value against its declared type, following arrays and
// objects down. A schema declaring `applied` as a string would pass a
// name-only check; this is what makes "validates a real response" the sentence
// it claims to be.
func kindOK(v any, schema map[string]any, where string) error {
	types := declaredTypes(schema)
	if len(types) == 0 {
		return nil // unconstrained: the schema claims nothing to contradict
	}
	actual := jsonKind(v)
	if !slices.Contains(types, actual) {
		return fmt.Errorf("%s: value is %s, schema declares %v", where, actual, types)
	}
	switch actual {
	case "array":
		items, _ := schema["items"].(map[string]any)
		if items == nil {
			return nil
		}
		for i, el := range v.([]any) {
			if err := kindOK(el, items, fmt.Sprintf("%s[%d]", where, i)); err != nil {
				return err
			}
		}
	case "object":
		obj := v.(map[string]any)
		if ap, ok := schema["additionalProperties"].(map[string]any); ok {
			for k, el := range obj {
				if err := kindOK(el, ap, where+"."+k); err != nil {
					return err
				}
			}
			return nil
		}
		if _, ok := schema["properties"]; ok {
			return validate(obj, schema, where)
		}
	}
	return nil
}

func declaredTypes(schema map[string]any) []string {
	switch t := schema["type"].(type) {
	case string:
		return []string{t}
	case []string:
		return t
	}
	return nil
}

// jsonKind names the JSON type of a decoded value. Numbers decode as float64,
// so an integral one is reported as integer AND number by the caller's check.
func jsonKind(v any) string {
	switch n := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case float64:
		if n == float64(int64(n)) {
			return "integer"
		}
		return "number"
	}
	return "unknown"
}
