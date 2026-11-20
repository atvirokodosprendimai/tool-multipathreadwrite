package rooted

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
)

// openGapFn runs between the last name-based step of Open and the open beneath
// the held root. It is a seam for a test to swap a directory for a link there
// (#727).
var openGapFn = func() {}

// fallbackGapFn runs between the by-name open of the fallback and its identity
// check. It is a seam for a test to swap a directory there (#727).
var fallbackGapFn = func() {}

// Reader opens the files under one root for reading through an os.Root held on
// it (#727, ADR-007). A walk holds one for its length; a single read uses
// OpenRead. It is safe for concurrent use.
//
// Resolve decides by name that a path is inside the root, and a later open by
// name can lead somewhere else if a directory was swapped for a link in
// between. Through the held root every component below the handle is resolved
// and one that leaves it is refused. The path is handed over RESOLVED as far as
// it exists, as apply's tree does (ADR-106): os.Root follows a relative link
// that stays inside but refuses an absolute one, and a link inside the root, or
// a Windows junction, must keep reading (ADR-071).
type Reader struct {
	abs string   // the canonical root (Abs)
	r   *os.Root // nil when the root cannot be held open: see fallback
	// walk makes Open try the held root with the path as given first: a walk's
	// discovered files were resolved, mrw's state refused and the links of their
	// directories left alone by the walk itself, so resolving each again doubled
	// the cost of a read of a large tree. Anything the held root refuses is
	// judged again, in full.
	walk bool
	// dirs holds the directories a walk is reading from, each opened once as an
	// os.Root of its own (confined by the held root), so that a file costs one
	// openat and not one per component: opening through the held root alone was
	// two thirds of a walk's time. A handle names the directory as it was when
	// opened, so a directory swapped afterwards is not followed by it.
	mu   sync.Mutex
	dirs map[string]*dirRoot
}

// dirRoot is a directory held open as an os.Root, shared by the reads in flight.
type dirRoot struct {
	r    *os.Root
	refs int
	temp bool // not kept in dirs (it was full): closed by its last release
}

// maxDirRoots bounds the directories a walk holds open at once; a walk reads a
// directory's files together, and there are more reads in flight than directories
// open only when a tree is wider than this.
const maxDirRoots = 32

// acquireDir returns the held handle for dir, a clean path below the root, or
// nil when it cannot be opened (the read then goes through the root alone).
func (rd *Reader) acquireDir(dir string) *dirRoot {
	rd.mu.Lock()
	if d, ok := rd.dirs[dir]; ok {
		d.refs++
		rd.mu.Unlock()
		return d
	}
	rd.mu.Unlock()
	sub, err := rd.r.OpenRoot(dir)
	if err != nil {
		return nil
	}
	rd.mu.Lock()
	defer rd.mu.Unlock()
	if d, ok := rd.dirs[dir]; ok { // another read opened it meanwhile
		_ = sub.Close()
		d.refs++
		return d
	}
	d := &dirRoot{r: sub, refs: 1}
	if rd.dirs == nil {
		rd.dirs = map[string]*dirRoot{}
	}
	if len(rd.dirs) >= maxDirRoots {
		for k, o := range rd.dirs {
			if o.refs == 0 {
				_ = o.r.Close()
				delete(rd.dirs, k)
				break
			}
		}
	}
	if len(rd.dirs) >= maxDirRoots {
		d.temp = true
		return d
	}
	rd.dirs[dir] = d
	return d
}

func (rd *Reader) releaseDir(d *dirRoot) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	if d.refs--; d.refs == 0 && d.temp {
		_ = d.r.Close()
	}
}

// openWalked opens rel, a clean path below the root, through its directory's
// handle.
func (rd *Reader) openWalked(rel string) (*os.File, fs.FileInfo, error) {
	dir, base := filepath.Split(rel)
	dir = filepath.Clean(dir)
	if dir == "." {
		return regular.OpenIn(rd.r, base)
	}
	d := rd.acquireDir(dir)
	if d == nil {
		return regular.OpenIn(rd.r, rel)
	}
	defer rd.releaseDir(d)
	return regular.OpenIn(d.r, base)
}

// NewReader holds root open for reading. A root the process may search but not
// read, mode 0311, cannot be opened as an os.Root; its reads then take the
// by-name fallback with its identity check.
func NewReader(root string) (*Reader, error) {
	abs, err := Abs(root)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(abs)
	if err != nil {
		if !fallsBack(err) {
			return nil, err
		}
		r = nil
	}
	return &Reader{abs: abs, r: r}, nil
}

// NewWalkReader is NewReader for the walk of a tree, whose files it has already
// resolved: see Reader.walk. A file the walk did not discover, one it was named,
// is opened by Open's full judgement when the held root refuses it as written.
func NewWalkReader(root string) (*Reader, error) {
	rd, err := NewReader(root)
	if rd != nil {
		rd.walk = true
	}
	return rd, err
}

// Close releases the held root.
func (rd *Reader) Close() {
	rd.mu.Lock()
	for _, d := range rd.dirs {
		_ = d.r.Close()
	}
	rd.dirs = nil
	rd.mu.Unlock()
	if rd.r != nil {
		_ = rd.r.Close()
	}
}

// OpenRead opens for reading the file full, which Resolve returned for a path
// under root, through a root held for the one open.
func OpenRead(root, full string) (*os.File, fs.FileInfo, error) {
	rd, err := NewReader(root)
	if err != nil {
		return nil, nil, err
	}
	defer rd.Close()
	return rd.Open(full)
}

// Open opens for reading the file full, which Resolve returned for a path under
// the root, and returns it with its descriptor's FileInfo (regular.OpenIn). A
// path that resolves outside the root now is refused with Resolve's own error,
// and one in mrw's own state with ADR-077's (the name was judged once, and may
// have changed). A link that resolves to nothing is a file that does not exist,
// whatever it points at.
func (rd *Reader) Open(full string) (*os.File, fs.FileInfo, error) {
	if rd.walk && rd.r != nil {
		if rel, err := filepath.Rel(rd.abs, full); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			openGapFn()
			if f, fi, err := rd.openWalked(rel); err == nil {
				return f, fi, nil
			}
		}
	}
	resolved := RealAsFarAsItExists(full)
	rel, err := filepath.Rel(rd.abs, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, nil, &OutsideError{Path: full, Resolved: resolved, Root: rd.abs, Exists: true}
	}
	if inState(resolved) {
		return nil, nil, fmt.Errorf("%s is inside mrw's own state directory; %w", rel, ErrOwnState)
	}
	openGapFn()
	if rd.r != nil {
		f, fi, err := regular.OpenIn(rd.r, rel)
		if err == nil {
			return f, fi, nil
		}
		if !fallsBack(err) {
			return nil, nil, rd.classify(rel, resolved, err)
		}
	}
	return rd.openByName(rel, resolved)
}

// classify turns the root's refusal of a link left at the end of the path, one
// that resolves to nothing, into the error a missing file gives: os.Root says
// "path escapes from parent" for an absolute target it will not follow, which
// is not what a dangling link inside the root, or pointing nowhere, is.
func (rd *Reader) classify(rel, resolved string, err error) error {
	if fi, lerr := rd.r.Lstat(rel); lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
		if _, serr := os.Stat(resolved); errors.Is(serr, fs.ErrNotExist) {
			// The system's own words for it: "no such file or directory".
			var pe *fs.PathError
			if errors.As(serr, &pe) {
				return &fs.PathError{Op: "open", Path: rel, Err: pe.Err}
			}
			return &fs.PathError{Op: "open", Path: rel, Err: fs.ErrNotExist}
		}
	}
	return err
}

// openByName is the fallback for what os.Root cannot do: it opens directories
// for reading, so a search-only directory (chmod 111), a root at mode 0311, and
// on Windows a reparse point that is no link, are refused by it with a
// permission or reparse error although the file may be read. The file is opened
// by name, and before anything is read from it the descriptor is compared with
// where the name resolves NOW: it must still be inside the root and be the file
// that was opened (os.SameFile), or the read is refused.
func (rd *Reader) openByName(rel, resolved string) (*os.File, fs.FileInfo, error) {
	f, fi, err := regular.Open(resolved)
	if err != nil {
		return nil, nil, err
	}
	fallbackGapFn()
	full, err := resolveIn(rd.abs, rel)
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	st, err := os.Stat(full)
	if err != nil || !os.SameFile(fi, st) {
		_ = f.Close()
		return nil, nil, &fs.PathError{Op: "open", Path: rel, Err: errors.New("changed while it was being opened")}
	}
	return f, fi, nil
}
