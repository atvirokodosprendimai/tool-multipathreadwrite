package adversarial

import (
	"strings"
	"testing"

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
