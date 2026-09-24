package main

import (
	"strings"
	"testing"
)

// ADR-066 (Codex review of #207, round 3). `mrw write --help` is served text
// too: after a commit failure some files may be written with a hunk failed,
// so the help must not promise that any failure writes nothing, and it must
// name the partial receipt a caller will see.
func TestWriteHelpSaysACommitFailureIsReportedPartial(t *testing.T) {
	d := writeCmd().Description
	if strings.Contains(d, "If any hunk fails, every hunk is reported and NOTHING is written") {
		t.Errorf("write help still promises nothing is written on any failure:\n%s", d)
	}
	if !strings.Contains(d, "PARTIALLY APPLIED") {
		t.Errorf("write help does not name the PARTIALLY APPLIED receipt:\n%s", d)
	}
}
