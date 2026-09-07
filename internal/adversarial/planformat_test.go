package adversarial

import (
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// body=N is the escape hatch that lets a body contain lines starting with
// "@@ ". It is a COUNT, and a count the document does not honour is the
// caller's picture of their own plan being wrong — the same class of mistake as
// a drifted line number, which this format fails loudly on.
func TestBodyCountShorterThanTheDocumentIsRejected(t *testing.T) {
	doc := "@@ a.go 1 replace body=5\none\ntwo\n"

	hunks, err := plan.Parse(strings.NewReader(doc))
	if err != nil {
		return // rejected, which is the promise
	}
	t.Errorf("body=5 with only 2 lines left in the plan parsed clean: hunk body is %q (%d lines)",
		hunks[0].Body, len(hunks[0].Body))
}

// body=0 must mean an empty body: it is the only reading under which body= is a
// count. Treating an exhausted count as "keep scanning" handed the hunk lines
// the caller wrote for something else.
func TestBodyZeroMeansAnEmptyBody(t *testing.T) {
	doc := "@@ a.go 1-2 delete body=0\n@@ b.go 1 replace\nreplacement\n"

	hunks, err := plan.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("body=0 followed by the next header should parse: %v", err)
	}
	if len(hunks) != 2 {
		t.Fatalf("want 2 hunks, got %d", len(hunks))
	}
	if len(hunks[0].Body) != 0 {
		t.Errorf("body=0 produced a %d-line body: %q", len(hunks[0].Body), hunks[0].Body)
	}
	if len(hunks[1].Body) != 1 {
		t.Errorf("the second hunk lost its body to the first: %q", hunks[1].Body)
	}
}

// The other half of the same rule: once a body= count is satisfied, a stray
// line is text the caller did not account for. Absorbing it silently is how
// body=0 came to mean "unbounded".
func TestTextAfterASatisfiedBodyCountIsRejected(t *testing.T) {
	doc := "@@ a.go 1 replace body=1\nthe body\nthis line belongs to nothing\n"

	if _, err := plan.Parse(strings.NewReader(doc)); err == nil {
		t.Error("a line after a satisfied body= count was absorbed rather than reported")
	}
}

// An empty anchor asserts nothing while looking exactly like an assertion, and
// apply reads an empty Anchor as "no anchor was given".
func TestEmptyAnchorIsNotSilentlyDropped(t *testing.T) {
	doc := "@@ a.go 1 replace anchor=\"\"\nnew line\n"

	hunks, err := plan.Parse(strings.NewReader(doc))
	if err != nil {
		return // rejected, which is one honest answer
	}
	if hunks[0].Anchor == "" {
		t.Errorf(`anchor="" parsed to an empty Anchor, which apply treats as no anchor at all: %+v`, hunks[0])
	}
}

// An anchor names a line of source, and source contains quotes. Without an
// escape the quote toggles the header's quoting state, the backslash survives
// into the value, and the caller gets a guard that cannot match the line they
// copied it from — a guard that fails loudly, but for the wrong reason, and on
// a plan that is correct.
func TestAnAnchorCanContainAQuote(t *testing.T) {
	doc := "@@ a.go 1 replace anchor=\"case \\\"anchor\\\":\"\nnew line\n"

	hunks, err := plan.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if want := `case "anchor":`; hunks[0].Anchor != want {
		t.Errorf("anchor parsed as %q, want %q", hunks[0].Anchor, want)
	}
}

// One mistake, one error. A satisfied body= count followed by a block of text
// is a single accounting mistake, and one error per line would bury every other
// hunk's diagnostic in the same report.
func TestAnUnaccountedBlockIsReportedOncePerHunk(t *testing.T) {
	doc := "@@ a.go 1 replace body=1\nthe body\nstray one\nstray two\nstray three\n"

	_, err := plan.Parse(strings.NewReader(doc))
	if err == nil {
		t.Fatal("three unaccounted lines parsed clean")
	}
	if n := strings.Count(err.Error(), "is not part of any hunk"); n != 1 {
		t.Errorf("three stray lines produced %d error(s), want 1:\n%s", n, err)
	}
}

// An OVERCOUNTED body=N is the last silent way to lose a hunk. An undercount is
// caught by the flush check and stray text after a satisfied count is caught by
// the stray check, but a count two too large simply consumes the following
// header and its body as content the caller meant to write — and the plan then
// applies, correctly by its own rules, missing an edit nobody will notice.
//
// It bit the author of the body= fix, in the commit that made it: a README hunk
// declared body=32 for a 30-line body, ate the next hunk's header, and the
// header landed IN the README as literal text.
//
// The rule that catches it without breaking the escape hatch: a counted body
// line is refused only when it parses as a COMPLETE, VALID header. Prose about
// the format does not — see the test below, which is the README's own example.
func TestAnOvercountedBodyIsRejected(t *testing.T) {
	doc := "@@ a.go 1 replace body=4\none\ntwo\n@@ b.go 1 replace\nswallowed\n@@ c.go 1 replace\nc body\n"

	_, err := plan.Parse(strings.NewReader(doc))
	if err == nil {
		t.Fatal("body=4 swallowed the next hunk's header and its body without a word")
	}
	if !strings.Contains(err.Error(), "b.go") {
		t.Errorf("the error does not name the hunk that was swallowed: %v", err)
	}
}

// The escape hatch has to keep working, and it is not hypothetical: the README
// documents anchor= with two example lines that begin with "@@ ", inside a hunk
// that edits the README. Those lines are not valid headers — the trailing prose
// is not key=value — so a body may still contain them.
func TestABodyMayContainLinesThatMerelyLookLikeHeaders(t *testing.T) {
	doc := "@@ README.md 1 replace body=2\n" +
		"@@ page.templ 12 replace anchor=\"class=\" ← what you meant\n" +
		"@@ page.templ 12 replace lines=1 ← and this\n"

	hunks, err := plan.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("a body of documentation about the format was rejected: %v", err)
	}
	if len(hunks) != 1 || len(hunks[0].Body) != 2 {
		t.Errorf("want one hunk with a 2-line body, got %d hunk(s) with %v", len(hunks), hunks[0].Body)
	}
}

// The escape hatch body= exists for, restored: a plan may write a REAL header
// as content when it says so. Without this, mrw could not apply a plan editing
// its own documentation or a contract fixture — the self-hosting cost the
// review named.
func TestARawBodyMayContainARealHeader(t *testing.T) {
	doc := "@@ doc.md 1 replace body=2 raw=true\n@@ a.go 1 replace\n@@ b.go 2-3 delete\n"

	hunks, err := plan.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("raw=true did not permit a real header in the body: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("want one hunk, got %d — the body was not taken whole", len(hunks))
	}
	if got := hunks[0].Body; len(got) != 2 || got[0] != "@@ a.go 1 replace" {
		t.Errorf("body = %q", got)
	}
}

// And the guard still bites without it: the same document, unmarked, is the
// overcount that loses a hunk.
func TestTheSameBodyWithoutRawIsRefused(t *testing.T) {
	doc := "@@ doc.md 1 replace body=2\n@@ a.go 1 replace\n@@ b.go 2-3 delete\n"

	if _, err := plan.Parse(strings.NewReader(doc)); err == nil {
		t.Error("a counted body swallowed two valid headers without raw=true")
	}
}

// raw= is deliberately not a general switch: it takes one value, so a typo is
// a parse error rather than a guard silently left on.
func TestRawTakesOnlyTrue(t *testing.T) {
	for _, v := range []string{"false", "1", "yes", ""} {
		doc := "@@ a.go 1 replace body=1 raw=" + v + "\nx\n"
		if _, err := plan.Parse(strings.NewReader(doc)); err == nil {
			t.Errorf("raw=%q was accepted", v)
		}
	}
}

// A `replace` with no body deletes the addressed lines, reports status "ok" and
// exits 0 — so a plan whose body was lost in transit (a truncated emission, an
// editor eating the last line) removes code while the receipt a hook reads says
// it succeeded. That is the exact shape this tool exists to refuse.
//
// When this test was written the parser policed the mirror image by refusing a
// `delete` WITH a body outright — one direction checked, the other not. ADR-008
// gave the other direction a meaning instead: a body on a `delete` is now the
// lines the caller expects to remove, checked against the file. The asymmetry
// this test was named for is gone; the rule it asserts is not.
//
// Nothing is lost by refusing it: deleting lines is what `delete` is for.
func TestAReplaceWithNoBodyIsRejected(t *testing.T) {
	for _, name := range []string{"last hunk in the plan", "mid-plan"} {
		doc := "@@ f.txt 2 replace\n"
		if name == "mid-plan" {
			doc += "@@ g.txt 1 replace\nreplacement\n"
		}
		t.Run(name, func(t *testing.T) {
			_, err := plan.Parse(strings.NewReader(doc))
			if err == nil {
				t.Fatal("an empty-bodied replace parsed clean; it silently deletes the line")
			}
			if !strings.Contains(err.Error(), "delete") {
				t.Errorf("the error does not point at the op that means this: %v", err)
			}
		})
	}
}

// body=0 is the same hunk written with an explicit count, and must fail the
// same way — otherwise the check is one spelling away from being bypassed.
func TestAReplaceWithAnExplicitlyEmptyBodyIsRejected(t *testing.T) {
	if _, err := plan.Parse(strings.NewReader("@@ f.txt 2 replace body=0\n@@ g.txt 1 delete\n")); err == nil {
		t.Error("replace body=0 parsed clean")
	}
}

// ADR-008. A `delete` may now carry the lines the caller expects to remove, and
// a mismatch must abandon the WHOLE plan, not just its own hunk. A partially
// applied plan is the worst outcome this tool has (ADR-001), and a guard that
// fails safe on its own hunk while its siblings land would produce exactly one.
func TestADeleteWhoseExpectedRemovalDiffersWritesNothing(t *testing.T) {
	root := tree(t, map[string]string{
		"a.go": goFile,
		"b.go": "package p\n\nvar X = 1\n",
	})

	res, err := apply.Apply(root, []apply.Input{
		// This one is fine, and must NOT land.
		{Path: "b.go", Start: 3, End: 3, Op: "replace", Body: []string{"var X = 2"}, Lines: unset, Index: 0},
		// The caller believes lines 4-5 are the body and the brace. They are
		// the body, the brace — and nothing else, because the range is right;
		// what is wrong is the picture, which is the case only a body catches.
		{Path: "a.go", Start: 4, End: 5, Op: "delete", Body: []string{"\treturn a - b", "}"}, Lines: unset, Index: 1},
	}, apply.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("a delete whose expected removal was wrong did not fail: %+v", res.Hunks)
	}
	if res.Applied {
		t.Error("the run reported applied")
	}
	if got := readFile(t, root, "b.go"); got != "package p\n\nvar X = 1\n" {
		t.Errorf("the innocent sibling hunk landed anyway: %q", got)
	}
	if got := readFile(t, root, "a.go"); got != goFile {
		t.Errorf("a.go was written: %q", got)
	}
}

// The rule ADR-006 installed is the mirror image of this one and must survive
// it: `replace` with no body is still refused, because it deletes while
// reporting a replacement. Only the delete direction changed meaning.
func TestAReplaceWithNoBodyIsStillRejectedNowThatDeleteTakesOne(t *testing.T) {
	if _, err := plan.Parse(strings.NewReader("@@ f.txt 2 replace\n")); err == nil {
		t.Error("an empty-bodied replace parsed clean")
	}
	if _, err := plan.Parse(strings.NewReader("@@ f.txt 2 delete\nx\n")); err != nil {
		t.Errorf("a delete with an expected body was rejected: %v", err)
	}
}

// TestTheEngineAndTheParserRefuseInTheSameWords pins the half of ADR-030 that
// its first cut only asserted in prose. The engine copies plan.validate's
// message strings, and the record claimed that keeps the two sites honest — but
// the engine-side test hardcoded those strings and never invoked the parser, so
// rewording validate alone left everything green. The review of PR #130 said so,
// and this is the fix: the expected text is TAKEN FROM THE PARSER at run time.
//
// One row per VERBATIM-mirrored validate branch. All ten are here.
//
// ⚠ An earlier cut said the tenth — `create` with a relative end — was
// unreachable from a plan document, because validate's numeric-address check
// fires first for `1,+2`. That was true of the spelling tried and false as a
// claim: `00,+2` reaches it, since CutRelative refuses only the exact bases "0"
// and "-" while "00" passes its digit check and converts to numeric zero. The
// review of PR #130 found the spelling. "I could not reach it" is not "it is
// unreachable", and this record has now made that mistake twice.
//
// The two remaining validate returns — replace at line zero, and a reversed
// range — are answered by the engine's own semantic checks with its own wording,
// not by a verbatim mirror, so they are not parity rows either.
//
// Each row is one malformed plan and the apply.Input a direct caller would build
// for the same mistake. Reword either site alone and this goes red.
func TestTheEngineAndTheParserRefuseInTheSameWords(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		in   apply.Input
	}{
		{"replace with an empty body", "@@ f.txt 1-2 replace\n",
			apply.Input{Path: "f.txt", Op: "replace", Start: 1, End: 2, Lines: unset}},
		{"insert-after over a range", "@@ f.txt 1-3 insert-after\nX\n",
			apply.Input{Path: "f.txt", Op: "insert-after", Start: 1, End: 3, Body: []string{"X"}, Lines: unset}},
		{"insert-before over a range", "@@ f.txt 1-3 insert-before\nX\n",
			apply.Input{Path: "f.txt", Op: "insert-before", Start: 1, End: 3, Body: []string{"X"}, Lines: unset}},
		{"create with a line address", "@@ n.txt 1 create\nX\n",
			apply.Input{Path: "n.txt", Op: "create", Start: 1, End: 1, Body: []string{"X"}, Lines: unset}},
		{"create with anchor=", "@@ n.txt - create anchor=\"zzz\"\nX\n",
			apply.Input{Path: "n.txt", Op: "create", Body: []string{"X"}, Lines: unset, Anchor: "zzz"}},
		{"create with lines=", "@@ n.txt - create lines=5\nX\n",
			apply.Input{Path: "n.txt", Op: "create", Body: []string{"X"}, Lines: 5}},
		{"a body-less create", "@@ n.txt - create\n",
			apply.Input{Path: "n.txt", Op: "create", Lines: unset}},
		{"create with a pattern address", "@@ n.txt /a/ create\nX\n",
			apply.Input{Path: "n.txt", Op: "create", Body: []string{"X"}, Lines: unset,
				StartPat: regexp.MustCompile(`^a$`)}},
		{"insert-after over a pattern range", "@@ f.txt /a/,/c/ insert-after\nX\n",
			apply.Input{Path: "f.txt", Op: "insert-after", Body: []string{"X"}, Lines: unset,
				StartPat: regexp.MustCompile(`^a$`), EndPat: regexp.MustCompile(`^c$`)}},
		{"insert-after with a relative end", "@@ f.txt 1,+2 insert-after\nX\n",
			apply.Input{Path: "f.txt", Op: "insert-after", Start: 1, End: 1, RelEnd: 2,
				Body: []string{"X"}, Lines: unset}},
		{"insert-after with an empty body", "@@ f.txt 1 insert-after\n",
			apply.Input{Path: "f.txt", Op: "insert-after", Start: 1, End: 1, Lines: unset}},
		{"insert-before with an empty body", "@@ f.txt 1 insert-before\n",
			apply.Input{Path: "f.txt", Op: "insert-before", Start: 1, End: 1, Lines: unset}},
		// ⚠ `00`, not `0`. CutRelative refuses the exact bases "0" and "-", so
		// `0,+2` is refused earlier with a different message — but "00" passes
		// its digit check and ParseAddr converts it to numeric zero, which slips
		// past the create-address check and lands on the relative-end branch.
		// An earlier cut of this test declared this branch UNREACHABLE from a
		// plan document and said so in a comment; the review of PR #130 found
		// the spelling that reaches it. Measured.
		{"create with a relative end", "@@ n.txt 00,+2 create\nX\n",
			apply.Input{Path: "n.txt", Op: "create", Start: 0, End: 0, RelEnd: 2,
				Body: []string{"X"}, Lines: unset}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := plan.Parse(strings.NewReader(c.doc))
			if err == nil {
				t.Fatalf("the parser accepted %q — this test's premise is that it refuses it", c.doc)
			}
			// "plan has 1 error(s):\n  line 1: <validate's own words>"
			_, want, found := strings.Cut(err.Error(), "line 1: ")
			if !found {
				t.Fatalf("cannot find the parser's message in %q", err)
			}
			want = strings.TrimSpace(want)

			root := tree(t, map[string]string{"f.txt": "a\nb\nc\nd\n"})
			res, aerr := apply.Apply(root, []apply.Input{c.in}, apply.Options{Force: true})
			if aerr != nil {
				t.Fatal(aerr)
			}
			if res.Failed != 1 {
				t.Fatalf("the engine accepted what the parser refuses: failed=%d", res.Failed)
			}
			if got := res.Hunks[0].Reason; got != want {
				t.Errorf("the two sites do not say the same thing about one mistake:\n  parser: %s\n  engine: %s", want, got)
			}
		})
	}
}
