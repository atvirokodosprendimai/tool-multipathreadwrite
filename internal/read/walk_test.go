package read

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// tree writes files into a fresh directory and returns it.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func paths(specs []Spec) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Path)
	}
	sort.Strings(out)
	return out
}

func walk(t *testing.T, root string, pattern string, in []string, exclude ...string) ([]Spec, []Problem) {
	t.Helper()
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatal(err)
	}
	specs, probs, err := Walk(root, in, WalkOptions{Pattern: re, Exclude: exclude})
	if err != nil {
		t.Fatalf("Walk returned a fatal error for a resolvable root: %v", err)
	}
	return specs, probs
}

func TestWalkReturnsOneSpecPerMatchingFile(t *testing.T) {
	root := tree(t, map[string]string{
		"a.go":        "package p\nfunc Target() {}\n",
		"sub/b.go":    "package s\nfunc Target() {}\n",
		"sub/c.go":    "package s\nfunc Other() {}\n",
		"deep/d/e.go": "package e\nfunc Target() {}\n",
	})
	specs, probs := walk(t, root, "Target", nil)
	if len(probs) != 0 {
		t.Errorf("problems on a clean tree: %v", probs)
	}
	want := []string{"a.go", "deep/d/e.go", "sub/b.go"}
	if got := paths(specs); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("walked %v, want %v", got, want)
	}
	for _, s := range specs {
		if len(s.Ranges) != 1 || s.Ranges[0].Re == nil {
			t.Errorf("%s got %d range(s); want one carrying the pattern", s.Path, len(s.Ranges))
		}
	}
}

func TestWalkSkipsAFileWithNoMatch(t *testing.T) {
	root := tree(t, map[string]string{"a.go": "package p\n", "b.go": "package p\nfunc Target() {}\n"})
	specs, _ := walk(t, root, "Target", nil)
	if got := paths(specs); len(got) != 1 || got[0] != "b.go" {
		t.Errorf("walked %v; a file with no match must not become a spec", got)
	}
}

func TestWalkHonoursExcludeGlobsAndPrunesDirectories(t *testing.T) {
	root := tree(t, map[string]string{
		"a.go":            "Target\n",
		"a_test.go":       "Target\n",
		"vendor/v.go":     "Target\n",
		"sub/vendor/w.go": "Target\n",
	})
	// The basename half of the rule: *_test.go must match at any depth, and a
	// directory name must prune the directory wherever it sits.
	specs, _ := walk(t, root, "Target", nil, "*_test.go", "vendor")
	if got := paths(specs); len(got) != 1 || got[0] != "a.go" {
		t.Errorf("walked %v, want just a.go", got)
	}
}

func TestWalkSkipsTheGitDirectory(t *testing.T) {
	root := tree(t, map[string]string{"a.go": "Target\n", ".git/COMMIT_EDITMSG": "Target\n"})
	if got := paths(mustSpecs(t, root, "Target")); len(got) != 1 || got[0] != "a.go" {
		t.Errorf("walked %v; .git is skipped unconditionally", got)
	}
}

func mustSpecs(t *testing.T, root, pattern string) []Spec {
	t.Helper()
	s, _ := walk(t, root, pattern, nil)
	return s
}

func TestWalkCannotLeaveTheRoot(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outer, "secret.txt"), []byte("Target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs, probs := walk(t, root, "Target", []string{"../secret.txt"})
	if len(specs) != 0 {
		t.Errorf("a path outside the root became a spec: %v", paths(specs))
	}
	if len(probs) != 1 || !strings.Contains(probs[0].Reason, "outside the root") {
		t.Errorf("problems = %v; want one naming the boundary", probs)
	}
}

func TestWalkDoesNotDescendASymlinkedDirectory(t *testing.T) {
	root := tree(t, map[string]string{"real/a.go": "Target\n"})
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got := paths(mustSpecs(t, root, "Target"))
	if len(got) != 1 || got[0] != "real/a.go" {
		t.Errorf("walked %v; the file is reached by its real name only", got)
	}
}

func TestWalkServesASymlinkToAFileInsideTheRoot(t *testing.T) {
	root := tree(t, map[string]string{"real.go": "Target\n"})
	if err := os.Symlink(filepath.Join(root, "real.go"), filepath.Join(root, "link.go")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// Rule 2 asks after Resolve, so a symlink to a file inside the root is a
	// candidate — `mrw read link.go` serves it today and --grep must not start
	// refusing it.
	specs, probs := walk(t, root, "Target", []string{"link.go"})
	if len(probs) != 0 || len(specs) != 1 || specs[0].Path != "link.go" {
		t.Errorf("specs=%v problems=%v; a symlinked file is a candidate", paths(specs), probs)
	}
}

func TestWalkSkipsADiscoveredFifoAndReportsANamedOne(t *testing.T) {
	root := tree(t, map[string]string{"a.go": "Target\n"})
	fifo := filepath.Join(root, "pipe")
	if err := exec.Command("mkfifo", fifo).Run(); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	// Discovered by walking: skipped silently, because the caller did not name
	// it and reading it would block forever.
	specs, probs := walk(t, root, "Target", nil)
	if got := paths(specs); len(got) != 1 || got[0] != "a.go" {
		t.Errorf("walked %v; a discovered fifo is skipped", got)
	}
	if len(probs) != 0 {
		t.Errorf("a discovered fifo produced problems: %v", probs)
	}
	// Named explicitly: refused BY NAME, because silence about something the
	// caller asked for is the failure this project refuses.
	specs, probs = walk(t, root, "Target", []string{"pipe"})
	if len(specs) != 0 || len(probs) != 1 {
		t.Errorf("specs=%v problems=%v; a named fifo is a problem, not silence", paths(specs), probs)
	}
}

func TestWalkReportsEveryBadPathAndServesTheGoodOnes(t *testing.T) {
	root := tree(t, map[string]string{"a.go": "Target\n"})
	specs, probs := walk(t, root, "Target", []string{"a.go", "nosuch.go", "../outside.go"})
	if got := paths(specs); len(got) != 1 || got[0] != "a.go" {
		t.Errorf("walked %v; a valid sibling is still served", got)
	}
	if len(probs) != 2 {
		t.Errorf("problems = %v; both bad paths are reported, not just the first", probs)
	}
}

func TestWalkDeduplicatesAPathNamedTwiceOrCoveredTwice(t *testing.T) {
	root := tree(t, map[string]string{"sub/a.go": "Target\n"})
	// Named twice, and named while also inside a named directory. Two specs for
	// one file would double its --max-lines budget, because Run resets the
	// budget per spec.
	specs, _ := walk(t, root, "Target", []string{"sub/a.go", "sub/a.go", "sub", "."})
	if got := paths(specs); len(got) != 1 || got[0] != "sub/a.go" {
		t.Errorf("walked %v, want one spec", got)
	}
}

func TestWalkKeepsTwoNamesForOneInode(t *testing.T) {
	root := tree(t, map[string]string{"a.go": "Target\n"})
	if err := os.Link(filepath.Join(root, "a.go"), filepath.Join(root, "b.go")); err != nil {
		t.Skipf("hardlinks unavailable: %v", err)
	}
	// mrw addresses files by path everywhere else; pretending two names are one
	// would make an address ambiguous.
	if got := paths(mustSpecs(t, root, "Target")); len(got) != 2 {
		t.Errorf("walked %v; a hardlink is two addresses", got)
	}
}
