package guide

import (
	"strings"
	"testing"
)

// ADR-218 Decision 7. The full instructions say what a caller does when a
// script runs its commands: return the lines to be edited, and under
// MRW_SHOWN=off name a line by pattern or anchor= and send a multi-line replace
// as a search_replace block. The core is not grown by it.
func TestTheInstructionsSayWhatAScriptedCallerDoes(t *testing.T) {
	core, full := Core(), CLI()
	if n := len(strings.Fields(core)); n >= 300 {
		t.Errorf("the core is %d words, want under 300", n)
	}
	if strings.Contains(core, "MRW_SHOWN") {
		t.Errorf("the core names the switch; Decision 7 leaves it to the full text")
	}
	for _, want := range []string{
		"MRW_SHOWN=off",
		"what reaches you is what the script returns",
		"records no new showing",
		"a showing already in the ledger for that version still counts",
		"whose SEARCH and REPLACE end in the same line",
		"mrw seen --forget",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("the full instructions do not hold %q", want)
		}
	}
}
