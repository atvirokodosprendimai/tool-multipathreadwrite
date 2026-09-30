package apply

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// swapAfterFirstStage makes the first staging call stage for real and then
// replace root/sub with a link to outside: the swap lands after validation
// judged every path, and before the plan's later steps reopen sub by name.
func swapAfterFirstStage(t *testing.T, root, outside string) {
	t.Helper()
	real := stageFileFn
	t.Cleanup(func() { stageFileFn = real })
	swapped := false
	stageFileFn = func(tr *tree, path string, tx text) (staged, error) {
		sf, err := real(tr, path, tx)
		if !swapped {
			swapped = true
			swapSub(t, root, outside)
		}
		return sf, err
	}
}

// swapSub replaces root/sub with a link to outside.
func swapSub(t *testing.T, root, outside string) {
	t.Helper()
	if err := os.Rename(filepath.Join(root, "sub"), filepath.Join(root, "sub.validated")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "sub")); err != nil {
		t.Skipf("symlinks cannot be made here: %v", err)
	}
}

// swapBeforeRename swaps root/sub for a link to outside the first time a
// commit rename matching at is about to run: after staging, resolution and
// the unlink's placeholder, immediately before the operation the swap is
// meant to redirect (the review of the record).
func swapBeforeRename(t *testing.T, root, outside string, at func(from, to string) bool) {
	t.Helper()
	real := commitRenameFn
	t.Cleanup(func() { commitRenameFn = real })
	swapped := false
	commitRenameFn = func(tr *tree, from, to string) error {
		if !swapped && at(from, to) {
			swapped = true
			swapSub(t, root, outside)
		}
		return real(tr, from, to)
	}
}

// listing names every path under dir, for comparing an outside directory
// before and after a plan.
func listing(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(dir, func(p string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// ADR-106 T1, the record's Enforced-by. rooted.Resolve judged each path, and
// staging, commit and the path operations then reopened it by name, so a parent
// directory swapped for a link out of the root after validation redirected
// them: an unlink moved the outside file into an aside beside it and removed
// it, and a rename built its destination directory outside and landed there.
// Whatever the plan's outcome, nothing outside the root may change.
func TestAWriteThroughASwappedParentStaysInTheRoot(t *testing.T) {
	t.Run("an unlink", func(t *testing.T) {
		root, outside := t.TempDir(), t.TempDir()
		write(t, root, "x.txt", abcde)
		write(t, root, filepath.Join("sub", "a.txt"), "inside\n")
		write(t, outside, "a.txt", "outside\n")
		before := listing(t, outside)
		swapAfterFirstStage(t, root, outside)
		res, err := Apply(root, []Input{
			{Path: "x.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
			{Path: "sub/a.txt", Op: "unlink", Lines: -1, Index: 1},
		}, Options{Force: true})
		if got := read(t, outside, "a.txt"); got != "outside\n" {
			t.Errorf("the outside a.txt holds %q after the plan (err %v, applied %v)", got, err, res.Applied)
		}
		if after := listing(t, outside); len(after) != len(before) {
			t.Errorf("the outside directory changed: %q -> %q", before, after)
		}
		if err == nil || res.Applied {
			t.Errorf("an unlink through a swapped parent was reported applied: %v %+v", err, res)
		}
	})

	t.Run("a rename's destination", func(t *testing.T) {
		root, outside := t.TempDir(), t.TempDir()
		write(t, root, "x.txt", abcde)
		write(t, root, "b.txt", "bee\n")
		write(t, root, filepath.Join("sub", "keep.txt"), "k\n")
		before := listing(t, outside)
		swapAfterFirstStage(t, root, outside)
		res, err := Apply(root, []Input{
			{Path: "x.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
			{Path: "b.txt", Op: "rename", Body: []string{"sub/deep/b.txt"}, Lines: -1, Index: 1},
		}, Options{Force: true})
		if after := listing(t, outside); len(after) != len(before) {
			t.Errorf("the rename reached outside the root: %q -> %q (err %v)", before, after, err)
		}
		if err == nil || res.Applied {
			t.Errorf("a rename through a swapped parent was reported applied: %v %+v", err, res)
		}
	})

	t.Run("an unlink swapped just before its rename", func(t *testing.T) {
		root, outside := t.TempDir(), t.TempDir()
		write(t, root, "x.txt", abcde)
		write(t, root, filepath.Join("sub", "a.txt"), "inside\n")
		write(t, outside, "a.txt", "outside\n")
		before := listing(t, outside)
		swapBeforeRename(t, root, outside, func(from, _ string) bool { return filepath.Base(from) == "a.txt" })
		res, err := Apply(root, []Input{
			{Path: "x.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
			{Path: "sub/a.txt", Op: "unlink", Lines: -1, Index: 1},
		}, Options{Force: true})
		if after := listing(t, outside); len(after) != len(before) || read(t, outside, "a.txt") != "outside\n" {
			t.Errorf("the unlink reached outside the root: %q -> %q (err %v)", before, after, err)
		}
		if err == nil || res.Applied {
			t.Errorf("an unlink through a parent swapped before its rename was reported applied: %v %+v", err, res)
		}
	})

	t.Run("a rename swapped just before its commit", func(t *testing.T) {
		root, outside := t.TempDir(), t.TempDir()
		write(t, root, "x.txt", abcde)
		write(t, root, "b.txt", "bee\n")
		write(t, root, filepath.Join("sub", "keep.txt"), "k\n")
		before := listing(t, outside)
		swapBeforeRename(t, root, outside, func(_, to string) bool { return filepath.Base(to) == "x.txt" })
		res, err := Apply(root, []Input{
			{Path: "x.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
			{Path: "b.txt", Op: "rename", Body: []string{"sub/deep/b.txt"}, Lines: -1, Index: 1},
		}, Options{Force: true})
		if after := listing(t, outside); len(after) != len(before) {
			t.Errorf("the rename reached outside the root: %q -> %q (err %v)", before, after, err)
		}
		if err == nil || res.Applied {
			t.Errorf("a rename through a parent swapped before its commit was reported applied: %v %+v", err, res)
		}
	})
}
