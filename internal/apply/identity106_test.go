package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// replaceWith puts a new file holding body at root/name, as an editor's
// save-by-rename does: a different file under the same name.
func replaceWith(t *testing.T, root, name, body string) {
	t.Helper()
	next := filepath.Join(root, name+".next")
	if err := os.WriteFile(next, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(next, filepath.Join(root, name)); err != nil {
		t.Fatal(err)
	}
}

// duringStage runs change right after name stages: after validation read
// every file, before the first commit rename.
func duringStage(t *testing.T, name string, change func()) {
	t.Helper()
	real := stageFileFn
	t.Cleanup(func() { stageFileFn = real })
	stageFileFn = func(tr *tree, path string, tx text) (staged, error) {
		sf, err := real(tr, path, tx)
		if filepath.Base(path) == name {
			change()
		}
		return sf, err
	}
}

// ADR-106 T3. Validation read a file; another process then replaced or
// rewrote it before the commit, and the rename replaced it in turn — a lost
// update os.Root cannot see, because nothing leaves the root. Every existing
// target is checked against what validation stat'ed before the first rename,
// where a change writes nothing, and again just before its own rename, where
// a change stops the commit at that file (ADR-066). A different file, size or
// modification time each count, and the refusal names the file.
func TestACommitRefusesATargetThatChangedAfterValidation(t *testing.T) {
	plan := []Input{
		{Path: "y.txt", Start: 1, End: 1, Op: "replace", Body: []string{"Y"}, Lines: -1, Index: 0},
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 1},
	}
	refused := func(t *testing.T, res Result, err error) {
		t.Helper()
		if err == nil || res.Applied || !strings.Contains(err.Error(), "a.txt") {
			t.Errorf("a target changed after validation was not refused by name: %v %+v", err, res)
		}
	}

	t.Run("replaced while staging: nothing is written", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "y.txt", abcde)
		write(t, root, "a.txt", abcde)
		duringStage(t, "a.txt", func() { replaceWith(t, root, "a.txt", "newcomer, a longer body\n") })
		res, err := Apply(root, plan, Options{Force: true})
		refused(t, res, err)
		if got := read(t, root, "a.txt"); got != "newcomer, a longer body\n" {
			t.Errorf("the newcomer was overwritten: a.txt holds %q", got)
		}
		if got := read(t, root, "y.txt"); got != abcde {
			t.Errorf("y.txt was written by a plan that stopped before its first rename: %q", got)
		}
	})

	t.Run("replaced just before its own rename: the commit stops there", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "y.txt", abcde)
		write(t, root, "a.txt", abcde)
		real := commitRenameFn
		t.Cleanup(func() { commitRenameFn = real })
		commitRenameFn = func(tr *tree, from, to string) error {
			err := real(tr, from, to)
			if filepath.Base(to) == "y.txt" {
				replaceWith(t, root, "a.txt", "newcomer, a longer body\n")
			}
			return err
		}
		res, err := Apply(root, plan, Options{Force: true})
		refused(t, res, err)
		if got := read(t, root, "a.txt"); got != "newcomer, a longer body\n" {
			t.Errorf("the newcomer was overwritten: a.txt holds %q", got)
		}
		if got := read(t, root, "y.txt"); got != "Y\nb\nc\nd\ne\n" {
			t.Errorf("y.txt, renamed before a.txt changed, should stay written (ADR-066): %q", got)
		}
	})

	one := []Input{{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0}}
	for _, c := range []struct {
		name   string
		change func(t *testing.T, root string, mtime time.Time)
	}{
		{"identity only: another file of equal size and the same time", func(t *testing.T, root string, mtime time.Time) {
			replaceWith(t, root, "a.txt", "v\nw\nx\ny\nz\n")
			if err := os.Chtimes(filepath.Join(root, "a.txt"), mtime, mtime); err != nil {
				t.Fatal(err)
			}
		}},
		{"size only: the same file rewritten longer", func(t *testing.T, root string, mtime time.Time) {
			f, err := os.OpenFile(filepath.Join(root, "a.txt"), os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString("more\n"); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(filepath.Join(root, "a.txt"), mtime, mtime); err != nil {
				t.Fatal(err)
			}
		}},
		{"time only: the same file, same size, a new time", func(t *testing.T, root string, mtime time.Time) {
			later := mtime.Add(time.Hour)
			if err := os.Chtimes(filepath.Join(root, "a.txt"), later, later); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "a.txt", abcde)
			fi, err := os.Stat(filepath.Join(root, "a.txt"))
			if err != nil {
				t.Fatal(err)
			}
			duringStage(t, "a.txt", func() { c.change(t, root, fi.ModTime()) })
			res, err := Apply(root, one, Options{Force: true})
			refused(t, res, err)
			if got := read(t, root, "a.txt"); strings.HasPrefix(got, "A\n") {
				t.Errorf("the changed a.txt was overwritten: %q", got)
			}
		})
	}
}
