package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-164 T5. -v and -o get a pointer, in every shape a caller meets them:
// `--grep -v a f` (no match, patternFlagHint), `--grep -v ' a' f` (refused for
// the padded value before the read runs, which -i shared), `read -v f` and
// `read -o f` (usageError). -z still gets none: it has no honest pointer.
func TestAnInvertedGrepSaysThereIsNone(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const inverted = "there is no inverted match: --grep serves the lines that match"
	const matchOnly = "there is no match-only output: --max-cols N cuts each line to a window round its match"
	for _, c := range []struct {
		argv []string
		code int
		want []string
	}{
		{[]string{"read", "--grep", "-v", "a", "f.txt"}, 1, []string{"no file matched /-v/", inverted}},
		{[]string{"read", "--grep", "-v", " a", "f.txt"}, 2, []string{"has edge whitespace", inverted}},
		{[]string{"read", "--grep", "-i", " a", "f.txt"}, 2, []string{"has edge whitespace", "(?i)"}},
		{[]string{"read", "-v", "f.txt"}, 2, []string{"flag provided but not defined: -v", inverted}},
		{[]string{"read", "-o", "f.txt"}, 2, []string{"flag provided but not defined: -o", matchOnly}},
	} {
		out, code := runIn(t, root, c.argv...)
		if code != c.code {
			t.Errorf("mrw %q: exit %d, want %d:\n%s", c.argv, code, c.code, out)
		}
		for _, w := range c.want {
			if !strings.Contains(out, w) {
				t.Errorf("mrw %q: lacks %q:\n%s", c.argv, w, out)
			}
		}
	}
	if out, _ := runIn(t, root, "read", "--grep", "-z", "f.txt"); strings.Contains(out, "took the value") {
		t.Errorf("-z got a pointer:\n%s", out)
	}
}
