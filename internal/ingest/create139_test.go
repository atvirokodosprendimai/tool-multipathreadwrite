package ingest

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

func parseOne(t *testing.T, compiled []byte) plan.Hunk {
	t.Helper()
	hunks, err := plan.Parse(bytes.NewReader(compiled))
	if err != nil {
		t.Fatalf("the compiled plan does not parse: %v\n%s", err, compiled)
	}
	if len(hunks) != 1 || hunks[0].Op != plan.OpCreate {
		t.Fatalf("want one create hunk, got %+v\n%s", hunks, compiled)
	}
	return hunks[0]
}

// ADR-139. A file's content becomes one create hunk, by the same emitter the
// foreign formats use: the lines of the content, a line beginning "@@" carried
// verbatim, a path with a space quoted. Empty content is refused (ADR-195).
func TestCompileCreateMakesACreatePlanOfStdin(t *testing.T) {
	out, err := CompileCreate("dir/a.txt", []byte("alpha\nbeta"))
	if err != nil {
		t.Fatal(err)
	}
	if h := parseOne(t, out); h.Path != "dir/a.txt" || !reflect.DeepEqual(h.Body, []string{"alpha", "beta"}) {
		t.Errorf("plain content: %+v", h)
	}
	doc := "# a plan, as content\n@@ other.go 1 replace\nnew\n@@ x.go 2 delete\n"
	out, err = CompileCreate("doc.md", []byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if h := parseOne(t, out); !reflect.DeepEqual(h.Body, strings.Split(strings.TrimSuffix(doc, "\n"), "\n")) {
		t.Errorf("a document holding @@ lines was not carried whole: %+v\n%s", h, out)
	}
	if _, err = CompileCreate("empty.txt", nil); err == nil || !strings.Contains(err.Error(), "standard input was empty") {
		t.Errorf("empty content was not refused as empty standard input (ADR-195): %v", err)
	}
	out, err = CompileCreate("with space.txt", []byte("x\n"))
	if err != nil {
		t.Fatal(err)
	}
	if h := parseOne(t, out); h.Path != "with space.txt" {
		t.Errorf("a path with a space: %q", h.Path)
	}
	for name, content := range map[string][]byte{"a UTF-16 mark": {0xff, 0xfe, 'a', 0}, "a NUL": []byte("a\x00b\n")} {
		if _, err := CompileCreate("bin.dat", content); err == nil {
			t.Errorf("content holding %s was compiled", name)
		}
	}
	for _, bad := range []string{"", "/abs/x.txt"} {
		if _, err := CompileCreate(bad, []byte("x\n")); err == nil {
			t.Errorf("path %q was accepted", bad)
		}
	}
}
