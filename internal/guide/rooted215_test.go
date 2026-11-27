package guide

import (
	"strings"
	"testing"
)

// ADR-215 T2. The full instructions say a plan path is relative to the root and
// that a rooted one is refused, and the core stays under 300 words (ADR-141).
func TestTheInstructionsSayAPlanPathIsRelativeToTheRoot(t *testing.T) {
	full := CLI()
	for _, want := range []string{
		"Every path in a plan is relative to the root",
		"begins with a separator",
		"names a drive",
		"is refused, for every op",
		"mrw read takes a real path, absolute included",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("the full instructions lost %q", want)
		}
	}
	if n := len(strings.Fields(Core())); n > 300 {
		t.Errorf("the core is %d words, want at most 300", n)
	}
}
