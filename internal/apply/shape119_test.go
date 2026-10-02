package apply

import (
	"fmt"
	"strings"
	"testing"
)

// shapeCase is one replace and what its receipt should say.
type shapeCase struct {
	path       string
	orig       []string
	start, end int
	body       []string
}

// fieldFailures are the three closer field reports, built to the offsets
// BACKLOG records under "From the field reports": Blade replaced 7 wrote 7-9
// orphan at 10; HTML replaced 5 wrote 5-7 orphan at 8; markdown fence replaced
// 333 wrote 333-336 orphan at 340. The orphan's new line number fixes which
// original line survived; the text around it is a reconstruction, the same one
// docs/break/shape-hints/stress.py measures. The two indentation reports are
// there too, and are not here: the indent hint was withdrawn (ADR-119).
func fieldFailures() []shapeCase {
	prose := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf("line %d of prose.", i+1)
		}
		return out
	}
	md := append(prose(332), "```bash", "make build", "make test", "make install", "```")
	md = append(md, prose(3)...)
	return []shapeCase{
		{"view.blade.php", []string{"<div>", "  <h1>{{ $title }}</h1>", "", "  <ul>", "  @foreach ($xs as $x)", "  @endforeach",
			"    @if ($items)", "    @endif", "  </ul>", "</div>"},
			7, 7, []string{"    @if ($items->isNotEmpty())", "        <li>{{ $items->first() }}</li>", "    @endif"}},
		{"page.html", []string{"<main>", "  <h1>Title</h1>", "  <p>Intro</p>", "", `  <div class="card">`, "  </div>", "</main>"},
			5, 5, []string{`  <div class="card card-wide">`, "    <p>Body</p>", "  </div>"}},
		{"README.md", md, 333, 333, []string{"```bash", "go build ./...", "go test ./...", "```"}},
	}
}

func (c shapeCase) input() Input {
	return Input{Path: c.path, Start: c.start, End: c.end, Op: "replace", Body: c.body, Lines: -1,
		Anchor: strings.TrimSpace(c.orig[c.start-1])}
}

func applyShape(t *testing.T, opt Options, cases ...shapeCase) Result {
	t.Helper()
	root := t.TempDir()
	var in []Input
	for _, c := range cases {
		write(t, root, c.path, strings.Join(c.orig, "\n")+"\n")
		in = append(in, c.input())
	}
	res, err := Apply(root, in, opt)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestTheFieldFailuresEachCarryAHint(t *testing.T) {
	// The line each survivor sits on in the written file, from the reports.
	want := map[string]int{"view.blade.php": 10, "page.html": 8, "README.md": 340}
	for _, c := range fieldFailures() {
		t.Run(c.path, func(t *testing.T) {
			h := applyShape(t, Options{}, c).Hunks[0]
			if h.Status != StatusOK {
				t.Fatalf("status %s: %s", h.Status, h.Reason)
			}
			if !strings.HasPrefix(h.Closer, fmt.Sprintf("line %d ", want[c.path])) {
				t.Fatalf("%s replaced %d-%d: closer %q; the bar asks this field failure to name the survivor at line %d",
					c.path, c.start, c.end, h.Closer, want[c.path])
			}
		})
	}
}

func TestACorrectReplaceIncludingItsCloserReportsNone(t *testing.T) {
	blade := fieldFailures()[0]
	blade.end = 8 // through the @endif the field report left behind
	html := fieldFailures()[1]
	html.end = 6
	// The review of #322: an inner block replaced through its own `}`, the
	// outer `}` right below. The tokens match; the range already ended in it.
	nested := shapeCase{"n.go", []string{"package n", "", "func A() {", "\tif x {", "\t\tf()", "\t}", "}"},
		4, 6, []string{"\tif y {", "\t\tg()", "\t}"}}
	fence := shapeCase{"n.md", []string{"intro", "```go", "x := 1", "```", "", "```sh", "ls", "```"},
		2, 4, []string{"```go", "x := 2", "```"}}
	for _, c := range []shapeCase{blade, html, nested, fence} {
		h := applyShape(t, Options{}, c).Hunks[0]
		if h.Status != StatusOK || h.Closer != "" {
			t.Fatalf("%s %d-%d: status %s, closer %q; a replace through its own closer is clean", c.path, c.start, c.end, h.Status, h.Closer)
		}
	}
}

func TestACloserNeedsACloserShapedLineWithinFour(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    shapeCase
		want bool
	}{
		// A repeated statement is usually a repeated statement, not a
		// survivor; only a closer-shaped line can be one.
		{"repeated statement", shapeCase{"a.go", []string{"func f() {", "\tx++", "\tx++", "}"}, 2, 2, []string{"\ty := 1", "\tx++"}}, false},
		{"fourth non-blank line", shapeCase{"b.go", []string{"if a {", "\tx()", "\ty()", "", "\tz()", "}"}, 1, 1, []string{"if b {", "}"}}, true},
		{"fifth non-blank line", shapeCase{"c.go", []string{"if a {", "\tw()", "\tx()", "\ty()", "\tz()", "}"}, 1, 1, []string{"if b {", "}"}}, false},
	} {
		h := applyShape(t, Options{}, tc.c).Hunks[0]
		if (h.Closer != "") != tc.want {
			t.Fatalf("%s: closer %q, want a hint %v", tc.name, h.Closer, tc.want)
		}
	}
}

func TestACloserIsJudgedOnTheWrittenFile(t *testing.T) {
	blade := fieldFailures()[0]
	root := t.TempDir()
	write(t, root, blade.path, strings.Join(blade.orig, "\n")+"\n")
	// The same plan also deletes the @endif that would have survived.
	del := Input{Path: blade.path, Start: 8, End: 8, Op: "delete", Lines: -1}
	res, err := Apply(root, []Input{blade.input(), del}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if h := res.Hunks[0]; h.Status != StatusOK || h.Closer != "" {
		t.Fatalf("status %s, closer %q; a later hunk removed the old closer, so nothing survives to repeat", h.Status, h.Closer)
	}
}

func TestACloserRunsOnProse(t *testing.T) {
	c := shapeCase{"notes.md", []string{"intro", "```go", "x := 1", "```", "outro"}, 3, 3, []string{"x := 2", "```"}}
	if h := applyShape(t, Options{}, c).Hunks[0]; h.Closer == "" {
		t.Fatal("no closer on a .md fence closed twice; closer runs on prose, which is the case it exists for")
	}
}

func TestHintsDoNotMoveAdvisoriesOrThePattern(t *testing.T) {
	hinted := fieldFailures()[1] // a closer, and no bracket in an HTML tag to move the balance
	balanced := shapeCase{"main.go", []string{"package main", "", "func f() {", "\treturn", "}"}, 4, 4, []string{"\tif x {"}}
	res := applyShape(t, Options{}, hinted, balanced)
	if res.Hunks[0].Closer == "" || res.Hunks[0].Balance != "" {
		t.Fatalf("hinted hunk: closer %q, balance %q; want a closer and no balance row", res.Hunks[0].Closer, res.Hunks[0].Balance)
	}
	if res.Hunks[1].Balance == "" || res.Hunks[1].Closer != "" {
		t.Fatalf("balanced hunk: balance %q, closer %q; want a balance row and no hint", res.Hunks[1].Balance, res.Hunks[1].Closer)
	}
	// advisories keeps counting balance rows only (ADR-111), and the pattern
	// line and pricing read advisories, so a hint that leaked into it would
	// move both.
	if res.Advisories != 1 || res.Hints != 1 {
		t.Fatalf("advisories %d, hints %d; want 1 and 1", res.Advisories, res.Hints)
	}
}

func TestASkippedHunkCarriesNoHints(t *testing.T) {
	hinted := fieldFailures()[0]
	broken := shapeCase{"other.go", []string{"package other", "var x = 1"}, 2, 2, []string{"var x = 2"}}
	root := t.TempDir()
	write(t, root, hinted.path, strings.Join(hinted.orig, "\n")+"\n")
	write(t, root, broken.path, strings.Join(broken.orig, "\n")+"\n")
	bad := broken.input()
	bad.Anchor = "not on that line"
	res, err := Apply(root, []Input{hinted.input(), bad}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if h := res.Hunks[0]; h.Status != StatusSkipped || h.Closer != "" || res.Hints != 0 {
		t.Fatalf("status %s, closer %q, hints %d; a skipped hunk was not written and carries no hint", h.Status, h.Closer, res.Hints)
	}
}

func TestStrictBalanceNeverRefusesOnAHint(t *testing.T) {
	ff := fieldFailures()
	res := applyShape(t, Options{StrictBalance: true}, ff[0], ff[1])
	for _, h := range res.Hunks {
		if h.Status != StatusOK {
			t.Fatalf("%s: status %s under --strict-balance: %s; a hint is advice and never refuses", h.Path, h.Status, h.Reason)
		}
	}
	if res.Hints != 2 {
		t.Fatalf("hints %d; the hints still report under --strict-balance", res.Hints)
	}
}
