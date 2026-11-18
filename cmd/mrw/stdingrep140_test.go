package main

import (
	"strings"
	"testing"
)

// A caller narrowing a command's output types grep's flag before the pattern:
// `cmd | mrw read --grep -v PATTERN -`. --grep takes -v as its pattern, PATTERN
// becomes a second thing to read, and the refusal used to say only that a read
// of standard input names nothing else. Counted 2026-11-19 across Claude Code
// transcripts outside this repository: 148 of 160 such refusals had this shape
// (-v 109, -i 29, -o 6, -c 3, -E 1). The refusal now names the flag and what
// mrw spells instead, as a read of files already does (ADR-140, ADR-164).
func TestAGrepFlagBeforeThePatternOnStandardInputIsNamed(t *testing.T) {
	run, _ := ifChangedRun(t, map[string]string{"f.txt": "one\n"})
	const in = "alpha\nbeta\n"
	for flag, want := range map[string]string{
		"-v": "there is no inverted match",
		"-i": "(?i)",
		"-o": "there is no match-only output",
		"-c": "--count",
	} {
		out, errOut, code := run(in, "read", "--grep", flag, "beta", "-")
		if code != 2 || out != "" {
			t.Errorf("read --grep %s beta -: exit %d stdout %q, want 2 and nothing", flag, code, out)
		}
		if !strings.Contains(errOut, `--grep took "`+flag+`" as its pattern`) || !strings.Contains(errOut, want) {
			t.Errorf("read --grep %s beta -: the refusal does not name the flag and mrw's spelling (%q):\n%s", flag, want, errOut)
		}
		if strings.Contains(errOut, "names nothing else") {
			t.Errorf("read --grep %s beta -: still the generic refusal:\n%s", flag, errOut)
		}
	}
	// The pair: two things to read with no grep flag in the pattern keep the old words,
	// and so does a pattern that merely starts with a dash.
	for _, args := range [][]string{{"read", "-", "f.txt"}, {"read", "--grep", "-x", "beta", "-"}, {"read", "--grep", "beta", "-", "f.txt"}} {
		_, errOut, code := run(in, args...)
		if code != 2 || !strings.Contains(errOut, "names nothing else") || strings.Contains(errOut, "as its pattern") {
			t.Errorf("mrw %s: exit %d, want 2 and the generic refusal:\n%s", strings.Join(args, " "), code, errOut)
		}
	}
	// And the form the refusal points to works.
	if out, errOut, code := run(in, "read", "--grep", "(?i)BETA", "-"); code != 0 || !strings.Contains(out, "beta") {
		t.Errorf("read --grep (?i)BETA -: exit %d:\n%s%s", code, out, errOut)
	}
}
