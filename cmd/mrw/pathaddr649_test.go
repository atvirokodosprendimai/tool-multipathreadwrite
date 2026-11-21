package main

import (
	"strings"
	"testing"
)

// #649. A path op has no line address: the plan writes `-`. Its FAIL and skip rows
// printed the unresolved bound, `0`, which no plan header says, though the row of a
// path op that applies already printed `-`. The JSON receipt's "addr" keeps the value
// it has (ADR-111).
func TestAPathOpRowPrintsTheAddressThePlanWrote(t *testing.T) {
	tree := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		writeFiles(t, root, map[string]string{"a.txt": "a\n", "b.txt": "b\n", "g.txt": "g\n"})
		if out, code := runIn(t, root, "read", "a.txt", "b.txt"); code != 0 {
			t.Fatalf("read: exit %d:\n%s", code, out)
		}
		return root
	}
	cases := []struct {
		name, plan string
		flags      []string
		want       string
		notWant    string
	}{
		{"an unlink of a missing file", "@@ nope.txt - unlink\n", nil, "FAIL nope.txt - unlink (plan line 1): nope.txt does not exist", "nope.txt 0 unlink"},
		{"a rename of a missing file", "@@ nope.txt - rename\nz.txt\n", nil, "FAIL nope.txt - rename (plan line 1): nope.txt does not exist", "nope.txt 0 rename"},
		{"a rename of a file never read", "@@ g.txt - rename\nz.txt\n", nil, "FAIL g.txt - rename (plan line 1): g.txt has not been read", "g.txt 0 rename"},
		{"an unlink beside a failing hunk is skipped", "@@ a.txt - unlink\n@@ b.txt 9 replace\nx\n", nil, "skip a.txt - unlink", "a.txt 0 unlink"},
		{"a rename beside a failing hunk is skipped", "@@ a.txt - rename\nz.txt\n@@ b.txt 9 replace\nx\n", nil, "skip a.txt - rename", "a.txt 0 rename"},
		{"--quiet keeps the failing row", "@@ nope.txt - unlink\n", []string{"--quiet"}, "FAIL nope.txt - unlink", "nope.txt 0 unlink"},
		{"--terse keeps the failing row", "@@ nope.txt - unlink\n", []string{"--terse"}, "FAIL nope.txt - unlink", "nope.txt 0 unlink"},
		{"a dry run of an unlink", "@@ a.txt - unlink\n", []string{"--dry-run"}, "ok   a.txt - unlink", "a.txt 0 unlink"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := tree(t)
			args := append([]string{"--no-check"}, c.flags...)
			out, _ := writeIn(t, root, append(args, planFile(t, c.plan))...)
			if !strings.Contains(out, c.want) {
				t.Errorf("want a row %q:\n%s", c.want, out)
			}
			if strings.Contains(out, c.notWant) {
				t.Errorf("the row names address 0 the plan never wrote (%q):\n%s", c.notWant, out)
			}
		})
	}
}
