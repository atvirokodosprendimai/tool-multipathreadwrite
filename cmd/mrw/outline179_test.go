package main

import (
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

const outline179Go = "package p\n\nfunc A() {}\n\nfunc B() {}\n"

// ADR-179 Decision (b). Files --outline cannot outline are named in ONE stderr
// line for the whole invocation (five names and a count), their headers still
// print, and the read exits 1 incomplete.
func TestOutlineNamesTheFilesItCannotOutlineOnce(t *testing.T) {
	files := map[string]string{"c.go": outline179Go}
	for _, n := range []string{"a.py", "b.py", "d.py", "e.py", "f.py", "g.py", "h.py"} {
		files[n] = "x = 1\n"
	}
	root := grepTree(t, files)
	out, errOut, code := project178Run(t, root, "", "read", "--outline", "a.py", "b.py", "c.go", "d.py", "e.py", "f.py", "g.py", "h.py")
	const want = "mrw: --outline reads .go and .md files; no outline for a.py, b.py, d.py, e.py, f.py and 2 more\n"
	if code != 1 || strings.Count(errOut, "--outline reads") != 1 || !strings.Contains(errOut, want) || !strings.Contains(out, "incomplete:") {
		t.Errorf("exit %d, want 1 with one line %q:\n%s", code, want, errOut)
	}
	if !strings.Contains(out, "==> a.py") || !strings.Contains(out, "3-3 func A\n5-5 func B\n") {
		t.Errorf("headers or the Go outline missing:\n%s", out)
	}
	out, errOut, code = project178Run(t, root, "", "read", "--outline", "c.go")
	if code != 0 || errOut != "" || !strings.Contains(out, "3-3 func A") {
		t.Errorf("a file it can outline exited %d with stderr %q:\n%s", code, errOut, out)
	}
}

// ADR-179 Decision (b). An outline licenses no line: the file is hashed and a
// write to it is refused as unread; the flag is refused, exit 2, by name, beside
// every flag that serves lines or plans, on standard input and with an address.
func TestOutlineLicensesNoLine(t *testing.T) {
	root := grepTree(t, map[string]string{"c.go": outline179Go, "f.json": `{"a": 1}` + "\n"})
	if _, code := runIn(t, root, "read", "--outline", "c.go"); code != 0 {
		t.Fatalf("--outline exited %d", code)
	}
	if l, err := seen.Snapshot(root); err != nil || len(l) != 1 {
		t.Errorf("the ledger holds %v (%v), want c.go hashed", l, err)
	}
	if out, code := runIn(t, root, "write", "--no-check", planFile(t, "@@ c.go 3 replace\nfunc A() { _ = 1 }\n")); code != 1 || !strings.Contains(out, "has not been read") {
		t.Errorf("a write after an outline exited %d, want 1 refused as unread:\n%s", code, out)
	}
	for _, argv := range [][]string{
		{"--outline", "--grep", "A", "c.go"},
		{"--outline", "--ast-grep", "A", "c.go"},
		{"--outline", "--sub", "x", "--grep", "A", "c.go"},
		{"--outline", "--count", "--grep", "A", "c.go"},
		{"--outline", "--stat", "c.go"},
		{"--outline", "--quiet", "c.go"},
		{"--outline", "-C", "2", "c.go"},
		{"--outline", "--fold", "c.go"},
		{"--outline", "--max-matches", "1", "--grep", "A", "c.go"},
		{"--outline", "--json-path", "$.a", "f.json"},
		{"--outline", "--if-changed", "c.go=00000000", "c.go"},
		{"--outline", "c.go:1-3"},
	} {
		if out, code := runIn(t, root, append([]string{"read"}, argv...)...); code != 2 || !strings.Contains(out, "--outline") || strings.Contains(out, "==>") {
			t.Errorf("mrw read %v exited %d, want 2 naming --outline:\n%s", argv, code, out)
		}
	}
	if out, _, code := project178Run(t, root, "package p\n", "read", "--outline", "-"); code != 2 || !strings.Contains(out, "--outline") {
		t.Errorf("--outline on standard input exited %d, want 2 naming it:\n%s", code, out)
	}
}
