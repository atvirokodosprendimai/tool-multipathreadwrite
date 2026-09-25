package guide

import (
	"regexp"
	"testing"
)

// ADR-070 T1. Blind reading 04: four of seven trials put body= on a line of
// its own, because mrw instructions described body= and never showed it on a
// header. The text must carry one worked header that counts its body.
func TestInstructionsShowBodyOnAHeader(t *testing.T) {
	// A replace or insert header counting a non-empty body: the existing
	// "@@ path 0 create ... body=0" sentence is not a worked header.
	header := regexp.MustCompile(`@@ \S+ \d+(-\d+)? (replace|insert-after|insert-before) [^\n]*\bbody=[1-9]\d*`)
	if !header.MatchString(CLI()) {
		t.Errorf("mrw instructions shows no @@ header carrying body=<n>:\n%s", CLI())
	}
}
