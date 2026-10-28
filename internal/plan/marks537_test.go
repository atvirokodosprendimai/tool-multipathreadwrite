package plan

import (
	"strings"
	"testing"
)

// Issue #537. Every leading U+FEFF is dropped for the header test, on the first
// line and on a later fragment's alike (a later one swallowed as a body was the
// silent corruption TestConcatenatedBOMFragmentsStayTwoHunks guards); a mark
// past the header test stays content.
func TestEveryLeadingByteOrderMarkIsDroppedForTheHeaderTest(t *testing.T) {
	doc := "\ufeff\ufeff@@ f.txt 1 replace\nFIRST\n" + "\ufeff\ufeff\ufeff@@ f.txt 3 replace\nTHIRD\n"
	hunks, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("a plan with two leading marks was refused: %v", err)
	}
	if len(hunks) != 2 || len(hunks[0].Body) != 1 || hunks[0].Body[0] != "FIRST" || hunks[1].Body[0] != "THIRD" {
		t.Errorf("hunks = %+v, want two hunks with FIRST and THIRD", hunks)
	}
	hunks, err = Parse(strings.NewReader("@@ f.txt 1 replace\n\ufeff\ufeffKEEP\n"))
	if err != nil || len(hunks) != 1 || hunks[0].Body[0] != "\ufeff\ufeffKEEP" {
		t.Errorf("marks in a body: %+v, %v, want both kept as content", hunks, err)
	}
}
