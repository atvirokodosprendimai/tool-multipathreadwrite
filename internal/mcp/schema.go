package mcp

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// SchemaOf derives a JSON Schema object from a Go struct, following the json
// tags a caller will actually see on the wire.
//
// It is generated rather than hand-written for one measured reason: a peer's
// hand-written schemas produced two failures that read perfectly in review — one
// attached to the WRONG tool, and one declaring "object, no properties", which
// validates anything and tells a caller nothing. A schema derived from the type
// the handler returns cannot drift from it, because there is nothing to keep in
// sync.
//
// It refuses a struct with no serialized fields rather than emitting the
// permissive form, so the second of those failures is unrepresentable.
func SchemaOf(v any) (map[string]any, error) {
	t := reflect.TypeOf(v)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("mcp: SchemaOf needs a struct, got %T — only a struct describes an object", v)
	}
	props, required := fieldsOf(t)
	if len(props) == 0 {
		return nil, fmt.Errorf("mcp: %s serializes no fields; a property-less object schema validates anything", t)
	}
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 && !hasCustomMarshal(t) {
		s["required"] = required
	}
	return s, nil
}

// hasCustomMarshal reports whether a type marshals itself. Reflection cannot see
// through MarshalJSON, so for such a type the FIELD NAMES are still a useful
// description while the presence of any of them is a guess.
//
// This is not hypothetical: apply.HunkResult emits removed_first and
// removed_last only for a successful delete, so a schema built from its struct
// tags required two fields an ordinary replace response does not carry —
// measured 2026-09-03, and missed by a conformance test that only compared
// top-level keys. Declaring a field required is a promise about the payload,
// and this is exactly where that promise cannot be kept.
func hasCustomMarshal(t reflect.Type) bool {
	marshaler := reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	return t.Implements(marshaler) || reflect.PointerTo(t).Implements(marshaler)
}

// fieldsOf walks a struct's exported fields, honouring json tags. Embedded
// structs are flattened the way encoding/json flattens them, so the schema
// matches the payload rather than the declaration.
func fieldsOf(t reflect.Type) (map[string]any, []string) {
	props := map[string]any{}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported: never serialized
		}
		tag := f.Tag.Get("json")
		name, opts, _ := strings.Cut(tag, ",")
		if name == "-" && opts == "" {
			continue
		}
		if f.Anonymous && name == "" {
			// Embedded and untagged: encoding/json inlines its fields, so the
			// schema must inline them too or it describes a shape nobody sends.
			inner, innerReq := fieldsOf(f.Type)
			for k, v := range inner {
				props[k] = v
			}
			required = append(required, innerReq...)
			continue
		}
		if name == "" {
			name = f.Name
		}
		props[name] = typeSchema(f.Type)
		// omitempty means a caller may not see it; anything else is always
		// present in the payload, so it is required.
		if !strings.Contains(opts, "omitempty") {
			required = append(required, name)
		}
	}
	return props, required
}

// typeSchema describes one field. The subset is deliberately small — object,
// string, boolean, integer, number, array — because that is what the two tool
// results are made of. ADR-011's Stop Condition says to say so rather than
// reach for a JSON Schema library if the types outgrow it.
func typeSchema(t reflect.Type) map[string]any {
	nullable := t.Kind() == reflect.Pointer
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	s := baseSchema(t)
	if nullable {
		s = orNull(s)
	}
	return s
}

// orNull widens a schema to admit null, for the Go shapes that marshal as null
// rather than as their zero value: a nil pointer, a nil slice, a nil map.
//
// Declaring only "array" for a nil slice makes a strict validator reject an
// ordinary response — seen.Observation.Spans is nil for a whole-file read,
// which is the COMMON case, and a schema that rejects the common case is worse
// than no schema at all.
func orNull(s map[string]any) map[string]any {
	switch v := s["type"].(type) {
	case string:
		s["type"] = []string{v, "null"}
	case nil:
		// An unconstrained schema already admits null.
	}
	return s
}

func baseSchema(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice:
		return orNull(map[string]any{"type": "array", "items": typeSchema(t.Elem())})
	case reflect.Array:
		// A fixed-size array is never nil.
		return map[string]any{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Map:
		// A map's keys are data, so its properties cannot be enumerated. This
		// is the one place a property-less object is honest, and it is marked
		// so a reader does not mistake it for the permissive form SchemaOf
		// refuses at the top level.
		return orNull(map[string]any{"type": "object", "additionalProperties": typeSchema(t.Elem())})
	case reflect.Struct:
		props, required := fieldsOf(t)
		if len(props) == 0 {
			return map[string]any{"type": "object"}
		}
		s := map[string]any{"type": "object", "properties": props}
		// Reflection cannot see through a custom MarshalJSON, so requiredness
		// is a promise this cannot keep for such a type. See hasCustomMarshal.
		if len(required) > 0 && !hasCustomMarshal(t) {
			s["required"] = required
		}
		return s
	case reflect.Interface:
		// Anything. Saying so beats claiming a shape we do not know.
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

// MaxResultChars is the largest tool result this server will produce. It is
// advertised in each tool's `_meta` and enforced on the read path — ONE
// constant with two readers, so the advertised limit and the enforced limit
// cannot drift.
//
// It is enforced in BYTES while the host's key is named in characters. That is
// deliberate and conservative in the safe direction: a UTF-8 character is at
// least one byte, so bounding bytes at N guarantees at most N characters, and
// the server can only ever come in UNDER what it advertised. The refusal
// message says bytes, because that is what was counted.
//
// The value is Claude Code's per-tool ceiling. Its global default is 25,000
// tokens (`MAX_MCP_OUTPUT_TOKENS`), and a tool may declare up to 500,000
// characters via `anthropic/maxResultSizeChars`; 200,000 leaves room for the
// envelope and for the serialized JSON that rides beside the report, since a
// result carries the receipt twice by design.
// https://code.claude.com/docs/en/mcp
const MaxResultChars = 200_000

// readSchema and writeSchema describe what each tool returns. They panic on a
// generation failure rather than returning an error, because the types are
// compile-time constants of this package: a failure here is a programming
// mistake that every test and every startup would hit immediately, not a
// runtime condition a caller could act on.
func readSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"observed": map[string]any{
				"type":                 "object",
				"description":          "What each served file was observed to hold, keyed by path.",
				"additionalProperties": mustSchema(seen.Observation{}),
			},
			"problems": map[string]any{
				"type":        "integer",
				"description": "How many requested ranges could not be served.",
			},
		},
		"required": []string{"observed", "problems"},
	}
}

func writeSchema() map[string]any { return mustSchema(apply.Result{}) }

func mustSchema(v any) map[string]any {
	s, err := SchemaOf(v)
	if err != nil {
		panic("mcp: " + err.Error())
	}
	return s
}
