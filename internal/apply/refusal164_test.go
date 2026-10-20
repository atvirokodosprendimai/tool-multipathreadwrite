package apply

import (
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
)

// ADR-164 T6. The not-read refusal says a line --max-cols served cut is not
// read: a peer was told line 43 "has not been read" when it had been served cut.
// The ledger does not record cut lines, so the clause is on every such refusal.
func TestANotReadRefusalNamesACutLine(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)
	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{"f.txt": {SHA: shaOfFile(t, root, "f.txt"), Spans: [][2]int{{3, 3}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("an unserved line was written: %+v", res.Hunks)
	}
	h := res.Hunks[0]
	for _, want := range []string{"has not been read", "a line --max-cols served cut is not read: read it whole, or --quiet"} {
		if !strings.Contains(h.Reason, want) {
			t.Errorf("reason lacks %q: %s", want, h.Reason)
		}
	}
	if h.Kind != refusal.NotRead {
		t.Errorf("kind %v, want refusal.NotRead (ADR-087)", h.Kind)
	}
}

// ADR-164 T6. An addressed create names both spellings it takes, from the parser
// and from the engine: the refusal said `use "-"` while `0 create`, which
// AGENTS.md teaches, also applies.
func TestACreateAddressNamesBothSpellings(t *testing.T) {
	const want = `use "-" or "0"`
	if _, err := plan.Parse(strings.NewReader("@@ n.txt 1 create\nhi\n")); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("the parser's create refusal: %v, want %q", err, want)
	}
	root := t.TempDir()
	res, err := Apply(root, []Input{
		{Path: "n.txt", Start: 1, End: 1, Op: "create", Body: []string{"hi"}, Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, want) {
		t.Errorf("the engine's create refusal: %+v, want %q", res.Hunks, want)
	}
	for _, doc := range []string{"@@ n.txt 0 create\nhi\n", "@@ n.txt - create\nhi\n"} {
		if _, err := plan.Parse(strings.NewReader(doc)); err != nil {
			t.Errorf("%q: %v", doc, err)
		}
	}
}
