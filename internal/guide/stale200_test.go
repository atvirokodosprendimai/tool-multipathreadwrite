package guide

import (
	"strings"
	"testing"
)

// ADR-200 T1. A file mrw just wrote is wholly known, which does not make the
// numbers of an earlier read still name the same lines: the full instructions
// say so, and the core, which has a word limit, is left as it is.
func TestInstructionsSayNumbersMoveAfterAWrite(t *testing.T) {
	const must = "a number from an earlier read names another line"
	if !strings.Contains(CLI(), must) {
		t.Errorf("CLI() does not say %q", must)
	}
	if strings.Contains(Core(), must) {
		t.Errorf("Core() carries the sentence, and it has a word limit")
	}
}
