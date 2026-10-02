package adversarial

import (
	"errors"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-087. The parser and the engine refuse the same mistakes (ADR-030), and
// TestTheEngineAndTheParserRefuseInTheSameWords keeps their TEXT equal. The
// kinds make the pairing structural: each mirrored rule carries one kind on
// both sides, so rewording a message cannot silently change what a caller
// that classifies refusals sees. The engine's not-read refusal carries its
// kind too, which the MCP surface keys its acknowledgement remedy on.
func TestTheEngineAndTheParserRefuseWithOneKind(t *testing.T) {
	cases := []struct {
		doc  string
		in   apply.Input
		kind refusal.Kind
	}{
		{"@@ f.txt 1-2 replace\n", apply.Input{Path: "f.txt", Op: "replace", Start: 1, End: 2, Lines: unset}, refusal.ReplaceEmptyBody},
		{"@@ f.txt 1-3 insert-after\nX\n", apply.Input{Path: "f.txt", Op: "insert-after", Start: 1, End: 3, Body: []string{"X"}, Lines: unset}, refusal.InsertRange},
		{"@@ n.txt 1 create\nX\n", apply.Input{Path: "n.txt", Op: "create", Start: 1, End: 1, Body: []string{"X"}, Lines: unset}, refusal.CreateAddress},
		{"@@ n.txt - create anchor=\"zzz\"\nX\n", apply.Input{Path: "n.txt", Op: "create", Body: []string{"X"}, Lines: unset, Anchor: "zzz"}, refusal.CreateGuard},
		{"@@ n.txt - create\n", apply.Input{Path: "n.txt", Op: "create", Lines: unset}, refusal.CreateEmptyBody},
		{"@@ f.txt 1 insert-after\n", apply.Input{Path: "f.txt", Op: "insert-after", Start: 1, End: 1, Lines: unset}, refusal.InsertEmptyBody},
		{"@@ n.txt 00,+2 create\nX\n", apply.Input{Path: "n.txt", Op: "create", RelEnd: 2, Body: []string{"X"}, Lines: unset}, refusal.CreateRelEnd},
		{"@@ f.txt 3 replace occurrence=2\nX\n", apply.Input{Path: "f.txt", Op: "replace", Start: 3, End: 3, Body: []string{"X"}, Lines: unset, Occurrence: 2}, refusal.OccurrenceAddress},
	}
	for _, c := range cases {
		t.Run(string(c.kind), func(t *testing.T) {
			_, err := plan.Parse(strings.NewReader(c.doc))
			var pe *plan.ParseError
			if !errors.As(err, &pe) || len(pe.Kinds) != 1 || pe.Kinds[0] != c.kind {
				t.Errorf("parser: %v, want one refusal of kind %q", err, c.kind)
			}
			root := tree(t, map[string]string{"f.txt": "a\nb\nc\nd\n"})
			res, aerr := apply.Apply(root, []apply.Input{c.in}, apply.Options{Force: true})
			if aerr != nil {
				t.Fatal(aerr)
			}
			if res.Failed != 1 || res.Hunks[0].Kind != c.kind {
				t.Errorf("engine: failed=%d kind=%q, want 1 of kind %q", res.Failed, res.Hunks[0].Kind, c.kind)
			}
		})
	}
	t.Run(string(refusal.NotRead), func(t *testing.T) {
		root := tree(t, map[string]string{"f.txt": "a\n"})
		res, err := apply.Apply(root, []apply.Input{{Path: "f.txt", Op: "replace", Start: 1, End: 1, Body: []string{"X"}, Lines: unset}}, apply.Options{Seen: seen.Ledger{}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Failed != 1 || res.Hunks[0].Kind != refusal.NotRead {
			t.Errorf("an unread file: failed=%d kind=%q, want %q", res.Failed, res.Hunks[0].Kind, refusal.NotRead)
		}
	})
}

// Kinds lines up with the errors the text reports, so a caller can pair them:
// an error the scan finds before validation has no kind, and must still take
// its place (the Codex review of #257).
func TestParseErrorKindsLineUpWithTheErrors(t *testing.T) {
	_, err := plan.Parse(strings.NewReader("stray\n@@ f.txt 1 replace\n"))
	var pe *plan.ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want a *plan.ParseError, got %v", err)
	}
	if n := strings.Count(err.Error(), "\n  line "); n != 2 {
		t.Fatalf("the fixture must report two errors, reported %d: %v", n, err)
	}
	want := []refusal.Kind{"", refusal.ReplaceEmptyBody}
	if len(pe.Kinds) != len(want) || pe.Kinds[0] != want[0] || pe.Kinds[1] != want[1] {
		t.Errorf("Kinds = %q, want %q", pe.Kinds, want)
	}
}
