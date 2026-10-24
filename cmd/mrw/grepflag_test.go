package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MRW-07: `--grep -C 2 PATTERN f` takes -C as the pattern, 2 and PATTERN as
// paths, and used to end `bad line number "?$"`. A --grep value that is exactly
// a flag read defines is refused by name, before any work; a dash pattern that
// is not one stays searchable.
func TestAGrepPatternThatIsAReadFlagIsRefusedByName(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.rb"), []byte("class A\n  def x\n  end\nend\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		argv []string
		code int
		want []string
	}{
		{[]string{"read", "--grep", "-C", "2", "def x|end:?$", "a.rb"}, 2, []string{`--grep took "-C" as its pattern`, "mrw read --grep PATTERN -C …", `--grep='(?:-C)'`}},
		{[]string{"read", "--grep=-C", "2", "a.rb"}, 2, []string{`--grep took "-C" as its pattern`}},
		{[]string{"read", "--grep", "--max-lines", "2", "def", "a.rb"}, 2, []string{`--grep took "--max-lines" as its pattern`}},
		{[]string{"read", "--grep", "def x", "-C", "2", "a.rb"}, 0, []string{"def x"}},
		{[]string{"read", "--grep", "-foo", "a.rb"}, 1, []string{"no file matched /-foo/"}},
		{[]string{"read", "--grep", "-i", "a.rb"}, 1, []string{"no file matched /-i/"}},
		{[]string{"read", "--grep=(?:-C)", "a.rb"}, 1, []string{"no file matched"}},
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
}
