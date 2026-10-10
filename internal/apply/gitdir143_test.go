package apply

import (
	"os"
	"path/filepath"
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
		// The entry is a link inside .git that leads elsewhere: unlink and rename
		// act on the entry, so the target's name is not what is judged (the Codex
		// review of #382).
		{"via link unlink", Input{Path: "alias/hooks/pre-commit", Op: "unlink", Lines: -1}},
		{"via link rename onto", Input{Path: "b.txt", Op: "rename", Body: []string{"alias/hooks/pre-commit"}, Lines: -1}},
	} {
		root := t.TempDir()
		write(t, root, "a.txt", "one\n")
		write(t, root, "b.txt", "two\n")
		write(t, root, ".git/config", "[core]\n")
		viaLink := strings.HasPrefix(c.name, "via link")
		if viaLink {
			write(t, root, "script.sh", "x\n")
			write(t, root, ".git/hooks/placeholder", "")
			if os.Symlink("../../script.sh", filepath.Join(root, ".git", "hooks", "pre-commit")) != nil || os.Symlink(".git", filepath.Join(root, "alias")) != nil {
				continue
			}
		}
		sibling := Input{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"changed"}, Lines: -1, Index: 0}
		c.hunk.Index = 1
		res, err := Apply(root, []Input{sibling, c.hunk}, Options{Seen: map[string]Seen{"a.txt": {SHA: shaOfFile(t, root, "a.txt")}, "b.txt": {SHA: shaOfFile(t, root, "b.txt")}}})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if res.Applied || len(res.Hunks) != 2 {
			t.Fatalf("%s: applied=%v hunks=%+v, want a refused plan with two verdicts", c.name, res.Applied, res.Hunks)
		}
		if r := res.Hunks[1].Reason; res.Hunks[1].Status != StatusFailed || !strings.Contains(r, ".git directory") || strings.Contains(r, "--root") {
			t.Errorf("%s: hunk = %+v, want a failure naming .git and giving no root advice", c.name, res.Hunks[1])
		}
		if res.Hunks[0].Status != StatusSkipped {
			t.Errorf("%s: the sibling is %q, want skipped", c.name, res.Hunks[0].Status)
		}
		if _, err := os.Lstat(filepath.Join(root, ".git", "hooks", "pre-commit")); viaLink && err != nil {
			t.Errorf("%s: the hook entry inside .git is gone: %v", c.name, err)
		}
		if read(t, root, "a.txt") != "one\n" || read(t, root, ".git/config") != "[core]\n" {
			t.Errorf("%s: a refused plan wrote the tree", c.name)
		}
	}
}

// ADR-143 left out a hard link in the tree to a file under .git. An edit is a
// temp file renamed over the name, so the other name keeps its content: the link
// is not a way into .git. Pinned here so a change to write in place is seen.
func TestAHardLinkToAFileUnderDotGitIsNotAWayIn(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".git/config", "[core]\n")
	if err := os.Link(filepath.Join(root, ".git", "config"), filepath.Join(root, "linked.txt")); err != nil {
		t.Skipf("no hard links here: %v", err)
	}
	res, err := Apply(root, []Input{{Path: "linked.txt", Start: 1, End: 1, Op: "replace", Body: []string{"[core]\thooksPath = x"}, Lines: -1}},
		Options{Seen: map[string]Seen{"linked.txt": {SHA: shaOfFile(t, root, "linked.txt")}}})
	if err != nil || !res.Applied {
		t.Fatalf("edit of the linked name: applied=%v err=%v hunks=%+v", res.Applied, err, res.Hunks)
	}
	if got := read(t, root, ".git/config"); got != "[core]\n" {
		t.Errorf(".git/config = %q after an edit of its hard link, want it untouched", got)
	}
	if got := read(t, root, "linked.txt"); got != "[core]\thooksPath = x\n" {
		t.Errorf("linked.txt = %q, want the edit", got)
	}
}
