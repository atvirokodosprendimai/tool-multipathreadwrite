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
// discovers. Where the two rules give one answer — a hit reached through a
// symlinked directory, an excluded real directory, a named path that passes
// through a link and then ".." — each finder serves exactly the expected file.
// A symlink to a single FILE is where they differ; see the test below.
func TestAstGrepAgreesWithWalkThroughSymlinks(t *testing.T) {
	root := tree(t, map[string]string{"real/sub/f.go": "Target\n", "real/f.go": "Target\n", "f.go": "Target\n"})
	if err := os.Symlink(filepath.Join(root, "real", "sub"), filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	target := regexp.MustCompile("Target")
	for _, tc := range []struct {
		name     string
		hit      string
		named    []string
		exclude  []string
		wantAG   []string
		wantWalk []string
	}{
		{"a hit through a linked directory, the link excluded", "link/f.go", nil, []string{"link"}, []string{"real/sub/f.go"}, []string{"f.go", "real/f.go", "real/sub/f.go"}},
		{"a hit through a linked directory, the real directory excluded", "link/f.go", nil, []string{"sub"}, []string{}, []string{"f.go", "real/f.go"}},
		{"a named path through a link and then ..", "link/../f.go", []string{"link/../f.go"}, nil, []string{"f.go"}, []string{"f.go"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installFakeAstGrepJSON(t, `[{"file":"`+tc.hit+`","range":{"start":{"line":0},"end":{"line":0}}}]`, 0)
			ag, agProblems, err := AstGrep(root, tc.named, "Target", tc.exclude, AstGrepOptions{})
			if err != nil {
				t.Fatal(err)
			}
			w, wProblems, err := Walk(root, tc.named, WalkOptions{Pattern: target, Exclude: tc.exclude})
			if err != nil {
				t.Fatal(err)
			}
			if len(agProblems) != 0 || len(wProblems) != 0 {
				t.Errorf("problems: ast-grep %v, walk %v", agProblems, wProblems)
			}
			if got := paths(ag); !reflect.DeepEqual(got, tc.wantAG) {
				t.Errorf("ast-grep served %v, want %v", got, tc.wantAG)
			}
			if got := paths(w); !reflect.DeepEqual(got, tc.wantWalk) {
				t.Errorf("the walk served %v, want %v", got, tc.wantWalk)
			}
		})
	}
}

// The accepted difference (Codex review of #256). For alias.go -> real.go with
// --exclude real.go, the walk judges the file by the name it discovers and
// serves alias.go, while AstGrep resolves a reported alias.go to real.go and
// drops it. Excluding a file's real name excludes its content from ast-grep;
// serving the alias instead would key AstGrep's hits by spelling, which is
// deferred in BACKLOG. Pinned so a change to either rule is a decision.
func TestAstGrepJudgesAFileSymlinkByItsTarget(t *testing.T) {
	root := tree(t, map[string]string{"real.go": "Target\n"})
	if err := os.Symlink(filepath.Join(root, "real.go"), filepath.Join(root, "alias.go")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	installFakeAstGrepJSON(t, `[{"file":"alias.go","range":{"start":{"line":0},"end":{"line":0}}}]`, 0)
	ag, _, err := AstGrep(root, nil, "Target", []string{"real.go"}, AstGrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("Target"), Exclude: []string{"real.go"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := paths(ag); len(got) != 0 {
		t.Errorf("ast-grep served %v; the accepted rule drops an alias whose target is excluded", got)
	}
	if got := paths(w); !reflect.DeepEqual(got, []string{"alias.go"}) {
		t.Errorf("the walk served %v, want [alias.go]: it judges a file by the name it discovers", got)
	}
}
