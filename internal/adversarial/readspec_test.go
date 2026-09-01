package adversarial

import (
	"bytes"
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
