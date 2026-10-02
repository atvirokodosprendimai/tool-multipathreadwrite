package plan_test

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
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

// TestEveryBuilderCarriesOccurrence: each place that turns a plan.Hunk into an
// apply.Input copies the address field by field, so a new field is dropped
// silently by any builder that forgets it. Every line that copies RelEnd must
// copy Occurrence too.
func TestEveryBuilderCarriesOccurrence(t *testing.T) {
	copies := regexp.MustCompile(`RelEnd: *(h\.Addr|i)\.RelEnd`)
	for _, f := range []string{"cmd/mrw/main.go", "internal/mcp/tools.go", "internal/curve/score.go", "internal/apply/apply.go"} {
		b, err := os.ReadFile(filepath.Join("..", "..", f))
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, line := range strings.Split(string(b), "\n") {
			if !copies.MatchString(line) {
				continue
			}
			n++
			if !strings.Contains(line, "Occurrence: h.Occurrence") && !strings.Contains(line, "Occurrence: i.Occurrence") {
				t.Errorf("%s: a builder copies RelEnd and not Occurrence: %s", f, strings.TrimSpace(line))
			}
		}
		if n == 0 {
			t.Errorf("%s: no builder line found; the scan would hold nothing", f)
		}
	}
}
