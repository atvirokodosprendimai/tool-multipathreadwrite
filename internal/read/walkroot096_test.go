package read

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-096 review: Walk and AstGrep build their root with rooted.Real, which on
// Windows also follows a junction (ADR-071); filepath.EvalSymlinks left one as
// written, and a walk from it served nothing. That half runs on Windows
// (cmd/mrw/junction096_windows_test.go). This is the half every platform can
// run: a root reached through a symlink is walked whether the caller names no
// path, ".", or a directory below it, under the root's own names.
func TestWalkUnderARootReachedThroughALinkServesEveryStart(t *testing.T) {
	root := tree(t, map[string]string{"top.go": "Target\n", "d/f.go": "Target\n"})
	linked := filepath.Join(t.TempDir(), "L")
	if err := os.Symlink(root, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cases := []struct {
		what string
		in   []string
		want string
	}{
		{"no path", nil, "d/f.go,top.go"},
		{".", []string{"."}, "d/f.go,top.go"},
		{"a directory below the root", []string{"d"}, "d/f.go"},
	}
	for _, c := range cases {
		specs, probs := walk(t, linked, "Target", c.in)
		if len(probs) != 0 {
			t.Errorf("%s: problems = %v; the root reached through a link is the root", c.what, probs)
		}
		if got := strings.Join(paths(specs), ","); got != c.want {
			t.Errorf("%s: walked %q, want %q", c.what, got, c.want)
		}
	}
}
