package plan_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
)

// ADR-118. occurrence= picks among a PATTERN's matches, so off a pattern it
// means nothing, and the parser and the engine refuse it with one kind
// (ADR-030, ADR-087).
func TestAnOccurrenceOnALineAddressIsRefusedByParserAndEngine(t *testing.T) {
	for _, doc := range []string{
		"@@ x.go 3 replace occurrence=2\nX\n",
		"@@ x.go - create occurrence=1\nX\n",
		"@@ x.go - unlink occurrence=1\n",
	} {
		_, err := plan.Parse(strings.NewReader(doc))
		var pe *plan.ParseError
		if !errors.As(err, &pe) || len(pe.Kinds) != 1 || pe.Kinds[0] != refusal.OccurrenceAddress {
			t.Errorf("parser, %q: %v, want one refusal of kind %q", doc, err, refusal.OccurrenceAddress)
		}
	}
	h, err := plan.Parse(strings.NewReader("@@ x.go /^func X/ replace occurrence=2\nX\n"))
	if err != nil || len(h) != 1 || h[0].Occurrence != 2 {
		t.Fatalf("occurrence= on a pattern: %v %+v", err, h)
	}
	for _, bad := range []string{"0", "-1", "two", "+2", "02", ""} {
		if _, err := plan.Parse(strings.NewReader("@@ x.go /^func X/ replace occurrence=" + bad + "\nX\n")); err == nil {
			t.Errorf("occurrence=%s parsed", bad)
		}
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x.go"), []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := apply.Apply(root, []apply.Input{{Path: "x.go", Op: "replace", Start: 2, End: 2, Body: []string{"X"}, Lines: -1, Occurrence: 2}}, apply.Options{Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 || res.Hunks[0].Kind != refusal.OccurrenceAddress {
		t.Errorf("engine: failed=%d kind=%q, want %q", res.Failed, res.Hunks[0].Kind, refusal.OccurrenceAddress)
	}
	// The mirror is checked first, so a create or an unlink on an unread file
	// is refused for occurrence= and not for what a later check finds.
	for _, in := range []apply.Input{
		{Path: "n.txt", Op: "create", Body: []string{"X"}, Lines: -1, Occurrence: 1},
		{Path: "x.go", Op: "unlink", Lines: -1, Occurrence: 1},
	} {
		res, err := apply.Apply(root, []apply.Input{in}, apply.Options{Seen: map[string]apply.Seen{}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Failed != 1 || res.Hunks[0].Kind != refusal.OccurrenceAddress {
			t.Errorf("engine, %s: failed=%d kind=%q, want %q", in.Op, res.Failed, res.Hunks[0].Kind, refusal.OccurrenceAddress)
		}
	}
}

// TestEveryBuilderCarriesOccurrence guards the route from each surface through
// the shared conversion and into the engine's resolved hunk.
func TestEveryBuilderCarriesOccurrence(t *testing.T) {
	for _, step := range []struct{ path, want string }{
		{"cmd/mrw/main.go", "in = append(in, writer.InputFromHunk(h, path))"},
		{"internal/mcp/tools.go", "in = append(in, writer.InputFromHunk(h, path))"},
		{"internal/curve/score.go", "in = append(in, writer.InputFromHunk(h, h.Path))"},
		{"internal/writer/flow.go", "Occurrence: h.Occurrence"},
		{"internal/apply/apply.go", "Occurrence: i.Occurrence"},
	} {
		b, err := os.ReadFile(filepath.Join("..", "..", step.path))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), step.want) {
			t.Errorf("%s: occurrence route missing %q", step.path, step.want)
		}
	}
}
