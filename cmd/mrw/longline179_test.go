package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// longline179Tree holds a.txt and b.txt, each with one line over 400 columns
// (500 and 700) between short ones, and c.txt whose longest line is exactly 400.
func longline179Tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{
		"a.txt": "one\n" + strings.Repeat("a", 500) + "\nthree\n",
		"b.txt": "one\n" + strings.Repeat("b", 700) + "\nthree\n",
		"c.txt": "one\n" + strings.Repeat("c", 400) + "\nthree\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// ADR-179 Decision (a). A read that serves a line over 400 columns whole says
// so once on stderr, naming how many and the longest, however many files held
// them; --max-cols given (any value) or no line over 400 says nothing; standard
// input is judged the same way.
func TestAReadOfAnOverlongLineHintsOnceOnStderr(t *testing.T) {
	t.Setenv("MRW_MAX_COLS", "")
	root := longline179Tree(t)
	const hint = "mrw: 2 line(s) over 400 columns served whole, the longest 700; --max-cols 200 cuts each to a window round the match, and a cut line licenses nothing until it is read whole"
	_, errOut, code := project178Run(t, root, "", "read", "a.txt", "b.txt")
	if code != 0 || strings.Count(errOut, "over 400 columns") != 1 || !strings.Contains(errOut, hint+"\n") {
		t.Errorf("two files with long lines exited %d, want one hint %q on stderr:\n%s", code, hint, errOut)
	}
	for _, argv := range [][]string{
		{"read", "--max-cols", "200", "a.txt", "b.txt"},
		{"read", "--max-cols", "0", "a.txt", "b.txt"},
		{"read", "c.txt"},
		{"read", "a.txt:3"},
	} {
		if _, errOut, code = project178Run(t, root, "", argv...); code != 0 || strings.Contains(errOut, "over 400 columns") {
			t.Errorf("mrw %v exited %d with a hint, want none:\n%s", argv, code, errOut)
		}
	}
	_, errOut, code = project178Run(t, root, "x\n"+strings.Repeat("s", 450)+"\n", "read", "-")
	if want := "mrw: 1 line(s) over 400 columns served whole, the longest 450;"; code != 0 || strings.Count(errOut, "over 400 columns") != 1 || !strings.Contains(errOut, want) {
		t.Errorf("a long line on standard input exited %d, want one hint %q:\n%s", code, want, errOut)
	}
}

// ADR-179 Decision (a). MRW_MAX_COLS=N is --max-cols N where the flag is
// absent, for a file and for standard input: the line is cut, a write to it is
// refused until it is read whole, --max-cols 0 on the command line turns it off,
// and a value that is not a non-negative integer is named and ignored.
func TestMRWMaxColsAppliesWhenTheFlagIsAbsent(t *testing.T) {
	root := longline179Tree(t)
	t.Setenv("MRW_MAX_COLS", "100")
	out, errOut, code := project178Run(t, root, "", "read", "a.txt")
	if code != 0 || !strings.Contains(out, "[cols ") || strings.Contains(out, strings.Repeat("a", 150)) || strings.Contains(errOut, "over 400 columns") {
		t.Fatalf("MRW_MAX_COLS=100 left the line whole or hinted (exit %d):\n%s\n%s", code, out, errOut)
	}
	plan := planFile(t, "@@ a.txt 2 replace\nnew\n")
	if out, code = runIn(t, root, "write", "--no-check", plan); code != 1 || strings.Contains(out, "applied") {
		t.Errorf("a write to the cut line exited %d, want 1 refused:\n%s", code, out)
	}
	out, _, code = project178Run(t, root, "x\n"+strings.Repeat("s", 450)+"\n", "read", "-")
	if code != 0 || !strings.Contains(out, "[cols ") || strings.Contains(out, strings.Repeat("s", 150)) {
		t.Errorf("MRW_MAX_COLS did not cut standard input (exit %d):\n%s", code, out)
	}
	out, _, code = project178Run(t, root, "", "read", "--max-cols", "0", "a.txt")
	if code != 0 || strings.Contains(out, "[cols ") || !strings.Contains(out, strings.Repeat("a", 500)) {
		t.Errorf("--max-cols 0 did not turn the variable off (exit %d):\n%s", code, out)
	}
	for _, bad := range []string{"abc", "-5", "1.5"} {
		t.Setenv("MRW_MAX_COLS", bad)
		out, errOut, code = project178Run(t, root, "", "read", "a.txt")
		if code != 0 || strings.Contains(out, "[cols ") || !strings.Contains(errOut, "MRW_MAX_COLS="+bad) || !strings.Contains(errOut, "ignored") || strings.Count(errOut, "MRW_MAX_COLS") != 1 {
			t.Errorf("MRW_MAX_COLS=%s exited %d, want it named once and ignored:\n%s\n%s", bad, code, out, errOut)
		}
	}
	// A boolean flag given false is not the flag: the variable still applies.
	t.Setenv("MRW_MAX_COLS", "100")
	for _, argv := range [][]string{
		{"read", "--quiet=false", "a.txt"},
		{"read", "--stat=false", "a.txt"},
		{"read", "--count=false", "--grep", "aaa", "a.txt"},
	} {
		if out, _, code = project178Run(t, root, "", argv...); code != 0 || !strings.Contains(out, "[cols ") {
			t.Errorf("mrw %v under the variable exited %d, want the line cut:\n%s", argv, code, out)
		}
	}
}

// ADR-179 Decision (a). The variable never applies to a read that exists to
// license (--quiet licenses the whole line), nor to --sub or --count, which it
// must not make a usage error.
func TestMRWMaxColsDoesNotApplyToAQuietRead(t *testing.T) {
	root := longline179Tree(t)
	t.Setenv("MRW_MAX_COLS", "100")
	if _, errOut, code := project178Run(t, root, "", "read", "--quiet", "a.txt"); code != 0 || strings.Contains(errOut, "MRW_MAX_COLS") {
		t.Fatalf("a --quiet read under the variable exited %d:\n%s", code, errOut)
	}
	plan := planFile(t, "@@ a.txt 2 replace\nnew\n")
	if out, code := runIn(t, root, "write", "--no-check", plan); code != 0 {
		t.Errorf("a write to the line a --quiet read licensed exited %d, want 0:\n%s", code, out)
	}
	for _, argv := range [][]string{
		{"read", "--grep", "one", "--sub", "uno", "b.txt"},
		{"read", "--grep", "one", "--count", "b.txt"},
		{"read", "--stat", "b.txt"},
	} {
		if out, code := runIn(t, root, argv...); code != 0 || strings.Contains(out, "MRW_MAX_COLS") {
			t.Errorf("mrw %v under the variable exited %d, want 0:\n%s", argv, code, out)
		}
	}
}
