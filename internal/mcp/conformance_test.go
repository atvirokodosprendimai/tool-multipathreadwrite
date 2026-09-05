package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
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

// TestEveryEmbeddedExamplePlanReallyApplies is ADR-012's Enforced-by.
//
// Every plan this server SHIPS is parsed and then dry-run applied against a
// tree built from the plan itself, through the same two tools a caller drives.
// An example asserted merely to be PRESENT stays green long after it has
// stopped being valid — and the example is the one thing a caller copies
// verbatim, so a stale one is worse than none.
func TestEveryEmbeddedExamplePlanReallyApplies(t *testing.T) {
	shipped := shippedPlans(t)
	if len(shipped) == 0 {
		t.Fatal("no example plan ships on the wire, so this test would pass vacuously")
	}
	for where, text := range shipped {
		t.Run(where, func(t *testing.T) { dryRunExample(t, text) })
	}
}

// shippedPlans collects every plan text a host can actually receive, read off
// the descriptor set rather than off the constants — the wire is what a caller
// copies, and an example that never reached `examples` is not shipped.
func shippedPlans(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{"examplePlan": examplePlan}
	for _, tl := range tools() {
		schema, ok := tl.InputSchema.(map[string]any)
		if !ok {
			t.Fatalf("tool %q has no object inputSchema", tl.Name)
		}
		props, _ := schema["properties"].(map[string]any)
		p, ok := props["plan"].(map[string]any)
		if !ok {
			continue
		}
		// Accept either JSON-ish shape. `plan` publishes []string and `specs`
		// publishes []any; a type assertion for one of them silently SKIPS an
		// example published as the other, which would make this whole test
		// vacuous without failing. Noted in review of PR #72.
		examples := stringExamples(t, tl.Name, p["examples"])
		if len(examples) == 0 {
			t.Errorf("tool %q declares a plan property with no examples; the format is bespoke and this is the only worked one a host sees", tl.Name)
			continue
		}
		for i, ex := range examples {
			out[fmt.Sprintf("%s.plan.examples[%d]", tl.Name, i)] = ex
		}
	}
	return out
}

// stringExamples reads a JSON Schema `examples` array whose entries are
// strings, accepting both the []string a Go literal produces and the []any a
// decoded JSON document produces. It FAILS on an unexpected shape rather than
// returning nothing: a helper that answers "no examples" for a shape it does
// not recognise makes its caller vacuous without making it red.
func stringExamples(t *testing.T, where string, raw any) []string {
	t.Helper()
	switch v := raw.(type) {
	case nil:
		return nil
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for i, el := range v {
			s, ok := el.(string)
			if !ok {
				t.Fatalf("%s: examples[%d] is %T, want a string", where, i, el)
			}
			out = append(out, s)
		}
		return out
	default:
		t.Fatalf("%s: examples is %T, want an array of strings", where, raw)
		return nil
	}
}

// TestTheStatusDescriptionNamesTheValuesTheEngineSends pins the prose to the
// constants.
//
// T2 enforced that every property is described and that no description names a
// property the schema dropped. Neither catches a description that names the
// right property and the WRONG VALUES — which is what shipped: `hunks.status`
// was documented as `fail`/`skip` while the engine sends `failed`/`skipped`, so
// a host filtering on the documented value would have read every failing run as
// clean. Caught in review of PR #72, on the field the record itself calls the
// most load-bearing in the receipt.
func TestTheStatusDescriptionNamesTheValuesTheEngineSends(t *testing.T) {
	schema, ok := writeSchema()["properties"].(map[string]any)
	if !ok {
		t.Fatal("the write schema declares no properties")
	}
	hunks, _ := schema["hunks"].(map[string]any)
	items, _ := hunks["items"].(map[string]any)
	props, _ := items["properties"].(map[string]any)
	status, _ := props["status"].(map[string]any)
	got, _ := status["description"].(string)
	if got == "" {
		t.Fatal("hunks.status carries no description")
	}
	for _, want := range []apply.Status{apply.StatusOK, apply.StatusFailed, apply.StatusSkipped} {
		if !strings.Contains(got, string(want)) {
			t.Errorf("the hunks.status description never names %q, which is a value the engine really sends:\n%s", want, got)
		}
	}
	// The near-miss forms, and the reason this test is not just a "contains"
	// check: "failed" contains "fail", so naming the wrong value is only
	// detectable by looking for it as a WHOLE word.
	for _, wrong := range []string{"fail", "skip"} {
		if regexp.MustCompile(`\b` + wrong + `\b`).MatchString(got) {
			t.Errorf("the hunks.status description names %q as a status; the engine sends %q-style values and a host filtering on %q sees no failure:\n%s", wrong, apply.StatusFailed, wrong, got)
		}
	}
}

// dryRunExample proves one shipped plan against the real engine: parse it,
// build the tree it addresses, read that tree through mrw_read so the ledger
// licenses the write, then dry-run it and demand every hunk pass.
func dryRunExample(t *testing.T, text string) {
	t.Helper()
	hunks, err := plan.Parse(strings.NewReader(text))
	if err != nil {
		t.Fatalf("a shipped example does not parse: %v\n%s", err, text)
	}
	if len(hunks) == 0 {
		t.Fatalf("a shipped example parsed to no hunks:\n%s", text)
	}
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	specs := treeFor(t, root, hunks)
	call(t, root, "mrw_read", map[string]any{"specs": specs})
	got := structured(t, call(t, root, "mrw_write", map[string]any{"plan": text, "dry_run": true}))
	if n, _ := got["failed"].(float64); n != 0 {
		t.Errorf("the shipped example failed %v hunk(s) on a real dry run: %v", n, got["hunks"])
	}
}

// treeFor builds the files a plan addresses, deriving each file's length from
// the highest line the plan names and planting any `anchor=` guard on the line
// it guards. The tree comes from the plan so that a plan naming a new path is
// still exercised, rather than the test quietly skipping what it cannot find.
//
// ⚠ WHAT THIS DOES NOT PROVE. The anchor is planted FROM the plan, so an
// `anchor=` guard here can never fail — measured as a surviving mutant on
// 2026-09-04, when changing the example's anchor text left this test green.
// That is the honest limit of an example naming files no repository has: the
// test proves the plan is grammatical and applies, not that its guard matches
// real code. The mutants that DO die are the ones that matter for an example a
// caller copies — an address form the plan grammar rejects, a header that does
// not parse.
func treeFor(t *testing.T, root string, hunks []plan.Hunk) []any {
	t.Helper()
	last := map[string]int{}
	anchors := map[string]map[int]string{}
	for _, h := range hunks {
		if h.Op == plan.OpCreate {
			continue
		}
		for _, n := range []int{h.Addr.Start, h.Addr.End} {
			if n > last[h.Path] {
				last[h.Path] = n
			}
		}
		if h.Anchor != "" {
			if anchors[h.Path] == nil {
				anchors[h.Path] = map[int]string{}
			}
			anchors[h.Path][h.Addr.Start] = h.Anchor
		}
	}
	var specs []any
	for p, n := range last {
		body := make([]string, max(n, 1))
		for i := range body {
			if a, ok := anchors[p][i+1]; ok {
				body[i] = a
			} else {
				body[i] = fmt.Sprintf("line %d", i+1)
			}
		}
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(strings.Join(body, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		specs = append(specs, p)
	}
	return specs
}

// TestEveryOutputSchemaPropertyIsDescribed holds the machine-readable half of
// the contract to the same standard as the prose half.
//
// ADR-011 made the shapes generated from the Go types, which is right and left
// them silent: a caller was told that `failed` is an integer and never what it
// counts. Coverage is total and at every depth, because the interesting fields
// are the ones one level down — a verdict's `status`, a file's `written`.
func TestEveryOutputSchemaPropertyIsDescribed(t *testing.T) {
	total := 0
	for _, tl := range tools() {
		schema, ok := tl.OutputSchema.(map[string]any)
		if !ok {
			t.Fatalf("tool %q declares no object outputSchema", tl.Name)
		}
		paths := describedPaths(t, tl.Name, schema, "")
		if len(paths) == 0 {
			t.Errorf("tool %q: the walk found no properties, so this test would pass vacuously", tl.Name)
		}
		total += len(paths)
	}
	// The two results carry more than twenty fields between them. A walk that
	// found only the top level would pass every assertion above.
	if total < 20 {
		t.Errorf("the walk found %d properties across both tools; the declared shapes carry more than that, so it is not descending", total)
	}
}

// describedPaths walks a schema, asserting a description on every property it
// finds, and returns the paths it checked. It is written here rather than
// reused from the production walker on purpose: a coverage test that shares its
// traversal with the code it checks agrees with itself by construction.
func describedPaths(t *testing.T, where string, schema map[string]any, prefix string) []string {
	t.Helper()
	var found []string
	props, _ := schema["properties"].(map[string]any)
	for name, raw := range props {
		p, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("%s: property %q is not an object", where, prefix+name)
			continue
		}
		path := prefix + name
		if d, _ := p["description"].(string); strings.TrimSpace(d) == "" {
			t.Errorf("%s: property %q has no description; a caller is told its type and not what it means", where, path)
		}
		found = append(found, path)
		found = append(found, describedPaths(t, where, p, path+".")...)
		if items, ok := p["items"].(map[string]any); ok {
			found = append(found, describedPaths(t, where, items, path+".")...)
		}
		if ap, ok := p["additionalProperties"].(map[string]any); ok {
			found = append(found, describedPaths(t, where, ap, path+".")...)
		}
	}
	return found
}

// receipt parses the serialized receipt out of content[1]. For mrw_read this
// is the ONLY machine-readable copy: ADR-023 removed structuredContent from
// every read result because a host that renders structuredContent in place of
// content showed the model the receipt and none of the served lines.
func receipt(t *testing.T, res map[string]any) map[string]any {
	t.Helper()
	content, _ := res["content"].([]any)
	if len(content) < 2 {
		t.Fatalf("want two content blocks (the answer, then the receipt), got %v", res["content"])
	}
	text, _ := content[1].(map[string]any)["text"].(string)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("content[1] is not the serialized receipt: %v\n%q", err, text)
	}
	return out
}

// TestAReadResultCarriesNoStructuredContent is ADR-023's Enforced-by.
//
// Measured 2026-09-05 on Claude Code 2.1.261: a successful tool result that
// carries structuredContent reaches the model AS the structuredContent — the
// content blocks are dropped. For mrw_write that is the verdict and fine; for
// mrw_read it is the receipt without the lines, while the ledger has already
// recorded those lines as seen. So no mrw_read result — served, paged, index —
// may carry structuredContent, and the receipt travels in content[1] alone.
// The write tool is asserted in the same test so the two cannot drift apart
// unnoticed: one keeps the envelope, one does not, on purpose.
func TestAReadResultCarriesNoStructuredContent(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\ntwo\n")
	served := call(t, root, "mrw_read", map[string]any{"specs": []any{path}})

	bigRoot, bigPath := bigCheckout(t, 12000)
	paged := call(t, bigRoot, "mrw_read", map[string]any{"specs": []any{bigPath}})

	idxRoot := grepTree(t, 60, 400)
	index := call(t, idxRoot, "mrw_read", map[string]any{"grep": "NEEDLE"})

	for name, res := range map[string]map[string]any{"served": served, "paged": paged, "index": index} {
		if _, has := res["structuredContent"]; has {
			t.Errorf("%s read result carries structuredContent; a host that renders it in place of content hides the served text", name)
		}
		if _, ok := receipt(t, res)["observed"]; !ok {
			t.Errorf("%s read result's content[1] carries no observed field; the receipt must still travel", name)
		}
	}
	if paged["isError"] != true {
		t.Fatal("the oversized read did not page; this fixture exists to produce a page")
	}
	if next, _ := receipt(t, paged)["next_read"].(string); next == "" {
		t.Error("a page's content[1] names no next_read; the continuation moved out of structuredContent and must still be findable")
	}
	if idx, _ := receipt(t, index)["index"].([]any); len(idx) == 0 {
		t.Error("an index's content[1] carries no index entries")
	}

	// mrw_write keeps its envelope, equal to its own content[1].
	w := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nONE\n"})
	sc, has := w["structuredContent"]
	if !has {
		t.Fatal("mrw_write lost its structuredContent; its answer IS the receipt and the measured host shows exactly that")
	}
	want, _ := json.Marshal(sc)
	got, _ := json.Marshal(receipt(t, w))
	if string(want) != string(got) {
		t.Errorf("mrw_write's content[1] and structuredContent disagree:\n got %s\nwant %s", got, want)
	}
}
