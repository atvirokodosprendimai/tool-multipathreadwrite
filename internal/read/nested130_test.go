package read

import (
	"regexp"
	"strings"
	"testing"
)

// ADR-130. Inside a checkout a directory holding its own .git — a repository,
// or a gitfile's submodule or worktree — is another project's, and git does
// not descend into it; mrw walked it with its own rules. It is not entered
// now and is counted; named, it is walked; NoIgnore walks it.
func TestANestedRepositoryIsNotEntered(t *testing.T) {
	needle := regexp.MustCompile("needle")
	root := t.TempDir()
	plant(t, root, map[string]string{
		".git/HEAD":        "x\n",
		"top.txt":          "needle\n",
		"nested/.git/HEAD": "x\n",
		"nested/a.txt":     "needle\n",
		"sub/.git":         "gitdir: ../elsewhere\n",
		"sub/b.txt":        "needle\n",
	})
	var sk WalkSkipped
	specs, _, _ := Walk(root, nil, WalkOptions{Pattern: needle, Skipped: &sk})
	if got := strings.Join(specPaths(specs), ","); got != "top.txt" || sk.Nested != 2 {
		t.Errorf("served %q, skipped %+v: want top.txt and two nested repositories counted", got, sk)
	}
	if note := SkipNote(sk, "--no-ignore"); !strings.Contains(note, "2 nested repositor") {
		t.Errorf("the skipped line does not name the nested repositories: %q", note)
	}
	sk = WalkSkipped{}
	specs, _, _ = Walk(root, []string{"nested"}, WalkOptions{Pattern: needle, Skipped: &sk})
	if got := strings.Join(specPaths(specs), ","); got != "nested/a.txt" || sk.Nested != 0 {
		t.Errorf("named: served %q, skipped %+v: want nested/a.txt and nothing counted", got, sk)
	}
	specs, _, _ = Walk(root, nil, WalkOptions{Pattern: needle, NoIgnore: true})
	if got := strings.Join(specPaths(specs), ","); got != "nested/a.txt,sub/b.txt,top.txt" {
		t.Errorf("NoIgnore: served %q, want every repository walked", got)
	}
	// A root that is no checkout walks each repository it holds, by its own
	// rules (ADR-116), and a repository nested in one of those is not entered.
	plain := t.TempDir()
	plant(t, plain, map[string]string{
		"r/.git/HEAD":       "x\n",
		"r/x.txt":           "needle\n",
		"r/inner/.git/HEAD": "x\n",
		"r/inner/y.txt":     "needle\n",
	})
	sk = WalkSkipped{}
	specs, _, _ = Walk(plain, nil, WalkOptions{Pattern: needle, Skipped: &sk})
	if got := strings.Join(specPaths(specs), ","); got != "r/x.txt" || sk.Nested != 1 {
		t.Errorf("a root that is no checkout: served %q, skipped %+v, want r/x.txt and r/inner counted", got, sk)
	}
}

// ADR-130 Decision 4. An ast-grep hit inside a nested repository is dropped and
// counted, as the walk would not have entered it (ADR-122).
func TestAnAstGrepHitInANestedRepositoryIsDropped(t *testing.T) {
	root := t.TempDir()
	plant(t, root, map[string]string{
		".git/HEAD":        "x\n",
		"b.go":             "package b\n",
		"nested/.git/HEAD": "x\n",
		"nested/n.go":      "package n\n",
	})
	installRecordingAstGrep(t, hit096("b.go", "nested/n.go"))
	var sk WalkSkipped
	specs, _, err := AstGrep(root, nil, "package $A", nil, AstGrepOptions{Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(servedPaths(specs), ","); got != "b.go" || sk.Nested != 1 {
		t.Fatalf("served %q, skipped %+v: want b.go and the nested repository counted", got, sk)
	}
}
