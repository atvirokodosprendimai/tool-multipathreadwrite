package read

import (
	"strings"
	"testing"
)

// ADR-136. The "a multi-line replace needs a served line after" note was
// printed at each qualifying range, so a read of three ranges repeated one rule
// three times. It is printed once per call, at the first range that qualifies; a
// read of one range, the common case, is unchanged.
func TestTheNeighbourNoteIsPrintedOncePerRead(t *testing.T) {
	root, opt := fixture(t)
	const note = "needs a served line after"
	out, problems := run(t, root, opt, "a.go:3-5", "a.go:7-8")
	if problems != 0 {
		t.Fatalf("problems=%d\n%s", problems, out)
	}
	if n := strings.Count(out, note); n != 1 {
		t.Errorf("two multi-line ranges printed the note %d times, want once:\n%s", n, out)
	}
	if !strings.Contains(out, "3-5") || !strings.Contains(out, "after 5") {
		t.Errorf("the one note is not the first range's:\n%s", out)
	}
	one, _ := run(t, root, opt, "a.go:3-5")
	if strings.Count(one, note) != 1 {
		t.Errorf("a single multi-line range lost its note:\n%s", one)
	}
}
