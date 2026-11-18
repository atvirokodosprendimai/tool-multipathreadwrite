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
	// Every one-letter flag of the table, with the whole sentence it must end in.
	want := map[string]string{}
	for flag, hint := range foreignFlags {
		if len(flag) == 1 {
			want["-"+flag] = hint
		}
	}
	for flag, hint := range stdinFlags {
		want["-"+flag] = hint
	}
	// Written out for the flags callers type most, so a wrong sentence in the table fails here.
	for flag, hint := range map[string]string{
		"-v": "there is no inverted match: --grep serves the lines that match",
		"-i": "case-insensitive: start the pattern with (?i), as in --grep '(?i)PATTERN'",
		"-o": "there is no match-only output: --max-cols N cuts each line to a window round its match",
		"-c": "there is no count of a stream: --grep serves the lines that match, and --count counts them in files",
		"-l": "there is no file list for a stream: --grep serves the lines that match",
		"-r": "a stream is not walked: drop the flag",
		"-m": "--max-matches N stops after N matching lines, counting the rest; --max-lines N caps the lines served per spec, context included",
	} {
		if want[flag] != hint {
			t.Errorf("the sentence for %s is %q, want %q", flag, want[flag], hint)
		}
	}
	if len(want) < 17 {
		t.Fatalf("only %d one-letter flags in the table", len(want))
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
