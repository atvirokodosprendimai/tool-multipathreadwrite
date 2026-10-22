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
	observed, problems := Run(NewBudget(&out, budget), root, specs, Options{Numbers: true, Withheld: &withheld})
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
	observed, _ = Run(NewBudget(&out, zero), root, specs, Options{Numbers: true, Withheld: &withheld})
	if out.Len() != 0 || len(observed) != 0 || strings.Count(withheld.String(), "WITHHELD") != 3 {
		t.Errorf("a budget of zero: out %q, observed %v, withheld %q", out.String(), observed, withheld.String())
	}

	// A file past the budget is named WITHHELD, not opened: a missing one is not
	// reported REFUSED, which only opening it could find. And a refused file the
	// budget reached first has its REFUSED line on the withheld writer, since
	// the line itself did not fit (go-reviewer of a043d346).
	out.Reset()
	withheld.Reset()
	Run(NewBudget(&out, zero), root, []Spec{{Path: "missing.txt"}}, Options{Numbers: true, Withheld: &withheld})
	if w := withheld.String(); w != "missing.txt: WITHHELD: --max-chars reached\n" {
		t.Errorf("a file past the budget was opened, or not named: %q", w)
	}
	var unbounded bytes.Buffer
	Run(&unbounded, root, []Spec{{Path: "a.txt"}}, Options{Numbers: true})
	aOnly := utf8.RuneCountInString(unbounded.String())
	out.Reset()
	withheld.Reset()
	_, problems = Run(NewBudget(&out, aOnly+5), root, []Spec{{Path: "a.txt"}, {Path: "missing.txt"}}, Options{Numbers: true, Withheld: &withheld}) // 5 left: missing.txt is opened, its line does not fit
	if !strings.Contains(withheld.String(), "missing.txt") || !strings.Contains(withheld.String(), "UNREADABLE") || problems != 1 || out.String() != unbounded.String() {
		t.Errorf("a refusal the budget dropped: out %q, withheld %q, problems %d", out.String(), withheld.String(), problems)
	}
	// --max-lines and --max-chars together: a --max-lines notice the budget
	// drops still names its range.
	one := 1
	out.Reset()
	withheld.Reset()
	Run(NewBudget(&out, aOnly/2), root, []Spec{{Path: "a.txt"}, {Path: "b.txt"}}, Options{Numbers: true, MaxLines: &one, Withheld: &withheld})
	if w := withheld.String(); !strings.Contains(w, "b.txt") {
		t.Errorf("the second file is named nowhere under both caps: %q", w)
	}
}

// Two withheld ranges of ONE file are two different lines on the withheld
// writer, each naming its range (the v1.71.0 field test: eight of twelve
// peers could not tell which range was cut). A whole file keeps its bare path.
func TestTwoWithheldRangesOfOneFileAreNamedApart(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("alpha\nbravo\ncharlie\ndelta\necho\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := func(s string) Spec {
		sp, err := ParseSpec(s)
		if err != nil {
			t.Fatal(err)
		}
		return sp
	}
	withheldBy := func(budget int, specs ...Spec) string {
		var out, withheld bytes.Buffer
		Run(NewBudget(&out, budget), root, specs, Options{Numbers: true, Withheld: &withheld})
		return withheld.String()
	}

	// Budget 0: both are named without being opened.
	w := withheldBy(0, spec("a.txt:1-2"), spec("a.txt:4-5"))
	if w != "a.txt:1-2: WITHHELD: --max-chars reached\na.txt:4-5: WITHHELD: --max-chars reached\n" {
		t.Errorf("two ranges past the budget are not named apart: %q", w)
	}
	if w := withheldBy(0, spec("a.txt")); w != "a.txt: WITHHELD: --max-chars reached\n" {
		t.Errorf("a whole file lost its bare-path form: %q", w)
	}

	// The first range fits; the second one's header does not.
	var first bytes.Buffer
	Run(&first, root, []Spec{spec("a.txt:1-2")}, Options{Numbers: true})
	w = withheldBy(utf8.RuneCountInString(first.String())+3, spec("a.txt:1-2"), spec("a.txt:4-5"))
	if w != "a.txt:4-5: WITHHELD: --max-chars reached\n" {
		t.Errorf("a range whose header did not fit is not named by its range: %q", w)
	}
}
