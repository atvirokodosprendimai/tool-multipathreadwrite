package apply

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// The go-reviewer's findings on ADR-209 T1 (#760).

// P2-1. On a volume that folds case a plan may name a.txt as A.TXT; the failed
// commit puts the file back under the spelling its directory listed, not the plan's.
func TestAFailedCommitKeepsTheSpellingTheDirectoryListed(t *testing.T) {
	u := newU209(t)
	if !caseInsensitiveFS(t, u.root) {
		t.Skip("this filesystem does not fold case")
	}
	u.refuse("b.txt")
	res, err := Apply(u.root, u.edit("A.TXT", "b.txt"), Options{})
	if err == nil || res.Applied {
		t.Fatalf("a refused rename was not reported: %v", err)
	}
	if got := names(t, u.root); !slices.Equal(got, []string{"a.txt", "b.txt", "c.txt"}) {
		t.Errorf("the directory lists %v, want a.txt as it was and not the plan's A.TXT", got)
	}
	u.restored("a.txt", "b.txt")
}

// P3-1. A created file the undo could not remove has no aside: left_behind must not
// name the empty string for it.
func TestAnUndoThatCannotRemoveACreatedFileNamesNoEmptyPath(t *testing.T) {
	u := newU209(t)
	u.refuse("a.txt")
	real := removeContentFn
	removeContentFn = func(tr *tree, p string, want os.FileInfo) (*vacated, error) {
		if filepath.Base(p) == "n.txt" {
			return nil, errors.New("injected: n.txt cannot be removed")
		}
		return real(tr, p, want)
	}
	in := append([]Input{{Path: "n.txt", Op: "create", Body: []string{"N"}, Lines: -1, Index: 0}}, u.edit("a.txt")...)
	in[1].Index = 1
	res, err := Apply(u.root, in, Options{})
	if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") || !strings.Contains(err.Error(), "n.txt") {
		t.Fatalf("an undo that could not remove n.txt: %v", err)
	}
	if slices.Contains(res.LeftBehind, "") {
		t.Errorf("left_behind = %q names the empty path", res.LeftBehind)
	}
}

// P3-2. A file with no aside that was changed after the rename is marked: its path
// does not hold mrw's bytes, so its hunk is failed and the ledger skips it.
func TestAFileWithNoAsideThatChangedAfterTheRenameIsMarked(t *testing.T) {
	u := newU209(t)
	realLink, realRename := asideLinkFn, commitRenameFn
	asideLinkFn = func(tr *tree, from, to string) error {
		if filepath.Base(from) == "b.txt" {
			return errors.New("injected: no hard links")
		}
		return realLink(tr, from, to)
	}
	commitRenameFn = func(tr *tree, from, to string) error {
		if filepath.Base(to) == "c.txt" {
			f, err := os.OpenFile(filepath.Join(u.root, "b.txt"), os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = f.WriteString("appended\n")
			_ = f.Close()
			return errors.New("injected: c.txt refused")
		}
		return realRename(tr, from, to)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil {
		t.Fatal("a refused rename was not reported")
	}
	var b FileResult
	for _, f := range res.Files {
		if f.Path == "b.txt" {
			b = f
		}
	}
	if !b.Written || !b.NotHeld {
		t.Errorf("b.txt's record %+v: want written and marked as not holding mrw's bytes", b)
	}
	if h := hunkFor(t, res, "b.txt"); h.Status != StatusFailed {
		t.Errorf("b.txt hunk %s, want failed: its path holds another process's change", h.Status)
	}
}

// P3-2. An undo that cannot remove a file it renamed in keeps its aside and names it.
func TestAnUndoThatCannotRemoveAFileNamesItsAside(t *testing.T) {
	u := newU209(t)
	u.refuse("c.txt")
	real := removeContentFn
	removeContentFn = func(tr *tree, p string, want os.FileInfo) (*vacated, error) {
		if filepath.Base(p) == "b.txt" {
			return nil, errors.New("injected: b.txt cannot be removed")
		}
		return real(tr, p, want)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("an undo that could not remove b.txt: %v", err)
	}
	asides := mrwLeftovers(t, u.root)
	if len(asides) != 1 || !slices.Contains(res.LeftBehind, asides[0]) {
		t.Errorf("aside %v, left_behind %v: want the aside of b.txt named", asides, res.LeftBehind)
	}
}

// P3-2, Codex #1 and the final pass. vacate moves the regular file it was told to
// expect into a private directory made for it and nothing else: a directory, a link
// or another file at the path is put back where it was and answered errNotTheFile,
// leaving no private directory, and a missing path is ErrNotExist.
func TestVacateTakesTheFileItWasToldToExpectAndNothingElse(t *testing.T) {
	root := t.TempDir()
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	write(t, root, "f.txt", "F\n")
	write(t, root, "g.txt", "G, longer\n")
	want, err := os.Lstat(filepath.Join(root, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	_ = sameFileFn(want, want)
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("f.txt", filepath.Join(root, "l.txt")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	for _, n := range []string{"empty", "l.txt", "g.txt"} {
		if v, err := tr.vacate(filepath.Join(root, n), want); !errors.Is(err, errNotTheFile) || v != nil {
			t.Errorf("vacate of %s, which is not the file expected: %v %v, want errNotTheFile", n, v, err)
		}
		if _, err := os.Lstat(filepath.Join(root, n)); err != nil {
			t.Errorf("%s is gone: %v", n, err)
		}
	}
	if left := mrwLeftovers(t, root); len(left) != 0 {
		t.Errorf("a refused vacate left %v", left)
	}
	v, err := tr.vacate(filepath.Join(root, "f.txt"), want)
	if err != nil || v == nil {
		t.Fatalf("the file expected: %v %v", v, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "f.txt")); !os.IsNotExist(err) {
		t.Errorf("the path is not vacant: %v", err)
	}
	if got, err := os.ReadFile(v.entry); err != nil || string(got) != "F\n" {
		t.Errorf("the vacated entry: %q %v", got, err)
	}
	if err := tr.discardVacated(v); err != nil {
		t.Errorf("discard: %v", err)
	}
	if left := mrwLeftovers(t, root); len(left) != 0 {
		t.Errorf("a discarded entry left %v", left)
	}
	if v, err := tr.vacate(filepath.Join(root, "f.txt"), want); !errors.Is(err, fs.ErrNotExist) || v != nil {
		t.Errorf("a missing file: %v %v", v, err)
	}
	if left := mrwLeftovers(t, root); len(left) != 0 {
		t.Errorf("a vacate of a missing path left %v", left)
	}
}

// Codex #1. An empty directory swapped in for the file after the undo looked at it is
// not removed: os.Root's Remove would take it, so the entry is moved aside and looked
// at there.
func TestAnEmptyDirectorySwappedInAfterTheLookIsNotRemoved(t *testing.T) {
	u := newU209(t)
	u.refuse("c.txt")
	real := removeContentFn
	removeContentFn = func(tr *tree, p string, want os.FileInfo) (*vacated, error) {
		if filepath.Base(p) == "b.txt" {
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		return real(tr, p, want)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("an undo that found a directory at b.txt: %v", err)
	}
	if fi, err := os.Lstat(filepath.Join(u.root, "b.txt")); err != nil || !fi.IsDir() {
		t.Errorf("the directory swapped in for b.txt was removed or replaced: %v", err)
	}
	asides := mrwLeftovers(t, u.root)
	if len(asides) != 1 || !slices.Contains(res.LeftBehind, asides[0]) {
		t.Errorf("aside %v, left_behind %v: want the aside of b.txt kept and named", asides, res.LeftBehind)
	}
	u.restored("a.txt")
}

// Codex #3. A file that vanished between the undo's look and its removal is not
// there to remove: the aside is linked back if the path is free, and the receipt
// says nothing is written.
func TestAFileThatVanishedBeforeTheUndoRemovedItIsPutBack(t *testing.T) {
	u := newU209(t)
	u.refuse("c.txt")
	real := removeContentFn
	removeContentFn = func(tr *tree, p string, want os.FileInfo) (*vacated, error) {
		if filepath.Base(p) == "b.txt" {
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
		}
		return real(tr, p, want)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil || strings.Contains(err.Error(), "UNDO INCOMPLETE") {
		t.Fatalf("a vanished file: %v, want a complete undo", err)
	}
	u.restored("a.txt", "b.txt", "c.txt")
	u.noMrwEntries()
	if w := writtenPaths(res); len(w) != 0 {
		t.Errorf("written = %v after the undo", w)
	}
	if h := hunkFor(t, res, "b.txt"); h.Status != StatusSkipped {
		t.Errorf("b.txt hunk %s, want skipped", h.Status)
	}
}

// Codex #2 and the final pass. Two names of one file, neither the plan's spelling: the
// undo does not guess which one the path meant. No aside is made for it, so a failed
// commit leaves it written and names it, rather than linking the old file back under a
// name chosen by comparing spellings. The other name is compared by identity whatever
// it is: a non-ASCII hard link, and a plain one (copy.txt) that a spelling filter would
// skip. On Windows this branch is not reachable: staging resolves A.TXT to the name
// the directory lists (a.txt), which is the plan's spelling by then, so the exact-name
// answer is given and the undo completes; the test says that and asserts it.
func TestAFileListedUnderTwoOtherNamesIsNotRestoredByGuessing(t *testing.T) {
	for _, other := range []string{"é.txt", "copy.txt"} {
		t.Run(other, func(t *testing.T) {
			u := newU209(t)
			if !caseInsensitiveFS(t, u.root) {
				t.Skip("this filesystem does not fold case, so the plan's spelling is always listed")
			}
			if err := os.Link(filepath.Join(u.root, "a.txt"), filepath.Join(u.root, other)); err != nil {
				t.Skipf("no hard links here: %v", err)
			}
			u.refuse("b.txt")
			res, err := Apply(u.root, u.edit("A.TXT", "b.txt"), Options{})
			if err == nil {
				t.Fatal("a refused rename was not reported")
			}
			if runtime.GOOS == "windows" {
				// Staging resolves A.TXT to the listed a.txt, so the plan's spelling is the
				// listed one and the ambiguity branch is not reached: the aside is made and
				// the undo completes.
				if strings.Contains(err.Error(), "UNDO INCOMPLETE") || strings.Contains(err.Error(), "more than one name") {
					t.Errorf("the undo was incomplete on Windows, where the staged spelling is the listed one: %v", err)
				}
				if w := writtenPaths(res); len(w) != 0 {
					t.Errorf("written = %v after the undo", w)
				}
				u.restored("a.txt", "b.txt")
				u.noMrwEntries()
				return
			}
			if !strings.Contains(err.Error(), "UNDO INCOMPLETE") || !strings.Contains(err.Error(), "more than one name") {
				t.Fatalf("a file listed under two names: %v", err)
			}
			if w := writtenPaths(res); !slices.Equal(w, []string{"A.TXT"}) && !slices.Equal(w, []string{"a.txt"}) {
				t.Errorf("written = %v, want the file left written", w)
			}
			for _, n := range mrwLeftovers(t, u.root) {
				t.Errorf("an aside or leftover was made for a file whose name is not known: %v", n)
			}
		})
	}
}

// P3-3. Both undos incomplete: one UNDO INCOMPLETE list, and a file the plan's own
// rename moved is not said to have "changed since mrw wrote it".
func TestBothUndosIncompleteGiveOneListAndTheRightReason(t *testing.T) {
	u := newU209(t)
	if err := os.Mkdir(filepath.Join(u.root, "e"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, u.root, "x.txt", "X\n")
	real := commitRenameFn
	commitRenameFn = func(tr *tree, from, to string) error {
		switch {
		case filepath.Base(from) == "x.txt" && strings.HasPrefix(filepath.Base(to), ".mrw-aside-"):
			return errors.New("injected: the unlink is refused")
		case filepath.Base(to) == "a.txt" && strings.HasSuffix(filepath.Dir(from), "e"):
			return errors.New("injected: the rename back is refused")
		}
		return real(tr, from, to)
	}
	in := append(u.edit("a.txt"), Input{Path: "a.txt", Op: "rename", Body: []string{"e/a.txt"}, Lines: -1, Index: 1}, Input{Path: "x.txt", Op: "unlink", Lines: -1, Index: 2})
	_, err := Apply(u.root, in, Options{})
	if err == nil {
		t.Fatal("a refused unlink was not reported")
	}
	if n := strings.Count(err.Error(), "UNDO INCOMPLETE"); n != 1 {
		t.Errorf("UNDO INCOMPLETE appears %d times: %v", n, err)
	}
	if !strings.Contains(err.Error(), "no longer at its path") || strings.Contains(err.Error(), "a.txt changed since mrw wrote it") {
		t.Errorf("the reason for a.txt: %v", err)
	}
}

// P3-5. A temp whose identity could not be taken before its rename is left alone by
// the undo and named, and its path is not marked as holding something else.
func TestATempWhoseIdentityCouldNotBeTakenIsNotMarkedAsChanged(t *testing.T) {
	u := newU209(t)
	u.refuse("c.txt")
	real := lstatFn
	t.Cleanup(func() { lstatFn = real })
	failed := false
	lstatFn = func(tr *tree, p string) (fs.FileInfo, error) {
		// the temps are the only .mrw- names looked at before a rename; fail the second
		if b := filepath.Base(p); strings.HasPrefix(b, ".mrw-") && !strings.HasPrefix(b, ".mrw-aside-") {
			if n := strings.Count(strings.Join(u.calls, ","), "rename"); n == 1 && !failed {
				failed = true
				return nil, errors.New("injected: the temp cannot be looked at")
			}
		}
		return real(tr, p)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
	if err == nil || !strings.Contains(err.Error(), "could not be identified") {
		t.Fatalf("a temp that could not be looked at: %v (failed=%v, calls %v)", err, failed, u.calls)
	}
	for _, f := range res.Files {
		if f.NotHeld {
			t.Errorf("%s is marked as not holding mrw's bytes though nothing says so", f.Path)
		}
	}
	if left := mrwLeftovers(t, u.root); len(left) != 1 || !slices.Contains(res.LeftBehind, left[0]) {
		t.Errorf("the aside of the unidentified file: %v, left_behind %v", left, res.LeftBehind)
	}
}
