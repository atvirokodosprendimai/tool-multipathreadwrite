package adversarial

import (
	"bytes"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// KNOWN GAP, decided rather than defective: a path that itself contains a colon
// cannot be addressed. ParseSpec's doc comment used to claim the opposite ("so
// a path containing a colon still works") while internal/read/read_test.go
// asserted that a non-range tail is an ERROR. The test won and the comment was
// corrected, because reading an unparseable tail as part of the filename turns
// a typo'd range into a "no such file" — and a typo is much likelier than a
// colon in a Go source path.
//
// This test holds that decision still. If mrw ever needs colon paths, the
// escape hatch to build is an explicit separator, not a guess.
func TestKnownGap_APathContainingAColonIsNotAddressable(t *testing.T) {
	if _, err := read.ParseSpec("weird:name.go"); err == nil {
		t.Errorf("weird:name.go now parses; the doc comment and read_test.go disagree about that, " +
			"so decide which one is right rather than letting this test drift")
	}
}

// A range that names lines the file does not have is a caller working from a
// stale picture — the premise ADR-002 protects on the write side.
func TestARangePastTheEndOfTheFileIsReported(t *testing.T) {
	root := tree(t, map[string]string{"short.txt": "one\ntwo\n"})

	var buf bytes.Buffer
	_, problems := read.Run(&buf, root,
		[]read.Spec{{Path: "short.txt", Ranges: []read.Range{{Start: 90, End: 99}}}}, read.Options{})

	if problems == 0 && !strings.Contains(buf.String(), "90") {
		t.Errorf("lines 90-99 of a 2-line file were served as silence: problems=%d, output:\n%s",
			problems, buf.String())
	}
}

// A pattern that matches nothing is the read-side twin of a missed anchor. The
// README's framing is that a read returning nothing is visible — which holds
// only if the output says the pattern found nothing.
func TestAPatternThatMatchesNothingSaysSo(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	spec, err := read.ParseSpec("a.go:/func NoSuchFunction/")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, problems := read.Run(&buf, root, []read.Spec{spec}, read.Options{})

	if problems == 0 && !strings.Contains(buf.String(), "NoSuchFunction") {
		t.Errorf("a pattern matching nothing produced no report of that: problems=%d, output:\n%s",
			problems, buf.String())
	}
}
