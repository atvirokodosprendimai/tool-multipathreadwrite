package read

import "testing"

// Issue #715 (a). A quoted JSON key may hold the characters the spec splitter
// reads as delimiters: a colon (the file/range separator) or a comma (the
// range separator). `$["build:prod"]` was cut at its colon and `$["a,b"]` at its
// comma, so a key the document really has could not be addressed.
func TestAQuotedJSONKeyMayHoldAColonOrACommaInASpec(t *testing.T) {
	for _, tc := range []struct {
		spec   string
		path   string
		ranges []string // the name each range's single JSON step carries
	}{
		{`doc.json:$["build:prod"]`, "doc.json", []string{"build:prod"}},
		{`doc.json:$["a,b"]`, "doc.json", []string{"a,b"}},
		{`doc.json:$["a:/b,c"]`, "doc.json", []string{"a:/b,c"}},
		{`doc.json:$["x,+3"]`, "doc.json", []string{"x,+3"}},
		{`doc.json:$["q\"r:s"]`, "doc.json", []string{`q"r:s`}},
		// A comma inside the quotes belongs to the key, an escaped quote included.
		{`doc.json:$["q\"r,s"]`, "doc.json", []string{`q"r,s`}},
		{`doc.json:$.a["b:c"]`, "doc.json", []string{"b:c"}},
	} {
		sp, err := ParseSpec(tc.spec)
		if err != nil {
			t.Errorf("%s: %v", tc.spec, err)
			continue
		}
		if sp.Path != tc.path || len(sp.Ranges) != len(tc.ranges) {
			t.Errorf("%s: path %q with %d range(s), want %q with %d", tc.spec, sp.Path, len(sp.Ranges), tc.path, len(tc.ranges))
			continue
		}
		for i, want := range tc.ranges {
			if j := sp.Ranges[i].JSON; len(j) == 0 || j[len(j)-1].name != want {
				t.Errorf("%s: range %d steps %+v, want the one name %q", tc.spec, i, j, want)
			}
		}
	}
	// A colon-and-slash pattern after a path keeps its meaning.
	if sp, err := ParseSpec(`x.go:/a:b/`); err != nil || sp.Path != "x.go" || len(sp.Ranges) != 1 || sp.Ranges[0].Re == nil {
		t.Errorf("a pattern holding a colon: %+v, %v", sp, err)
	}
}

// Review of #722. A file name may hold `:$.` or `:$[`: those served on main and
// still do, because the JSON cut applies only when the last colon sits inside a
// quoted key.
func TestAFileNameHoldingADollarStepStillSplitsAtItsLastColon(t *testing.T) {
	for _, tc := range []struct{ spec, path string }{
		{"a:$.txt:2", "a:$.txt"},
		{"b:$[1].txt:1", "b:$[1].txt"},
		{"d:$.x/f.go:2", "d:$.x/f.go"},
	} {
		sp, err := ParseSpec(tc.spec)
		if err != nil || sp.Path != tc.path || len(sp.Ranges) != 1 || sp.Ranges[0].JSON != nil {
			t.Errorf("%s: path %q, ranges %+v, err %v; want %q with one line range", tc.spec, sp.Path, sp.Ranges, err, tc.path)
		}
	}
}
