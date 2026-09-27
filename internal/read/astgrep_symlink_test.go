package read

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
)

// ADR-064 left ast-grep's symlink spellings unpromised (BACKLOG): AstGrep
// judges a hit by the path astGrepRel resolves, Walk by the spelling it
// discovers. Where mrw's own normalisation decides the answer — a hit reached
// through a symlinked directory, an excluded real directory, a named path that
// passes through a link and then ".." — the two agree. A symlink to a single
// file is not pinned here: whether the binary reports the alias at all is the
// binary's own symlink policy, and filtering its hits cannot add one.
func TestAstGrepAgreesWithWalkThroughSymlinks(t *testing.T) {
	root := tree(t, map[string]string{"real/sub/f.go": "Target\n", "real/f.go": "Target\n", "f.go": "Target\n"})
	if err := os.Symlink(filepath.Join(root, "real", "sub"), filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	target := regexp.MustCompile("Target")
	for _, tc := range []struct {
		name    string
		hit     string
		named   []string
		exclude []string
	}{
		{"a hit through a linked directory, the link excluded", "link/f.go", nil, []string{"link"}},
		{"a hit through a linked directory, the real directory excluded", "link/f.go", nil, []string{"sub"}},
		{"a named path through a link and then ..", "link/../f.go", []string{"link/../f.go"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installFakeAstGrepJSON(t, `[{"file":"`+tc.hit+`","range":{"start":{"line":0},"end":{"line":0}}}]`, 0)
			ag, agProblems, err := AstGrep(root, tc.named, "Target", tc.exclude)
			if err != nil {
				t.Fatal(err)
			}
			w, _, err := Walk(root, tc.named, WalkOptions{Pattern: target, Exclude: tc.exclude})
			if err != nil {
				t.Fatal(err)
			}
			walked := map[string]bool{}
			for _, p := range paths(w) {
				walked[p] = true
			}
			for _, p := range paths(ag) {
				if !walked[p] {
					t.Errorf("ast-grep served %s, which the walk does not (walk: %v, ast-grep: %v, problems %v)", p, paths(w), paths(ag), agProblems)
				}
			}
			if tc.named != nil && !reflect.DeepEqual(paths(ag), paths(w)) {
				t.Errorf("a named path: ast-grep %v, walk %v", paths(ag), paths(w))
			}
		})
	}
}
