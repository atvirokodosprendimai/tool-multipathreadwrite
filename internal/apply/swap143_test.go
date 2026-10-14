package apply

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-143: validation judged sub/hooks/x as an ordinary path; a directory swapped
// for a link to .git after that must not carry the write there. The swap seams
// are ADR-106's: stageFileFn swaps after the first staging, so the later hunks
// reopen sub by name.
func TestAWriteThroughAParentSwappedForDotGitStaysOut(t *testing.T) {
	for _, c := range []struct {
		name    string
		hunk    Input
		staging bool
	}{
		{"a create", Input{Path: "sub/hooks/pre-commit", Op: "create", Body: []string{"#!/bin/sh"}, Lines: -1, Index: 1}, true},
		{"an unlink", Input{Path: "sub/config", Op: "unlink", Lines: -1, Index: 1}, false},
		{"a rename's destination", Input{Path: "b.txt", Op: "rename", Body: []string{"sub/hooks/moved.txt"}, Lines: -1, Index: 1}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "x.txt", abcde)
			write(t, root, "b.txt", "bee\n")
			write(t, root, filepath.Join("sub", "config"), "ordinary\n")
			write(t, root, filepath.Join("sub", "hooks", "keep"), "k\n")
			write(t, root, filepath.Join(".git", "config"), "[core]\n")
			write(t, root, filepath.Join(".git", "hooks", "keep"), "k\n")
			before := listing(t, filepath.Join(root, ".git"))
			swapAfterFirstStage(t, root, filepath.Join(root, ".git"))
			res, err := Apply(root, []Input{
				{Path: "x.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
				c.hunk,
			}, Options{Force: true})
			if after := listing(t, filepath.Join(root, ".git")); len(after) != len(before) {
				t.Errorf("the plan changed .git: %q -> %q (err %v)", before, after, err)
			}
			if got := read(t, root, ".git/config"); got != "[core]\n" {
				t.Errorf(".git/config = %q after the plan", got)
			}
			if _, statErr := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); statErr == nil {
				t.Error("a hook was made in .git")
			}
			if res.Applied {
				t.Errorf("a write through a parent swapped for .git was reported applied: %v %+v", err, res)
			}
			// A refusal at staging is a failed hunk, exit 1; only a failure while
			// committing is an error (exit 2).
			if c.staging && (err != nil || res.Failed == 0) {
				t.Errorf("the refusal at staging came back as err %v, failed %d; want a failed hunk and no error", err, res.Failed)
			}
		})
	}
}

// ADR-143: tree.rel judges where a path lands NOW. A parent swapped for a
// relative link to .git after staging, immediately before the commit rename,
// carries sub/hooks/moved.txt into .git although its spelling holds no .git.
func TestARenameThroughAParentSwappedForDotGitAfterStagingStaysOut(t *testing.T) {
	root := t.TempDir()
	write(t, root, "b.txt", "bee\n")
	write(t, root, filepath.Join("sub", "hooks", "keep"), "k\n")
	write(t, root, filepath.Join(".git", "config"), "[core]\n")
	write(t, root, filepath.Join(".git", "hooks", "keep"), "k\n")
	before := listing(t, filepath.Join(root, ".git"))
	real := commitRenameFn
	t.Cleanup(func() { commitRenameFn = real })
	swapped := false
	commitRenameFn = func(tr *tree, from, to string) error {
		if !swapped {
			swapped = true
			if err := os.Rename(filepath.Join(root, "sub"), filepath.Join(root, "sub.validated")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(".git", filepath.Join(root, "sub")); err != nil {
				t.Skipf("symlinks cannot be made here: %v", err)
			}
		}
		return real(tr, from, to)
	}
	res, err := Apply(root, []Input{
		{Path: "b.txt", Op: "rename", Body: []string{"sub/hooks/moved.txt"}, Lines: -1, Index: 0},
	}, Options{Force: true})
	if after := listing(t, filepath.Join(root, ".git")); len(after) != len(before) {
		t.Errorf("the rename changed .git: %q -> %q (err %v)", before, after, err)
	}
	if res.Applied {
		t.Errorf("a rename through a parent swapped for .git was reported applied: %v %+v", err, res)
	}
	if got := read(t, root, "b.txt"); got != "bee\n" {
		t.Errorf("b.txt = %q, want it left where it was", got)
	}
}
