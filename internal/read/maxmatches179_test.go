package read

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// maxMatches179Specs writes the files and returns a --grep spec for each, in
// the order given: what the walk hands Run.
func maxMatches179Specs(t *testing.T, re string, files [][2]string) (string, []Spec) {
	t.Helper()
	root := t.TempDir()
	rx := regexp.MustCompile(re)
	var specs []Spec
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(root, f[0]), []byte(f[1]), 0o644); err != nil {
			t.Fatal(err)
		}
		specs = append(specs, Spec{Path: f[0], Raw: f[0], Ranges: []Range{{Re: rx, Text: "/" + re + "/"}}})
	}
	return root, specs
}

// ADR-179 Decision (g). Three files of four matches and a budget of five: the
// first file shows four, the second one, the third none (no header either);
// the seven left are counted in two files; the read is incomplete; and nothing
// beyond what was shown is observed. A budget of zero shows none and counts all.
func TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest(t *testing.T) {
	body := "hit 1\nfill\nhit 2\nfill\nhit 3\nfill\nhit 4\n"
	root, specs := maxMatches179Specs(t, "hit", [][2]string{{"a.txt", body}, {"b.txt", body}, {"c.txt", body}})
	var out bytes.Buffer
	observed, problems := Run(&out, root, specs, Options{Numbers: true, NextLine: true, MaxMatches: intp(5)})
	text := out.String()
	if n := strings.Count(text, "hit "); n != 5 {
		t.Errorf("%d matches shown, want 5:\n%s", n, text)
	}
	if want := "-- max-matches 5 reached: 7 more matching line(s) in 2 file(s) not shown\n"; !strings.HasSuffix(text, want) {
		t.Errorf("the answer does not end %q:\n%s", want, text)
	}
	if strings.Contains(text, "==> c.txt") {
		t.Errorf("a file with nothing shown got a header:\n%s", text)
	}
	if problems == 0 {
		t.Errorf("a budget that withheld matches left the read complete (%d problems)", problems)
	}
	if _, ok := observed["c.txt"]; ok {
		t.Errorf("c.txt was observed though none of it was shown: %v", observed["c.txt"])
	}
	if sp := observed["b.txt"].Spans; len(sp) != 1 || sp[0][0] > 1 {
		t.Errorf("b.txt observed %v, want only the span of its first match", sp)
	}

	out.Reset()
	observed, problems = Run(&out, root, specs, Options{Numbers: true, NextLine: true, MaxMatches: intp(0)})
	if want := "-- max-matches 0 reached: 12 more matching line(s) in 3 file(s) not shown\n"; out.String() != want || problems == 0 || len(observed) != 0 {
		t.Errorf("a budget of 0 printed %q (problems %d, observed %v), want %q alone", out.String(), problems, observed, want)
	}

	// --max-lines cuts a span after the budget was charged: only the matches
	// that reached the answer are charged. The first file prints its context
	// line and no match, so its match is returned to the count and the second
	// file's match is still shown.
	root2, specs2 := maxMatches179Specs(t, "hit", [][2]string{{"a.txt", "a\nhit 1\nb\nc\nhit 2\nd\n"}, {"b.txt", "hit 3\nz\n"}})
	out.Reset()
	_, problems = Run(&out, root2, specs2, Options{Numbers: true, NextLine: true, Context: 1, MaxLines: intp(1), MaxMatches: intp(1)})
	if text := out.String(); !strings.Contains(text, "hit 3") || strings.Contains(text, "hit 1") ||
		!strings.HasSuffix(text, "-- max-matches 1 reached: 2 more matching line(s) in 1 file(s) not shown\n") || problems == 0 {
		t.Errorf("--max-lines charged a match it did not print (problems %d):\n%s", problems, text)
	}

	// A hit is charged only when all of it reached the answer: a multi-line
	// ast-grep hit cut after its first line returns to the count, and the next
	// file's whole hit is still shown.
	rootH, _ := maxMatches179Specs(t, "x", [][2]string{{"a.txt", "x\nA1\nA2\nA3\ny\n"}, {"b.txt", "x\nB1\ny\n"}})
	hitSpecs := []Spec{
		{Path: "a.txt", Raw: "a.txt", Ranges: []Range{{Start: 2, End: 4, Text: "2-4"}}},
		{Path: "b.txt", Raw: "b.txt", Ranges: []Range{{Start: 2, End: 2, Text: "2"}}},
	}
	out.Reset()
	_, problems = Run(&out, rootH, hitSpecs, Options{Numbers: true, MaxLines: intp(1), MaxMatches: intp(1)})
	if text := out.String(); !strings.Contains(text, "B1") || !strings.HasSuffix(text, "-- max-matches 1 reached: 1 more matching line(s) in 1 file(s) not shown\n") || problems == 0 {
		t.Errorf("a hit cut by --max-lines stayed charged (problems %d):\n%s", problems, text)
	}

	// A file whose header --max-chars rejects shows nothing, so none of its
	// matches is charged or counted, and no note speaks of them.
	rootS, specsS := maxMatches179Specs(t, "hit", [][2]string{{"a.txt", "hit 1\nfill\nfill\nhit 2\nfill\nfill\nhit 3\n"}})
	var sink, withheld bytes.Buffer
	_, problems = Run(NewBudget(&sink, 10), rootS, specsS, Options{Numbers: true, NextLine: true, MaxMatches: intp(1), Withheld: &withheld})
	if strings.Contains(sink.String()+withheld.String(), "max-matches") || strings.Contains(sink.String(), "hit") || problems == 0 {
		t.Errorf("a file withheld whole was still counted (problems %d):\n%q\n%q", problems, sink.String(), withheld.String())
	}
}

// ADR-179 Decision (g). A match inside a span already served — as context or as
// the line after a range — is shown and counts: two adjacent matches with -C 1
// and a budget of 1 show the second as context, complete the span, and open no
// span for the far third.
func TestMaxMatchesCountsAMatchShownAsContext(t *testing.T) {
	body := "x1\nhit A\nhit B\nx4\nx5\nx6\nhit C\nx8\n"
	root, specs := maxMatches179Specs(t, "hit", [][2]string{{"a.txt", body}})
	var out bytes.Buffer
	observed, problems := Run(&out, root, specs, Options{Numbers: true, NextLine: true, Context: 1, MaxMatches: intp(1)})
	text := out.String()
	if !strings.Contains(text, "hit A") || !strings.Contains(text, "hit B") || strings.Contains(text, "hit C") || strings.Count(text, "@@") != 1 {
		t.Errorf("want the span of A with B as context, one span, no C:\n%s", text)
	}
	if want := "-- max-matches 1 reached: 1 more matching line(s) in 1 file(s) not shown\n"; !strings.HasSuffix(text, want) || problems == 0 {
		t.Errorf("the answer does not end %q (problems %d):\n%s", want, problems, text)
	}
	if sp := observed["a.txt"].Spans; len(sp) != 1 || sp[0] != [2]int{1, 5} {
		t.Errorf("observed %v, want [1 5]: B's own context and the line after it are served with it", sp)
	}

	// A budget no smaller than the matches serves and records what the read
	// without the flag does: adjacent hits, context, and ast-grep style
	// multi-line hits (numeric ranges) next to another.
	adjacent := "x\nMATCH a\nMATCH b\nq\nr\ns\nt\n"
	rootA, specsA := maxMatches179Specs(t, "MATCH", [][2]string{{"f.txt", adjacent}})
	hits := []Spec{{Path: "f.txt", Raw: "f.txt", Ranges: []Range{{Start: 2, End: 4, Text: "2-4"}, {Start: 5, End: 6, Text: "5-6"}}}}
	for name, c := range map[string]struct {
		specs []Spec
		opt   Options
		n     int
	}{
		"grep":          {specsA, Options{Numbers: true, NextLine: true}, 2},
		"grep over":     {specsA, Options{Numbers: true, NextLine: true}, 100},
		"grep context":  {specsA, Options{Numbers: true, NextLine: true, Context: 1}, 2},
		"ast-grep hits": {hits, Options{Numbers: true, NextLine: true}, 2},
	} {
		var plain, budgeted bytes.Buffer
		wantObs, _ := Run(&plain, rootA, c.specs, c.opt)
		c.opt.MaxMatches = intp(c.n)
		gotObs, problems := Run(&budgeted, rootA, c.specs, c.opt)
		if plain.String() != budgeted.String() || problems != 0 || !reflect.DeepEqual(wantObs, gotObs) {
			t.Errorf("%s: a budget of %d serves %q (problems %d, observed %v), the read without it %q (observed %v)",
				name, c.n, budgeted.String(), problems, gotObs, plain.String(), wantObs)
		}
	}

	// A match on the line after a span is shown, so a budget of 1 over two
	// adjacent matches leaves nothing unshown.
	out.Reset()
	if _, problems = Run(&out, rootA, specsA, Options{Numbers: true, NextLine: true, MaxMatches: intp(1)}); problems != 0 || strings.Contains(out.String(), "max-matches") || !strings.Contains(out.String(), "MATCH b") {
		t.Errorf("the line after the span was not counted shown (problems %d):\n%s", problems, out.String())
	}
}
