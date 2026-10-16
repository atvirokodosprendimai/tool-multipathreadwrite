package read

import (
	"reflect"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-161 Decision 10. A ranged read serves the line after each range that ends
// before the file does, numbered and licensed, so a multi-line replace of
// exactly the lines read is not refused for the line below it (ADR-052). A
// whole-file read, `$` and a range ending on the last line are unchanged;
// --max-lines counts the extra line; --quiet licenses it; ADR-136's note is
// gone. The fixture's a.go has nine lines.
func TestARangedReadServesTheLineAfterIt(t *testing.T) {
	root, opt := fixture(t)
	opt.NextLine = true
	for _, c := range []struct {
		spec   string
		header string // the range line served
		spans  [][2]int
	}{
		{"a.go:3", "@@ 3-4", [][2]int{{3, 4}}},
		{"a.go:3-5", "@@ 3-6", [][2]int{{3, 6}}},
		{"a.go:3,+1", "@@ 3-5", [][2]int{{3, 5}}},
		{"a.go:/Foo/", "@@ 3-4", [][2]int{{3, 4}}},
		{"a.go:/doThing/", "@@ 4-5", [][2]int{{4, 5}}},
		{"a.go:/Foo/,/^}/", "@@ 3-6", [][2]int{{3, 6}}},
		{"a.go:$", "@@ 9-9", [][2]int{{9, 9}}},
		{"a.go:8-9", "@@ 8-9", [][2]int{{8, 9}}},
	} {
		sp, err := ParseSpec(c.spec)
		if err != nil {
			t.Fatal(err)
		}
		var sb strings.Builder
		obs, problems := Run(&sb, root, []Spec{sp}, opt)
		out := sb.String()
		if problems != 0 || !strings.Contains(out, c.header+"\n") {
			t.Errorf("%s: problems %d, want the range line %q:\n%s", c.spec, problems, c.header, out)
		}
		if got := obs["a.go"].Spans; !reflect.DeepEqual(got, c.spans) {
			t.Errorf("%s: licensed %v, want %v", c.spec, got, c.spans)
		}
		if strings.Contains(out, "needs a served line after") {
			t.Errorf("%s: ADR-136's note is still printed:\n%s", c.spec, out)
		}
	}

	// A whole-file read is unchanged: the whole file, licensed whole.
	if obs, _ := runObs(t, root, opt, "a.go"); obs["a.go"].Spans != nil {
		t.Errorf("a whole-file read licensed %v, want the whole file", obs["a.go"].Spans)
	}

	// --max-lines counts the extra line: 3-5 becomes 3-6, capped at 2 lines.
	two := 2
	capped := opt
	capped.MaxLines = &two
	obs, out := runObs(t, root, capped, "a.go:3-5")
	if got := obs["a.go"].Spans; !reflect.DeepEqual(got, [][2]int{{3, 4}}) || !strings.Contains(out, "2 more line(s) withheld") {
		t.Errorf("--max-lines 2 over 3-5: licensed %v, want [[3 4]] and 2 withheld:\n%s", got, out)
	}

	// --quiet licenses the extra line without printing it.
	quiet := opt
	quiet.Quiet = true
	obs, out = runObs(t, root, quiet, "a.go:3-5")
	if got := obs["a.go"].Spans; !reflect.DeepEqual(got, [][2]int{{3, 6}}) || !strings.Contains(out, "-- licensed 4 line(s) in 1 file(s)") {
		t.Errorf("--quiet over 3-5: licensed %v, want [[3 6]] counted as 4:\n%s", got, out)
	}
}

func runObs(t *testing.T, root string, opt Options, specs ...string) (map[string]seen.Observation, string) {
	t.Helper()
	var parsed []Spec
	for _, s := range specs {
		sp, err := ParseSpec(s)
		if err != nil {
			t.Fatal(err)
		}
		parsed = append(parsed, sp)
	}
	var sb strings.Builder
	obs, _ := Run(&sb, root, parsed, opt)
	return obs, sb.String()
}
