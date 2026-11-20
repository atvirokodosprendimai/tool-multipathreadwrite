package plan

import (
	"strings"
	"testing"
)

// Issue #719. A plan refusing a JSON-path address printed it with its quotes
// stripped, because the header tokenizer consumed them before the refusal
// formatted the address: `$["a,b"]` was named "$[a,b]". It is named as written.
func TestAJSONPathRefusalNamesTheAddressAsWritten(t *testing.T) {
	for _, tc := range []struct{ header, want string }{
		{`@@ doc.json $["a,b"] replace`, `"$[\"a,b\"]"`},
		{`@@ doc.json $.q["x y"] replace`, `"$.q[\"x y\"]"`},
		{`@@ "my doc.json" $["k"] replace anchor="z"`, `"$[\"k\"]"`},
		{`@@ doc.json $.plain replace`, `"$.plain"`},
		{`@@ doc.json "$".q["a,b"] replace`, `"\"$\".q[\"a,b\"]"`}, // a quoted `$` hides nothing, but the key's quotes are the address
		{`@@ doc.json $["$"] replace`, `"$[\"$\"]"`},               // a key named `$` keeps its quotes: only a quoted ROOT marker is left out
	} {
		_, err := Parse(strings.NewReader(tc.header + "\nz\n"))
		if err == nil || !strings.Contains(err.Error(), "a JSON path is a read address") || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want a refusal naming %s", tc.header, err, tc.want)
		}
	}
	// The pair: a path with a quote in it and a pattern with quotes are unaffected.
	// The path itself is read, not only parsed: the address as written keeps its
	// quotes for the refusal, and the path must not.
	if h, err := Parse(strings.NewReader("@@ \"my doc.json\" 1 replace\nz\n")); err != nil || len(h) != 1 || h[0].Path != "my doc.json" {
		t.Errorf("a quoted path was not read as one: %+v, %v", h, err)
	}
	// A quoted `$` hides nothing of the address: the refusal is the one the parsed
	// address always had, naming `$.x`, not a line-number error.
	if _, err := Parse(strings.NewReader("@@ f.txt \"$\".x replace\nz\n")); err == nil || !strings.Contains(err.Error(), "a JSON path is a read address") || !strings.Contains(err.Error(), `"$.x"`) || strings.Contains(err.Error(), "bad line number") {
		t.Errorf(`a quoted "$" address: %v, want the JSON-path refusal naming "$.x"`, err)
	}
	if h, err := Parse(strings.NewReader("@@ f.txt /^\"foo\"$/ replace\nz\n")); err != nil || len(h) != 1 || h[0].Addr.StartPat == nil || !strings.Contains(h[0].Addr.StartPat.String(), `"foo"`) {
		t.Errorf("a pattern holding quotes changed: %+v, %v", h, err)
	}
}
