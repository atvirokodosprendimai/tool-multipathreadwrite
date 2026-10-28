package main

import (
	"strings"
	"testing"
)

const comments179Go = "package p\n\n// Doc one.\n// Doc two.\nfunc A() {}\n"

// ADR-179 Decision (h). An elided comment line was not shown, so it is not
// licensed: a write to it is refused as unread, one to a line that was shown
// applies; the flag is refused, exit 2, beside every flag that serves no line,
// prints a plan, hides lines of its own or counts what it shows.
func TestAnElidedCommentLineIsNotLicensed(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": comments179Go, "f.json": `{"a": 1}` + "\n"})
	out, code := runIn(t, root, "read", "--no-comments", "a.go")
	if code != 0 || !strings.Contains(out, "-- … 2 comment line(s), 3-4") || strings.Contains(out, "Doc one") {
		t.Fatalf("--no-comments exited %d:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ a.go 3 replace\n// x\n")); code != 1 || !strings.Contains(out, "has not been read") {
		t.Errorf("a write to an elided line exited %d, want 1 refused as unread:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ a.go 5 replace\nfunc A() { _ = 1 }\n")); code != 0 {
		t.Errorf("a write to a shown line exited %d, want 0:\n%s", code, out)
	}
	for _, argv := range [][]string{
		{"--no-comments", "--quiet", "a.go"},
		{"--no-comments", "--stat", "a.go"},
		{"--no-comments", "--grep", "A", "--count", "a.go"},
		{"--no-comments", "--grep", "A", "--sub", "B", "a.go"},
		{"--no-comments", "--grep", "A", "--max-matches", "1", "a.go"},
		{"--no-comments", "--fold", "a.go"},
		{"--no-comments", "--outline", "a.go"},
		{"--no-comments", "--json-path", "$.a", "f.json"},
	} {
		if out, code := runIn(t, root, append([]string{"read"}, argv...)...); code != 2 || !strings.Contains(out, "--no-comments") || strings.Contains(out, "==>") {
			t.Errorf("mrw read %v exited %d, want 2 naming --no-comments:\n%s", argv, code, out)
		}
	}
}

// ADR-179 Decision (h). A file that is not .go, and standard input, is served
// whole, and ONE stderr line names them for the whole invocation.
func TestNoCommentsNamesTheFilesItServedWholeOnce(t *testing.T) {
	root := grepTree(t, map[string]string{"a.py": "# c\n# d\n", "b.py": "# e\n# f\n", "c.go": comments179Go})
	out, errOut, code := project178Run(t, root, "", "read", "--no-comments", "a.py", "b.py", "c.go")
	const want = "mrw: --no-comments reads .go files; served whole: a.py, b.py\n"
	if code != 0 || strings.Count(errOut, "--no-comments reads") != 1 || !strings.Contains(errOut, want) || !strings.Contains(out, "# c\n") || !strings.Contains(out, "-- … 2 comment line(s), 3-4") {
		t.Errorf("exit %d, want 0 with one line %q:\n%s\n%s", code, want, errOut, out)
	}
	out, errOut, code = project178Run(t, root, "// a\n// b\n", "read", "--no-comments", "-")
	if code != 0 || !strings.Contains(errOut, "mrw: --no-comments reads .go files; served whole: -\n") || !strings.Contains(out, "// a") {
		t.Errorf("standard input exited %d:\n%s\n%s", code, errOut, out)
	}
}
