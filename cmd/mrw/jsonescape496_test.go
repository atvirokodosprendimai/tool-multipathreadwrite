package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Issue #496. Go's JSON encoder writes <, > and & as \u003c, \u003e and \u0026
// unless told not to, so `mrw read --json` showed the header's `==>` as
// `==\u003e` to every adapter and person that read the document raw (the
// v1.71.0 field test: five peers). The CLI's --json documents carry them
// literally; the decoded values are unchanged. Every encoder the CLI writes a
// document with is driven here but stats, whose document holds none of the
// three characters to look for.
func TestJSONDocumentsCarryAngleBracketsAndAmpersandsLiterally(t *testing.T) {
	t.Setenv("MRW_SHELL", "on")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("x<a>&\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(root, "p.mrw")
	if err := os.WriteFile(plan, []byte("@@ f.txt 1 replace anchor=\"q<z>&\"\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runSplit(t, "-C", root, "read", "f.txt"); err != nil { // licenses the plan's line
		t.Fatal(err)
	}
	cases := []struct {
		name string
		argv []string
		want string // a substring the raw document must hold
	}{
		{"read", []string{"read", "--json", "f.txt"}, "==> f.txt"},
		{"read content", []string{"read", "--json", "f.txt"}, "x<a>&"},
		{"read refusal", []string{"read", "--json", "f.txt:a<b"}, `bad line number \"a<b\"`},
		{"write", []string{"write", "--json", "--no-check", plan}, `anchor \"q<z>&\"`},
	}
	if _, err := exec.LookPath("sh"); err == nil {
		cases = append(cases, struct {
			name string
			argv []string
			want string
		}{"check", []string{"check", "--json", "--then-sh", `echo "<a>"`}, `echo \"<a>\"`})
	}
	for _, c := range cases {
		out, _ := runSplit(t, append([]string{"-C", root}, c.argv...)...) // the exit code is not this test's question
		for _, escaped := range []string{`\u003c`, `\u003e`, `\u0026`} {
			if strings.Contains(out, escaped) {
				t.Errorf("%s: the document escapes a character it need not (%s):\n%s", c.name, escaped, out)
			}
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: the document lacks %q:\n%s", c.name, c.want, out)
		}
		if !json.Valid([]byte(out)) {
			t.Errorf("%s: not one JSON document:\n%s", c.name, out)
		}
	}
}
