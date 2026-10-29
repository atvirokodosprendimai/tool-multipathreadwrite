package main

import (
	"strings"
	"testing"
)

// ADR-182. `mrw seen` and a refusal describe a quiet read, a --sub plan and a
// write by the licence the ledger holds, never by a command that may not have
// run (issue #555).
func TestSeenNamesNoProvenanceTheLedgerDoesNotHold(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{
		"h.txt": "a\nb\nc\nd\ne\n",
		"g.txt": "a\nb\n",
		"e.txt": "a\nb\n",
		"s.txt": "Old\nb\n",
	})
	for _, args := range [][]string{
		{"read", "--quiet", "h.txt:1-2"},
		{"read", "--quiet", "g.txt"},
		{"read", "--quiet", "e.txt:/nomatch/"},
		{"read", "--grep", "Old", "--sub", "New", "s.txt"},
	} {
		run("", args...)
	}
	seen := func() string {
		o, e, c := run("", "seen")
		if c != 0 {
			t.Fatalf("mrw seen: exit %d:\n%s%s", c, o, e)
		}
		return o
	}
	line := func(out, file string) string {
		for _, l := range strings.Split(out, "\n") {
			if strings.HasSuffix(l, "  "+file) {
				return l
			}
		}
		return ""
	}
	out := seen()
	for file, want := range map[string]string{
		"h.txt": "no lines, and licensed lines 1-3",
		"s.txt": "no lines, and licensed lines 1",
		"g.txt": "no lines, and licensed the whole file",
		"e.txt": "no lines, and licensed none",
	} {
		if l := line(out, file); !strings.Contains(l, "  "+want+" ") {
			t.Errorf("mrw seen for %s = %q, want %q:\n%s", file, l, want, out)
		}
	}
	for _, old := range []string{"since mrw wrote it", "a --sub plan licensed"} {
		if strings.Contains(out, old) {
			t.Errorf("mrw seen still names %q:\n%s", old, out)
		}
	}
	// The pair: line 4 is not licensed, line 2 is.
	if o, e, c := run("@@ h.txt 4 replace\nzzz\n", "write", "-"); c != 1 || !strings.Contains(o+e, "mrw served no lines, and licensed lines 1-3") || subFile(t, root, "h.txt") != "a\nb\nc\nd\ne\n" {
		t.Errorf("write to an unlicensed line: exit %d, want 1 naming the licence, nothing written:\n%s%s", c, o, e)
	}
	if o, e, c := run("@@ h.txt 2 replace\nzzz\n", "write", "-"); c != 0 {
		t.Errorf("write to a licensed line: exit %d:\n%s%s", c, o, e)
	}
	if l := line(seen(), "h.txt"); !strings.Contains(l, "  no lines, and licensed the whole file ") {
		t.Errorf("mrw seen after the write = %q", l)
	}
	run("", "read", "h.txt:2") // serves 2 and the line after it (ADR-161): shown lines beside a whole-file licence
	if l := line(seen(), "h.txt"); !strings.Contains(l, "  lines 2-3, and licensed the whole file ") {
		t.Errorf("mrw seen after a read of a written file = %q", l)
	}
}
