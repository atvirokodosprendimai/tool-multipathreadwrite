package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// ADR-165 T2, through the built binary, since the budget is the whole of
// standard output and the withheld list is on standard error: an in-process
// run shares neither stream with the test. Standard output never passes N
// characters, the walk's own lines (the skip line) included; what was cut is
// named on standard error before ADR-146's incomplete: line, which says the
// budget was reached; the flag's refusals; and a withheld line stays unwritable.
func TestMaxCharsBoundsTheWholeAnswer(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "mrw.exe") // .exe: Windows exec will not launch an extensionless file
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	tree, state := t.TempDir(), t.TempDir()
	for name, body := range map[string]string{
		"a.txt":   "alpha\nbravo\ncharlie\ndelta\necho\n",
		"b.txt":   "foxtrot\ngolf\nhotel\nindia\njuliet\n",
		"c.txt":   "kilo\nlima\nmike\nnovember\noscar\n",
		"bin.dat": "a\x00binary\n", // a walk skips it and says so: a stdout line outside read.Run
	} {
		if err := os.WriteFile(filepath.Join(tree, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(stdin string, argv ...string) (stdout, stderr string, code int) {
		t.Helper()
		cmd := exec.Command(bin, argv...)
		cmd.Dir = tree
		cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
		cmd.Stdin = strings.NewReader(stdin)
		var o, e bytes.Buffer
		cmd.Stdout, cmd.Stderr = &o, &e
		err := cmd.Run()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return o.String(), e.String(), code
	}

	whole, _, code := run("", "read", "--grep", "a", ".")
	if code != 0 || !strings.Contains(whole, "-- skipped:") {
		t.Fatalf("the unbudgeted walk: exit %d, want 0 and a skip line:\n%s", code, whole)
	}
	size := utf8.RuneCountInString(whole)
	for _, n := range []int{0, 1, 40, 90, size - 1, size, size + 100} {
		out, errOut, code := run("", "read", "--grep", "a", "--max-chars", fmt.Sprint(n), ".")
		if got := utf8.RuneCountInString(out); got > n {
			t.Errorf("--max-chars %d: standard output is %d characters:\n%s", n, got, out)
		}
		if !strings.HasPrefix(whole, out) {
			t.Errorf("--max-chars %d: standard output is not a prefix of the unbudgeted answer:\n%s", n, out)
		}
		if n >= size {
			if code != 0 || out != whole {
				t.Errorf("--max-chars %d fits the whole answer: exit %d, out %q", n, code, out)
			}
			continue
		}
		last := strings.TrimRight(errOut, "\n")
		last = last[strings.LastIndex(last, "\n")+1:]
		if code != 1 || !strings.HasPrefix(last, "mrw: incomplete: ") || !strings.HasSuffix(last, fmt.Sprintf("; --max-chars %d reached", n)) {
			t.Errorf("--max-chars %d: exit %d, last stderr line %q", n, code, last)
		}
	}

	out, errOut, code := run("", "read", "--max-chars", "0", "a.txt", "b.txt", "c.txt")
	if code != 1 || out != "" || strings.Count(errOut, ": WITHHELD: --max-chars reached") != 3 {
		t.Errorf("--max-chars 0: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	if _, errOut, code := run("", "read", "--max-chars", "-1", "a.txt"); code != 2 || !strings.Contains(errOut, "a budget cannot be negative") {
		t.Errorf("--max-chars -1: exit %d, %q", code, errOut)
	}
	for _, argv := range [][]string{
		{"read", "--grep", "a", "--sub", "b", "--max-chars", "9", "."},
		{"read", "--grep", "a", "--count", "--max-chars", "9", "."},
		{"read", "--stat", "--max-chars", "9", "a.txt"},
	} {
		if _, errOut, code := run("", argv...); code != 2 || !strings.Contains(errOut, "--max-chars") {
			t.Errorf("%q: exit %d, %q; want 2 naming --max-chars", argv, code, errOut)
		}
	}
	if out, _, _ := run("", "read", "--quiet", "--max-chars", "40", "a.txt", "b.txt", "c.txt"); utf8.RuneCountInString(out) > 40 {
		t.Errorf("--quiet --max-chars 40: %d characters:\n%s", utf8.RuneCountInString(out), out)
	}

	// The header, the range line and alpha fit; bravo on are withheld, and a
	// plan to delta is refused as unread.
	out, _, _ = run("", "read", "a.txt")
	budget := 0
	for _, l := range strings.SplitAfter(out, "\n")[:3] {
		budget += utf8.RuneCountInString(l)
	}
	run("", "seen", "--prune") // nothing to keep from the unbudgeted read above
	if err := os.RemoveAll(state); err != nil {
		t.Fatal(err)
	}
	if out, errOut, code := run("", "read", "--max-chars", fmt.Sprint(budget), "a.txt"); code != 1 || !strings.Contains(out, "1| alpha") || strings.Contains(out, "bravo") || !strings.Contains(errOut, "a.txt: lines 2-5 WITHHELD (4 line(s))") {
		t.Fatalf("a budget for one line: exit %d, out %q, err %q", code, out, errOut)
	}
	if _, errOut, code := run("@@ a.txt 4 replace\nDELTA\n", "write", "--no-check", "-"); code != 1 {
		t.Errorf("a plan to a withheld line applied: exit %d, %s", code, errOut)
	}
}
