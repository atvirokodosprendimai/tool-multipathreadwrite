package apply

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// before runs inject the first time commitRenameFn is asked for a rename that
// match accepts, right before the real step (or, with fail, instead of it): the
// instant another process can act between mrw's last look and its rename.
func before(t *testing.T, match func(oldpath, newpath string) bool, fail bool, inject func()) {
	t.Helper()
	real := commitRenameFn
	t.Cleanup(func() { commitRenameFn = real })
	done := false
	commitRenameFn = func(tr *tree, oldpath, newpath string) error {
		if !done && match(oldpath, newpath) {
			done = true
			inject()
			if fail {
				return errors.New("injected rename failure")
			}
		}
		return real(tr, oldpath, newpath)
	}
}

func base(p string) string { return filepath.Base(p) }

// #726. The undo of a rename put the file back with a replacing rename, so a
// file another process made at the source's path while the plan was being undone
// was overwritten, at an error that said nothing was written (ADR-066: no undo
// step overwrites a file). Now the foreign file stays, the renamed file stays
// where the plan put it, and the error says the undo is incomplete.
func TestAnUndoOfARenameDoesNotOverwriteAFileThatAppearedAtTheSource(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	write(t, root, "c.txt", "C\n")
	before(t, func(_, n string) bool { return base(n) == "d.txt" }, true, func() { write(t, root, "a.txt", "FOREIGN\n") })

	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1, Index: 0},
		{Path: "c.txt", Op: "rename", Body: []string{"d.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("an undo that could not put a file back did not say so: %v", err)
	}
	if got := read(t, root, "a.txt"); got != "FOREIGN\n" {
		t.Errorf("a.txt = %q: the undo overwrote the file that appeared there", got)
	}
	if got := read(t, root, "b.txt"); got != "A\n" {
		t.Errorf("b.txt = %q, want the renamed file kept", got)
	}
	var written bool
	for _, f := range res.Files {
		written = written || (f.Path == "a.txt" && f.Written)
	}
	if !written {
		t.Errorf("the rename that could not be undone is not reported written: %+v", res.Files)
	}
}

// The pair: with nothing in the way the same plan is undone whole.
func TestAnUndoOfARenameWithNothingInTheWayPutsTheFileBack(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	write(t, root, "c.txt", "C\n")
	before(t, func(_, n string) bool { return base(n) == "d.txt" }, true, func() {})

	_, err := Apply(root, []Input{
		{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1, Index: 0},
		{Path: "c.txt", Op: "rename", Body: []string{"d.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err == nil || strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("a clean undo: %v", err)
	}
	if got := read(t, root, "a.txt"); got != "A\n" {
		t.Errorf("a.txt = %q, want A", got)
	}
	if _, err := os.Lstat(filepath.Join(root, "b.txt")); !os.IsNotExist(err) {
		t.Errorf("b.txt is still there: %v", err)
	}
}

// #726. The restore of an unlinked file's aside had the same flaw.
func TestAnUndoOfAnUnlinkDoesNotOverwriteAFileThatAppearedAtThePath(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	write(t, root, "z.txt", "Z\n")
	// The second unlink fails; by then a.txt is in an aside.
	before(t, func(o, _ string) bool { return base(o) == "z.txt" }, true, func() { write(t, root, "a.txt", "FOREIGN\n") })

	_, err := Apply(root, []Input{
		{Path: "a.txt", Op: "unlink", Lines: -1, Index: 0},
		{Path: "z.txt", Op: "unlink", Lines: -1, Index: 1},
	}, Options{})
	if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("an aside that could not be restored did not say so: %v", err)
	}
	if got := read(t, root, "a.txt"); got != "FOREIGN\n" {
		t.Errorf("a.txt = %q: the restore overwrote the file that appeared there", got)
	}
	a := asideFiles(t, root)
	if len(a) != 1 || read(t, root, a[0]) != "A\n" {
		t.Errorf("the unlinked file is not kept in its aside: %v", a)
	}
}

// #731. A rename destination that appears between mrw's look and the rename was
// replaced; the rename now fails and the plan is undone.
func TestARenameDestinationThatAppearsBeforeTheRenameIsNotReplaced(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	before(t, func(_, n string) bool { return base(n) == "b.txt" }, false, func() { write(t, root, "b.txt", "MINE\n") })

	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1, Index: 0}}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a rename onto a file that appeared was applied: %v %+v", err, res)
	}
	if got := read(t, root, "b.txt"); got != "MINE\n" {
		t.Errorf("b.txt = %q: the file that appeared was replaced", got)
	}
	if got := read(t, root, "a.txt"); got != "A\n" {
		t.Errorf("a.txt = %q, want the source kept", got)
	}
}

// #731. The same for a create's target.
func TestACreateTargetThatAppearsBeforeTheRenameIsNotReplaced(t *testing.T) {
	root := t.TempDir()
	before(t, func(_, n string) bool { return base(n) == "n.txt" }, false, func() { write(t, root, "n.txt", "MINE\n") })

	res, err := Apply(root, []Input{{Path: "n.txt", Op: "create", Body: []string{"PLAN"}, Lines: -1, Index: 0}}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a create over a file that appeared was applied: %v %+v", err, res)
	}
	if got := read(t, root, "n.txt"); got != "MINE\n" {
		t.Errorf("n.txt = %q: the file that appeared was replaced", got)
	}
}

// The pair for both: an edit of a file that exists is still a replacing rename.
func TestAnEditOfAnExistingFileStillReplacesIt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "e.txt", "old\n")
	if _, err := Apply(root, []Input{{Path: "e.txt", Start: 1, End: 1, Op: "replace", Body: []string{"new"}, Lines: -1, Index: 0}}, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, root, "e.txt"); got != "new\n" {
		t.Errorf("e.txt = %q", got)
	}
}

// renameNew itself. A directory cannot be hard-linked, so onto a free name it takes
// the fallback (ask, then rename); onto a taken name the OS already answers exists,
// so the fallback's own ask is not reached by a real filesystem here.
func TestTheNoReplaceMoveRefusesATakenNameForAFileAndForADirectory(t *testing.T) {
	root := t.TempDir()
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	write(t, root, "f.txt", "F\n")
	write(t, root, "g.txt", "G\n")
	for _, d := range []string{"d1", "d2"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p := func(n string) string { return filepath.Join(root, n) }
	if err := tr.renameNew(p("f.txt"), p("g.txt")); !errors.Is(err, fs.ErrExist) {
		t.Errorf("a file onto a taken name: %v, want fs.ErrExist", err)
	}
	if err := tr.renameNew(p("d1"), p("d2")); !errors.Is(err, fs.ErrExist) {
		t.Errorf("a directory onto a taken name: %v, want fs.ErrExist", err)
	}
	if read(t, root, "f.txt") != "F\n" || read(t, root, "g.txt") != "G\n" {
		t.Error("a refused move changed a file")
	}
	// The pair: a free name is taken, by a file and by a directory.
	if err := tr.renameNew(p("f.txt"), p("h.txt")); err != nil || read(t, root, "h.txt") != "F\n" {
		t.Errorf("a file onto a free name: %v", err)
	}
	if _, err := os.Lstat(p("f.txt")); !os.IsNotExist(err) {
		t.Errorf("the old name is still there: %v", err)
	}
	if err := tr.renameNew(p("d1"), p("d3")); err != nil {
		t.Errorf("a directory onto a free name: %v", err)
	}
}
