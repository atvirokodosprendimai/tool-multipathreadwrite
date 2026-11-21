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

// The targeted final pass on ADR-209 T1 (#760), findings 1, 2 and 5: the restore does not
// depend on deleting the moved entry, and the private entry is named whatever it holds.

// F1. The staged file is moved into its private directory, the previous version is
// linked back into the vacant path, and the private entry is deleted last: a deletion
// that fails (EIO, injected AFTER the move) is a named leftover and the undo is complete.
func TestAnUndoWhoseLastDeletionFailsIsCompleteAndNamesTheLeftover(t *testing.T) {
	u := newU209(t)
	u.refuse("c.txt")
	realDiscard := discardVacatedFn
	t.Cleanup(func() { discardVacatedFn = realDiscard })
	discardVacatedFn = func(*tree, *vacated) error {
		return errors.New("injected: EIO deleting the private entry")
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil || strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("a failed commit whose undo only left a private entry: %v, want a complete undo", err)
	}
	u.restored("a.txt", "b.txt", "c.txt")
	if w := writtenPaths(res); len(w) != 0 {
		t.Errorf("written = %v after the undo", w)
	}
	for _, f := range res.Files {
		if f.NotHeld {
			t.Errorf("%s is marked not held though its path holds the original", f.Path)
		}
	}
	left := mrwLeftovers(t, u.root)
	if len(left) != 2 {
		t.Fatalf("leftovers %v, want the two private directories (a.txt and b.txt were renamed in)", left)
	}
	for _, n := range left {
		if !strings.HasPrefix(n, ".mrw-undo-") || !slices.Contains(res.LeftBehind, n) {
			t.Errorf("%s is not named in left_behind %v", n, res.LeftBehind)
		}
	}
}

// F5. A non-empty directory swapped in for the file is moved into the private
// directory with what it holds; when it cannot be put back it stays there, the private
// directory is named in left_behind (a directory that still holds something is not
// hidden by noteIfLeft), and the path is marked as not holding mrw's bytes.
func TestANonEmptyDirectoryThatCannotBePutBackIsNamedAndNothingIsDeleted(t *testing.T) {
	u := newU209(t)
	u.refuse("c.txt")
	realV := removeContentFn
	realL := fallbackLstatFn
	t.Cleanup(func() { removeContentFn, fallbackLstatFn = realV, realL })
	removeContentFn = func(tr *tree, p string, want os.FileInfo) (*vacated, error) {
		if filepath.Base(p) == "b.txt" {
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(p, "precious.txt"), []byte("someone's only copy\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return realV(tr, p, want)
	}
	fallbackLstatFn = func(tr *tree, rt string) (fs.FileInfo, error) {
		if filepath.Base(rt) == "b.txt" {
			return nil, errors.New("injected: EIO looking at the target")
		}
		return realL(tr, rt)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("an undo that displaced a directory and could not put it back: %v", err)
	}
	var private string
	for _, n := range mrwLeftovers(t, u.root) {
		if strings.HasPrefix(n, ".mrw-undo-") {
			private = n
		}
	}
	if private == "" {
		t.Fatalf("no private directory left: the displaced directory was deleted: %v", mrwLeftovers(t, u.root))
	}
	if got, err := os.ReadFile(filepath.Join(u.root, private, "e", "precious.txt")); err != nil || string(got) != "someone's only copy\n" {
		t.Errorf("what the displaced directory held: %q %v", got, err)
	}
	if !slices.Contains(res.LeftBehind, private) {
		t.Errorf("left_behind %v does not name %s, though it holds a non-empty directory", res.LeftBehind, private)
	}
	var b FileResult
	for _, f := range res.Files {
		if f.Path == "b.txt" {
			b = f
		}
	}
	if !b.NotHeld {
		t.Errorf("b.txt's record %+v: want it marked as not holding mrw's bytes", b)
	}
}

// Finding 3. If another file stood at the path for the instant the aside was linked and
// the original came back before the commit's own check, the aside is the stranger. It is
// checked under its own name against what the plan loaded before the first rename:
// nothing is written, exit 1, and no aside is left.
func TestAnAsideLinkedToATransientFileRefusesTheWrite(t *testing.T) {
	u := newU209(t)
	real := asideLinkFn
	t.Cleanup(func() { asideLinkFn = real })
	asideLinkFn = func(tr *tree, from, to string) error {
		if filepath.Base(from) != "b.txt" {
			return real(tr, from, to)
		}
		orig := from + ".orig"
		if err := os.Rename(from, orig); err != nil {
			t.Fatal(err)
		}
		write(t, u.root, "b.txt", "orig b.txt\n") // another inode, the same bytes and size
		err := real(tr, from, to)
		if rerr := os.Remove(from); rerr != nil {
			t.Fatal(rerr)
		}
		if rerr := os.Rename(orig, from); rerr != nil {
			t.Fatal(rerr)
		}
		return err
	}
	u.refuse("c.txt")
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err != nil || res.Applied || res.Failed == 0 {
		t.Fatalf("a transient file at the path: applied %v failed %d err %v, want a refused plan", res.Applied, res.Failed, err)
	}
	if h := hunkFor(t, res, "b.txt"); !strings.Contains(h.Reason, "when its aside was made") {
		t.Errorf("reason %q", h.Reason)
	}
	u.restored("a.txt", "b.txt", "c.txt")
	u.noMrwEntries()
}
