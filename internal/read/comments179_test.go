package read

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func comments179Run(t *testing.T, src string, ranges ...Range) (string, [][2]int, int) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	observed, problems := Run(&out, root, []Spec{{Path: "a.go", Raw: "a.go", Ranges: ranges}}, Options{Numbers: true, NoComments: true})
	return out.String(), observed["a.go"].Spans, problems
}

const comments179Src = "package p\n\n// Doc line one.\n// Doc line two.\nfunc A() {\n\tx := \"// not a comment\"\n\ty := 1 // trailing\n\t// single\n\tz := 2\n\t/* block start\n\t   block mid\n\t   block end */\n\t_ = x + y + z\n}\n"

// ADR-179 Decision (h). A run of two or more whole-line comment lines (// or a
// /* */ block) in a .go file is `-- … N comment line(s), A-B` and is not
// recorded as read; a // inside a string, a trailing comment and a single
// comment line stay; a range that starts inside a block comment is judged by a
// scan of the whole file.
func TestNoCommentsElidesGoCommentBlocks(t *testing.T) {
	text, spans, problems := comments179Run(t, comments179Src)
	for _, want := range []string{
		"-- … 2 comment line(s), 3-4\n",
		"-- … 3 comment line(s), 10-12\n",
		"  6| \tx := \"// not a comment\"\n",
		"  7| \ty := 1 // trailing\n",
		"  8| \t// single\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the elision lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Doc line") || strings.Contains(text, "block mid") {
		t.Errorf("an elided comment line was printed:\n%s", text)
	}
	if want := [][2]int{{1, 2}, {5, 9}, {13, 14}}; problems != 0 || !reflect.DeepEqual(spans, want) {
		t.Errorf("observed %v (problems %d), want %v", spans, problems, want)
	}
	text, _, _ = comments179Run(t, comments179Src, Range{Start: 11, End: 13, Text: "11-13"})
	if !strings.Contains(text, "-- … 2 comment line(s), 11-12\n") || !strings.Contains(text, "_ = x + y + z") {
		t.Errorf("a range starting inside a block comment printed:\n%s", text)
	}
	text, _, _ = comments179Run(t, comments179Src, Range{Start: 12, End: 13, Text: "12-13"})
	if strings.Contains(text, "comment line(s)") || !strings.Contains(text, "block end */") {
		t.Errorf("a single comment line of a range was elided:\n%s", text)
	}
}

// ADR-179 Decision (h). //go: and //line directives and the comment above
// import "C" are never elided, however many lines they run.
func TestNoCommentsKeepsDirectivesAndTheCgoPreamble(t *testing.T) {
	src := "//go:build linux\n// +build linux\n\npackage p\n\n//go:generate echo a\n//go:generate echo b\n\n// #include <stdio.h>\n// static void f() {}\nimport \"C\"\n\n//line foo.go:10\n//line foo.go:20\n// c1\n// c2\nfunc F() {}\n"
	text, _, problems := comments179Run(t, src)
	if problems != 0 || strings.Count(text, "comment line(s)") != 1 || !strings.Contains(text, "-- … 2 comment line(s), 15-16\n") {
		t.Errorf("want only the plain pair 15-16 elided (problems %d):\n%s", problems, text)
	}
	for _, kept := range []string{"//go:build linux", "//go:generate echo b", "// #include <stdio.h>", "// static void f() {}", "//line foo.go:20"} {
		if !strings.Contains(text, kept) {
			t.Errorf("%q was elided:\n%s", kept, text)
		}
	}
}
