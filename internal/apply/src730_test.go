package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// swapSourceAt replaces root/name with a different file holding body the first
// time the commit asks whether that path can be replaced: after validation read
// it, before the first rename.
func swapSourceAt(t *testing.T, root, name, body string) {
	t.Helper()
	real := replaceableFn
	t.Cleanup(func() { replaceableFn = real })
	done := false
	replaceableFn = func(tr *tree, p string) error {
		if !done && filepath.Base(p) == name {
			done = true
			replaceWith(t, root, name, body)
		}
		return real(tr, p)
	}
}

// #730. An unlink and a rename acted on whatever was at the path at commit: a
// file swapped in after validation was removed, or moved, unread. The source is
// now the file validation saw, or the step is refused before anything is written.
func TestAnUnlinkRefusesASourceThatWasReplacedAfterValidation(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	write(t, root, "b.txt", "B\n")
	swapSourceAt(t, root, "b.txt", "NEVER-READ-FOREIGN, longer\n")

	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"EDITED"}, Lines: -1, Index: 0},
		{Path: "b.txt", Op: "unlink", Lines: -1, Index: 1},
	}, Options{})
	if err != nil || res.Applied || res.Failed != 1 {
		t.Fatalf("a replaced unlink source was not refused as a hunk: %v %+v", err, res)
	}
	if got := read(t, root, "b.txt"); got != "NEVER-READ-FOREIGN, longer\n" {
		t.Errorf("b.txt = %q: the file that replaced it was removed", got)
	}
	if got := read(t, root, "a.txt"); got != "A\n" {
		t.Errorf("a.txt = %q: a plan refused before its first rename wrote", got)
	}
	if h := hunkFor(t, res, "b.txt"); h.Status != StatusFailed || !strings.Contains(h.Reason, "b.txt") {
		t.Errorf("hunk %s %q, want failed and naming b.txt", h.Status, h.Reason)
	}
}

func TestARenameRefusesASourceThatWasReplacedAfterValidation(t *testing.T) {
	root := t.TempDir()
	write(t, root, "b.txt", "B\n")
	swapSourceAt(t, root, "b.txt", "NEVER-READ-FOREIGN, longer\n")

	res, err := Apply(root, []Input{{Path: "b.txt", Op: "rename", Body: []string{"c.txt"}, Lines: -1, Index: 0}}, Options{})
	if err != nil || res.Applied || res.Failed != 1 {
		t.Fatalf("a replaced rename source was not refused as a hunk: %v %+v", err, res)
	}
	if _, err := os.Lstat(filepath.Join(root, "c.txt")); !os.IsNotExist(err) {
		t.Errorf("the file that replaced the source was moved to c.txt: %v", err)
	}
	if got := read(t, root, "b.txt"); got != "NEVER-READ-FOREIGN, longer\n" {
		t.Errorf("b.txt = %q", got)
	}
}

// Between the first check and the step's own: the commit stops at that step,
// the file that replaced the source stays, and what already landed is named.
func TestAnUnlinkStopsTheCommitWhenItsSourceIsReplacedAfterTheFirstCheck(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	write(t, root, "z.txt", "Z\n")
	real := commitRenameFn
	t.Cleanup(func() { commitRenameFn = real })
	commitRenameFn = func(tr *tree, from, to string) error {
		if filepath.Base(to) == "a.txt" { // the edit's install, before the path ops
			replaceWith(t, root, "z.txt", "NEVER-READ-FOREIGN, longer\n")
		}
		return real(tr, from, to)
	}
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"EDITED"}, Lines: -1, Index: 0},
		{Path: "z.txt", Op: "unlink", Lines: -1, Index: 1},
	}, Options{})
	if err == nil || res.Applied || !strings.Contains(err.Error(), "z.txt") || !strings.Contains(err.Error(), "nothing was written") || read(t, root, "a.txt") != "A\n" {
		t.Fatalf("the replaced source did not stop the commit by name: %v", err)
	}
	if got := read(t, root, "z.txt"); got != "NEVER-READ-FOREIGN, longer\n" {
		t.Errorf("z.txt = %q: the file that replaced it was removed", got)
	}
	if a := asideFiles(t, root); len(a) != 0 {
		t.Errorf("an aside was made for a file nobody read: %v", a)
	}
}

// The pair: an untouched source is unlinked and renamed as before, and a file the
// plan edits and renames moves the edited file (ADR-114), though the edit made it
// a different file from the one validation read.
func TestUntouchedSourcesAreStillUnlinkedAndRenamed(t *testing.T) {
	root := t.TempDir()
	write(t, root, "u.txt", "U\n")
	write(t, root, "r.txt", "R\n")
	write(t, root, "e.txt", "old\n")
	res, err := Apply(root, []Input{
		{Path: "u.txt", Op: "unlink", Lines: -1, Index: 0},
		{Path: "r.txt", Op: "rename", Body: []string{"r2.txt"}, Lines: -1, Index: 1},
		{Path: "e.txt", Start: 1, End: 1, Op: "replace", Body: []string{"new"}, Lines: -1, Index: 2},
		{Path: "e.txt", Op: "rename", Body: []string{"e2.txt"}, Lines: -1, Index: 3},
	}, Options{})
	if err != nil || !res.Applied {
		t.Fatalf("a plain plan: %v %+v", err, res)
	}
	if _, err := os.Lstat(filepath.Join(root, "u.txt")); !os.IsNotExist(err) {
		t.Errorf("u.txt is still there: %v", err)
	}
	if read(t, root, "r2.txt") != "R\n" || read(t, root, "e2.txt") != "new\n" {
		t.Error("a rename did not land")
	}
}

func TestARenameStopsTheCommitWhenItsSourceIsReplacedAfterTheFirstCheck(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	write(t, root, "z.txt", "Z\n")
	real := commitRenameFn
	t.Cleanup(func() { commitRenameFn = real })
	commitRenameFn = func(tr *tree, from, to string) error {
		if filepath.Base(to) == "a.txt" {
			replaceWith(t, root, "z.txt", "NEVER-READ-FOREIGN, longer\n")
		}
		return real(tr, from, to)
	}
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"EDITED"}, Lines: -1, Index: 0},
		{Path: "z.txt", Op: "rename", Body: []string{"y.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err == nil || res.Applied || !strings.Contains(err.Error(), "z.txt") || !strings.Contains(err.Error(), "nothing was written") || read(t, root, "a.txt") != "A\n" {
		t.Fatalf("the replaced source did not stop the commit by name: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "y.txt")); !os.IsNotExist(err) {
		t.Errorf("the file that replaced the source was moved: %v", err)
	}
}

// changeSourceAt runs change the first time the commit asks whether root/name can
// be replaced: after validation read it, before the first rename.
func changeSourceAt(t *testing.T, name string, change func()) {
	t.Helper()
	real := replaceableFn
	t.Cleanup(func() { replaceableFn = real })
	done := false
	replaceableFn = func(tr *tree, p string) error {
		if !done && filepath.Base(p) == name {
			done = true
			change()
		}
		return real(tr, p)
	}
}

// Review of #744. Each criterion alone refuses a changed source, with the others
// held equal: a different file of the same size and time (identity), the same
// file grown with its time kept (size), the same file with its time moved (time).
// Where all three vary, the test of one hides a mutant of another.
func TestEachCriterionAloneRefusesAChangedSource(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(t *testing.T, root string, was os.FileInfo)
	}{
		{"identity: another file, the same size and time", func(t *testing.T, root string, was os.FileInfo) {
			replaceWith(t, root, "b.txt", "X\n")
			if err := os.Chtimes(filepath.Join(root, "b.txt"), was.ModTime(), was.ModTime()); err != nil {
				t.Fatal(err)
			}
		}},
		{"size: the same file grown, its time kept", func(t *testing.T, root string, was os.FileInfo) {
			p := filepath.Join(root, "b.txt")
			f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString("more\n"); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(p, was.ModTime(), was.ModTime()); err != nil {
				t.Fatal(err)
			}
		}},
		{"time: the same file, its time moved", func(t *testing.T, root string, was os.FileInfo) {
			later := was.ModTime().Add(time.Hour)
			if err := os.Chtimes(filepath.Join(root, "b.txt"), later, later); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		for _, op := range []string{"unlink", "rename"} {
			t.Run(op+" "+tc.name, func(t *testing.T) {
				root := t.TempDir()
				write(t, root, "b.txt", "B\n")
				was, err := os.Lstat(filepath.Join(root, "b.txt"))
				if err != nil {
					t.Fatal(err)
				}
				changeSourceAt(t, "b.txt", func() { tc.change(t, root, was) })
				in := Input{Path: "b.txt", Op: op, Lines: -1, Index: 0}
				if op == "rename" {
					in.Body = []string{"c.txt"}
				}
				res, err := Apply(root, []Input{in}, Options{})
				if err != nil || res.Applied || res.Failed != 1 {
					t.Fatalf("a changed source was not refused as a hunk: %v %+v", err, res)
				}
				if _, err := os.Lstat(filepath.Join(root, "b.txt")); err != nil {
					t.Errorf("b.txt was removed or moved: %v", err)
				}
				if _, err := os.Lstat(filepath.Join(root, "c.txt")); !os.IsNotExist(err) {
					t.Errorf("c.txt exists: %v", err)
				}
			})
		}
	}
}

// Review of #744. On Windows os.Lstat leaves the file ID unloaded, to be read
// lazily from the path when compared, which blinds the check; the ID is loaded
// when the source is taken, and a source whose ID cannot be loaded is refused,
// as ADR-107 refuses a target.
func TestASourceWhoseIdentityCannotBeReadIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "t.txt", "T\n")
	if err := os.Symlink("t.txt", filepath.Join(root, "l.txt")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	// The link's target loads its ID at validation (os.Stat follows the link); the
	// entry the unlink acts on is the link itself, whose ID only lstatSource loads.
	real := sameFileFn
	t.Cleanup(func() { sameFileFn = real })
	sameFileFn = func(a, b os.FileInfo) bool { return a.Mode()&os.ModeSymlink == 0 && real(a, b) }
	res, err := Apply(root, []Input{{Path: "l.txt", Op: "unlink", Lines: -1, Index: 0}}, Options{})
	if err != nil || res.Applied || res.Failed != 1 {
		t.Fatalf("a source whose identity cannot be read was not refused: %v %+v", err, res)
	}
	if h := hunkFor(t, res, "l.txt"); !strings.Contains(h.Reason, "identity could not be read") {
		t.Errorf("reason %q", h.Reason)
	}
	if _, err := os.Lstat(filepath.Join(root, "l.txt")); err != nil {
		t.Errorf("l.txt: %v", err)
	}
}
