package main

import (
	"regexp"
	"strings"
	"testing"
)

// ADR-203 Decision 8. The plan `read --sub` prints carries anchor= on each hunk,
// so written by a later call to a file mrw wrote, it lands and is not refused as
// a number alone. The anchor is a bounded piece of the line, valid UTF-8, free
// of CR; a line with no such piece says so on stderr.
func TestAReadSubPlanLandsWhenWrittenLaterOnAWrittenFile(t *testing.T) {
	long := "Old " + strings.Repeat("x", 5000)
	lines := []string{
		`say "hi" Old`,
		`back\slash Old \" end`,
		`trailing Old\`,
		"tab\tOld",
		"héllo Old wörld ✓",
		long,
		"ab\rOld\rcd", // a stray CR in an LF file
		"plain",
	}
	root, run := subRun(t)
	subTree(t, root, map[string]string{"f.txt": strings.Join(lines, "\n") + "\n"})
	if _, e, c := run("", "read", "f.txt"); c != 0 {
		t.Fatalf("read: exit %d\n%s", c, e)
	}
	// A position-only write leaves the file written, with nothing shown.
	if out, e, c := run("@@ f.txt $ insert-after\nlast\n", "write", "--no-check", "-"); c != 0 {
		t.Fatalf("seed write: exit %d\n%s%s", c, out, e)
	}
	plan, errOut, code := run("", "read", "--grep", "Old", "--sub", "New", "f.txt")
	if code != 0 || strings.Count(plan, " replace sha=") != 7 || strings.Contains(errOut, "carry no anchor=") {
		t.Fatalf("read --sub: exit %d, want seven anchored hunks and no note:\n%.600s\n%s", code, plan, errOut)
	}
	for _, h := range regexp.MustCompile(`(?m)^@@ .*$`).FindAllString(plan, -1) {
		if len(h) > 400 {
			t.Errorf("a header of %d bytes: the anchor is not bounded: %.120s…", len(h), h)
		}
	}
	if !strings.Contains(plan, ` anchor="Old `+strings.Repeat("x", 196)+`"`) {
		t.Errorf("the long line is not anchored on its first 200 bytes")
	}
	if !strings.Contains(plan, `anchor="ab"`) {
		t.Errorf("the line with a stray CR is not anchored on a CR-free segment:\n%.900s", plan)
	}
	if out, e, c := run(plan, "write", "--no-check", "-"); c != 0 {
		t.Fatalf("the plan written by a later call: exit %d\n%s%s", c, out, e)
	}
	got := subFile(t, root, "f.txt")
	if strings.Contains(strings.ReplaceAll(got, "\r", ""), "Old") || strings.Count(got, "New") != 7 {
		t.Errorf("not every Old became New:\n%.400s", got)
	}
}

// A line with no piece an anchor can name gets none, and the read says how many.
func TestAReadSubPlanNamesTheHunksThatCarryNoAnchor(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{"f.txt": "a\n\nb\n"})
	plan, errOut, code := run("", "read", "--grep", "^$", "--sub", "BLANK", "f.txt")
	if code != 0 || strings.Contains(plan, "anchor=") || !strings.Contains(errOut, "-- note: 1 hunk(s) carry no anchor=") || !strings.Contains(errOut, "read; or use write --grep --sub") {
		t.Errorf("exit %d, plan %q, stderr:\n%s", code, plan, errOut)
	}
}
