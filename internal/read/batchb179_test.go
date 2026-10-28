package read

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func bb179Run(t *testing.T, name, body string, ranges []Range, opt Options) (string, string, int) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, withheld bytes.Buffer
	opt.Withheld = &withheld
	_, problems := Run(&out, root, []Spec{{Path: name, Raw: name, Ranges: ranges}}, opt)
	return out.String(), withheld.String(), problems
}

// The fold compares lines by their non-digit text, so a literal # is no digit
// placeholder: `id #` shares a run with nothing, `id 7` and `id 42` fold.
func TestFoldDoesNotTakeAHashForADigitRun(t *testing.T) {
	text, _, _ := bb179Run(t, "f.txt", "id #\nid 7\nid 42\n", nil, Options{Numbers: true, Fold: true})
	if !strings.Contains(text, "    1| id #\n    2| id 7\n-- ×2: lines 2-3 like 2, differing only in digits\n") {
		t.Errorf("the fold printed\n%s", text)
	}
}

// A line a finder matched is never folded away, and the line after a range
// (ADR-161) is served for its licence, not folded into the run it ends.
func TestFoldNeverHidesAMatchedOrRequestedLine(t *testing.T) {
	body := "row 1\nrow 2\nrow 3\nrow 4\nrow 5\nrow 6\n"
	text, _, _ := bb179Run(t, "f.txt", body, []Range{{Re: regexp.MustCompile(`row [2-6]`), Text: "/row [2-6]/"}}, Options{Numbers: true, Fold: true})
	for _, l := range []string{"row 2", "row 3", "row 4", "row 5", "row 6"} {
		if !strings.Contains(text, l) {
			t.Errorf("a matched line %q was folded away:\n%s", l, text)
		}
	}
	text, _, _ = bb179Run(t, "f.txt", body, []Range{{Start: 2, End: 2, Text: "2"}}, Options{Numbers: true, NextLine: true, Fold: true})
	if strings.Contains(text, "×") || !strings.Contains(text, "row 2") {
		t.Errorf("the requested line was folded with the line after it:\n%s", text)
	}
}

// With --max-chars a fold or comment note that does not fit is withheld and
// named, never dropped: wherever the budget ends, a read that printed the first
// line of a run but not its note says WITHHELD.
func TestAFoldOrCommentNoteThatDoesNotFitIsWithheld(t *testing.T) {
	cases := []struct {
		name, body string
		opt        Options
	}{
		{"f.txt", "x 1\nx 2\nx 3\n", Options{Numbers: true, Fold: true}},
		{"a.go", "package p\n// a\n// b\n// c\n", Options{Numbers: true, NoComments: true}},
	}
	for _, c := range cases {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, c.name), []byte(c.body), 0o644); err != nil {
			t.Fatal(err)
		}
		hit := false
		for n := 0; n < 140; n++ {
			var sink, withheld bytes.Buffer
			opt := c.opt
			opt.Withheld = &withheld
			Run(NewBudget(&sink, n), root, []Spec{{Path: c.name, Raw: c.name}}, opt)
			shownLast := strings.Contains(sink.String(), "| x 1\n") || strings.Contains(sink.String(), "| package p\n")
			noted := strings.Contains(sink.String(), "-- ×3") || strings.Contains(sink.String(), "comment line(s)")
			if shownLast && !noted && !strings.Contains(withheld.String(), "WITHHELD") {
				t.Errorf("%s with --max-chars %d printed the first line and lost the note unnamed:\n%q\n%q", c.name, n, sink.String(), withheld.String())
			}
			hit = hit || (shownLast && !noted)
		}
		if !hit {
			t.Errorf("%s: no budget exercised a note that does not fit", c.name)
		}
	}
}

// The line after a range never counts toward a comment run, and a line a grep
// matched is never elided.
func TestNoCommentsNeverElidesARequestedOrMatchedLine(t *testing.T) {
	src := "package p\n\n// a\n// b\nfunc A() {}\n"
	text, _, _ := bb179Run(t, "a.go", src, []Range{{Start: 3, End: 3, Text: "3"}}, Options{Numbers: true, NextLine: true, NoComments: true})
	if strings.Contains(text, "comment line(s)") || !strings.Contains(text, "// a") {
		t.Errorf("the requested line was elided with the line after it:\n%s", text)
	}
	text, _, _ = bb179Run(t, "a.go", src, []Range{{Re: regexp.MustCompile(`// a`), Text: "/// a/"}}, Options{Numbers: true, NoComments: true})
	if strings.Contains(text, "comment line(s)") || !strings.Contains(text, "// a") {
		t.Errorf("a matched line was elided:\n%s", text)
	}
}

// Every C import spec keeps its preamble, grouped imports included, and an
// //export line is a cgo directive.
func TestNoCommentsKeepsEveryCgoPreambleAndExport(t *testing.T) {
	src := "package p\n\nimport (\n\t\"fmt\"\n\n\t// #include <stdio.h>\n\t// static void f() {}\n\t\"C\"\n)\n\n//export Foo\n//export Bar\nfunc Foo() {}\n\n// c1\n// c2\nvar _ = fmt.Sprint\n"
	text, _, _ := bb179Run(t, "a.go", src, nil, Options{Numbers: true, NoComments: true})
	if strings.Count(text, "comment line(s)") != 1 || !strings.Contains(text, "-- … 2 comment line(s), 15-16\n") {
		t.Errorf("want only the plain pair 15-16 elided:\n%s", text)
	}
	for _, kept := range []string{"// #include <stdio.h>", "// static void f() {}", "//export Foo", "//export Bar"} {
		if !strings.Contains(text, kept) {
			t.Errorf("%q was elided:\n%s", kept, text)
		}
	}
}

// A .go file that does not scan is served whole and named with the files that
// are not .go.
func TestNoCommentsNamesAGoFileThatDoesNotScan(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package p\n// a\n// b\nvar s = `never closed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	var names []string
	Run(&out, root, []Spec{{Path: "a.go", Raw: "a.go"}}, Options{Numbers: true, NoComments: true, WholeNames: &names})
	if strings.Contains(out.String(), "comment line(s)") || !strings.Contains(out.String(), "// a") || len(names) != 1 || names[0] != "a.go" {
		t.Errorf("a file that does not scan: names %v, output\n%s", names, out.String())
	}
}

// A fence closes only on the same character, at least as long as its opener,
// with nothing after it; ~~~ fences and indents of up to three spaces count; four
// spaces make code, not a heading.
func TestOutlineFencesFollowCommonMark(t *testing.T) {
	src := "# A\n````\n```\n# no1\n```info\n# no2\n````\n~~~\n# no3\n```\n# no4\n~~~\n   ## B\n    # code\n## C\n"
	text, _, _ := outline179Run(t, map[string]string{"a.md": src}, "a.md")
	want := "1-15 # A\n13-14 ## B\n15-15 ## C\n"
	if !strings.Contains(text, want) || strings.Contains(text, "no") || strings.Contains(text, "code") {
		t.Errorf("the outline is\n%s\nwant it to hold\n%s", text, want)
	}
}

// A closing sequence needs whitespace before it, so `# C#` is `# C#`; an empty
// heading is a heading.
func TestOutlineATXHeadings(t *testing.T) {
	src := "# C#\n#\n## closed ##\n#hashtag\n####### seven\n"
	text, _, _ := outline179Run(t, map[string]string{"a.md": src}, "a.md")
	for _, want := range []string{"1-1 # C#\n", "2-5 #\n", "3-5 ## closed\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("the outline lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "hashtag") || strings.Contains(text, "seven") {
		t.Errorf("a non-heading was listed:\n%s", text)
	}
}

// A declaration cut off by a parse error has no end: its row says so instead of
// 3-0; a //line directive that precedes a decl by a blank line moves the file
// set's numbers, not the outline's; more than four names are cut with an ellipsis.
func TestOutlineGoEdgeCases(t *testing.T) {
	text, _, _ := outline179Run(t, map[string]string{"c.go": "package p\nfunc A() {}\nfunc B(\n"}, "c.go")
	if !strings.Contains(text, "3-? func B\n") || strings.Contains(text, "3-0") {
		t.Errorf("a decl cut by a parse error printed\n%s", text)
	}
	text, _, _ = outline179Run(t, map[string]string{"d.go": "package p\n//line x.go:100\n\nfunc L() {}\n\nvar a, b, c, d, e = 1, 2, 3, 4, 5\n"}, "d.go")
	if !strings.Contains(text, "4-4 func L\n") || !strings.Contains(text, "6-6 var a, b, c, d, …\n") {
		t.Errorf("the outline printed\n%s", text)
	}
}
