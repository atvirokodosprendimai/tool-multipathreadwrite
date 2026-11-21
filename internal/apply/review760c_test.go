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

// The last round on ADR-209 T1 (#760): the aside's name is a window of its own, and
// the cheaper name lookup, the leftover texts and the link-back failure.

// asideHolding is the aside (a .mrw-aside-* entry) that holds the bytes of name.
func asideHolding(t *testing.T, root, name string) string {
	t.Helper()
	for _, n := range mrwLeftovers(t, root) {
		if strings.HasPrefix(n, ".mrw-aside-") && read(t, root, n) == "orig "+name+"\n" {
			return n
		}
	}
	t.Fatalf("no aside holds the original of %s: %v", name, mrwLeftovers(t, root))
	return ""
}

// swapAside puts another file at the aside's name, as another process may in the shared
// directory after the commit checked it.
func swapAside(t *testing.T, root, aside string) {
	t.Helper()
	write(t, root, "stranger.tmp", "STRANGER\n")
	if err := os.Rename(filepath.Join(root, "stranger.tmp"), filepath.Join(root, aside)); err != nil {
		t.Fatal(err)
	}
}

// P2, the undo path. An aside replaced after it was verified is not linked back as the
// original: the file mrw wrote stays at the path, and the stranger is named and kept.
func TestAnAsideReplacedAfterItWasCheckedIsNotLinkedBack(t *testing.T) {
	u := newU209(t)
	real := commitRenameFn
	var aside string
	commitRenameFn = func(tr *tree, from, to string) error {
		if filepath.Base(to) == "c.txt" {
			aside = asideHolding(t, u.root, "b.txt")
			swapAside(t, u.root, aside)
			return errors.New("injected: c.txt refused")
		}
		return real(tr, from, to)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("an undo whose aside was replaced: %v", err)
	}
	if got := read(t, u.root, "b.txt"); got != "new b.txt\n" {
		t.Errorf("b.txt = %q, want the file mrw wrote, not the stranger", got)
	}
	if got := read(t, u.root, aside); got != "STRANGER\n" {
		t.Errorf("the stranger at %s = %q, want it left", aside, got)
	}
	if !slices.Contains(res.LeftBehind, aside) {
		t.Errorf("left_behind %v does not name %s", res.LeftBehind, aside)
	}
	for _, f := range res.Files {
		if f.Path == "b.txt" && f.NotHeld {
			t.Errorf("b.txt is marked not held though the file mrw wrote stands there")
		}
	}
	u.restored("a.txt")
}

// P2, the landed path. An aside replaced after it was verified is not removed: it is
// someone's file, and it is named.
func TestAnAsideReplacedBeforeTheLandedWriteRemovesItIsNotDeleted(t *testing.T) {
	u := newU209(t)
	real := commitRenameFn
	var aside string
	commitRenameFn = func(tr *tree, from, to string) error {
		err := real(tr, from, to)
		if filepath.Base(to) == "c.txt" && err == nil {
			aside = asideHolding(t, u.root, "b.txt")
			swapAside(t, u.root, aside)
		}
		return err
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err != nil && !res.Applied {
		t.Fatalf("a landed write: %v", err)
	}
	if got := read(t, u.root, aside); got != "STRANGER\n" {
		t.Errorf("the entry at %s = %q, want the stranger left", aside, got)
	}
	if !slices.Contains(res.LeftBehind, aside) {
		t.Errorf("left_behind %v does not name %s", res.LeftBehind, aside)
	}
	if left := mrwLeftovers(t, u.root); len(left) != 1 {
		t.Errorf("leftovers %v, want only the stranger: the other asides are mrw's and go", left)
	}
}

// P3-1. A file with another name in its directory (copy.txt), planned under its own
// spelling, lists that spelling exactly: it gets an aside, and a failed commit is put
// back whole. Without the exact-name answer the two names would read as ambiguous. The
// descriptor path is switched off so the directory is listed, as on a volume where
// it is.
func TestAFileWithAnotherNameIsListedUnderItsOwnSpelling(t *testing.T) {
	u := newU209(t)
	if !caseInsensitiveFS(t, u.root) {
		t.Skip("this filesystem keeps case apart: the listing is not reached")
	}
	if err := os.Link(filepath.Join(u.root, "a.txt"), filepath.Join(u.root, "copy.txt")); err != nil {
		t.Skipf("no hard links here: %v", err)
	}
	real := spelledByDescriptorFn
	t.Cleanup(func() { spelledByDescriptorFn = real })
	spelledByDescriptorFn = func(*tree, string) (string, bool, bool) { return "", false, false }
	u.refuse("b.txt")
	res, err := Apply(u.root, u.edit("a.txt", "b.txt"), Options{})
	if err == nil || strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("a failed commit of a file with another name: %v, want a complete undo", err)
	}
	if w := writtenPaths(res); len(w) != 0 {
		t.Errorf("written = %v after the undo", w)
	}
	u.restored("a.txt", "b.txt")
}

// P3-4. An aside that cannot be looked at (an I/O error) is the environment's doing,
// not another file at the path: the plan stops with an error (exit 2), nothing written,
// no aside left.
func TestAnAsideThatCannotBeLookedAtIsAnErrorNotARefusal(t *testing.T) {
	u := newU209(t)
	real := lstatFn
	t.Cleanup(func() { lstatFn = real })
	lstatFn = func(tr *tree, p string) (fs.FileInfo, error) {
		if strings.HasPrefix(filepath.Base(p), ".mrw-aside-") {
			return nil, errors.New("injected: EIO looking at the aside")
		}
		return real(tr, p)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt"), Options{})
	if err == nil || res.Applied || !strings.Contains(err.Error(), "its aside could not be looked at") {
		t.Fatalf("an aside that cannot be looked at: applied %v, err %v, want an error", res.Applied, err)
	}
	u.restored("a.txt", "b.txt")
	u.noMrwEntries()
}

// P3-5. For a created file, which has no aside, the text does not end in a previous
// version "kept in" nothing.
func TestAChangedCreatedFileIsNotSaidToHaveAPreviousVersion(t *testing.T) {
	u := newU209(t)
	u.refuse("a.txt")
	real := removeContentFn
	t.Cleanup(func() { removeContentFn = real })
	removeContentFn = func(tr *tree, p string, want os.FileInfo) (*vacated, error) {
		if filepath.Base(p) == "n.txt" {
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		return real(tr, p, want)
	}
	in := append([]Input{{Path: "n.txt", Op: "create", Body: []string{"N"}, Lines: -1, Index: 0}}, u.edit("a.txt")...)
	in[1].Index = 1
	_, err := Apply(u.root, in, Options{})
	if err == nil || !strings.Contains(err.Error(), "n.txt was created by mrw and changed since") || strings.Contains(err.Error(), "kept in ;") || strings.HasSuffix(err.Error(), "kept in ") {
		t.Fatalf("a created file replaced by a directory: %v", err)
	}
}

// P3-6. When the previous version cannot be linked back for a reason other than the
// path being taken, the staged file is moved back to the path, so a file stands there
// (the bytes mrw wrote, and the receipt says so), and nothing is deleted.
func TestAFailedLinkBackLeavesTheStagedFileAtThePath(t *testing.T) {
	u := newU209(t)
	u.refuse("c.txt")
	real := asideLinkFn
	t.Cleanup(func() { asideLinkFn = real })
	asideLinkFn = func(tr *tree, from, to string) error {
		if strings.HasPrefix(filepath.Base(from), ".mrw-aside-") && filepath.Base(to) == "b.txt" {
			return errors.New("injected: EIO linking the aside back")
		}
		return real(tr, from, to)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("an undo whose link back failed: %v", err)
	}
	if got := read(t, u.root, "b.txt"); got != "new b.txt\n" {
		t.Errorf("b.txt = %q, want the file mrw wrote moved back to the path", got)
	}
	if left := mrwLeftovers(t, u.root); len(left) != 1 || !strings.HasPrefix(left[0], ".mrw-aside-") || read(t, u.root, left[0]) != "orig b.txt\n" {
		t.Errorf("leftovers %v, want only the aside holding the original", left)
	}
	for _, f := range res.Files {
		if f.Path == "b.txt" && (!f.Written || f.NotHeld) {
			t.Errorf("b.txt's record %+v: want written, holding the bytes mrw wrote", f)
		}
	}
}
