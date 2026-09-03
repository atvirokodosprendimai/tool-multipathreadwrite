package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		props, _ := schema["properties"].(map[string]any)
		if len(props) == 0 {
			t.Errorf("tool %q declares a property-less schema, which validates anything", tool.Name)
			continue
		}
		// Every property the response carries must be one the schema declares.
		// This is the direction that catches a schema attached to the WRONG
		// tool — the peer's failure — because the names simply will not match.
		for k := range got {
			if _, ok := props[k]; !ok {
				t.Errorf("tool %q returned field %q which its outputSchema does not declare; "+
					"schema declares %v", tool.Name, k, keysOf(props))
			}
		}
		// And every required property must actually be present.
		if req, ok := schema["required"].([]string); ok {
			for _, k := range req {
				if _, ok := got[k]; !ok {
					t.Errorf("tool %q schema requires %q, which the real response does not carry", tool.Name, k)
				}
			}
		}
	}
	if declared != len(tools()) {
		t.Errorf("%d of %d tools declare an outputSchema", declared, len(tools()))
	}
}

func TestTheFirstContentBlockIsTheSerializedStructuredContent(t *testing.T) {
	// The spec's SHOULD: "a tool that returns structured content SHOULD also
	// return the serialized JSON in a TextContent block." Before this, content
	// carried a rendered report and a host reading only content got prose.
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
		first, _ := content[0].(map[string]any)
		text, _ := first["text"].(string)
		var decoded map[string]any
		if err := json.Unmarshal([]byte(text), &decoded); err != nil {
			t.Fatalf("%s: content[0] is not JSON: %v\n%q", name, err, text)
		}
		// One marshal used twice, not two marshals of one value — two is how
		// the halves start to disagree.
		want, _ := json.Marshal(res["structuredContent"])
		var wantMap map[string]any
		_ = json.Unmarshal(want, &wantMap)
		gotJSON, _ := json.Marshal(decoded)
		wantJSON, _ := json.Marshal(wantMap)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("%s: content[0] and structuredContent disagree.\n got %s\nwant %s", name, gotJSON, wantJSON)
		}
		// The human-readable report is still there, one block later.
		second, _ := content[1].(map[string]any)
		if txt, _ := second["text"].(string); txt == "" {
			t.Errorf("%s: content[1] is empty; the report must survive the change", name)
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
