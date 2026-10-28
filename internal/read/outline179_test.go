package read

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func outline179Run(t *testing.T, files map[string]string, paths ...string) (string, map[string][][2]int, int) {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var specs []Spec
	for _, p := range paths {
		specs = append(specs, Spec{Path: p, Raw: p})
	}
	var out bytes.Buffer
	observed, problems := Run(&out, root, specs, Options{Numbers: true, Outline: true})
	spans := map[string][][2]int{}
	for k, o := range observed {
		spans[k] = o.Spans
	}
	return out.String(), spans, problems
}

// ADR-179 Decision (b). A Go file's top-level func (with its receiver), type,
// var and const declarations, one line each with the lines they span from the
// file set, the doc comment included; imports are not listed; the file is hashed
// with no span; a file that does not parse prints what parsed, `!! parse error`,
// and is a problem.
func TestOutlineListsGoDeclarationsWithTheirLines(t *testing.T) {
	src := "package p\n\nimport \"fmt\"\n\n// Get returns.\nfunc (s *Store) Get(k string) string {\n\treturn fmt.Sprint(k)\n}\n\ntype T struct {\n\tA int\n}\n\nvar (\n\tx = 1\n\ty = 2\n)\n\nconst C = 3\n\nfunc (*Store) Put() {}\n"
	text, spans, problems := outline179Run(t, map[string]string{"a.go": src}, "a.go")
	want := "5-8 func (s *Store) Get\n10-12 type T\n14-17 var x, y\n19-19 const C\n21-21 func (*Store) Put\n"
	if !strings.Contains(text, want) || problems != 0 || strings.Contains(text, "import") || strings.Contains(text, "|") {
		t.Errorf("the outline (problems %d) is\n%s\nwant it to hold\n%s", problems, text, want)
	}
	if got, ok := spans["a.go"]; !ok || !reflect.DeepEqual(got, [][2]int{}) {
		t.Errorf("observed %#v, want the file hashed with no span", spans["a.go"])
	}
	text, _, problems = outline179Run(t, map[string]string{"b.go": "package p\nfunc Ok() {}\nfunc Bad( {\n"}, "b.go")
	if !strings.Contains(text, "2-2 func Ok\n") || !strings.Contains(text, "!! parse error: ") || problems == 0 {
		t.Errorf("a file that does not parse printed (problems %d):\n%s", problems, text)
	}
	// A //line directive moves the file set's line numbers, not the file's (and is the doc comment of the func below it).
	text, _, _ = outline179Run(t, map[string]string{"c.go": "package p\n//line x.go:100\nfunc L() {}\n"}, "c.go")
	if !strings.Contains(text, "2-3 func L\n") {
		t.Errorf("a //line directive moved the outline:\n%s", text)
	}
}

// ADR-179 Decision (b). A Markdown file's ATX headings, one line each with the
// lines they span: a `#` inside a fenced block is no heading, and a heading's
// range ends before the next heading of its level or a higher one.
func TestOutlineListsMarkdownHeadingsOutsideFences(t *testing.T) {
	src := "# Title\nintro\n## A\n```\n# not a heading\n```\n### A1\n## B\n"
	text, _, problems := outline179Run(t, map[string]string{"a.md": src}, "a.md")
	want := "1-8 # Title\n3-7 ## A\n7-7 ### A1\n8-8 ## B\n"
	if !strings.Contains(text, want) || strings.Contains(text, "not a heading") || problems != 0 {
		t.Errorf("the outline (problems %d) is\n%s\nwant it to hold\n%s", problems, text, want)
	}
}
