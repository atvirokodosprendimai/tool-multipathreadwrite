package adversarial

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// KNOWN GAP: a path that itself contains a colon cannot be addressed.
//
// ParseSpec's doc comment used to claim the opposite ("so a path containing a
// colon still works") while internal/read/read_test.go asserted that a
// non-range tail is an ERROR. Those cannot both hold, and no ADR settles it.
// The comment was corrected to match the shipped behaviour, on the reasoning
// that reading an unparseable tail as part of the filename turns a typo'd range
// into a "no such file" — and a typo is much likelier than a colon in a Go
// source path.
//
// That is a reading of the evidence, not a decision anyone made. This test pins
// it so the next person to want colon paths has to change it deliberately; the
// escape hatch to build then is an explicit separator, not a guess.
func TestKnownGap_APathContainingAColonIsNotAddressable(t *testing.T) {
	if _, err := read.ParseSpec("weird:name.go"); err == nil {
		t.Errorf("weird:name.go now parses; decide whether ParseSpec's doc comment or " +
			"read_test.go is right and delete this test")
	}
}

// A range naming lines the file does not have is a caller working from a stale
// picture — the premise ADR-002 protects on the write side. Both halves are
// asserted separately: a count with no diagnostic is as useless as a diagnostic
// the exit status does not back.
func TestARangePastTheEndOfTheFileIsReported(t *testing.T) {
	root := tree(t, map[string]string{"short.txt": "one\ntwo\n"})

	// Built through ParseSpec rather than by hand: a Range carries the text it
	// was written as, and Run prints that text in its diagnostic. A hand-built
	// Range has none, and the report then reads "no match for  " — worth
	// knowing if you ever drive this package directly.
	spec, err := read.ParseSpec("short.txt:90-99")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, problems := read.Run(&buf, root, []read.Spec{spec}, read.Options{})

	if problems == 0 {
		t.Error("lines 90-99 of a 2-line file were served without being counted as a problem")
	}
	if !strings.Contains(buf.String(), "90") {
		t.Errorf("the output does not name the range that could not be served:\n%s", buf.String())
	}
}

// The read-side twin of a missed anchor. The README's framing is that a read
// returning nothing is visible — which holds only if the output says the
// pattern found nothing AND the caller's exit status reflects it.
func TestAPatternThatMatchesNothingSaysSo(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	spec, err := read.ParseSpec("a.go:/func NoSuchFunction/")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, problems := read.Run(&buf, root, []read.Spec{spec}, read.Options{})

	if problems == 0 {
		t.Error("a pattern matching nothing was not counted as a problem")
	}
	if !strings.Contains(buf.String(), "NoSuchFunction") {
		t.Errorf("the output does not name the pattern that matched nothing:\n%s", buf.String())
	}
}

// DECISION POINT, pinned rather than changed: `read` exits 1 whenever its
// answer is incomplete — including when a --max-lines cap the caller ASKED FOR
// fires. internal/read/read_test.go::TestMaxLinesReportsWhatItWithheld already
// asserts that, so it is the project's stated intent and not an accident.
//
// The argument against it is real and worth writing down: an agent that runs
// `mrw read --max-lines 200 …` routinely gets a non-zero status on every
// ordinary call, and a status that cries wolf is one people learn to ignore —
// the exact harm this tool is built around. The argument FOR it is the tool's
// own thesis: an incomplete answer must be visible, and the exit code is the
// machine-readable half of visible. "Some of what you asked for is missing" is
// true of a withheld span and of an unreadable file alike.
//
// Left as it is, because changing it is a served-path change to the exit
// contract and that is M's call, not mine. These tests hold the current answer
// in BOTH directions so it cannot drift unnoticed while the question is open.
func TestKnownGap_ARequestedCapCountsAsAProblem(t *testing.T) {
	root := tree(t, map[string]string{"big.txt": strings.Repeat("line\n", 40)})

	var buf bytes.Buffer
	_, problems := read.Run(&buf, root, []read.Spec{{Path: "big.txt"}}, read.Options{MaxLines: 5})

	if problems == 0 {
		t.Error("a fired --max-lines cap no longer counts as a problem — if that is deliberate, " +
			"say so in the README's exit table and delete this test")
	}
	if !strings.Contains(buf.String(), "withheld") {
		t.Errorf("the cap fired without saying so:\n%s", buf.String())
	}
}

// The same for a span withheld whole rather than cut in half — the other branch
// of the budget check, and the one a multi-range read hits first.
func TestKnownGap_AWhollyWithheldSpanCountsAsAProblem(t *testing.T) {
	root := tree(t, map[string]string{"big.txt": strings.Repeat("line\n", 40)})

	spec, err := read.ParseSpec("big.txt:1-3,20-25")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, problems := read.Run(&buf, root, []read.Spec{spec}, read.Options{MaxLines: 3})

	if problems == 0 {
		t.Error("a wholly withheld span no longer counts as a problem")
	}
	if !strings.Contains(buf.String(), "WITHHELD") {
		t.Errorf("the withheld span was not reported:\n%s", buf.String())
	}
}

// Whatever is decided above, these two must keep their verdict: the answer the
// caller asked for does not exist at all.
func TestAnUnreadableFileAndAnUnmatchedPatternStayProblems(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	if _, problems := read.Run(io.Discard, root, []read.Spec{{Path: "nosuch.go"}}, read.Options{}); problems == 0 {
		t.Error("an unreadable file is not a problem any more")
	}

	spec, err := read.ParseSpec("a.go:/func NoSuchFunction/")
	if err != nil {
		t.Fatal(err)
	}
	if _, problems := read.Run(io.Discard, root, []read.Spec{spec}, read.Options{}); problems == 0 {
		t.Error("a pattern matching nothing is not a problem any more")
	}
}
