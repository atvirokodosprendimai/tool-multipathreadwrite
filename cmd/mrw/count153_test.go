package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// servedLine is a line read.Run serves: its number, a bar, the content.
var servedLine = regexp.MustCompile(`(?m)^ *[0-9]+\| `)

// ADR-153. `--grep P --count` says how many lines match in each file and in
// all, serves none of them and licenses none: the count is the number of
// matching lines a plain `--grep P` serves (it serves each one's next line too,
// ADR-161), a line matching twice counts once, and a write to a counted line is
// refused as unread.
func TestAGrepCountSaysHowManyLinesMatchAndLicensesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":    "x NEEDLE\ny\nNEEDLE NEEDLE\n",
		"b/c.txt": "NEEDLE\n",
		"d.txt":   "nothing\n",
	})
	out, err := readIn(t, root, "--grep", "NEEDLE", "--count")
	if err != nil {
		t.Fatalf("--count failed: %v\n%s", err, out)
	}
	for _, want := range []string{"==> a.go  3L  ", "  2 matching line(s)", "==> b/c.txt  1L  ", "  1 matching line(s)", "-- 3 matching line(s) in 2 file(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("the count does not say %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "d.txt") {
		t.Errorf("a file with no match was counted:\n%s", out)
	}
	if servedLine.MatchString(out) {
		t.Errorf("--count served a line:\n%s", out)
	}
	served, err := readIn(t, root, "--grep", "NEEDLE")
	if err != nil {
		t.Fatalf("--grep failed: %v\n%s", err, served)
	}
	if n := len(regexp.MustCompile(`(?m)^ *[0-9]+\| .*NEEDLE`).FindAllString(served, -1)); n != 3 {
		t.Fatalf("the same --grep serves %d matching lines, want the 3 --count counted:\n%s", n, served)
	}

	// A fresh ledger, so the serve above licenses nothing: only the count has
	// been read.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if out, err := readIn(t, root, "--grep", "NEEDLE", "--count"); err != nil {
		t.Fatalf("--count failed: %v\n%s", err, out)
	}
	before, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ a.go 1 replace\nchanged\n")); code != 1 {
		t.Errorf("a write to a line only counted exited %d, want 1 (refused as unread):\n%s", code, out)
	}
	after, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("a write to a line only counted changed the file: %q", after)
	}
}

// ADR-153 Decisions 3 and 4: --count answers a --grep and serves no line, so
// without a --grep, beside --ast-grep, and beside a flag that acts on served
// lines it is a usage error that names the flag.
func TestACountWithoutAGrepOrBesideAServedLineFlagIsRefused(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "NEEDLE\n"})
	for _, c := range []struct {
		args []string
		name string
	}{
		{[]string{"--count", "a.go"}, "--grep"},
		{[]string{"--count", "a.go:1"}, "--grep"},
		{[]string{"--count", "--ast-grep", "x"}, "--ast-grep"},
		{[]string{"--count", "--grep", "NEEDLE", "--stat"}, "--stat"},
		{[]string{"--count", "--grep", "NEEDLE", "-C", "1"}, "-C"},
		{[]string{"--count", "--grep", "NEEDLE", "--max-lines", "3"}, "--max-lines"},
		{[]string{"--count", "--grep", "NEEDLE", "--max-cols", "10"}, "--max-cols"},
		{[]string{"--count", "--grep", "NEEDLE", "-N"}, "-N"},
	} {
		out, err := readIn(t, root, c.args...)
		if err == nil || exitCode(err) != 2 {
			t.Errorf("mrw read %s: exit %d, want 2\n%s", strings.Join(c.args, " "), exitCode(err), out)
			continue
		}
		if msg := errString(err); !strings.Contains(msg, "--count") || !strings.Contains(msg, c.name) {
			t.Errorf("mrw read %s: %q does not name --count and %s", strings.Join(c.args, " "), msg, c.name)
		}
		if out != "" {
			t.Errorf("mrw read %s: a usage error printed an answer:\n%s", strings.Join(c.args, " "), out)
		}
	}
}

// ADR-153 Decision 2: a count that matched no file exits 1 with ADR-146's
// `no match:`, as a grep does.
func TestACountThatMatchesNothingExitsAsAGrep(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "NEEDLE\n"})
	out, err := readIn(t, root, "--grep", "ABSENT", "--count")
	if err == nil || exitCode(err) != 1 {
		t.Fatalf("a count that matched nothing: exit %d, want 1\n%s", exitCode(err), out)
	}
	if got := errString(err); !strings.HasPrefix(got, "no match: no file matched /ABSENT/") {
		t.Errorf("a count that matched nothing: %q, want it to begin `no match: no file matched /ABSENT/`", got)
	}
}
