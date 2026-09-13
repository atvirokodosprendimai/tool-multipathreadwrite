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

// TestWriteHelpNamesApplyPatchFormat is ADR-051 T2: a PATH caller who reads
// write --help must learn --format=apply_patch and that a git patch is not one.
func TestWriteHelpNamesApplyPatchFormat(t *testing.T) {
	got := writeCmd().Description
	for _, must := range []string{
		"--format=apply_patch",
		"--format=search_replace",
		"git patch",
		"not an apply_patch",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("write --help does not teach %q:\n%s", must, got)
		}
	}
}

// TestWriteHelpNamesEchoPad is ADR-052 T2: write --help names the opt-in pad
// and that it is not a checker.
func TestWriteHelpNamesEchoPad(t *testing.T) {
	got := writeCmd().Description
	for _, must := range []string{
		"--echo-pad",
		"not a checker",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("write --help does not teach %q:\n%s", must, got)
		}
	}
}

// TestWriteHelpNamesNoCheck is ADR-054 T4: a PATH caller who reads write
// --help must learn that the check now runs by default on a non-prose write,
// that --no-check opts out, and that the balance row never fails a hunk.
func TestWriteHelpNamesNoCheck(t *testing.T) {
	got := writeCmd().Description
	for _, must := range []string{
		"--no-check",
		"by default",
		"prose",
		"balance",
		"does not fail the hunk",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("write --help does not teach %q:\n%s", must, got)
		}
	}
}
