package read

import (
	"regexp"
	"strings"
	"testing"
)

// ADR-164 Decision 7. The window centred on where the match starts, so a match
// no longer than the window was still cut: --max-cols 8 over "Alpha NEEDLE x"
// showed "pha NEED". It centres on the match itself: a match that fits is
// whole, a longer one starts the window, and an empty one does not panic.
func TestAMatchThatFitsIsWholeInItsWindow(t *testing.T) {
	needle := []*regexp.Regexp{regexp.MustCompile("NEEDLE")}
	text := func(w string) string { return w[:strings.Index(w, "  [cols")] }
	if got := window("Alpha NEEDLE x", 8, needle); !strings.Contains(text(got), "NEEDLE") {
		t.Errorf("a 6-character match in an 8-column window must be whole: %q", got)
	}
	long := []*regexp.Regexp{regexp.MustCompile("M{30}")}
	if got := window(strings.Repeat("a", 50)+strings.Repeat("M", 30)+strings.Repeat("b", 50), 20, long); text(got) != "…"+strings.Repeat("M", 20)+"…" || !strings.Contains(got, "[cols 51-70 of 130]") {
		t.Errorf("a 30-character match in a 20-column window must start the window: %q", got)
	}
	if got := window(strings.Repeat("a", 50)+"NEEDLE"+strings.Repeat("b", 50), 6, needle); text(got) != "…NEEDLE…" {
		t.Errorf("a window as wide as the match must be exactly the match: %q", got)
	}
	empty := []*regexp.Regexp{regexp.MustCompile("x*")}
	if got := window(strings.Repeat("a", 50), 10, empty); !strings.Contains(got, "[cols 1-10 of 50]") {
		t.Errorf("an empty match at the start: %q", got)
	}
}
