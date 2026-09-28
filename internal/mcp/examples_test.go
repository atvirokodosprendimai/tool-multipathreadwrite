package mcp

import (
	"testing"
)

// TestTheShippedReadExampleServesEverySpec is ADR-090's read half.
//
// The spec list mrw_read publishes as its worked example was checked only for
// its SHAPE — a list of more than one item — so a spec whose regexp matched
// nothing, or whose range named a file no tree has, shipped green. It is sent
// here as one call, the way a caller copies it, to the tree the examples are
// written against, and every spec must be served.
func TestTheShippedReadExampleServesEverySpec(t *testing.T) {
	var specs []any
	for _, tl := range tools() {
		if tl.Name != "mrw_read" {
			continue
		}
		props, _ := tl.InputSchema.(map[string]any)["properties"].(map[string]any)
		p, _ := props["specs"].(map[string]any)
		examples, _ := p["examples"].([]any)
		if len(examples) == 0 {
			t.Fatal("mrw_read publishes no worked spec list")
		}
		for _, s := range stringExamples(t, "mrw_read.specs.examples[0]", examples[0]) {
			specs = append(specs, s)
		}
	}
	if len(specs) < 2 {
		t.Fatalf("the worked spec list has %d spec(s); it exists to show several address forms in one call", len(specs))
	}
	root := exampleTree(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	res := call(t, root, "mrw_read", map[string]any{"specs": specs})
	got := receipt(t, res)
	n, ok := got["problems"].(float64)
	if !ok {
		t.Fatalf("the read receipt carries no numeric problems field: %v", got)
	}
	if n != 0 {
		t.Errorf("the shipped read example reported %v problem(s) on the tree it is written against:\n%s", n, served0(t, res))
	}
	// The keys are compared as they come: ADR-091 spells them with "/" on every
	// platform, and on Windows this is the check that the receipt builders
	// really call slashKeys (windows-shard 1, #261).
	observed, ok := got["observed"].(map[string]any)
	if !ok {
		t.Fatalf("the read receipt carries no observed map: %v", got)
	}
	for _, want := range []string{"internal/store/store.go", "cmd/app/main.go"} {
		if _, ok := observed[want]; !ok {
			t.Errorf("the shipped read example served nothing from %s; observed %v", want, observed)
		}
	}
}

// TestTheDescriptionWalkNamesItsSentinels is ADR-090's T2.
//
// TestEveryOutputSchemaPropertyIsDescribed holds every property it finds to a
// description, but "finds" is guarded only by a floor: the walk counted 27 and
// the floor is 20, so losing a whole container of described fields passed.
// These paths are required by name, one from each container a caller reads —
// a hunk's verdict, a file's landing, the recent-window pattern, and the read
// receipt's served spans.
func TestTheDescriptionWalkNamesItsSentinels(t *testing.T) {
	shapes := map[string]map[string]any{"mrw_read receipt": readSchema()}
	for _, tl := range tools() {
		if schema, ok := tl.OutputSchema.(map[string]any); ok {
			shapes[tl.Name] = schema
		}
	}
	walked := map[string]bool{}
	for name, schema := range shapes {
		for _, p := range describedPaths(t, name, schema, "") {
			walked[name+":"+p] = true
		}
	}
	for _, want := range []string{
		"mrw_write:hunks.status",
		"mrw_write:files.written",
		"mrw_write:pattern.fires",
		"mrw_read receipt:observed.Spans",
	} {
		if !walked[want] {
			t.Errorf("the description walk never reached %s; a container a caller reads went undescribed or away", want)
		}
	}
}
