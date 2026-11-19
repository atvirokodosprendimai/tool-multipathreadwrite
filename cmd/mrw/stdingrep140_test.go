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
	// Every one-letter flag, with the whole sentence it must end in, written out
	// here and not read from the table, so a wrong sentence in the table fails.
	shared := map[string]string{
		"-i": "case-insensitive: start the pattern with (?i), as in --grep '(?i)PATTERN'",
		"-n": "line numbers are printed by default; -N drops them",
		"-A": "context is symmetric: -C N prints N lines either side of a match",
		"-B": "context is symmetric: -C N prints N lines either side of a match",
		"-e": `the pattern is the value of --grep; one that starts with a dash is written --grep=-PATTERN, or --grep="(?:-C)" when it spells a flag of mrw read`,
		"-E": "--grep patterns are already extended regular expressions (Go RE2)",
		"-P": "--grep patterns are Go RE2: no look-around and no backreferences",
		"-F": `--grep patterns are regular expressions; wrap a literal in \Q…\E, or escape its metacharacters one by one if it contains \E`,
		"-w": `wrap the word in \b…\b; Go's \b is ASCII-only, so a word with non-ASCII letters is not bounded correctly`,
		"-H": "every file is announced by a ==> header; there is no flag for it",
		"-m": "--grep PATTERN --max-matches N stops after N matching lines and counts the rest (a match inside a span already being served still prints); --max-lines N caps the lines served per spec, context included",
		"-v": "there is no inverted match: --grep serves the lines that match; an inverted match of a stream is --grep PATTERN --invert -",
		"-o": "there is no match-only output: --max-cols N cuts each line to a window round its match",
	}
	want := map[string]string{
		"-c": "there is no count of a stream: --grep serves the lines that match, and --count counts them in files",
		"-l": "there is no file list for a stream: --grep serves the lines that match",
		"-r": "a stream is not walked: drop the flag",
		"-R": "a stream is not walked: drop the flag",
	}
	for flag, hint := range shared {
		want[flag] = hint
		// The shared sentences are the ones a read of files ends in too.
		if _, errOut, code := run("", "read", "--grep", flag, "f.txt"); code != 1 || !strings.Contains(errOut, hint) {
			t.Errorf("read --grep %s f.txt: exit %d, want 1 ending in %q:\n%s", flag, code, hint, errOut)
		}
	}
	n := 0
	for flag := range foreignFlags {
		if len(flag) == 1 {
			n++
			if _, ok := want["-"+flag]; !ok {
				t.Errorf("the table's -%s has no expected sentence here", flag)
			}
		}
	}
	if n != len(want) {
		t.Fatalf("%d one-letter flags in the table, %d expected here", n, len(want))
	}
	for flag, hint := range want {
		out, errOut, code := run(in, "read", "--grep", flag, "beta", "-")
		if code != 2 || out != "" {
			t.Errorf("read --grep %s beta -: exit %d stdout %q, want 2 and nothing", flag, code, out)
		}
		full := `--grep took "` + flag + `" as its pattern, and a read of standard input names nothing else beside -; ` + hint
		if !strings.Contains(errOut, full) {
			t.Errorf("read --grep %s beta -: want the refusal %q:\n%s", flag, full, errOut)
		}
	}
	// An attached value and a flag typed last reach the same refusal, which claims no order.
	for _, args := range [][]string{{"read", "--grep=-v", "beta", "-"}, {"read", "-", "f.txt", "--grep", "-v"}} {
		if _, errOut, code := run(in, args...); code != 2 || !strings.Contains(errOut, `--grep took "-v" as its pattern, and a read of standard input names nothing else beside -`) {
			t.Errorf("mrw %s: exit %d:\n%s", strings.Join(args, " "), code, errOut)
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
