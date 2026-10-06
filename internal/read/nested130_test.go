package read

import (
	"os"
	"path/filepath"
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
	// Named as the start, the nested repository is walked, by its own rules
	// (Decision 2 on the ast-grep surface; the in-process review of #348).
	sk = WalkSkipped{}
	specs, _, err = AstGrep(root, []string{"nested"}, "package $A", nil, AstGrepOptions{Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	// The fake ast-grep answers every hit whatever is named; the judge's part
	// is that nested/n.go is served and nothing is counted nested.
	if got := strings.Join(servedPaths(specs), ","); !strings.Contains(got, "nested/n.go") || sk.Nested != 0 {
		t.Fatalf("named: served %q, skipped %+v: want nested/n.go and nothing counted", got, sk)
	}
}

// The Codex review of #348. On a filesystem that folds case, a start named
// NESTED walks nested; compared as strings, the walk that also met nested from
// the root kept it counted as not entered.
func TestANestedRepositoryNamedInAnotherCaseIsNotCounted(t *testing.T) {
	root := t.TempDir()
	plant(t, root, map[string]string{".git/HEAD": "x\n", "nested/.git/HEAD": "x\n", "nested/a.txt": "needle\n"})
	if _, err := os.Stat(filepath.Join(root, "NESTED")); err != nil {
		t.Skip("this filesystem does not fold case")
	}
	var sk WalkSkipped
	specs, _, _ := Walk(root, []string{".", "NESTED"}, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk})
	if len(specs) != 1 || sk.Nested != 0 {
		t.Errorf("served %v, skipped %+v: the repository a named start walked is still counted", specPaths(specs), sk)
	}
}

// The Codex review of #348. A directory that is both ignored and a nested
// repository counted as ignored_dirs under --grep and as nested under
// --ast-grep: both finders now ask "ignored" first.
func TestADirectoryBothIgnoredAndNestedCountsAlikeOnBothFinders(t *testing.T) {
	root := t.TempDir()
	plant(t, root, map[string]string{
		".git/HEAD": "x\n", ".gitignore": "nested/\n", "b.go": "package b\n",
		"nested/.git/HEAD": "x\n", "nested/a.go": "package a\n",
	})
	var walked WalkSkipped
	Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("package"), Skipped: &walked})
	installRecordingAstGrep(t, hit096("b.go", "nested/a.go"))
	var judged WalkSkipped
	if _, _, err := AstGrep(root, nil, "package $A", nil, AstGrepOptions{Skipped: &judged}); err != nil {
		t.Fatal(err)
	}
	want := WalkSkipped{IgnoredDirs: 1}
	if walked != want || judged != want {
		t.Errorf("walk skipped %+v, ast-grep skipped %+v, want both %+v", walked, judged, want)
	}
}

// The Codex re-review of #348: the index key folds as strings.EqualFold does,
// so names it calls equal are never kept in two buckets.
func TestTheFoldKeyMatchesEqualFold(t *testing.T) {
	for _, p := range [][2]string{{"nested", "NESTED"}, {"σ", "ς"}, {"Σ", "ς"}, {"k", "K"}} {
		if !strings.EqualFold(p[0], p[1]) || foldKey(p[0]) != foldKey(p[1]) {
			t.Errorf("%q and %q: EqualFold %v, keys %q %q", p[0], p[1], strings.EqualFold(p[0], p[1]), foldKey(p[0]), foldKey(p[1]))
		}
	}
	if foldKey("a") == foldKey("b") {
		t.Error("a and b share a key")
	}
}
