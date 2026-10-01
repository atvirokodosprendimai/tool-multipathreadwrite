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

// rel spells p relative to the root, and refuses a path the root does not
// hold: resolution followed a link out of it.
func (t *tree) rel(p string) (string, error) {
	for _, base := range []string{t.abs, t.alias} {
		r, err := filepath.Rel(base, p)
		if err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r) {
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
