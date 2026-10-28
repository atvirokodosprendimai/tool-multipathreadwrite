package main

import (
	"strings"
	"testing"
)

const fold179Body = "head\nx 1\nx 22\nx 3\nmid\nsame\nsame\nsame\nend\n"

// ADR-179 Decision (c). A folded line was not shown, so it is not licensed: a
// write to the second line of a run is refused as unread, one to the run's first
// line applies; standard input folds too; and the flag is refused, exit 2, beside
// every flag that serves no line, prints a plan or counts what it shows.
func TestAFoldedLineIsNotLicensed(t *testing.T) {
	root := grepTree(t, map[string]string{"f.txt": fold179Body, "f.json": `{"a": 1}` + "\n"})
	out, code := runIn(t, root, "read", "--fold", "f.txt")
	if code != 0 || !strings.Contains(out, "-- ×3: lines 2-4 like 2, differing only in digits") || strings.Contains(out, "x 22") {
		t.Fatalf("--fold exited %d:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ f.txt 3 replace\nz\n")); code != 1 || !strings.Contains(out, "has not been read") {
		t.Errorf("a write to a folded line exited %d, want 1 refused as unread:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ f.txt 2 replace\nz\n")); code != 0 {
		t.Errorf("a write to the run's first line exited %d, want 0:\n%s", code, out)
	}
	piped, _, code := project178Run(t, root, "r 1\nr 2\nr 3\n", "read", "--fold", "-")
	if code != 0 || !strings.Contains(piped, "-- ×3: lines 1-3 like 1, differing only in digits") {
		t.Errorf("--fold on standard input exited %d:\n%s", code, piped)
	}
	for _, argv := range [][]string{
		{"--fold", "--quiet", "f.txt"},
		{"--fold", "--stat", "f.txt"},
		{"--fold", "--grep", "x", "--count", "f.txt"},
		{"--fold", "--grep", "x", "--sub", "y", "f.txt"},
		{"--fold", "--grep", "x", "--max-matches", "1", "f.txt"},
		{"--fold", "--json-path", "$.a", "f.json"},
	} {
		if out, code := runIn(t, root, append([]string{"read"}, argv...)...); code != 2 || !strings.Contains(out, "--fold") || strings.Contains(out, "==>") {
			t.Errorf("mrw read %v exited %d, want 2 naming --fold:\n%s", argv, code, out)
		}
	}
}
