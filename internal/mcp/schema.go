package mcp

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
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
	return mustDescribe(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"observed": map[string]any{
				"type":                 "object",
				"additionalProperties": mustSchema(seen.Observation{}),
			},
			"problems": map[string]any{"type": "integer"},
		},
		"required": []string{"observed", "problems"},
	}, readDescriptions)
}

func writeSchema() map[string]any { return mustDescribe(mustSchema(apply.Result{}), writeDescriptions) }

// readDescriptions and writeDescriptions say what each property MEANS. The
// shapes stay generated — ADR-011 measured what happens to a hand-written
// schema — and only the prose is authored here.
//
// ⚠ THE PROSE LIVES IN THIS PACKAGE, NOT IN A STRUCT TAG. `internal/apply` and
// `internal/seen` are two of the six engine directories every ADR-010 and
// ADR-011 fence asserts byte-identical, and that boundary is permanent; a
// `desc:` tag is inert at runtime and still changes those bytes. ADR-011's
// boundary table already gives this package the tool descriptors, and prose
// about a returned field is a descriptor.
//
// Keys are dotted property paths, so a field one level down is nameable:
// `hunks.status`, not `status`. Coverage is enforced in both directions — an
// undescribed property fails TestEveryOutputSchemaPropertyIsDescribed, and an
// entry here naming a property the schema no longer declares is refused at
// construction, which is the quieter of the two drifts.
var readDescriptions = map[string]string{
	"observed":       "What THIS call observed of each served file, keyed by path. It is merged into the per-checkout ledger rather than replacing it, so a later write is authorised by the accumulated spans for the same sha — not by this response alone.",
	"observed.SHA":   "The sha256 of the whole file as it was when served. A later write is refused if the file no longer hashes to this.",
	"observed.Spans": "The line spans this call rendered, as [start, end] pairs; null means the whole file. Authorisation is per LINE: a write to a line no read has served is refused, though a line served by an EARLIER read of the same sha is still licensed.",
	"problems":       "How many requested ranges could not be served. Non-zero means part of what you asked for is missing from `observed` — the call itself still answered.",
}

var writeDescriptions = map[string]string{
	"root":               "The checkout the plan was applied in. Every path in the plan is relative to it.",
	"dry_run":            "True when the plan was only validated. Every other field means what it would have meant, and nothing was written.",
	"applied":            "True when every hunk passed and the new content reached disk. False on a dry run and on any refusal.",
	"failed":             "How many hunks FAILED. A hunk that was valid but abandoned because a sibling failed is `skipped` and is NOT counted here, so this is not the number of hunks that did not apply — read `hunks[].status` for that. Non-zero means NOTHING was written: a plan is all or nothing.",
	"files":              "One entry per file the plan addressed, including files it could not validate. If the run died on an I/O error partway through, files already written are named and the rest may be missing.",
	"files.path":         "The file's path, relative to root.",
	"files.created":      "True when the file did not exist and a create hunk made it.",
	"files.written":      "True when this file's new content reached disk. False on a dry run, and false for every file when any hunk failed.",
	"files.sha_before":   "The sha256 of the file before the plan was applied.",
	"files.sha_after":    "The sha256 the file WOULD have after this plan. Computed before the write, so on a dry run or a failed plan it describes proposed content that is not on disk — `written` says which.",
	"files.lines_before": "How many lines the file held before the plan was applied.",
	"files.lines_after":  "How many lines the file would hold after this plan. Proposed, on the same terms as `sha_after`.",
	"hunks":              "One verdict per hunk, in plan order. This is the field to read: a replacement that matched nothing is reported here rather than silently skipped.",
	"hunks.path":         "The file this hunk addressed, as written in the plan.",
	"hunks.addr":         "The address as written in the plan, so a verdict can be matched back to the plan line that produced it.",
	"hunks.op":           "The op as written: replace, insert-after, insert-before, delete or create.",
	// ⚠ THESE THREE ARE THE WIRE VALUES, verbatim from apply.Status. An earlier
	// draft of this line taught `fail` and `skip`, which the engine never
	// sends: a host filtering hunks[].status == "fail" would have seen a clean
	// run through every failure. Caught in review of PR #72 and pinned by
	// TestTheStatusDescriptionNamesTheValuesTheEngineSends.
	"hunks.status":        "`ok` when this hunk applied, `failed` when it did not, `skipped` when a sibling failed and the whole plan was abandoned. A skipped hunk is never an applied one.",
	"hunks.reason":        "Why a failing hunk failed: a guard that did not hold, a file mrw had not served, a path outside the root. Absent when the hunk passed.",
	"hunks.removed":       "How many lines this hunk removes. Computed during validation, so on a dry run or a failed plan it is a proposed delta.",
	"hunks.added":         "How many lines this hunk adds, on the same terms as `removed`.",
	"hunks.plan_line":     "The line of the plan document this hunk's header was on.",
	"hunks.removed_first": "The first line a delete removes, trimmed for display, so a caller can see what it is losing. Present whenever a delete reaches `ok` — a dry run included, where nothing was actually removed.",
	"hunks.removed_last":  "The last line a delete removes, on the same terms as `removed_first`.",
}

// describeResult attaches the table's prose to a generated schema, in place,
// and REFUSES a table entry that matches no declared property.
//
// The refusal is the point. A missing description is loud — the coverage test
// names the property. A stale one is silent: the schema still validates every
// response, and the entry sits there describing a field nobody sends until
// somebody reads it and believes it.
func describeResult(schema map[string]any, table map[string]string) (map[string]any, error) {
	used := map[string]bool{}
	attachDescriptions(schema, "", table, used)
	var stale []string
	for k := range table {
		if !used[k] {
			stale = append(stale, k)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		return nil, fmt.Errorf("mcp: described propert(ies) the schema does not declare: %s", strings.Join(stale, ", "))
	}
	return schema, nil
}

// attachDescriptions walks properties, array items and map values alike, so a
// field one level down is reachable by a dotted path. Items and additional
// properties share their parent's prefix because neither is separately named
// on the wire — an element of `hunks` is a hunk, not a `hunks[]`.
func attachDescriptions(schema map[string]any, prefix string, table map[string]string, used map[string]bool) {
	props, _ := schema["properties"].(map[string]any)
	for name, raw := range props {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		path := prefix + name
		if d, ok := table[path]; ok {
			p["description"] = d
			used[path] = true
		}
		attachDescriptions(p, path+".", table, used)
		if items, ok := p["items"].(map[string]any); ok {
			attachDescriptions(items, path+".", table, used)
		}
		if ap, ok := p["additionalProperties"].(map[string]any); ok {
			attachDescriptions(ap, path+".", table, used)
		}
	}
}

// mustDescribe is the panicking form, for the same reason mustSchema is: the
// tables and the types are compile-time constants of this package, so a
// mismatch is a programming mistake every startup and every test hits at once,
// not a runtime condition a caller could act on.
func mustDescribe(schema map[string]any, table map[string]string) map[string]any {
	s, err := describeResult(schema, table)
	if err != nil {
		panic(err.Error())
	}
	return s
}

func mustSchema(v any) map[string]any {
	s, err := SchemaOf(v)
	if err != nil {
		panic("mcp: " + err.Error())
	}
	return s
}
