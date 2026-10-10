package apply

import (
	"strings"
	"testing"
)

// ADR-143. A plan with a hunk that lands in .git fails that hunk and writes
// nothing: the sibling edit of an ordinary file skips, whatever the op.
func TestAPlanThatTouchesDotGitWritesNothing(t *testing.T) {
	for _, c := range []struct {
		name string
		hunk Input
	}{
		{"create", Input{Path: ".git/hooks/pre-commit", Op: "create", Body: []string{"x"}, Lines: -1}},
		{"replace", Input{Path: ".git/config", Start: 1, End: 1, Op: "replace", Body: []string{"x"}, Lines: -1}},
		{"delete", Input{Path: ".git/config", Start: 1, End: 1, Op: "delete", Lines: -1}},
		{"unlink", Input{Path: ".git/config", Op: "unlink", Lines: -1}},
		{"rename into", Input{Path: "b.txt", Op: "rename", Body: []string{".git/b.txt"}, Lines: -1}},
		{"rename out", Input{Path: ".git/config", Op: "rename", Body: []string{"c.txt"}, Lines: -1}},
	} {
		root := t.TempDir()
		write(t, root, "a.txt", "one\n")
		write(t, root, "b.txt", "two\n")
		write(t, root, ".git/config", "[core]\n")
		sibling := Input{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"changed"}, Lines: -1, Index: 0}
		c.hunk.Index = 1
		res, err := Apply(root, []Input{sibling, c.hunk}, Options{Seen: map[string]Seen{"a.txt": {SHA: shaOfFile(t, root, "a.txt")}, "b.txt": {SHA: shaOfFile(t, root, "b.txt")}}})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if res.Applied || len(res.Hunks) != 2 {
			t.Fatalf("%s: applied=%v hunks=%+v, want a refused plan with two verdicts", c.name, res.Applied, res.Hunks)
		}
		if r := res.Hunks[1].Reason; res.Hunks[1].Status != StatusFailed || !strings.Contains(r, "inside a .git directory") || strings.Contains(r, "--root") {
			t.Errorf("%s: hunk = %+v, want a failure naming .git and giving no root advice", c.name, res.Hunks[1])
		}
		if res.Hunks[0].Status != StatusSkipped {
			t.Errorf("%s: the sibling is %q, want skipped", c.name, res.Hunks[0].Status)
		}
		if read(t, root, "a.txt") != "one\n" || read(t, root, ".git/config") != "[core]\n" {
			t.Errorf("%s: a refused plan wrote the tree", c.name)
		}
	}
}
