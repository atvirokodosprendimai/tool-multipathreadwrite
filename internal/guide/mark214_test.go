package guide

import (
	"strings"
	"testing"
)

// ADR-214 T2. The full instructions say a file's leading byte order mark is part
// of line 1, that an edit keeps it at byte zero, how a pattern and an old side
// meet it, and the one-plan way to remove it; the core stays under 300 words
// (ADR-141).
func TestTheInstructionsSayAFilesMarkIsKept(t *testing.T) {
	full := CLI()
	for _, want := range []string{
		"leading UTF-8 byte order mark",
		"part of line 1",
		"^one misses it while ^.one and ^\\x{FEFF}one match",
		"keeps the mark at byte zero",
		"printf '\\357\\273\\277'",
		"in ONE plan, unlink the marked path and rename",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("the full instructions lost %q", want)
		}
	}
	if n := len(strings.Fields(Core())); n >= 300 {
		t.Errorf("the core is %d words, want under 300", n)
	}
}
