package read

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// jsonFixture is a package.json-shaped document whose values sit on known
// lines: 3-6 the scripts object, 5 one script, 9-10 an array element that
// spans two lines, 12 a name holding a dot, 13 a name written with an escape.
const jsonFixture = `{
  "name": "demo",
  "scripts": {
    "build": "go build",
    "test": "go test"
  },
  "list": [
    1,
    {"x": 2,
     "y": 3}
  ],
  "a.b": {"c": true},
  "esc\u0061pe": 7
}
`

// serveJSON reads root/f.json at addr and returns the answer, the problem count
// and the spans the read recorded.
func serveJSON(t *testing.T, body, addr string) (string, int, [][2]int) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := ParseSpec("f.json:" + addr)
	if err != nil {
		t.Fatalf("%s: %v", addr, err)
	}
	var sb strings.Builder
	observed, problems := Run(&sb, root, []Spec{sp}, Options{Numbers: true})
	return sb.String(), problems, observed["f.json"].Spans
}

// ADR-155 Decisions 2-5. A path serves the lines its value spans, whatever the
// file's line endings or a leading byte order mark; every way it can fail is a
// range that could not be served, naming why; a name twice in one object on
// the path is refused with both lines, and a duplicate off the path is not
// looked at.
func TestAJSONPathAddressesTheLinesOfItsValue(t *testing.T) {
	variants := map[string]string{
		"LF":      jsonFixture,
		"CRLF":    strings.ReplaceAll(jsonFixture, "\n", "\r\n"),
		"CR-only": strings.ReplaceAll(jsonFixture, "\n", "\r"),
		"BOM":     "\uFEFF" + jsonFixture,
	}
	cases := []struct {
		addr string
		want [][2]int
	}{
		{"$.scripts", [][2]int{{3, 6}}},
		{"$.scripts.test", [][2]int{{5, 5}}},
		{"$.name", [][2]int{{2, 2}}},
		{"$.list[0]", [][2]int{{8, 8}}},
		{"$.list[1]", [][2]int{{9, 10}}},
		{"$.list[1].y", [][2]int{{10, 10}}},
		{`$["a.b"].c`, [][2]int{{12, 12}}},
		{"$.escape", [][2]int{{13, 13}}},
		{"$", [][2]int{{14, 14}}},
	}
	for name, body := range variants {
		for _, c := range cases {
			out, problems, spans := serveJSON(t, body, c.addr)
			if problems != 0 {
				t.Errorf("%s %s: %d problem(s):\n%s", name, c.addr, problems, out)
				continue
			}
			if !reflect.DeepEqual(spans, c.want) {
				t.Errorf("%s %s: recorded %v, want %v:\n%s", name, c.addr, spans, c.want, out)
			}
		}
	}

	misses := []struct{ body, addr, why string }{
		{jsonFixture, "$.missing", `no name "missing"`},
		{jsonFixture, "$.list[5]", "past the end"},
		{jsonFixture, "$.name.x", "a string, not an object"},
		{jsonFixture, "$.scripts[0]", "an object, not an array"},
		{"{\"a\":1}\n{\"a\":2}\n", "$.a", "not one JSON document"},
		{"{\"a\":1} x\n", "$.a", "not one JSON document"},
		{"{\n\"a\": 1,,\n}\n", "$.a", "line 2"},
		{"{\n  \"a\": 1,\n  \"a\": 2,\n  \"c\": 3\n}\n", "$.a", "twice"},
		{"{\n  \"b\": {\"d\": 1,\n   \"d\": 2}\n}\n", "$.b.d", "lines 2 and 3"},
	}
	for _, m := range misses {
		out, problems, spans := serveJSON(t, m.body, m.addr)
		if problems == 0 || !strings.Contains(out, "!! no match for "+m.addr+" (") || !strings.Contains(out, m.why) {
			t.Errorf("%s on %q: want a missed range naming %q, got %d problem(s):\n%s", m.addr, m.body, m.why, problems, out)
		}
		if len(spans) != 0 {
			t.Errorf("%s on %q: a failed path recorded %v", m.addr, m.body, spans)
		}
	}

	// A duplicate the path does not go through is not looked at.
	dup := "{\n  \"a\": 1,\n  \"a\": 2,\n  \"c\": 3\n}\n"
	if out, problems, spans := serveJSON(t, dup, "$.c"); problems != 0 || !reflect.DeepEqual(spans, [][2]int{{4, 4}}) {
		t.Errorf("$.c beside a duplicated a: %d problem(s), recorded %v:\n%s", problems, spans, out)
	}
}

// ADR-155 Decision 1. The form begins `$.` or `$[`; a bare `$`, `$-N` and
// `N-$` keep meaning the last line; everything the form does not have is
// refused at parse, naming it.
func TestAJSONPathIsParsedOnlyInTheFormsItHas(t *testing.T) {
	for _, a := range []string{"$", "$-2", "2-$"} {
		sp, err := ParseSpec("f.json:" + a)
		if err != nil {
			t.Errorf("%s: %v", a, err)
			continue
		}
		if r := sp.Ranges[0]; r.JSON != nil || (r.Start != lastLine && r.End != lastLine) {
			t.Errorf("%s no longer names the last line: %+v", a, r)
		}
	}
	for _, a := range []string{"$.a", "$.a[0]", `$["a b"]`, `$["x/y"].z`, "$[0]", "$.a.b.c"} {
		sp, err := ParseSpec("f.json:" + a)
		if err != nil {
			t.Errorf("%s was refused: %v", a, err)
			continue
		}
		if len(sp.Ranges) != 1 || sp.Ranges[0].JSON == nil {
			t.Errorf("%s did not parse as one JSON path: %+v", a, sp.Ranges)
		}
	}
	for _, a := range []string{"$..a", "$.*", "$['a']", "$.", "$[x]", "$.a]", "$[-1]", `$["a"`, "$.a[]", "$[01]"} {
		if _, err := ParseSpec("f.json:" + a); err == nil {
			t.Errorf("%s was accepted", a)
		}
	}
	if _, err := ParseSpec("f.json:$.a,+3"); err == nil || !strings.Contains(err.Error(), "relative end") {
		t.Errorf("a relative end on a JSON path was not refused by name: %v", err)
	}
	// A slash inside a quoted name is the name's, so a pattern after the path in
	// the same comma list is a second range, not the end of a first.
	sp, err := ParseSpec(`f.json:$["x/y"],/demo/`)
	if err != nil || len(sp.Ranges) != 2 || sp.Ranges[0].JSON == nil || sp.Ranges[1].Re == nil {
		t.Errorf(`$["x/y"],/demo/ is not a JSON path and a pattern: %+v, %v`, sp.Ranges, err)
	}
}
