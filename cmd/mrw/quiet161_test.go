package main

import (
	"regexp"
	"strings"
	"testing"
)

// ledgerLines is what `mrw seen` says of each file, without the state
// directory and the comment lines, which name a run and not a licence.
func ledgerLines(t *testing.T, run func(string, ...string) (string, string, int)) string {
	t.Helper()
	out, errOut, code := run("", "seen")
	if code != 0 {
		t.Fatalf("mrw seen: exit %d\n%s", code, errOut)
	}
	var keep []string
	for i, l := range strings.Split(out, "\n") {
		if i == 0 || strings.HasPrefix(l, "#") {
			continue
		}
		keep = append(keep, l)
	}
	return strings.Join(keep, "\n")
}

// ADR-161 Decision 9. `read --quiet` licenses what the same read without it
// licenses and prints no line of it: each file's header, then
// `-- licensed N line(s) in M file(s)`; and it is refused beside --count,
// --stat and --sub, which serve or license differently.
func TestAQuietReadLicensesAndPrintsNoLine(t *testing.T) {
	files := map[string]string{
		"f.txt":    "a\nb\nc\nd\ne\n",
		"long.txt": "short\n" + strings.Repeat("x", 40) + "c\nend\n",
	}
	numbered := regexp.MustCompile(`(?m)^ *\d+\| |^@@ \d`)
	for _, args := range [][]string{
		{"f.txt"},
		{"f.txt:2-3"},
		{"f.txt:/c/", "long.txt:2"},
		{"--max-lines", "1", "f.txt"},
		{"--max-lines", "0", "f.txt"},
		{"--max-lines", "1", "f.txt:/a|e/"},
		{"--max-cols", "10", "long.txt:1-3"},
		{"--grep", "c", "."},
	} {
		qRoot, quiet := subRun(t)
		pRoot, plain := subRun(t)
		subTree(t, qRoot, files)
		subTree(t, pRoot, files)
		qOut, qErr, qCode := quiet("", append([]string{"read", "--quiet"}, args...)...)
		pOut, _, pCode := plain("", append([]string{"read"}, args...)...)
		name := strings.Join(args, " ")
		if qCode != pCode {
			t.Errorf("%s: --quiet exited %d, the plain read %d", name, qCode, pCode)
		}
		if numbered.MatchString(qOut) {
			t.Errorf("%s: --quiet printed a served line or a range line:\n%s", name, qOut)
		}
		if !strings.Contains(qOut, "==> ") || !regexp.MustCompile(`(?m)^-- licensed \d+ line\(s\) in \d+ file\(s\)$`).MatchString(qOut) {
			t.Errorf("%s: --quiet lacks a header or the licensed line:\n%s%s", name, qOut, qErr)
		}
		if q, p := ledgerLines(t, quiet), ledgerLines(t, plain); q != p {
			t.Errorf("%s: the ledger after --quiet\n%s\nis not the plain read's\n%s\n(plain stdout:\n%s)", name, q, p, pOut)
		}
	}

	// The count is the lines licensed: 2-3 is two lines, and a plan for one
	// of them applies with no further read.
	root, run := subRun(t)
	subTree(t, root, files)
	out, _, code := run("", "read", "--quiet", "f.txt:2-3")
	if code != 0 || !strings.Contains(out, "-- licensed 2 line(s) in 1 file(s)") {
		t.Errorf("read --quiet f.txt:2-3: exit %d, want 0 and 2 lines licensed:\n%s", code, out)
	}
	if o, e, c := run("@@ f.txt 2 replace\nB\n", "write", "--no-check", "-"); c != 0 || subFile(t, root, "f.txt") != "a\nB\nc\nd\ne\n" {
		t.Errorf("a plan of a quietly read line: exit %d\n%s%s", c, o, e)
	}

	for _, args := range [][]string{
		{"--quiet", "--count", "--grep", "c", "."},
		{"--quiet", "--stat", "f.txt"},
		{"--quiet", "--grep", "c", "--sub", "C", "."},
	} {
		if o, e, c := run("", append([]string{"read"}, args...)...); c != 2 || o != "" {
			t.Errorf("mrw read %s: exit %d stdout %q, want 2 and nothing:\n%s", strings.Join(args, " "), c, o, e)
		}
	}
}
