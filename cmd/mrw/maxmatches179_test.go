package main

import (
	"strings"
	"testing"
)

const maxMatches179Body = "hit 1\nfill\nfill\nhit 2\nfill\nfill\nhit 3\n"

// ADR-179 Decision (g). Exactly N matches: nothing is said and the read exits
// 0, and a budget above N likewise. The flag is refused, exit 2, without a
// finder, beside --count, --sub, --rewrite and --json-path (ADR-178, which says
// so first), and for a negative number.
func TestMaxMatchesOfExactlyNSaysNothing(t *testing.T) {
	root := grepTree(t, map[string]string{"a.txt": maxMatches179Body, "f.json": `{"a": 1}` + "\n"})
	for _, n := range []string{"3", "4"} {
		out, code := runIn(t, root, "read", "--grep", "hit", "--max-matches", n, "a.txt")
		if code != 0 || strings.Count(out, "hit ") != 3 || strings.Contains(out, "max-matches") {
			t.Errorf("--max-matches %s over 3 matches exited %d, want 0 and no note:\n%s", n, code, out)
		}
	}
	for _, c := range []struct {
		argv []string
		want string
	}{
		{[]string{"--max-matches", "1", "a.txt"}, "--max-matches caps the matches a --grep or --ast-grep finds"},
		{[]string{"--grep", "hit", "--max-matches", "-1", "a.txt"}, "--max-matches needs"},
		{[]string{"--grep", "hit", "--max-matches", "1", "--count", "a.txt"}, "--count"},
		{[]string{"--grep", "hit", "--max-matches", "1", "--sub", "x", "a.txt"}, "--sub"},
		{[]string{"--grep", "hit", "--max-matches", "1", "--stat", "a.txt"}, "--stat"},
		{[]string{"--json-path", "$.a", "--max-matches", "1", "f.json"}, "--json-path prints values and serves no line, so --max-matches has nothing to act on"},
		{[]string{"--max-matches", "1", "--json-path", "$.a", "f.json"}, "--json-path prints values and serves no line, so --max-matches has nothing to act on"},
	} {
		if out, code := runIn(t, root, append([]string{"read"}, c.argv...)...); code != 2 || !strings.Contains(out, c.want) || strings.Contains(out, "==>") {
			t.Errorf("mrw read %v exited %d, want 2 saying %q:\n%s", c.argv, code, c.want, out)
		}
	}
}

// ADR-179 Decision (g). Only what was shown is licensed: a write to a match the
// budget counted and did not show is refused as unread, the read exits 1
// incomplete, and the shown match can be written.
func TestMaxMatchesLicensesOnlyWhatWasShown(t *testing.T) {
	root := grepTree(t, map[string]string{"a.txt": maxMatches179Body})
	out, code := runIn(t, root, "read", "--grep", "hit", "--max-matches", "1", "a.txt")
	if code != 1 || !strings.Contains(out, "incomplete: --max-matches 1 reached, 2 matching line(s) not shown") || !strings.Contains(out, "-- max-matches 1 reached: 2 more matching line(s) in 1 file(s) not shown") || strings.Contains(out, "hit 2") {
		t.Fatalf("the read exited %d, want 1 incomplete with the note and only the first match:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ a.txt 7 replace\nz\n")); code != 1 || !strings.Contains(out, "has not been read") {
		t.Errorf("a write to a counted, unshown match exited %d, want 1 refused as unread:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ a.txt 1 replace\nz\n")); code != 0 {
		t.Errorf("a write to the shown match exited %d, want 0:\n%s", code, out)
	}
}
