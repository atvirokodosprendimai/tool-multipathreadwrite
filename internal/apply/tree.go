package apply

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// tree is the checkout a write may change, held open as an os.Root for the
// length of one Apply (ADR-106). rooted.Resolve judges a path by its name, and
// every later step used to reopen it by name, so a directory swapped for a link
// after validation redirected the step: an unlink moved the file the link led
// to into an aside and removed it, and a rename built its directories wherever
// the link pointed. Through the root, each step resolves every component below
// the handle and the OS refuses one that leaves it.
//
// Its methods take the absolute paths the write path already carries, RESOLVED
// (rooted.RealAsFarAsItExists), so no link is left in them: os.Root follows a
// relative link that stays inside but refuses an absolute one, and an in-root
// link or Windows junction must stay writable. A link the root meets was
// swapped in after resolution.
type tree struct {
	r   *os.Root
	abs string // the canonical root the handle was opened on (rooted.Abs)
	// alias is the root as the caller spelled it, made absolute: the same
	// directory reached through a link (/var and /private/var on macOS). A
	// path spelled under it is beneath the same handle.
	alias string
	// keep makes rename refuse to replace (noReplace).
	keep bool
}

// openTree opens absRoot, which must be canonical, as the tree of one Apply;
// root is the caller's spelling of it.
func openTree(absRoot, root string) (*tree, error) {
	r, err := os.OpenRoot(absRoot)
	if err != nil {
		return nil, err
	}
	alias, err := filepath.Abs(root)
	if err != nil {
		alias = absRoot
	}
	return &tree{r: r, abs: absRoot, alias: alias}, nil
}

// close releases the root handle. A failed close changes nothing on disk.
func (t *tree) close() { _ = t.r.Close() }

// dotGitError is a path refused because it lands in a .git directory (ADR-143).
// Its cause is the target's, so a refusal while staging is a failed hunk.
type dotGitError struct{ rel string }

func (e *dotGitError) Error() string {
	return fmt.Sprintf("%s resolves into a .git directory; mrw does not write there (use git for it)", filepath.ToSlash(e.rel))
}

// landsInGit reports whether the entry p names sits in a .git directory NOW:
// its parent resolved as it is at this instant, and the leaf as written, since a
// rename or an unlink acts on the entry and not on what a link at its end leads
// to. A parent swapped for a link to .git after validation shows here.
func (t *tree) landsInGit(p string) bool {
	landing := filepath.Join(rooted.RealAsFarAsItExists(filepath.Dir(p)), filepath.Base(p))
	r, err := filepath.Rel(t.abs, landing)
	return err == nil && rooted.HasGitComponent(r)
}

// rel spells p relative to the root, and refuses a path the root does not
// hold: resolution followed a link out of it.
func (t *tree) rel(p string) (string, error) {
	for _, base := range []string{t.abs, t.alias} {
		r, err := filepath.Rel(base, p)
		if err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r) {
			// ADR-143: every staging, rename and removal reopens its path by name, so
			// a directory swapped for a link to .git after validation is refused here
			// too, where the path is judged by where it now lands.
			if rooted.HasGitComponent(r) || t.landsInGit(p) {
				return "", &dotGitError{rel: r}
			}
			return r, nil
		}
	}
	return "", fmt.Errorf("%s resolves outside the root %s", p, t.abs)
}

// mkdirAll makes p and its missing parents beneath the root.
func (t *tree) mkdirAll(p string, perm fs.FileMode) error {
	r, err := t.rel(p)
	if err != nil {
		return err
	}
	return t.r.MkdirAll(r, perm)
}

// createExcl creates p, which must not exist, for writing.
func (t *tree) createExcl(p string, perm fs.FileMode) (*os.File, error) {
	r, err := t.rel(p)
	if err != nil {
		return nil, err
	}
	return t.r.OpenFile(r, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
}

// createTemp creates a new file in dir named prefix plus a random suffix, as
// os.CreateTemp does — os.Root has no CreateTemp — and returns it with its
// absolute path.
func (t *tree) createTemp(dir, prefix string) (*os.File, string, error) {
	for range 100 {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, "", err
		}
		name := filepath.Join(dir, prefix+hex.EncodeToString(b[:]))
		f, err := t.createExcl(name, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		return f, name, nil
	}
	return nil, "", fmt.Errorf("could not make a temp file in %s: every name tried exists", dir)
}

// rename moves from to to, both beneath the root.
func (t *tree) rename(from, to string) error {
	if t.keep {
		return t.renameNew(from, to)
	}
	rf, err := t.rel(from)
	if err != nil {
		return err
	}
	rt, err := t.rel(to)
	if err != nil {
		return err
	}
	return t.r.Rename(rf, rt)
}

// noReplace is this tree with a rename that refuses to replace.
func (t *tree) noReplace() *tree {
	c := *t
	c.keep = true
	return &c
}

// extraNameError is renameNew's failure when it linked the file under its new
// name and could neither remove the old name nor take the new one back: the file
// has two names, and name is the one mrw made. The caller names it as left behind.
type extraNameError struct {
	err  error
	name string
}

func (e *extraNameError) Error() string { return e.err.Error() }
func (e *extraNameError) Unwrap() error { return e.err }

// afterLink runs in renameNew between the link and the removal of the old name,
// the instant other processes can act on either: a seam for the tests of it.
var afterLink = func(from, to string) {}

// linkFn and fallbackLstatFn are the seams of renameNew's two system calls: a
// test makes a filesystem that cannot hard-link, or one whose lookups fail.
var (
	linkFn = func(t *tree, rf, rt string) error { return t.r.Link(rf, rt) }

	fallbackLstatFn = func(t *tree, rt string) (fs.FileInfo, error) { return t.r.Lstat(rt) }
)

// renameNew moves from to to and never replaces: a file another process made at
// to since mrw looked is left alone and the move fails with an error that
// satisfies fs.ErrExist (#726, #731). It is a hard link followed by the removal
// of the old name, since os.Root offers no no-replace rename and the link is
// refused by the OS when the name is taken, atomically. Only where the OS says
// the entry cannot be hard-linked (noHardLinks: a filesystem without hard links
// such as exFAT or FAT, a protected_hardlinks refusal, another device) does it
// fall back to asking whether to is free and then renaming, which leaves the
// window between the two calls: there a name made in between is replaced. Any
// other link error is the move's, and so is a failed lookup in that fallback: a
// name that cannot be looked at is not known to be free. A destination that is
// the source itself is a plain rename only when it is a respelling of the leaf
// on a filesystem that folds case (ADR-129); two names of one file otherwise are
// a hard link, which rename(2) reports done and changes nothing for, so the move
// is refused as a taken name.
//
// After the link, to is looked at again: it must still be the file linked, or
// another file took the name and the move failed with the source where it was.
// Then the source is looked at: gone, or another file now (an editor's atomic
// save), it is not removed, and the move stands with the name it has. The
// instant between that look and the removal remains. Any failure after the link
// leaves two names or none of ours, and says which: the new name is taken back,
// and when that fails too, or when it cannot be looked at, the error carries it
// (extraNameError) for left_behind.
func (t *tree) renameNew(from, to string) error {
	rf, err := t.rel(from)
	if err != nil {
		return err
	}
	rt, err := t.rel(to)
	if err != nil {
		return err
	}
	linked, err := t.r.Lstat(rf)
	if err != nil {
		return err
	}
	lerr := linkFn(t, rf, rt)
	switch {
	case lerr == nil:
		afterLink(from, to)
		return t.finishLink(rf, rt, to, linked)
	case errors.Is(lerr, fs.ErrExist):
		if respelling(from, to) == respelled {
			return t.r.Rename(rf, rt)
		}
		return lerr
	case noHardLinks(lerr):
		return t.checkThenRename(from, to, rf, rt)
	}
	return lerr
}

// finishLink is the end of a move that linked: to must be the file linked, the
// source is removed unless it is no longer that file.
func (t *tree) finishLink(rf, rt, to string, linked fs.FileInfo) error {
	b, berr := t.r.Lstat(rt)
	switch {
	case errors.Is(berr, fs.ErrNotExist):
		return berr // the name made is gone and the source was not touched: nothing moved
	case berr != nil:
		// Both names may exist and neither can be looked at: say which one is ours.
		return &extraNameError{err: fmt.Errorf("the new name %s cannot be looked at after the link: %w", to, berr), name: to}
	case !os.SameFile(b, linked):
		// Another file took the name after the link: the move failed, the source
		// stands where it was, and the temp or old name is the caller's to clean.
		return &fs.PathError{Op: "rename", Path: to, Err: fs.ErrExist}
	}
	a, aerr := t.r.Lstat(rf)
	switch {
	case errors.Is(aerr, fs.ErrNotExist):
		return nil // another process removed the source: to holds the file now
	case aerr == nil && !os.SameFile(a, linked):
		return nil // the source is another file now: not ours to remove
	}
	if rerr := t.r.Remove(rf); rerr != nil {
		if errors.Is(rerr, fs.ErrNotExist) {
			return nil
		}
		// Two names for one file: take the new one back so the move is whole.
		if uerr := t.r.Remove(rt); uerr != nil {
			return &extraNameError{err: fmt.Errorf("%w; and the extra name %s could not be removed: %w", rerr, to, uerr), name: to}
		}
		return rerr
	}
	return nil
}

// checkThenRename is the move where hard links are unavailable: to must be free,
// or the source under another spelling of its leaf, and a lookup that fails for
// any other reason refuses it.
func (t *tree) checkThenRename(from, to, rf, rt string) error {
	_, serr := fallbackLstatFn(t, rt)
	switch {
	case serr == nil:
		if respelling(from, to) == respelled {
			return t.r.Rename(rf, rt)
		}
		return &fs.PathError{Op: "rename", Path: to, Err: fs.ErrExist}
	case errors.Is(serr, fs.ErrNotExist):
		return t.r.Rename(rf, rt)
	}
	return serr
}

// remove removes p, a file or an empty directory, beneath the root.
func (t *tree) remove(p string) error {
	r, err := t.rel(p)
	if err != nil {
		return err
	}
	return t.r.Remove(r)
}

// lstat describes p beneath the root without following a link at its end.
func (t *tree) lstat(p string) (fs.FileInfo, error) {
	r, err := t.rel(p)
	if err != nil {
		return nil, err
	}
	return t.r.Lstat(r)
}

// stat describes p beneath the root, following a link at its end.
func (t *tree) stat(p string) (fs.FileInfo, error) {
	r, err := t.rel(p)
	if err != nil {
		return nil, err
	}
	return t.r.Stat(r)
}

// openRead opens p for reading beneath the root, without blocking (regular.OpenIn).
func (t *tree) openRead(p string) (*os.File, fs.FileInfo, error) {
	r, err := t.rel(p)
	if err != nil {
		return nil, nil, err
	}
	return regular.OpenIn(t.r, r)
}

// readDir lists the directory p beneath the root.
func (t *tree) readDir(p string) ([]fs.DirEntry, error) {
	r, err := t.rel(p)
	if err != nil {
		return nil, err
	}
	// ADR-109: without blocking, so a directory swapped for a FIFO is an error
	// from ReadDir rather than an open that waits for a writer.
	f, err := t.r.OpenFile(r, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return f.ReadDir(-1)
}
