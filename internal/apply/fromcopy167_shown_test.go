package apply

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
)

// ADR-167, Zy 2026-10-22: a from= source line must have been SHOWN to the
// caller, not merely licensed (ADR-118's rule for occurrence=). A file mrw
// wrote, a --sub plan's lines and a --quiet read license an edit and show
// nothing.
func TestAFromCopyNeedsTheLinesShownNotMerelyLicensed(t *testing.T) {
	newTree := func(t *testing.T) string {
		root := t.TempDir()
		write(t, root, "a.txt", srcText167)
		write(t, root, "b.txt", "x\ny\n")
		return root
	}
	ledger := func(t *testing.T, root string, a Seen) map[string]Seen {
		a.SHA = shaOfFile(t, root, "a.txt")
		return map[string]Seen{"a.txt": a, "b.txt": {SHA: shaOfFile(t, root, "b.txt")}}
	}
	in := func() []Input { return []Input{copyInput167("b.txt", "insert-after", 1, "a.txt", 3, 5)} }
	for _, c := range []struct {
		name string
		obs  Seen
		ok   bool
	}{
		{"written, nothing shown since", Seen{Written: true}, false},
		{"licensed by a --sub plan, not shown", Seen{Written: true, Spans: [][2]int{{3, 5}}, Shown: [][2]int{}}, false},
		{"written, line 5 not shown", Seen{Written: true, Shown: [][2]int{{3, 4}}}, false},
		{"written, lines 3-5 shown since", Seen{Written: true, Shown: [][2]int{{3, 5}}}, true},
		{"read whole", Seen{}, true},
		{"read 3-5", Seen{Spans: [][2]int{{3, 5}}}, true},
	} {
		for _, force := range []bool{false, true} {
			root := newTree(t)
			res, err := Apply(root, in(), Options{Seen: ledger(t, root, c.obs), Force: force})
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			landed := res.Failed == 0 && read(t, root, "b.txt") == "x\nsecret-3\nsecret-4\nsecret-5\ny\n"
			if landed != c.ok {
				t.Errorf("%s force=%v: landed=%v, want %v: %+v", c.name, force, landed, c.ok, res.Hunks)
			}
			if !c.ok && (res.Hunks[0].Kind != refusal.SourceNotRead || read(t, root, "b.txt") != "x\ny\n") {
				t.Errorf("%s force=%v: want a SourceNotRead refusal and nothing written: %+v", c.name, force, res.Hunks[0])
			}
		}
	}
}

// ADR-076: a source spelled as a directory is refused as written, however the
// clean would fold it.
func TestAFromSourceSpelledAsADirectoryIsRefused(t *testing.T) {
	for _, spelling := range []string{"a.txt/", "a.txt/."} {
		root := t.TempDir()
		write(t, root, "a.txt", srcText167)
		write(t, root, "b.txt", "x\ny\n")
		l := ledgerOf167(t, root, map[string][][2]int{"a.txt": nil, "b.txt": nil})
		res, err := Apply(root, []Input{copyInput167("b.txt", "insert-after", 1, spelling, 3, 5)}, Options{Seen: l})
		if err != nil || res.Failed != 1 || read(t, root, "b.txt") != "x\ny\n" {
			t.Errorf("%q: err=%v failed=%d %+v", spelling, err, res.Failed, res.Hunks)
		}
	}
}

// ADR-167 Decision 4: a refused copy discloses no copied line. That holds when
// the SOURCE was fine and the TARGET failed, and for a skipped sibling; body_from
// stays, with the resolved span.
func TestARefusedCopyDisclosesNoLineWhateverRefusedIt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", srcText167)
	write(t, root, "exists.txt", "already\n")
	l := ledgerOf167(t, root, map[string][][2]int{"a.txt": nil, "exists.txt": nil})
	res, err := Apply(root, []Input{
		{Path: "exists.txt", Op: "create", Lines: -1, From: "a.txt", FromStart: 3, FromEnd: 5}, // the target refuses
		{Path: "c.txt", Op: "create", Lines: -1, From: "a.txt", FromStart: 1, FromEnd: 2},      // skipped
	}, Options{Seen: l})
	if err != nil || res.Failed != 1 {
		t.Fatalf("err=%v failed=%d %+v", err, res.Failed, res.Hunks)
	}
	for i, h := range res.Hunks {
		if h.CopiedFirst != "" || h.CopiedLast != "" {
			t.Errorf("hunk %d (%s) discloses %q..%q", i, h.Status, h.CopiedFirst, h.CopiedLast)
		}
		b, _ := json.Marshal(h)
		for _, k := range []string{`"body_from":"a.txt:`, `"copied_first":""`, `"copied_last":""`} {
			if !strings.Contains(string(b), k) {
				t.Errorf("hunk %d (%s): receipt lacks %s: %s", i, h.Status, k, b)
			}
		}
	}
}

// A refused copy's body_from is the resolved span when the address resolved, so
// `$` and `A,+N` read as the lines they are; one that did not resolve keeps the
// address as written.
func TestARefusedCopyNamesTheResolvedSpan(t *testing.T) {
	for _, c := range []struct {
		start, end, rel int
		want            string
	}{
		{EOF, EOF, 0, "a.txt:6"},
		{2, 2, 2, "a.txt:2-4"},
		{5, EOF, 0, "a.txt:5-6"},
		{5, 9, 0, "a.txt:5-9"}, // past the end: not resolved, as written
	} {
		root := t.TempDir()
		write(t, root, "a.txt", srcText167)
		write(t, root, "b.txt", "x\ny\n")
		l := ledgerOf167(t, root, map[string][][2]int{"a.txt": {{1, 1}}, "b.txt": nil}) // nothing wanted is served
		in := Input{Path: "b.txt", Op: "insert-after", Start: 1, End: 1, Lines: -1, From: "a.txt", FromStart: c.start, FromEnd: c.end, FromRelEnd: c.rel}
		res, err := Apply(root, []Input{in}, Options{Seen: l})
		if err != nil || res.Failed != 1 || res.Hunks[0].BodyFrom != c.want {
			t.Errorf("%+v: err=%v body_from=%q, want %q (%+v)", c, err, res.Hunks[0].BodyFrom, c.want, res.Hunks[0])
		}
	}
}
