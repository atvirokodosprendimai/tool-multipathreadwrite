package apply

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// refuseRemove makes removeFn refuse every path for which refused says so,
// leaving it on disk, and remove every other path for real.
func refuseRemove(t *testing.T, refused func(p string) bool) {
	t.Helper()
	real := removeFn
	t.Cleanup(func() { removeFn = real })
	removeFn = func(p string) error {
		if refused(p) {
			return errors.New("remove refused")
		}
		return real(p)
	}
}

// failSecondStage makes the second staging call fail, after the first staged
// for real, so the abort has one temp file and its directories to take back.
func failSecondStage(t *testing.T) {
	t.Helper()
	real := stageFileFn
	t.Cleanup(func() { stageFileFn = real })
	calls := 0
	stageFileFn = func(path string, tx text) (staged, error) {
		calls++
		if calls == 2 {
			return staged{}, errors.New("staging refused")
		}
		return real(path, tx)
	}
}

// leftOnDisk reports whether every root-relative entry of left exists under root.
func leftOnDisk(t *testing.T, root string, left []string) {
	t.Helper()
	for _, p := range left {
		if filepath.IsAbs(p) {
			t.Errorf("left_behind entry %q is not root-relative", p)
			continue
		}
		if _, err := os.Lstat(filepath.Join(root, p)); err != nil {
			t.Errorf("left_behind names %q, which is not there: %v", p, err)
		}
	}
}

func isTemp(p string) bool  { return strings.HasPrefix(filepath.Base(p), ".mrw-") }
func isAside(p string) bool { return strings.HasPrefix(filepath.Base(p), ".mrw-aside-") }

// ADR-105 T1, the record's Enforced-by. Every removal apply attempts on
// something it made in the tree goes through removeFn, and a path still there
// afterwards is named in LeftBehind, root-relative. Before, each site dropped
// the error with `_ = os.Remove`, so a staging abort could leave a .mrw- temp
// in the tree beside a receipt that said nothing was written.
func TestEveryFailedCleanupIsNamedInLeftBehind(t *testing.T) {
	two := func(t *testing.T) (string, []Input) {
		root := t.TempDir()
		write(t, root, "one.txt", abcde)
		write(t, root, "two.txt", abcde)
		return root, []Input{
			{Path: "one.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
			{Path: "two.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 1},
		}
	}

	t.Run("a staging abort names the temp it could not remove", func(t *testing.T) {
		root, in := two(t)
		failSecondStage(t)
		refuseRemove(t, isTemp)
		res, err := Apply(root, in, Options{Force: true})
		if err == nil || res.Applied {
			t.Fatalf("the abort was not reported: %v %+v", err, res)
		}
		if len(res.LeftBehind) != 1 || !isTemp(res.LeftBehind[0]) {
			t.Fatalf("LeftBehind = %q, want the one staged temp", res.LeftBehind)
		}
		leftOnDisk(t, root, res.LeftBehind)
	})

	t.Run("with removal working nothing is named", func(t *testing.T) {
		root, in := two(t)
		failSecondStage(t)
		res, err := Apply(root, in, Options{Force: true})
		if err == nil || len(res.LeftBehind) != 0 {
			t.Fatalf("err %v, LeftBehind %q, want an error and nothing left", err, res.LeftBehind)
		}
	})

	t.Run("a failed stage hands its own temp to the abort", func(t *testing.T) {
		root, in := two(t)
		real := stageFileFn
		t.Cleanup(func() { stageFileFn = real })
		stageFileFn = func(path string, tx text) (staged, error) {
			sf, err := real(path, tx)
			if err != nil {
				return sf, err
			}
			return staged{tmp: sf.tmp, dirs: sf.dirs}, errors.New("close failed")
		}
		refuseRemove(t, isTemp)
		res, err := Apply(root, in, Options{Force: true})
		if err == nil || len(res.LeftBehind) != 1 || !isTemp(res.LeftBehind[0]) {
			t.Fatalf("err %v, LeftBehind %q, want the failed stage's temp named", err, res.LeftBehind)
		}
		leftOnDisk(t, root, res.LeftBehind)
	})

	t.Run("a directory it made is named only when empty", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "two.txt", abcde)
		in := []Input{
			{Path: "newdir/deep/new.txt", Op: "create", Body: []string{"n"}, Lines: -1, Index: 0},
			{Path: "two.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 1},
		}
		real := stageFileFn
		t.Cleanup(func() { stageFileFn = real })
		calls := 0
		stageFileFn = func(path string, tx text) (staged, error) {
			calls++
			if calls == 2 {
				// Something else puts a file into newdir while the plan stages.
				if err := os.WriteFile(filepath.Join(root, "newdir", "foreign.txt"), []byte("f\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return staged{}, errors.New("staging refused")
			}
			return real(path, tx)
		}
		refuseRemove(t, func(p string) bool { return filepath.Base(p) == "deep" })
		res, err := Apply(root, in, Options{Force: true})
		if err == nil {
			t.Fatalf("the abort was not reported: %+v", res)
		}
		if !slices.Equal(res.LeftBehind, []string{filepath.Join("newdir", "deep")}) {
			t.Fatalf("LeftBehind = %q, want only newdir/deep: the empty directory it could not remove, not newdir, which holds another file", res.LeftBehind)
		}
		leftOnDisk(t, root, res.LeftBehind)
		if read(t, root, filepath.Join("newdir", "foreign.txt")) != "f\n" {
			t.Error("the foreign file was disturbed")
		}
	})

	t.Run("an applied plan names the aside it could not remove", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "gone.txt", abcde)
		refuseRemove(t, isAside)
		calls := 0
		real := removeFn
		removeFn = func(p string) error {
			// The placeholder's own removal, before the commit rename, is let
			// through; the final removal of the aside is refused.
			if isAside(p) {
				calls++
				if calls == 1 {
					return os.Remove(p)
				}
			}
			return real(p)
		}
		res, err := Apply(root, []Input{{Path: "gone.txt", Op: "unlink", Lines: -1, Index: 0}}, Options{Force: true})
		if err != nil || !res.Applied {
			t.Fatalf("an unlink whose aside stayed was not applied: %v %+v", err, res)
		}
		if len(res.LeftBehind) != 1 || !isAside(res.LeftBehind[0]) {
			t.Fatalf("LeftBehind = %q, want the aside", res.LeftBehind)
		}
		leftOnDisk(t, root, res.LeftBehind)
		if got := read(t, root, res.LeftBehind[0]); got != abcde {
			t.Errorf("the aside holds %q, want the removed file's content", got)
		}
	})

	t.Run("a placeholder that could not be removed is named", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "gone.txt", abcde)
		refuseRemove(t, isAside)
		res, err := Apply(root, []Input{{Path: "gone.txt", Op: "unlink", Lines: -1, Index: 0}}, Options{Force: true})
		if err == nil || res.Applied {
			t.Fatalf("a refused placeholder removal did not fail the commit: %v %+v", err, res)
		}
		if len(res.LeftBehind) != 1 || !isAside(res.LeftBehind[0]) {
			t.Fatalf("LeftBehind = %q, want the placeholder", res.LeftBehind)
		}
		leftOnDisk(t, root, res.LeftBehind)
		if read(t, root, "gone.txt") != abcde {
			t.Error("gone.txt was disturbed")
		}
	})

	t.Run("a probe that could not be removed is named", func(t *testing.T) {
		root := t.TempDir()
		refuseRemove(t, func(p string) bool { return filepath.Base(p) == "new.txt" })
		res, err := Apply(root, []Input{{Path: "new.txt", Op: "create", Body: []string{"n"}, Lines: -1, Index: 0}}, Options{Force: true})
		if err == nil || res.Applied {
			t.Fatalf("a probe that stayed did not abort: %v %+v", err, res)
		}
		if !slices.Equal(res.LeftBehind, []string{"new.txt"}) {
			t.Fatalf("LeftBehind = %q, want the probe new.txt", res.LeftBehind)
		}
		leftOnDisk(t, root, res.LeftBehind)
	})

	t.Run("a path gone despite the error is not named", func(t *testing.T) {
		root, in := two(t)
		failSecondStage(t)
		real := removeFn
		t.Cleanup(func() { removeFn = real })
		removeFn = func(p string) error {
			_ = os.Remove(p)
			return errors.New("reported failure, but the path is gone")
		}
		res, err := Apply(root, in, Options{Force: true})
		if err == nil || len(res.LeftBehind) != 0 {
			t.Fatalf("err %v, LeftBehind %q, want nothing named", err, res.LeftBehind)
		}
	})

	t.Run("a rename destination's probe that could not be removed is named", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "b.txt", "bee\n")
		refuseRemove(t, func(p string) bool { return filepath.Base(p) == "x.txt" })
		res, err := Apply(root, []Input{{Path: "b.txt", Op: "rename", Body: []string{"x.txt"}, Lines: -1, Index: 0}}, Options{Force: true})
		if err == nil || res.Applied {
			t.Fatalf("a probe that stayed did not abort: %v %+v", err, res)
		}
		if !slices.Equal(res.LeftBehind, []string{"x.txt"}) {
			t.Fatalf("LeftBehind = %q, want the probe x.txt", res.LeftBehind)
		}
		leftOnDisk(t, root, res.LeftBehind)
		if read(t, root, "b.txt") != "bee\n" {
			t.Error("b.txt was disturbed")
		}
	})

	t.Run("an aside the undo keeps is named and holds the file", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "b.txt", "bee\n")
		write(t, root, "c.txt", "sea\n")
		write(t, root, "d.txt", "dee\n")
		real := commitRenameFn
		t.Cleanup(func() { commitRenameFn = real })
		// The last unlink fails, and the undo cannot move c.txt back to b.txt,
		// so the aside holding the unlinked c.txt stays as a recovery file.
		commitRenameFn = func(from, to string) error {
			if filepath.Base(from) == "d.txt" || (filepath.Base(from) == "c.txt" && filepath.Base(to) == "b.txt") {
				return errors.New("rename refused")
			}
			return real(from, to)
		}
		res, err := Apply(root, []Input{
			{Path: "c.txt", Op: "unlink", Lines: -1, Index: 0},
			{Path: "b.txt", Op: "rename", Body: []string{"c.txt"}, Lines: -1, Index: 1},
			{Path: "d.txt", Op: "unlink", Lines: -1, Index: 2},
		}, Options{Force: true})
		if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
			t.Fatalf("the undo was not reported incomplete: %v", err)
		}
		if len(res.LeftBehind) != 1 || !isAside(res.LeftBehind[0]) {
			t.Fatalf("LeftBehind = %q, want the kept aside", res.LeftBehind)
		}
		leftOnDisk(t, root, res.LeftBehind)
		if got := read(t, root, res.LeftBehind[0]); got != "sea\n" {
			t.Errorf("the kept aside holds %q, want the unlinked c.txt", got)
		}
	})
}
