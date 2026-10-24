package main

import (
	"strings"
	"testing"
)

// ADR-171 Decision 5. mrw write --help opens its body= lines with the fact that
// a count is optional for an ordinary body, and keeps the line-count sentence.
func TestTheWriteHelpSaysBodyIsOptional(t *testing.T) {
	out, code := runIn(t, t.TempDir(), "write", "--help")
	if code != 0 {
		t.Fatalf("write --help: exit %d:\n%.300s", code, out)
	}
	for _, want := range []string{"body= is optional for an ordinary body", "body= is a line count, not a character count"} {
		if !strings.Contains(out, want) {
			t.Errorf("write --help lost %q:\n%s", want, out)
		}
	}
}
