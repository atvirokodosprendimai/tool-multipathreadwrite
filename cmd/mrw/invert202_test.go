package main

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
)

var (
	inv202ServedLine = regexp.MustCompile(`(?m)^\s*(\d+)\| `)
	inv202RunHeader  = regexp.MustCompile(`(?m)^@@ (\d+-\d+)`)
)

// inv202Served are the line numbers an answer printed, in order.
func inv202Served(out string) []int {
	var ns []int
	for _, m := range inv202ServedLine.FindAllStringSubmatch(out, -1) {
		n, _ := strconv.Atoi(m[1])
		ns = append(ns, n)
	}
	return ns
}

// inv202NonMatching is what the test itself says the inverted read must serve: the
// numbers of the lines of in that pattern does not match, split as the engine
// splits them.
func inv202NonMatching(t *testing.T, pattern, in string) []int {
	t.Helper()
	re := regexp.MustCompile(pattern)
	ls, _, _ := lines.Split(in)
	var ns []int
	for i, l := range ls {
		if !re.MatchString(l) {
			ns = append(ns, i+1)
		}
	}
	return ns
}

func inv202LastLine(s string) string {
	s = strings.TrimRight(s, "\n")
	return s[strings.LastIndex(s, "\n")+1:]
}

func inv202Same(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ADR-202. `cmd | mrw read --grep PATTERN --invert -` serves the lines of
// standard input that do not match, numbered as in the input, and records
// nothing. Everywhere else the flag is refused, before anything is read.
func TestAnInvertedMatchServesTheLinesOfStandardInputThatDoNotMatch(t *testing.T) {
	run, _ := ifChangedRun(t, map[string]string{"f.txt": "one\n"})
	const a = "alpha\nbeta\ngamma\nbeta2\ndelta\n"
	const form = "--grep PATTERN --invert -"

	// Fixture A: the served numbers are every line that does not match, and no other.
	out, errOut, code := run(a, "read", "--grep", "beta", "--invert", "-")
	if code != 0 {
		t.Fatalf("A inverted: exit %d:\n%s%s", code, out, errOut)
	}
	if got, want := inv202Served(out), inv202NonMatching(t, "beta", a); !inv202Same(got, want) || len(want) != 3 {
		t.Errorf("A inverted served %v, the lines that do not match are %v:\n%s", got, want, out)
	}
	for _, h := range []string{"@@ 1-1", "@@ 3-3", "@@ 5-5"} {
		if !strings.Contains(out, h+"\n") {
			t.Errorf("A inverted lacks %q:\n%s", h, out)
		}
	}
	if strings.Contains(out, "beta") || strings.Contains(out, "-- note:") {
		t.Errorf("A inverted served a matching line or a note:\n%s", out)
	}
	if o, _, _ := run("", "seen"); !strings.Contains(o, "0 file(s) seen") {
		t.Errorf("an inverted read recorded something:\n%s", o)
	}

	// Fixture B: a run of two lines, and no note about a replace.
	out, errOut, code = run("one\ntwo\nskip\nfour\n", "read", "--grep", "skip", "--invert", "-")
	if got := inv202RunHeader.FindAllString(out, -1); code != 0 || !inv202Same(inv202Served(out), []int{1, 2, 4}) || strings.Join(got, "|") != "@@ 1-2|@@ 4-4" || strings.Contains(out, "-- note:") {
		t.Errorf("B inverted: exit %d, runs %v:\n%s%s", code, got, out, errOut)
	}

	// The caps.
	out, errOut, code = run(a, "read", "--grep", "beta", "--invert", "--max-lines", "2", "-")
	if code != 0 || !inv202Same(inv202Served(out), []int{1, 3}) || !strings.Contains(out, "@@ 5-5  WITHHELD 1 line(s): --max-lines reached") || !strings.HasPrefix(inv202LastLine(errOut), "mrw: capped:") {
		t.Errorf("--max-lines 2: exit %d:\n%s%s", code, out, errOut)
	}
	out, errOut, code = run(a, "read", "--grep", "beta", "--invert", "--max-chars", "1", "-")
	if code != 1 || len(inv202Served(out)) != 0 || !strings.Contains(errOut, "mrw: incomplete:") || !strings.Contains(inv202LastLine(errOut), "--max-chars 1 reached") {
		t.Errorf("--max-chars 1: exit %d:\n%s%s", code, out, errOut)
	}
	out, _, code = run(a, "read", "--grep", "beta", "--invert", "-N", "-")
	if code != 0 || len(inv202Served(out)) != 0 || !strings.Contains(out, "@@ 1-1\nalpha\n@@ 3-3\ngamma\n@@ 5-5\ndelta\n") {
		t.Errorf("-N: exit %d:\n%s", code, out)
	}
	out, _, code = run(a, "read", "--grep", "beta", "--invert", "--fold", "-")
	if code != 0 || !inv202Same(inv202Served(out), []int{1, 3, 5}) {
		t.Errorf("--fold over unlike lines: exit %d:\n%s", code, out)
	}
	out, _, code = run("x 1\nx 2\nx 3\nskip\n", "read", "--grep", "skip", "--invert", "--fold", "-")
	if code != 0 || !strings.Contains(out, "-- ×3: lines 1-3 like 1") || strings.Contains(out, "x 2") {
		t.Errorf("--fold: exit %d, the run is not folded as numbered lines fold:\n%s", code, out)
	}
	out, _, code = run(strings.Repeat("abcdefghij", 4)+"\nskip\n", "read", "--grep", "skip", "--invert", "--max-cols", "10", "-")
	if code != 0 || !strings.Contains(out, "1| abcdefghij") || !strings.Contains(out, "[cols 1-10 of 40]") || strings.Contains(out, "abcdefghijabcdefghij") {
		t.Errorf("--max-cols 10: exit %d, the line is not cut from column 1:\n%s", code, out)
	}

	// Nothing left: no numbered line, the kind on the last line of stderr.
	out, errOut, code = run(a, "read", "--grep", "a", "--invert", "-")
	if code != 1 || len(inv202Served(out)) != 0 || !strings.HasPrefix(out, "==> -  5L") || !strings.HasSuffix(out, "\n!! every line matches /a/\n") || inv202LastLine(errOut) != "mrw: no match: every line of standard input matched /a/" {
		t.Errorf("every line matches: exit %d:\n%s%s", code, out, errOut)
	}
	out, errOut, code = run("", "read", "--grep", "a", "--invert", "-")
	if code != 1 || !strings.HasPrefix(out, "==> -  0L") || !strings.HasSuffix(out, "\n!! standard input is empty\n") || strings.Count(out, "\n") != 2 || inv202LastLine(errOut) != "mrw: no match: standard input is empty" {
		t.Errorf("empty input: exit %d:\n%s%s", code, out, errOut)
	}
	// Capped is not nothing left.
	if _, errOut, code = run(a, "read", "--grep", "a", "--invert", "--max-lines", "0", "-"); code != 1 {
		t.Errorf("every line matches beside --max-lines 0: exit %d:\n%s", code, errOut)
	}

	// The engine and this read split the same bytes the same way.
	for _, c := range []struct{ name, in, pattern string }{
		{"CRLF", "a\r\nskip\r\nb\r\n", "skip"},
		{"CR only", "a\rskip\rb", "skip"},
		{"no final newline", "a\nskip\nb", "skip"},
		{"one empty line", "\n", "x"},
		{"mixed", "a\r\nskip\nb\r\n", "skip"},
	} {
		out, errOut, code := run(c.in, "read", "--grep", c.pattern, "--invert", "-")
		want := inv202NonMatching(t, c.pattern, c.in)
		if code != 0 || !inv202Same(inv202Served(out), want) || len(want) == 0 {
			t.Errorf("%s: exit %d, served %v, want %v:\n%s%s", c.name, code, inv202Served(out), want, out, errOut)
		}
	}
	if out, _, _ := run("\n", "read", "--grep", "x", "--invert", "-"); !strings.Contains(out, "@@ 1-1\n   1| \n") && !strings.Contains(out, "1| \n") {
		t.Errorf("an empty line is not served as line 1:\n%s", out)
	}

	// The refusals: exit 2, nothing served, the sentence and the form named.
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"read", "--invert", "-"}, "--invert needs --grep"},
		{[]string{"read", "--invert", "--count", "f.txt"}, "--invert needs --grep"},
		{[]string{"read", "--grep", "beta", "--invert", "f.txt"}, "a read of files serves the lines that match"},
		{[]string{"read", "--grep", "beta", "--invert", "."}, "a read of files serves the lines that match"},
		{[]string{"read", "--grep", "beta", "--invert", "./-"}, "a read of files serves the lines that match"},
		{[]string{"read", "--grep", "beta", "--invert"}, "--invert reads standard input and nothing else"},
		{[]string{"read", "--grep", "beta", "--invert", "-:1-2"}, "--invert reads standard input and nothing else"},
		{[]string{"read", "--grep", "beta", "--invert", "-", "f.txt"}, "a read of files serves the lines that match"},
		{[]string{"read", "--grep", "beta", "--invert", "--sub", "X", "f.txt"}, "a read of files serves the lines that match"},
		{[]string{"read", "--grep", "beta", "--invert", "-C", "1", "-"}, "--invert allows only --grep, --max-lines, --max-cols, --max-chars, --fold and -N beside it, and -C is not one of them"},
		{[]string{"read", "--grep", "beta", "--invert", "--max-matches", "1", "-"}, "and --max-matches is not one of them"},
		{[]string{"read", "--grep", "beta", "--invert", "--no-comments", "-"}, "and --no-comments is not one of them"},
		{[]string{"read", "--grep", "beta", "--invert", "--quiet", "-"}, "and --quiet is not one of them"},
		{[]string{"read", "--grep", "beta", "--invert", "--count", "-"}, "and --count is not one of them"},
		{[]string{"read", "--grep", "beta", "--invert", "--sub", "X", "-"}, "and --sub is not one of them"},
	} {
		out, errOut, code := run(a, c.args...)
		name := strings.Join(c.args, " ")
		if code != 2 || out != "" || !strings.Contains(errOut, c.want) || !strings.Contains(errOut, form) {
			t.Errorf("mrw %s: exit %d, stdout %q, want 2, nothing and %q beside %q:\n%s", name, code, out, c.want, form, errOut)
		}
	}
	// --invert=false is the flag not given: the plain read, the line after each match included.
	if out, _, code := run(a, "read", "--grep", "beta", "--invert=false", "-"); code != 0 || !inv202Same(inv202Served(out), []int{2, 3, 4, 5}) {
		t.Errorf("--invert=false: exit %d, want the plain read:\n%s", code, out)
	}
	// --json answers a refusal as a document on standard output, so it is judged on the sentence alone.
	if out, errOut, code := run(a, "read", "--grep", "beta", "--invert", "--json", "-"); code != 2 || len(inv202Served(out)) != 0 || !strings.Contains(errOut, "and --json is not one of them") {
		t.Errorf("--invert --json: exit %d:\n%s%s", code, out, errOut)
	}
	// The plain read of a stream is as it was: the lines that match, and the line after each.
	if out, _, code := run(a, "read", "--grep", "beta", "-"); code != 0 || !inv202Same(inv202Served(out), []int{2, 3, 4, 5}) {
		t.Errorf("a plain read of A: exit %d:\n%s", code, out)
	}

	// --invert taken as the pattern, on a stream and on a file.
	for _, args := range [][]string{{"read", "--grep", "--invert", "beta", "-"}, {"read", "--grep", "--invert", "beta", "f.txt"}} {
		out, errOut, code := run(a, args...)
		if code != 2 || out != "" || !strings.Contains(errOut, `--grep took "--invert" as its pattern`) {
			t.Errorf("mrw %s: exit %d:\n%s%s", strings.Join(args, " "), code, out, errOut)
		}
	}
	// The literal is written as ADR-140 says, and serves its line.
	if out, _, code := run("x\n--invert\n", "read", "--grep=(?:--invert)", "-"); code != 0 || !inv202Same(inv202Served(out), []int{2}) {
		t.Errorf(`--grep="(?:--invert)": exit %d:\n%s`, code, out)
	}

	// Every way to type -v ends in the form.
	const pointer = "there is no inverted match: --grep serves the lines that match; an inverted match of a stream is --grep PATTERN --invert -"
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{"read", "--grep", "-v", "beta", "-"}, 2},
		{[]string{"read", "--grep", "beta", "-v", "-"}, 2},
		{[]string{"read", "-v", "--grep", "beta", "-"}, 2},
		{[]string{"read", "-v", "f.txt"}, 2},
		{[]string{"read", "--grep", "-v", "beta", "f.txt"}, 1},
	} {
		out, errOut, code := run(a, c.args...)
		if code != c.code || out != "" && c.code == 2 || !strings.Contains(errOut, pointer) {
			t.Errorf("mrw %s: exit %d, want %d and %q:\n%s%s", strings.Join(c.args, " "), code, c.code, pointer, out, errOut)
		}
	}

	// The flag is discoverable.
	if out, _, code := run("", "read", "--help"); code != 0 || !strings.Contains(out, "--invert") || !strings.Contains(out, "standard input") {
		t.Errorf("read --help does not name --invert and standard input (exit %d)", code)
	}
}
