package apply

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// #744 final pass, P1-1. A source that is gone when its identity is taken after
// its bytes were read changed: a nil snapshot is not a reason to skip both checks.
func TestASourceThatVanishesAfterItsReadIsRefused(t *testing.T) {
	for _, op := range []string{"unlink", "rename"} {
		t.Run(op, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "a.txt", "A\n")
			real := loadFn
			t.Cleanup(func() { loadFn = real })
			loadFn = func(r, p string) (text, bool, error) {
				o, e, err := real(r, p)
				if filepath.Base(p) == "a.txt" {
					if rerr := os.Remove(p); rerr != nil {
						t.Fatal(rerr)
					}
				}
				return o, e, err
			}
			in := Input{Path: "a.txt", Op: op, Lines: -1, Index: 0}
			if op == "rename" {
				in.Body = []string{"b.txt"}
			}
			res, err := Apply(root, []Input{in}, Options{})
			if err != nil || res.Applied || res.Failed != 1 {
				t.Fatalf("a source that vanished after its read: %v %+v", err, res)
			}
			if h := hunkFor(t, res, "a.txt"); !strings.Contains(h.Reason, "changed while mrw read it") {
				t.Errorf("reason %q", h.Reason)
			}
		})
	}
}

// #744 final pass, P1-2. The destination is looked at once more right before the
// source is removed: a file swapped in after the first look keeps the source.
func TestADestinationSwappedAfterItWasVerifiedKeepsTheSource(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	real := beforeRemoveSource
	t.Cleanup(func() { beforeRemoveSource = real })
	beforeRemoveSource = func(to string) {
		if filepath.Base(to) == "b.txt" {
			replaceWith(t, root, "b.txt", "FOREIGN\n")
		}
	}
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1, Index: 0}}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a rename whose destination was swapped before the source was removed reported success: %v", err)
	}
	if read(t, root, "a.txt") != "A\n" || read(t, root, "b.txt") != "FOREIGN\n" {
		t.Errorf("a.txt %q b.txt %q: the source must stay and the foreign file too", read(t, root, "a.txt"), read(t, root, "b.txt"))
	}
}

// #744 final pass, P2-3. A source replaced between its look and the link makes a
// name for the replacement: mrw made it, so it takes it back, or says it did not.
func TestASourceReplacedBeforeTheLinkLeavesNoExtraName(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	real := linkFn
	t.Cleanup(func() { linkFn = real })
	linkFn = func(tr *tree, rf, rt string) error {
		if filepath.Base(rf) == "a.txt" {
			replaceWith(t, root, "a.txt", "REPLACEMENT\n")
		}
		return real(tr, rf, rt)
	}
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1, Index: 0}}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a rename of a replaced source reported success: %v", err)
	}
	if read(t, root, "a.txt") != "REPLACEMENT\n" {
		t.Errorf("a.txt = %q, want the replacement left in place", read(t, root, "a.txt"))
	}
	if _, err := os.Lstat(filepath.Join(root, "b.txt")); err == nil {
		t.Error("b.txt, the name mrw made for the replacement, was left")
	}
	if len(res.LeftBehind) != 0 {
		t.Errorf("left_behind = %v, want none: the extra name was taken back", res.LeftBehind)
	}
}

func TestAnExtraNameForAReplacedSourceThatCannotBeTakenBackIsNamed(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A\n")
	d := filepath.Join(root, "d")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "d/a.txt", "A\n")
	real := linkFn
	realRemove := removeNameFn
	t.Cleanup(func() { linkFn, removeNameFn = real, realRemove })
	linkFn = func(tr *tree, rf, rt string) error {
		if filepath.Base(rt) == "b.txt" {
			replaceWith(t, root, "d/a.txt", "REPLACEMENT\n")
		}
		return real(tr, rf, rt)
	}
	// The removal is refused by a seam, not by a permission: Windows deletes in a
	// read-only directory, and the production code is the same on every OS.
	removeNameFn = func(tr *tree, rt string) error {
		if filepath.Base(rt) == "b.txt" {
			return errors.New("injected: the extra name cannot be removed")
		}
		return realRemove(tr, rt)
	}
	res, err := Apply(root, []Input{{Path: "d/a.txt", Op: "rename", Body: []string{"d/b.txt"}, Lines: -1, Index: 0}}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a rename of a replaced source reported success: %v", err)
	}
	if !slices.Contains(res.LeftBehind, "d/b.txt") {
		t.Errorf("left_behind = %v, want d/b.txt", res.LeftBehind)
	}
}

// A directory is never hard-linked, on any OS: the move asks and renames without
// a link attempt, whose refusal Windows words as access denied (which noHardLinks
// deliberately does not read as "no hard links").
func TestADirectoryMoveNeverAttemptsALink(t *testing.T) {
	root := t.TempDir()
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	for _, d := range []string{"d1", "d2"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	real := linkFn
	t.Cleanup(func() { linkFn = real })
	calls := 0
	linkFn = func(*tree, string, string) error {
		calls++
		return &os.LinkError{Op: "link", Err: errors.New("Access is denied.")}
	}
	p := func(n string) string { return filepath.Join(root, n) }
	if err := tr.renameNew(p("d1"), p("d2")); !errors.Is(err, fs.ErrExist) {
		t.Errorf("a directory onto a taken name: %v, want fs.ErrExist", err)
	}
	if err := tr.renameNew(p("d1"), p("d3")); err != nil {
		t.Errorf("a directory onto a free name: %v", err)
	}
	if calls != 0 {
		t.Errorf("a link was attempted %d time(s) for a directory", calls)
	}
}

// #744 final pass, P2-4. A case-only rename undone by identity, not by what the
// directory lists: another file under the original spelling is not the renamed one.
func TestACaseOnlyUndoDoesNotTrustTheSpelling(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem does not fold case")
	}
	write(t, root, "a.txt", "A\n")
	write(t, root, "x.txt", "X\n")
	real := commitRenameFn
	t.Cleanup(func() { commitRenameFn = real })
	commitRenameFn = func(tr *tree, from, to string) error {
		if filepath.Base(to) == "y.txt" {
			if err := os.Rename(filepath.Join(root, "A.txt"), filepath.Join(root, "saved.txt")); err != nil {
				t.Fatal(err)
			}
			write(t, root, "a.txt", "UNRELATED\n")
			return errors.New("injected failure")
		}
		return real(tr, from, to)
	}
	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "rename", Body: []string{"A.txt"}, Lines: -1, Index: 0},
		{Path: "x.txt", Op: "rename", Body: []string{"y.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a plan whose later rename failed: %v applied %v", err, res.Applied)
	}
	if !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Errorf("err = %v, want UNDO INCOMPLETE: A.txt is not where the undo would put it back", err)
	}
	var kept fs.FileInfo
	if kept, _ = os.Lstat(filepath.Join(root, "saved.txt")); kept == nil {
		t.Error("saved.txt is gone")
	}
}
