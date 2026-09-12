package main

import (
	"strings"
	"testing"
)

// TestWriteHelpNamesHowToQuoteAHeaderOption is ADR-040 T1: a PATH caller who
// reads write --help must learn how to spell a spaced anchor=, that body= is
// a line count (Python str is characters), that lines= is a range guard, and
// that the checkout is named by global -C / --root before the subcommand.
func TestWriteHelpNamesHowToQuoteAHeaderOption(t *testing.T) {
	got := writeCmd().Description
	for _, must := range []string{
		`anchor="`,
		"single-quot",
		"body=",
		"line count",
		"lines=",
		"-C",
		"--root",
		"Python",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("write --help does not teach %q:\n%s", must, got)
		}
	}
}
