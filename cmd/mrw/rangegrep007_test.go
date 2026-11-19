package main

import (
	"strings"
	"testing"
)

// `mrw read --grep P f.go:10-40` asks two things, and the refusal said only
// that: "a range and --grep are two answers to one question". Counted
// 2026-11-19 across Claude Code transcripts outside this repository (a one-off
// tally, aggregates only): 52 calls. It now ends with the two forms that are
// one question each, with the path the caller gave (ADR-007).
func TestARangeBesideGrepSaysTheTwoFormsThatWork(t *testing.T) {
	run, _ := ifChangedRun(t, map[string]string{"f.txt": "one\ntwo\nthree\n"})
	const kept = "f.txt:1-2: a range and --grep are two answers to one question"
	const added = "; drop the range to search the whole file (--grep PATTERN f.txt), or drop --grep and address the lines by the pattern (f.txt:/PATTERN/)"
	out, errOut, code := run("", "read", "--grep", "two", "f.txt:1-2")
	if code != exitUsage || out != "" || !strings.HasSuffix(strings.TrimSpace(errOut), kept+added) {
		t.Errorf("read --grep two f.txt:1-2: exit %d, stdout %q, want exit %d ending in %q:\n%s", code, out, exitUsage, kept+added, errOut)
	}
	// The pair: each form the refusal names is served.
	for _, args := range [][]string{{"--grep", "two", "f.txt"}, {"f.txt:/two/"}} {
		if out, errOut, code := run("", append([]string{"read"}, args...)...); code != 0 || !strings.Contains(out, "| two") {
			t.Errorf("read %q: exit %d, want line 2 served:\n%s%s", args, code, out, errOut)
		}
	}
}
