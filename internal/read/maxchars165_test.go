package read

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// ADR-165 T2. A shell host cuts mrw's answer at its own output limit, where it
// likes, and lines mrw licensed never reach the agent. --max-chars N is a
// budget on the whole answer: at most N characters on standard output, served
// in whole lines, and every range and file not served named on the withheld
// writer — standard error on the CLI — which the budget does not charge, so
// the report of what was cut cannot itself be cut. Only printed lines are
// licensed, and a file the budget never reached is not opened to be named.
func TestAReadStopsAtItsCharacterBudgetAndNamesWhatItWithheld(t *testing.T) {
	root := t.TempDir()
	// DISTINCT lines per file, so a licence that names the wrong lines shows.
	for name, words := range map[string]string{
		"a.txt": "alpha\nbravo\ncharlie\ndelta\necho\n",
		"b.txt": "foxtrot\ngolf\nhotel\nindia\njuliet\n",
		"c.txt": "kilo\nlima\nmike\nnovember\noscar\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(words), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	specs := []Spec{{Path: "a.txt"}, {Path: "b.txt"}, {Path: "c.txt"}}

	// The unbounded answer fixes the budget: all of a.txt, then b.txt's header,
	// its range line and its first two lines, measured in runes.
	var whole bytes.Buffer
	Run(&whole, root, specs, Options{Numbers: true})
	all := strings.SplitAfter(whole.String(), "\n")
	bHeader := -1
	for i, l := range all {
		if strings.HasPrefix(l, "==> b.txt") {
			bHeader = i
		}
	}
	if bHeader < 0 {
		t.Fatalf("no b.txt header in the unbounded answer:\n%s", whole.String())
	}
	budget := 0
	for _, l := range all[:bHeader+4] { // b's header, its @@ line, golf's line before it is cut: foxtrot, golf
		budget += utf8.RuneCountInString(l)
	}

	var out, withheld bytes.Buffer
	observed, problems := Run(&out, root, specs, Options{Numbers: true, MaxChars: &budget, Withheld: &withheld})
	if n := utf8.RuneCountInString(out.String()); n > budget {
		t.Errorf("the answer is %d characters, over the budget of %d:\n%s", n, budget, out.String())
	}
	if want := strings.Join(all[:bHeader+4], ""); out.String() != want {
		t.Errorf("the budget did not stop at the last whole line that fit:\n got %q\nwant %q", out.String(), want)
	}
	w := withheld.String()
	for _, want := range []string{"b.txt: lines 3-5 WITHHELD (3 line(s)): --max-chars reached", "c.txt: WITHHELD: --max-chars reached"} {
		if !strings.Contains(w, want) {
			t.Errorf("the withheld report does not say %q:\n%s", want, w)
		}
	}
	if strings.Contains(out.String(), "WITHHELD") {
		t.Errorf("the withheld report was charged to the answer:\n%s", out.String())
	}
	if problems < 2 {
		t.Errorf("problems = %d; each withheld range and file is one", problems)
	}
	if o := observed["a.txt"]; o.Spans != nil {
		t.Errorf("a.txt was served whole and is licensed as %v", o.Spans)
	}
	if o, ok := observed["b.txt"]; !ok || len(o.Spans) != 1 || o.Spans[0] != [2]int{1, 2} {
		t.Errorf("b.txt is licensed as %+v; only lines 1-2 were printed", o)
	}
	if _, ok := observed["c.txt"]; ok {
		t.Error("c.txt was never served and is recorded as read")
	}

	// A budget of zero prints nothing and names every file.
	zero := 0
	out.Reset()
	withheld.Reset()
	observed, _ = Run(&out, root, specs, Options{Numbers: true, MaxChars: &zero, Withheld: &withheld})
	if out.Len() != 0 || len(observed) != 0 || strings.Count(withheld.String(), "WITHHELD") != 3 {
		t.Errorf("a budget of zero: out %q, observed %v, withheld %q", out.String(), observed, withheld.String())
	}
}
