package apply

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ADR-209 T1. A content commit that fails partway puts the files it renamed back.

// u209 is a tree of a.txt, b.txt and c.txt, one line each, read whole, the
// identity of each taken before the write, and the seams of ADR-209 recording
// each call in order.
type u209 struct {
	t      *testing.T
	root   string
	before map[string]os.FileInfo
	calls  []string
}

func newU209(t *testing.T) *u209 {
	t.Helper()
	u := &u209{t: t, root: t.TempDir(), before: map[string]os.FileInfo{}}
	for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
		write(t, u.root, n, "orig "+n+"\n")
		fi, err := os.Stat(filepath.Join(u.root, n))
		if err != nil {
			t.Fatal(err)
		}
		u.before[n] = fi
	}
	realRename, realLink, realRemove := commitRenameFn, asideLinkFn, removeContentFn
	t.Cleanup(func() { commitRenameFn, asideLinkFn, removeContentFn = realRename, realLink, realRemove })
	commitRenameFn = func(tr *tree, from, to string) error {
		u.calls = append(u.calls, "rename "+filepath.Base(to))
		return realRename(tr, from, to)
	}
	asideLinkFn = func(tr *tree, from, to string) error {
		u.calls = append(u.calls, "link "+filepath.Base(from)+" "+filepath.Base(to))
		return realLink(tr, from, to)
	}
	removeContentFn = func(tr *tree, p string, want os.FileInfo) (*vacated, error) {
		u.calls = append(u.calls, "remove "+filepath.Base(p))
		return realRemove(tr, p, want)
	}
	return u
}

func (u *u209) edit(names ...string) []Input {
	var in []Input
	for i, n := range names {
		in = append(in, Input{Path: n, Start: 1, End: 1, Op: "replace", Body: []string{"new " + n}, Lines: -1, Index: i})
	}
	return in
}

// restored says a.txt holds its original bytes and is the original file.
func (u *u209) restored(names ...string) {
	u.t.Helper()
	for _, n := range names {
		if got := read(u.t, u.root, n); got != "orig "+n+"\n" {
			u.t.Errorf("%s = %q, want its original", n, got)
		}
		fi, err := os.Stat(filepath.Join(u.root, n))
		if err != nil || !os.SameFile(fi, u.before[n]) {
			u.t.Errorf("%s is not the original file: %v", n, err)
		}
	}
}

func (u *u209) noMrwEntries() {
	u.t.Helper()
	var left []string
	filepath.WalkDir(u.root, func(p string, d os.DirEntry, _ error) error {
		if strings.HasPrefix(d.Name(), ".mrw-") {
			left = append(left, p)
		}
		return nil
	})
	if len(left) != 0 {
		u.t.Errorf("the tree holds %v", left)
	}
}

func (u *u209) refuse(base string) {
	real := commitRenameFn
	commitRenameFn = func(tr *tree, from, to string) error {
		if filepath.Base(to) == base && !strings.HasPrefix(filepath.Base(from), ".mrw-aside-") {
			u.calls = append(u.calls, "rename "+base+" refused")
			return errors.New("injected: " + base + " refused")
		}
		return real(tr, from, to)
	}
}

// withoutAsides makes every aside link refuse, as on a filesystem with no hard
// links (ADR-209 Decision 2): a commit that fails partway then leaves the files it
// renamed written, as it did before asides. For the tests of what a receipt says
// of a partial commit, which an undo that put everything back no longer reaches.
func withoutAsides(t *testing.T) {
	t.Helper()
	real := asideLinkFn
	t.Cleanup(func() { asideLinkFn = real })
	asideLinkFn = func(*tree, string, string) error { return errors.New("no hard links here") }
}

func writtenPaths(res Result) []string {
	var w []string
	for _, f := range res.Files {
		if f.Written {
			w = append(w, f.Path)
		}
	}
	return w
}

func TestAContentCommitThatFailsPartwayUndoesTheFilesItRenamed(t *testing.T) {
	t.Run("third_rename_refused", func(t *testing.T) {
		u := newU209(t)
		u.refuse("c.txt")
		res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
		if err == nil || res.Applied {
			t.Fatalf("a refused rename was not reported: %v", err)
		}
		u.restored("a.txt", "b.txt", "c.txt")
		u.noMrwEntries()
		if bi, ai := slices.Index(u.calls, "remove b.txt"), slices.Index(u.calls, "remove a.txt"); bi < 0 || ai < 0 || bi > ai {
			t.Errorf("the undo ran oldest first or not at all: %v", u.calls)
		}
		if h := hunkFor(t, res, "c.txt"); h.Status != StatusFailed {
			t.Errorf("c.txt hunk %s, want failed", h.Status)
		}
		for _, n := range []string{"a.txt", "b.txt"} {
			if h := hunkFor(t, res, n); h.Status != StatusSkipped {
				t.Errorf("%s hunk %s, want skipped", n, h.Status)
			}
		}
		if w := writtenPaths(res); len(w) != 0 {
			t.Errorf("files written after the undo: %v", w)
		}
		if strings.Contains(err.Error(), "ALREADY WRITTEN") || !strings.Contains(err.Error(), "nothing was written") {
			t.Errorf("error = %v, want it built after the undo", err)
		}
	})

	t.Run("path_op_refused", func(t *testing.T) {
		u := newU209(t)
		if err := os.Mkdir(filepath.Join(u.root, "d"), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, u.root, "d/x.txt", "X\n")
		real := commitRenameFn
		commitRenameFn = func(tr *tree, from, to string) error {
			if filepath.Base(from) == "x.txt" && strings.HasPrefix(filepath.Base(to), ".mrw-aside-") {
				return errors.New("injected: the unlink is refused")
			}
			return real(tr, from, to)
		}
		in := append(u.edit("a.txt"), Input{Path: "d/x.txt", Op: "unlink", Lines: -1, Index: 1})
		res, err := Apply(u.root, in, Options{})
		if err == nil || res.Applied {
			t.Fatalf("a refused unlink was not reported: %v", err)
		}
		u.restored("a.txt")
		u.noMrwEntries()
		if read(t, u.root, "d/x.txt") != "X\n" {
			t.Error("d/x.txt is not in place")
		}
		if strings.Contains(err.Error(), "ALREADY WRITTEN") || !strings.Contains(err.Error(), "nothing was written") {
			t.Errorf("error = %v, want it built after both undos", err)
		}
	})

	t.Run("nested_create", func(t *testing.T) {
		u := newU209(t)
		u.refuse("a.txt")
		in := []Input{{Path: "n/m/k.txt", Op: "create", Body: []string{"K"}, Lines: -1, Index: 0}}
		in = append(in, u.edit("a.txt")...)
		in[1].Index = 1
		res, err := Apply(u.root, in, Options{})
		if err == nil || res.Applied {
			t.Fatalf("a refused rename was not reported: %v", err)
		}
		if bi, ai := slices.Index(u.calls, "rename k.txt"), slices.Index(u.calls, "rename a.txt refused"); bi < 0 || ai < bi {
			t.Fatalf("k.txt must land before a.txt is refused: %v", u.calls)
		}
		if _, err := os.Stat(filepath.Join(u.root, "n")); !os.IsNotExist(err) {
			t.Errorf("the created directories are still there: %v", err)
		}
		u.restored("a.txt")
		u.noMrwEntries()
		if len(res.DirsCreated) != 0 {
			t.Errorf("dirs_created = %v after the undo", res.DirsCreated)
		}
	})

	t.Run("edited_in_place_after_rename", func(t *testing.T) {
		u := newU209(t)
		real := commitRenameFn
		commitRenameFn = func(tr *tree, from, to string) error {
			if filepath.Base(to) == "c.txt" {
				f, err := os.OpenFile(filepath.Join(u.root, "a.txt"), os.O_APPEND|os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = f.WriteString("appended\n")
				_ = f.Close()
				return errors.New("injected: c.txt refused")
			}
			return real(tr, from, to)
		}
		res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
		if err == nil || res.Applied {
			t.Fatalf("a refused rename was not reported: %v", err)
		}
		if got := read(t, u.root, "a.txt"); !strings.Contains(got, "appended") {
			t.Fatalf("a.txt = %q, want it left with the append", got)
		}
		u.restored("b.txt", "c.txt")
		asides := mrwLeftovers(t, u.root)
		if len(asides) != 1 || read(t, u.root, asides[0]) != "orig a.txt\n" {
			t.Fatalf("the aside of a.txt: %v", asides)
		}
		if !slices.Contains(res.LeftBehind, asides[0]) || !strings.Contains(err.Error(), "UNDO INCOMPLETE") {
			t.Errorf("left_behind %v, error %v: want the aside named under UNDO INCOMPLETE", res.LeftBehind, err)
		}
		if w := writtenPaths(res); !slices.Equal(w, []string{"a.txt"}) {
			t.Errorf("written = %v, want a.txt only", w)
		}
		if h := hunkFor(t, res, "a.txt"); h.Status != StatusFailed || h.Reason == "" {
			t.Errorf("a.txt hunk %s %q, want failed with the undo's reason", h.Status, h.Reason)
		}
		if h := hunkFor(t, res, "b.txt"); h.Status != StatusSkipped {
			t.Errorf("b.txt hunk %s, want skipped", h.Status)
		}
		if !res.Files[0].NotHeld {
			t.Errorf("a.txt's record does not carry the mark that its path does not hold mrw's bytes: %+v", res.Files[0])
		}
	})

	t.Run("path_taken_before_restore", func(t *testing.T) {
		u := newU209(t)
		u.refuse("c.txt")
		real := removeContentFn
		removeContentFn = func(tr *tree, p string, want os.FileInfo) (*vacated, error) {
			v, err := real(tr, p, want)
			if filepath.Base(p) == "b.txt" && err == nil {
				write(t, u.root, "b.txt", "OTHER\n")
			}
			return v, err
		}
		res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
		if err == nil || res.Applied {
			t.Fatalf("a refused rename was not reported: %v", err)
		}
		if read(t, u.root, "b.txt") != "OTHER\n" {
			t.Errorf("b.txt = %q, want the file that took the path", read(t, u.root, "b.txt"))
		}
		u.restored("a.txt")
		asides := mrwLeftovers(t, u.root)
		if len(asides) != 1 || read(t, u.root, asides[0]) != "orig b.txt\n" || !slices.Contains(res.LeftBehind, asides[0]) {
			t.Errorf("the aside of b.txt: %v, left_behind %v", asides, res.LeftBehind)
		}
		if w := writtenPaths(res); !slices.Equal(w, []string{"b.txt"}) {
			t.Errorf("written = %v, want b.txt only", w)
		}
		if h := hunkFor(t, res, "b.txt"); h.Status != StatusFailed {
			t.Errorf("b.txt hunk %s, want failed", h.Status)
		}
	})

	t.Run("no_aside_for_one_file", func(t *testing.T) {
		u := newU209(t)
		u.refuse("c.txt")
		real := asideLinkFn
		asideLinkFn = func(tr *tree, from, to string) error {
			if filepath.Base(from) == "b.txt" {
				u.calls = append(u.calls, "link b.txt refused")
				return errors.New("injected: no hard links")
			}
			return real(tr, from, to)
		}
		res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
		if err == nil || res.Applied {
			t.Fatalf("a refused rename was not reported: %v", err)
		}
		if slices.Contains(u.calls, "remove b.txt") {
			t.Errorf("a file with no aside was removed by the undo: %v", u.calls)
		}
		u.restored("a.txt", "c.txt")
		if got := read(t, u.root, "b.txt"); got != "new b.txt\n" {
			t.Errorf("b.txt = %q, want mrw's bytes", got)
		}
		if w := writtenPaths(res); !slices.Equal(w, []string{"b.txt"}) {
			t.Errorf("written = %v, want b.txt only", w)
		}
		if h := hunkFor(t, res, "b.txt"); h.Status != StatusOK {
			t.Errorf("b.txt hunk %s, want ok: its path holds what the hunk wrote", h.Status)
		}
		if !strings.Contains(err.Error(), "UNDO INCOMPLETE") || !strings.Contains(err.Error(), "b.txt") || !strings.Contains(err.Error(), "no hard links") {
			t.Errorf("error = %v, want UNDO INCOMPLETE naming b.txt and the link's error", err)
		}
	})

	t.Run("edit_and_rename_move_back_refused", func(t *testing.T) {
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
			case filepath.Base(from) == "a.txt" && filepath.Base(to) == "a.txt" && filepath.Dir(from) != filepath.Dir(to):
				if strings.HasSuffix(filepath.Dir(from), "e") {
					return errors.New("injected: the rename back is refused")
				}
			}
			return real(tr, from, to)
		}
		in := append(u.edit("a.txt"), Input{Path: "a.txt", Op: "rename", Body: []string{"e/a.txt"}, Lines: -1, Index: 1}, Input{Path: "x.txt", Op: "unlink", Lines: -1, Index: 2})
		res, err := Apply(u.root, in, Options{})
		if err == nil || res.Applied {
			t.Fatalf("a refused unlink was not reported: %v", err)
		}
		if got := read(t, u.root, "e/a.txt"); got != "new a.txt\n" {
			t.Errorf("e/a.txt = %q, want the edit", got)
		}
		asides := mrwLeftovers(t, u.root)
		if len(asides) != 1 || read(t, u.root, asides[0]) != "orig a.txt\n" || !slices.Contains(res.LeftBehind, asides[0]) {
			t.Errorf("the aside of a.txt: %v, left_behind %v", asides, res.LeftBehind)
		}
		for _, h := range res.Hunks {
			if h.Path == "a.txt" && h.Status != StatusOK {
				t.Errorf("a.txt %s hunk %s, want ok: e/a.txt holds the edit", h.Op, h.Status)
			}
		}
	})

	t.Run("landed", func(t *testing.T) {
		u := newU209(t)
		res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
		if err != nil || !res.Applied {
			t.Fatalf("a plan with nothing failing: %v", err)
		}
		if n := strings.Count(strings.Join(u.calls, ","), "link "); n != 3 {
			t.Errorf("asides linked %d times, want 3: %v", n, u.calls)
		}
		u.noMrwEntries()
		if len(res.LeftBehind) != 0 {
			t.Errorf("left_behind = %v", res.LeftBehind)
		}
	})

	t.Run("aside_removal_refused", func(t *testing.T) {
		u := newU209(t)
		realRemove := removeFn
		t.Cleanup(func() { removeFn = realRemove })
		removeFn = func(tr *tree, p string) error {
			if strings.HasPrefix(filepath.Base(p), ".mrw-aside-") && !slices.ContainsFunc(u.calls, func(c string) bool { return c == "aside removal refused" }) {
				u.calls = append(u.calls, "aside removal refused")
				return errors.New("injected: the aside cannot be removed")
			}
			return realRemove(tr, p)
		}
		res, err := Apply(u.root, u.edit("a.txt", "b.txt", "c.txt"), Options{})
		if err != nil || !res.Applied {
			t.Fatalf("a landed plan whose aside cannot be removed: %v", err)
		}
		left := mrwLeftovers(t, u.root)
		if len(left) != 1 || !slices.Contains(res.LeftBehind, left[0]) {
			t.Errorf("leftover %v, left_behind %v: want the aside named", left, res.LeftBehind)
		}
	})
}
