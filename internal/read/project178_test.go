package read

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// project178 reads root/f.json with the projections parsed from args, and
// returns the answer, the problem count, the withheld list and how many files
// the read recorded.
func project178(t *testing.T, body string, args ...string) (string, int, string, int) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	proj, err := ParseProjections(args)
	if err != nil {
		t.Fatalf("%q: %v", args, err)
	}
	sp, err := ParseSpec("f.json")
	if err != nil {
		t.Fatal(err)
	}
	var sb, withheld strings.Builder
	observed, problems := Run(&sb, root, []Spec{sp}, Options{Numbers: true, Project: proj, Withheld: &withheld})
	return sb.String(), problems, withheld.String(), len(observed)
}

// ADR-178 Decisions 2 and 3. A projection prints one `EXPR = VALUE` line per
// expression under the file's header, VALUE the value with its insignificant
// whitespace removed and every literal's bytes kept, from a single-line and a
// pretty-printed document alike; it records nothing.
func TestAJSONPathProjectionPrintsCompactValues(t *testing.T) {
	pretty := "{\n  \"scripts\": {\n    \"build\": \"go build\",\n    \"test\": \"go test\"\n  },\n  \"list\": [\n    1,\n    {\"x\": 2}\n  ]\n}\n"
	oneLine := `{"scripts":{"build":"go build","test":"go test"},"list":[1,{"x":2}]}`
	for name, body := range map[string]string{"pretty": pretty, "one line": oneLine} {
		out, problems, _, recorded := project178(t, body, "$.scripts,$.list[1].x")
		want := `$.scripts = {"build":"go build","test":"go test"}` + "\n" + `$.list[1].x = 2` + "\n"
		if !strings.HasPrefix(out, "==> f.json  ") || !strings.HasSuffix(out, "\n"+want) || strings.Count(out, "\n") != 3 {
			t.Errorf("%s: the answer is not a header and the two values:\n%s", name, out)
		}
		if problems != 0 || recorded != 0 {
			t.Errorf("%s: problems %d, recorded %d, want 0 and 0", name, problems, recorded)
		}
	}
	// A string keeps its escapes and a number its spelling: no decode and re-encode.
	out, _, _, _ := project178(t, `{ "s" : "a\u00e9\n\"q\"" , "n" : 1.50e+2 , "z" : [ ] }`, "$.s", "$.n", "$.z")
	for _, want := range []string{`$.s = "a\u00e9\n\"q\""`, `$.n = 1.50e+2`, `$.z = []`} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("the answer lacks %q:\n%s", want, out)
		}
	}
}

// ADR-178 Decision 1. A comma separates expressions only outside a quoted
// step, whose JSON string honours \" and \; several uses of the flag keep
// their command-line order.
func TestACommaInsideABracketedNameDoesNotSplit(t *testing.T) {
	for _, c := range []struct {
		args []string
		want []string
	}{
		{[]string{`$["a,b"],$["x\"y"],$.c`}, []string{`$["a,b"]`, `$["x\"y"]`, `$.c`}},
		{[]string{`$["a\\"],$.c`}, []string{`$["a\\"]`, `$.c`}},
		{[]string{`$.b`, `$.a,$[0]`}, []string{`$.b`, `$.a`, `$[0]`}},
		{[]string{`$.a,$.b`}, []string{`$.a`, `$.b`}},
	} {
		proj, err := ParseProjections(c.args)
		if err != nil {
			t.Errorf("%q: %v", c.args, err)
			continue
		}
		var got []string
		for _, p := range proj {
			got = append(got, p.Expr)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q split into %q, want %q", c.args, got, c.want)
		}
	}
	// An escaped quote inside the step keeps the comma after it in the name:
	// the document's key is a",b, and the read finds it.
	out, problems, _, _ := project178(t, `{"a\",b":1}`, `$["a\",b"]`)
	if want := `$["a\",b"] = 1` + "\n"; problems != 0 || !strings.HasSuffix(out, want) {
		t.Errorf("a name holding an escaped quote and a comma: problems %d, want %q last:\n%s", problems, want, out)
	}
}

// ADR-178 Decision 4. An expression the document does not hold is named with
// the resolver's reason and the rest are still printed; a file of several JSON
// values prints its header and NOT JSON once; either is a problem, and a
// problem makes the read exit 1.
func TestAMissingJSONPathIsIncomplete(t *testing.T) {
	out, problems, _, recorded := project178(t, `{"a":1,"b":[true]}`, "$.a,$.nope,$.b[0],$.b[3]")
	for _, want := range []string{
		"$.a = 1\n",
		`$.nope  NOT FOUND: no name "nope" in the object at $, line 1` + "\n",
		"$.b[0] = true\n",
		"$.b[3]  NOT FOUND: index 3 is past the end of the 1-element array at $.b, line 1\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the answer lacks %q:\n%s", want, out)
		}
	}
	if problems != 2 || recorded != 0 {
		t.Errorf("problems %d, recorded %d, want 2 and 0", problems, recorded)
	}

	out, problems, _, recorded = project178(t, "{\"a\":1}\n{\"a\":2}\n", "$.a,$.b")
	if !strings.HasPrefix(out, "==> f.json  ") || strings.Count(out, "NOT JSON: ") != 1 || !strings.Contains(out, "at line 2\n") || strings.Contains(out, "$.a") {
		t.Errorf("a JSON Lines file did not print its header and one NOT JSON line:\n%s", out)
	}
	if problems != 1 || recorded != 0 {
		t.Errorf("JSON Lines: problems %d, recorded %d, want 1 and 0", problems, recorded)
	}
}
