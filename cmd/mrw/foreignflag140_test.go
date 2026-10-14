package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-140. A flag grep has and `mrw read` lacks is still a usage error, exit 2
// with the old prefix, and now says what mrw spells instead. `--grep -i NEEDLE`
// is no unknown flag: -i is taken as the pattern and the read ends "no file
// matched /-i/", which carries the same hint. Anything else is worded as before.
func TestAForeignReadFlagIsAnsweredWithMrwsSpelling(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for flag, sentence := range foreignFlags {
		dash := "-"
		if len(flag) > 1 {
			dash = "--"
		}
		out, code := runIn(t, root, "read", dash+flag, "--grep", "x")
		if code != 2 || !strings.Contains(out, "flag provided but not defined: -"+flag) || !strings.HasSuffix(strings.TrimSpace(out), "; "+sentence) {
			t.Errorf("mrw read %s%s: exit %d, want 2 with the old prefix and %q last:\n%s", dash, flag, code, sentence, out)
		}
	}
	for flag, sentence := range foreignFlags {
		if len(flag) != 1 {
			continue
		}
		out, code := runIn(t, root, "read", "--grep", "-"+flag, "f.txt")
		if code != 1 || !strings.Contains(out, "no file matched /-"+flag+"/ — "+sentence) || !strings.Contains(out, "took the value after --grep as the pattern") {
			t.Errorf("mrw read --grep -%s f.txt: exit %d, want 1 with the hint:\n%s", flag, code, out)
		}
	}
	for _, c := range []struct {
		argv []string
		code int
		want string // a substring the answer must not carry: any table sentence
	}{
		{[]string{"read", "--bogus"}, 2, "; "},
		{[]string{"write", "-i"}, 2, "(?i)"},
		{[]string{"read", "--grep", "-v", "f.txt"}, 1, "took the value"},
		{[]string{"read", "--grep", "-ii", "f.txt"}, 1, "took the value"},
		{[]string{"read", "--grep", "-count", "f.txt"}, 1, "took the value"}, // a real search string that is also a table key: only a dash and ONE letter is a flag
	} {
		out, code := runIn(t, root, c.argv...)
		if code != c.code || strings.Contains(out, c.want) {
			t.Errorf("mrw %s: exit %d (want %d), carried %q:\n%s", strings.Join(c.argv, " "), code, c.code, c.want, out)
		}
	}
}

// The runnable equivalents the table names do what their sentences say.
func TestEveryForeignFlagHintIsATrueSpelling(t *testing.T) {
	root := t.TempDir()
	body := "Alpha NEEDLE x\n-dash line\na.b word\nfoo(bar)\nother\n"
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		why  string
		argv []string
		want string
	}{
		{"-i is (?i)", []string{"read", "--grep", "(?i)needle", "f.txt"}, "Alpha NEEDLE x"},
		{"-e of a dash pattern is --grep=-PATTERN", []string{"read", "--grep=-dash", "f.txt"}, "-dash line"},
		{"-F is \\Q…\\E", []string{"read", "--grep", `\Qfoo(bar)\E`, "f.txt"}, "foo(bar)"},
		{"-w is \\b…\\b", []string{"read", "--grep", `\bword\b`, "f.txt"}, "a.b word"},
		{"-A/-B is -C N", []string{"read", "--grep", "dash", "-C", "1", "f.txt"}, "Alpha NEEDLE x"},
		{"-n is on, -N drops", []string{"read", "-N", "f.txt:1"}, "Alpha NEEDLE x"},
	} {
		out, code := runIn(t, root, c.argv...)
		if code != 0 || !strings.Contains(out, c.want) {
			t.Errorf("%s: exit %d, want %q:\n%s", c.why, code, c.want, out)
		}
	}
	if out, _ := runIn(t, root, "read", "-N", "f.txt:1"); strings.Contains(out, "1|") {
		t.Errorf("-N did not drop the numbers:\n%s", out)
	}
	// -l is --grep P --stat: the matching files, with no content.
	out, code := runIn(t, root, "read", "--grep", "(?i)needle", "--stat")
	if code != 0 || !strings.Contains(out, "==> f.txt") || strings.Contains(out, "@@") {
		t.Errorf("-l is --grep P --stat: exit %d\n%s", code, out)
	}
}
