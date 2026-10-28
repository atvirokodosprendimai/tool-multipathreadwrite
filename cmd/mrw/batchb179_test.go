package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ADR-179 Decision (b). --outline is refused beside --max-lines and --max-cols
// (a cap on a structure means nothing); under --max-chars a file whose rows do
// not fit is named on standard error, the read exits 1, and the --json receipt's
// problems agrees with the exit.
func TestOutlineLimits(t *testing.T) {
	root := grepTree(t, map[string]string{"c.go": "package p\n\nfunc A() {}\n\nfunc B() {}\n\nfunc C() {}\n"})
	for _, argv := range [][]string{
		{"--outline", "--max-lines", "1", "c.go"},
		{"--outline", "--max-cols", "10", "c.go"},
	} {
		if out, code := runIn(t, root, append([]string{"read"}, argv...)...); code != 2 || !strings.Contains(out, "--outline") || strings.Contains(out, "==>") {
			t.Errorf("mrw read %v exited %d, want 2 naming --outline:\n%s", argv, code, out)
		}
	}
	out, errOut, code := project178Run(t, root, "", "read", "--outline", "--max-chars", "60", "c.go")
	if code != 1 || !strings.Contains(errOut, "c.go: outline WITHHELD: --max-chars reached") {
		t.Errorf("--max-chars 60 exited %d, want 1 naming the withheld outline:\n%s\n%s", code, out, errOut)
	}
	jout, _, jcode := project178Run(t, root, "", "read", "--json", "--outline", "--max-chars", "60", "c.go")
	var doc struct {
		Receipt struct {
			Problems int `json:"problems"`
		} `json:"receipt"`
	}
	if i := strings.Index(jout, "{"); jcode != 1 || i < 0 || json.Unmarshal([]byte(jout[i:strings.LastIndex(jout, "}")+1]), &doc) != nil || doc.Receipt.Problems != 1 {
		t.Errorf("--json exited %d with problems %d, want 1 and 1:\n%s", jcode, doc.Receipt.Problems, jout)
	}
}

// Each refusal between two flags is true of both: --fold beside --no-comments
// does not say that either serves no line.
func TestFoldAndNoCommentsRefusalsAreTrueOfBoth(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package p\n"})
	out, code := runIn(t, root, "read", "--fold", "--no-comments", "a.go")
	if code != 2 || !strings.Contains(out, "--fold") || !strings.Contains(out, "--no-comments") || strings.Contains(out, "serves no line") {
		t.Errorf("--fold --no-comments exited %d:\n%s", code, out)
	}
}

// A requested line is never hidden by the line after it, and a grep match is
// never folded away, through the binary: the ledger records what was shown.
func TestNoCommentsAndFoldLeaveTheRequestedLineServed(t *testing.T) {
	root := grepTree(t, map[string]string{"c.go": "package p\n\n// a\n// b\nfunc A() {}\n"})
	out, code := runIn(t, root, "read", "--no-comments", "c.go:3")
	if code != 0 || strings.Contains(out, "comment line(s)") || !strings.Contains(out, "// a") {
		t.Errorf("--no-comments c.go:3 exited %d:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ c.go 3 replace\n// x\n")); code != 0 {
		t.Errorf("a write to the requested line exited %d, want 0:\n%s", code, out)
	}
}
