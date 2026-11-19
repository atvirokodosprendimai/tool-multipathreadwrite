package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

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
	inv := newInv202(t, map[string]string{"f.txt": "one\n"})
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
	if code != 1 || len(inv202Served(out)) != 0 || !strings.HasPrefix(inv202LastLine(errOut), "mrw: incomplete:") || !strings.Contains(inv202LastLine(errOut), "--max-chars 1 reached") {
		t.Errorf("--max-chars 1: exit %d:\n%s%s", code, out, errOut)
	}
	// Lines that all do match nothing is not what withheld them: the kind stays incomplete.
	if out, errOut, code = run("alpha\ngamma\n", "read", "--grep", "beta", "--invert", "--max-chars", "1", "-"); code != 1 || !strings.HasPrefix(inv202LastLine(errOut), "mrw: incomplete:") {
		t.Errorf("--max-chars 1 over lines that do not match: exit %d:\n%s%s", code, out, errOut)
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

	// Nothing left: no numbered line, the kind on the last line of stderr, and the
	// header the plain read prints for the same input.
	plainHeader := func(in string) string {
		o, _, _ := run(in, "read", "--grep", "zzz", "-")
		return strings.SplitN(o, "\n", 2)[0]
	}
	out, errOut, code = run(a, "read", "--grep", "a", "--invert", "-")
	if code != 1 || len(inv202Served(out)) != 0 || out != plainHeader(a)+"\n!! every line matches /a/\n" || inv202LastLine(errOut) != "mrw: no match: every line of standard input matched /a/" {
		t.Errorf("every line matches: exit %d:\n%s%s", code, out, errOut)
	}
	out, errOut, code = run("", "read", "--grep", "a", "--invert", "-")
	if code != 1 || out != plainHeader("")+"\n!! standard input is empty\n" || !strings.HasPrefix(out, "==> -  0L  0B") || inv202LastLine(errOut) != "mrw: no match: standard input is empty" {
		t.Errorf("empty input: exit %d:\n%s%s", code, out, errOut)
	}
	// Capped is not nothing left.
	if _, errOut, code = run(a, "read", "--grep", "a", "--invert", "--max-lines", "0", "-"); code != 1 {
		t.Errorf("every line matches beside --max-lines 0: exit %d:\n%s", code, errOut)
	}

	// --max-chars bounds the two lines of nothing left as it bounds a plain read's.
	for _, c := range []struct{ in, budget, last string }{
		{"a\nab\n", "5", "mrw: no match: every line of standard input matched /a/; --max-chars 5 reached"},
		{"", "3", "mrw: no match: standard input is empty; --max-chars 3 reached"},
	} {
		out, errOut, code := run(c.in, "read", "--grep", "a", "--invert", "--max-chars", c.budget, "-")
		plainOut, plainErr, plainCode := run(c.in, "read", "--grep", "zzz", "--max-chars", c.budget, "-")
		const withheld = "-: WITHHELD: --max-chars reached\n"
		if code != 1 || plainCode != 1 || out != "" || plainOut != "" || !strings.HasPrefix(errOut, withheld) || !strings.HasPrefix(plainErr, withheld) || inv202LastLine(errOut) != c.last {
			t.Errorf("nothing left beside --max-chars %s over %q: exit %d (plain %d), stdout %q:\n%s", c.budget, c.in, code, plainCode, out, errOut)
		}
	}

	// A budget that holds the header and not the `!!` line holds neither: all or nothing.
	if out, errOut, code := run("a\n", "read", "--grep", "a", "--invert", "--max-chars", "28", "-"); code != 1 || out != "" || !strings.HasPrefix(errOut, "-: WITHHELD: --max-chars reached\n") || inv202LastLine(errOut) != "mrw: no match: every line of standard input matched /a/; --max-chars 28 reached" {
		t.Errorf("a budget between the header and both lines: exit %d, stdout %q:\n%s", code, out, errOut)
	}

	// A line over the long-line threshold: the hint for an inverted read says where --max-cols cuts and
	// nothing of licensing; the plain read keeps its own words.
	long := strings.Repeat("x", 450) + "\nskip\n"
	if _, errOut, code := inv.do(t, strings.NewReader(long), nil, "read", "--grep", "skip", "--invert", "-"); code != 0 || !strings.Contains(errOut, "1 line(s) over 400 columns served whole, the longest 450; --max-cols 200 cuts each from its first column") || strings.Contains(errOut, "licens") {
		t.Errorf("the long-line hint of an inverted read: exit %d:\n%s", code, errOut)
	}
	if _, errOut, code := inv.do(t, strings.NewReader("skip\n"+strings.Repeat("x", 450)+"\n"), nil, "read", "--grep", "x", "-"); code != 0 || !strings.Contains(errOut, "--max-cols 200 cuts each to a window round the match, and a cut line licenses nothing until it is read whole") {
		t.Errorf("the long-line hint of a plain read changed: exit %d:\n%s", code, errOut)
	}

	// The PowerShell shape, a stream with a byte that is not UTF-8, and MRW_MAX_COLS.
	if out, errOut, code := run("a\nskip\nb\r\n", "read", "--grep", "skip", "--invert", "-"); code != 0 || !inv202Same(inv202Served(out), []int{1, 3}) || !strings.HasPrefix(out, "==> -  3L") || !strings.Contains(errOut, "so it was dropped") {
		t.Errorf("PowerShell's CRLF: exit %d:\n%s%s", code, out, errOut)
	}
	if out, errOut, code := run("a\nskip\n\xff\n", "read", "--grep", "skip", "--invert", "-"); code != 0 || !inv202Same(inv202Served(out), []int{1, 3}) || !strings.Contains(out, "\xff") {
		t.Errorf("a line that is not UTF-8: exit %d:\n%q%s", code, out, errOut)
	}
	if out, errOut, code := inv.do(t, strings.NewReader(strings.Repeat("abcdefghij", 4)+"\nskip\n"), []string{"MRW_MAX_COLS=10"}, "read", "--grep", "skip", "--invert", "-"); code != 0 || !strings.Contains(out, "1| abcdefghij") || !strings.Contains(out, "[cols 1-10 of 40]") || strings.Contains(out, "abcdefghijabcdefghij") {
		t.Errorf("MRW_MAX_COLS=10: exit %d:\n%s%s", code, out, errOut)
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
	if out, _, _ := run("\n", "read", "--grep", "x", "--invert", "-"); !strings.HasSuffix(out, "@@ 1-1\n 1| \n") {
		t.Errorf("an empty line is not served as line 1:\n%q", out)
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
		{[]string{"read", "--invert", "-C", "-1", "-"}, "--invert needs --grep"},
		{[]string{"read", "--grep", "x", "--invert", "--stat", "--max-chars", "1", "-"}, "and --stat is not one of them"},
		{[]string{"read", "--grep=-C", "--invert", "f.txt"}, "a read of files serves the lines that match"},
	} {
		out, errOut, code := inv.do(t, nil, nil, c.args...) // standard input never ends: a refusal that read it would hang
		name := strings.Join(c.args, " ")
		if code != 2 || out != "" || !strings.Contains(errOut, c.want) || !strings.Contains(errOut, form) {
			t.Errorf("mrw %s: exit %d, stdout %q, want 2, nothing and %q beside %q:\n%s", name, code, out, c.want, form, errOut)
		}
	}
	// --invert=false is the flag not given: the plain read, the line after each match included.
	if out, _, code := run(a, "read", "--grep", "beta", "--invert=false", "-"); code != 0 || !inv202Same(inv202Served(out), []int{2, 3, 4, 5}) {
		t.Errorf("--invert=false: exit %d, want the plain read:\n%s", code, out)
	}
	// --json answers a refusal as one document on standard output, with the sentence under "error".
	jout, jerr, jcode := inv.do(t, nil, nil, "read", "--grep", "beta", "--invert", "--json", "-")
	var doc map[string]string
	dec := json.NewDecoder(strings.NewReader(jout))
	if err := dec.Decode(&doc); err != nil || dec.More() || len(doc) != 1 || jcode != 2 || !strings.Contains(doc["error"], "and --json is not one of them") || !strings.Contains(doc["error"], form) {
		t.Errorf("--invert --json: exit %d, document %v (%v):\n%s%s", jcode, doc, err, jout, jerr)
	}
	// The plain read of a stream is as it was: the lines that match, and the line after each.
	if out, _, code := run(a, "read", "--grep", "beta", "-"); code != 0 || !inv202Same(inv202Served(out), []int{2, 3, 4, 5}) {
		t.Errorf("a plain read of A: exit %d:\n%s", code, out)
	}
	// Without --invert the order of ADR-140's refusal and the flag-domain checks is as it was.
	if out, errOut, code := run(a, "read", "--grep=-C", "-C", "-1", "f.txt"); code != 2 || out != "" || !strings.Contains(errOut, "-C -1: context cannot be negative") {
		t.Errorf("--grep=-C -C -1 f.txt: exit %d:\n%s%s", code, out, errOut)
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

	// The flag is discoverable: its own line in the help names it as a boolean (no
	// value placeholder, which a backticked word in the usage would make) and says
	// where it works.
	out, _, code = run("", "read", "--help")
	helpLine := regexp.MustCompile(`(?m)^\s+--invert\s{2,}with --grep, serve the lines that do not match.*$`).FindString(out)
	if code != 0 || helpLine == "" || !strings.Contains(helpLine, "Standard input only") {
		t.Errorf("read --help has no --invert line that says standard input only (exit %d):\n%s", code, out)
	}
}

// inv202 is a built mrw with a tree and a state directory of its own, run with
// a standard input the caller chooses: ifChangedRun's runner takes a string,
// which cannot say "never ends".
type inv202 struct{ bin, tree, state string }

func newInv202(t *testing.T, files map[string]string) inv202 {
	t.Helper()
	e := inv202{bin: filepath.Join(t.TempDir(), "mrw.exe"), tree: t.TempDir(), state: t.TempDir()}
	if out, err := exec.Command("go", "build", "-o", e.bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(e.tree, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return e
}

// do runs mrw with stdin, or with a pipe that never delivers and never closes
// when stdin is nil: a call that reads it does not finish, and fails the test
// at the deadline instead of hanging it.
func (e inv202) do(t *testing.T, stdin io.Reader, env []string, args ...string) (string, string, int) {
	t.Helper()
	if stdin == nil {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = w.Close(); _ = r.Close() })
		stdin = r
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.bin, args...)
	cmd.Dir = e.tree
	// MRW_MAX_COLS is the developer's, and would change what a long line prints: only the caller's env sets it.
	cmd.Env = []string{"XDG_STATE_HOME=" + e.state}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "MRW_MAX_COLS=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdin = stdin
	var o, er strings.Builder
	cmd.Stdout, cmd.Stderr = &o, &er
	cmd.WaitDelay = time.Second
	code := 0
	var exit *exec.ExitError
	if err := cmd.Run(); errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil {
		t.Errorf("mrw %s did not finish: it waited for standard input", strings.Join(args, " "))
	}
	return o.String(), er.String(), code
}
