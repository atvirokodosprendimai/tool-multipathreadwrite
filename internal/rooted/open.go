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
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// openGapFn runs between the last name-based step of Open and the open beneath
// the held root. It is a seam for a test to swap a directory for a link there
// (#727).
var openGapFn = func() {}

// byNameGapFn runs between the root's refusal and the by-name open of the
// fallback. It is a seam for a test to swap a directory there (#727).
var byNameGapFn = func() {}

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
//
// THE WINDOWS THAT REMAIN, by disposition (#738 final pass). (1) The root
// replaced while a reader is held: the held os.Root names the directory it was
// opened on, so reads stay inside that directory; the names this reader resolves
// (Abs at NewReader, then RealAsFarAsItExists) are looked up afresh, so a root
// swapped for another directory makes reads fail or find the new directory's
// files, never an outside path's. Main re-resolved the root's name on EVERY
// open (internal/read/read.go:78 regular.Open(path), no held handle), so this is
// narrower than main, not a regression. (2) The FIFO that replaces a directory
// between openDirProbe and OpenRoot (see openDir). (3) A platform with no
// descriptor path (fdPath: neither linux nor darwin) keeps the name's state check
// and, in the fallback, the identity comparison by a second lookup.
type Reader struct {
	abs string   // the canonical root (Abs)
	r   *os.Root // nil when the root cannot be held open: see fallback
	// walk makes Open try the held root with the path as given first: a walk's
	// discovered files were resolved, mrw's state refused and the links of their
	// directories left alone by the walk itself, so resolving each again doubled
	// the cost of a read of a large tree. Anything the held root refuses is
	// judged again, in full.
	walk bool
	// stateIn is set for a walk whose root and mrw's state directory contain one
	// another (XDG_STATE_HOME inside the checkout, or the root inside the state):
	// then the walk's files are judged again, state check included (ADR-077),
	// because a swap could lead a discovered file into the state. Computed once.
	stateIn bool
	// dirs holds the directories a walk is reading from, each opened once as an
	// os.Root of its own (confined by the held root), so that a file costs one
	// openat and not one per component: opening through the held root alone was
	// two thirds of a walk's time. A handle names the directory as it was when
	// opened, so a directory swapped afterwards is not followed by it.
	mu   sync.Mutex
	dirs map[string]*dirRoot
	// maxDirs bounds dirs (dirBudget); zero keeps none.
	maxDirs int
	// opening counts the handles being opened without the lock held, so the
	// bound holds across them; closed is set by Close, after which nothing is
	// cached or handed out.
	opening int
	closed  bool
}

// dirRoot is a directory held open as an os.Root, shared by the reads in flight.
type dirRoot struct {
	r    *os.Root
	refs int
}

// dirEventFn is told "open" before a directory handle is opened and "evict"
// when one is closed to make room: a seam for a test to hold the order (#727).
var dirEventFn = func(string) {}

// acquireGapFn runs just after a handle has been opened, with the lock released: a
// seam for a test to close the reader there (#727).
var acquireGapFn = func() {}

// acquireDir returns the held handle for dir, a clean path below the root, or
// nil when there is none to give: the directory cannot be opened, is not a
// directory, the reader is closed, or the cache is full of handles in use (the
// read then goes through the root alone). The cache never holds more than
// rd.maxDirs handles, counting those being opened: one is evicted BEFORE the
// next is opened, so the descriptors the walk holds are bounded by what the
// process may open (#727: a walk under ulimit -n 24 read nothing, having opened
// a 33rd handle before closing one). The lock is not held while a directory is
// opened; the slot is reserved, and a duplicate opened meanwhile is closed.
func (rd *Reader) acquireDir(dir string) *dirRoot {
	rd.mu.Lock()
	if rd.closed {
		rd.mu.Unlock()
		return nil
	}
	if d, ok := rd.dirs[dir]; ok {
		d.refs++
		rd.mu.Unlock()
		return d
	}
	if rd.maxDirs <= 0 || (len(rd.dirs)+rd.opening >= rd.maxDirs && !rd.evictOneLocked()) {
		rd.mu.Unlock()
		return nil
	}
	rd.opening++
	rd.mu.Unlock()
	sub := rd.openDir(dir)
	acquireGapFn() // the handle is open and the lock is not held (a test closes the reader here)
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.opening--
	if sub == nil {
		return nil
	}
	if rd.closed { // Close ran while it was being opened: nothing is kept
		_ = sub.Close()
		return nil
	}
	if d, ok := rd.dirs[dir]; ok { // another read opened it meanwhile
		_ = sub.Close()
		d.refs++
		return d
	}
	d := &dirRoot{r: sub, refs: 1}
	if rd.dirs == nil {
		rd.dirs = map[string]*dirRoot{}
	}
	rd.dirs[dir] = d
	return d
}

// openDir opens dir as an os.Root of its own, without the lock. A directory
// replaced by a FIFO blocks OpenRoot until something writes to it (Go's OpenRoot
// opens without O_NONBLOCK), so the directory is first opened through the parent
// root with O_DIRECTORY|O_NONBLOCK, which a FIFO fails at once. THE WINDOW THAT
// REMAINS is the instant between that probe and OpenRoot: a FIFO swapped in
// there can still hold this one read until a writer comes, and can lead nowhere
// (OpenRoot confines).
func (rd *Reader) openDir(dir string) *os.Root {
	dirEventFn("open")
	if !openDirProbe(rd.r, dir) {
		return nil
	}
	sub, err := rd.r.OpenRoot(dir)
	if err != nil && fdExhausted(err) {
		rd.mu.Lock()
		rd.dropIdleLocked()
		rd.mu.Unlock()
		sub, err = rd.r.OpenRoot(dir)
	}
	if err != nil {
		return nil
	}
	return sub
}

// evictOneLocked closes one handle nobody is using and reports whether there was one.
func (rd *Reader) evictOneLocked() bool {
	for k, o := range rd.dirs {
		if o.refs == 0 {
			dirEventFn("evict")
			_ = o.r.Close()
			delete(rd.dirs, k)
			return true
		}
	}
	return false
}

// dropIdleLocked closes every handle nobody is using, and stops the cache
// growing again: the process is out of descriptors.
func (rd *Reader) dropIdleLocked() {
	for rd.evictOneLocked() {
	}
	rd.maxDirs = 0
}

func (rd *Reader) releaseDir(d *dirRoot) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	d.refs--
}

// openWalked opens rel, a clean path below the root, through its directory's
// handle. A process out of descriptors drops the cache and tries once more.
func (rd *Reader) openWalked(rel string) (*os.File, fs.FileInfo, error) {
	f, fi, err := rd.openWalkedOnce(rel)
	if err != nil && fdExhausted(err) {
		rd.mu.Lock()
		rd.dropIdleLocked()
		rd.mu.Unlock()
		return rd.openWalkedOnce(rel)
	}
	return f, fi, err
}

func (rd *Reader) openWalkedOnce(rel string) (*os.File, fs.FileInfo, error) {
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

// stateTouches reports whether mrw's state directory lies inside abs, or abs
// inside it: only then can a discovered file be led into the state (ADR-077).
func stateTouches(abs string) bool {
	base, err := state.Base()
	if err != nil {
		return false
	}
	b, _ := resolvedBase(base)
	return fileUnder(abs, b) || fileUnder(b, abs)
}

// fileUnder reports whether p is top or lies below it, comparing as FILES where
// the strings differ (os.SameFile), as inState does: on a filesystem that folds
// case /work/Repo and /work/repo are one directory (#727).
func fileUnder(top, p string) bool {
	if Contains(top, p) {
		return true
	}
	ti, err := os.Stat(top)
	if err != nil {
		return false
	}
	for q := p; ; {
		if qi, err := os.Stat(q); err == nil && os.SameFile(ti, qi) {
			return true
		}
		parent := filepath.Dir(q)
		if parent == q {
			return false
		}
		q = parent
	}
}

// NewWalkReader is NewReader for the walk of a tree, whose files it has already
// resolved: see Reader.walk. A file the walk did not discover, one it was named,
// is opened by Open's full judgement when the held root refuses it as written.
func NewWalkReader(root string) (*Reader, error) {
	rd, err := NewReader(root)
	if rd != nil {
		rd.walk = true
		rd.stateIn = stateTouches(rd.abs)
		rd.maxDirs = dirBudget()
	}
	return rd, err
}

// Close releases the held root.
func (rd *Reader) Close() {
	rd.mu.Lock()
	rd.closed = true
	rd.mu.Unlock()
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
	if rd.walk && !rd.stateIn && rd.r != nil {
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
			// The state check above judged the NAME; a link swapped in since is
			// caught by where the open descriptor is (ADR-077, #727). Where a
			// platform cannot say (neither linux nor darwin), the name's check is all
			// there is.
			if p, ok := fdPath(f); ok && inState(p) {
				_ = f.Close()
				return nil, nil, fmt.Errorf("%s is inside mrw's own state directory; %w", rel, ErrOwnState)
			}
			return f, fi, nil
		}
		if !fallsBack(err) {
			return nil, nil, rd.classify(full, rel, resolved, err)
		}
	}
	return rd.openByName(rel, resolved)
}

// StatRead describes the file full, under root, through a root held for the
// one call: its type and size are those of the file the root reaches, and an
// outside file's are never reported (#727). For what OpenRead cannot open, a
// FIFO or a file the process may not read.
func StatRead(root, full string) (fs.FileInfo, error) {
	rd, err := NewReader(root)
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	resolved := RealAsFarAsItExists(full)
	rel, err := filepath.Rel(rd.abs, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, &OutsideError{Path: full, Resolved: resolved, Root: rd.abs, Exists: true}
	}
	if rd.r == nil { // a root that cannot be held open: the name, as the fallback does
		return os.Stat(resolved)
	}
	return rd.r.Stat(rel)
}

// classify gives the root's refusal the words a lookup by name gave (#727). A
// link on the way, at any component, that resolves to nothing is a file that
// does not exist: os.Root says "path escapes from parent" for an absolute
// target it will not follow, which is not what a dangling link inside the root,
// or pointing nowhere, is. Any other error names the path as it was looked up,
// in full ("open /root/README.md/nope: not a directory"), not as the root saw it.
func (rd *Reader) classify(full, rel, resolved string, err error) error {
	p := ""
	for _, comp := range strings.Split(rel, string(filepath.Separator)) {
		p = filepath.Join(p, comp)
		fi, lerr := rd.r.Lstat(p)
		if lerr != nil {
			break
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			// A link on the way that does not lead to a file: the lookup by name
			// says why, as it did (ENOENT, ENOTDIR for "/repo/README.md/nope", ...),
			// where os.Root only says it will not follow an absolute target.
			if _, serr := os.Stat(resolved); serr != nil {
				var pe *fs.PathError
				if errors.As(serr, &pe) {
					return &fs.PathError{Op: "open", Path: full, Err: pe.Err}
				}
				return &fs.PathError{Op: "open", Path: full, Err: serr}
			}
			break
		}
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return &fs.PathError{Op: "open", Path: full, Err: pe.Err}
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
	byNameGapFn()
	f, fi, err := regular.Open(resolved)
	if err != nil {
		return nil, nil, err
	}
	fallbackGapFn()
	// Where the system says the open file is, not a second lookup of the name:
	// the descriptor is bound to what it is, wherever the name leads now.
	if p, ok := fdPath(f); ok {
		if !fileUnder(rd.abs, p) {
			_ = f.Close()
			return nil, nil, &OutsideError{Path: resolved, Resolved: p, Root: rd.abs, Exists: true}
		}
		if inState(p) {
			_ = f.Close()
			return nil, nil, fmt.Errorf("%s is inside mrw's own state directory; %w", rel, ErrOwnState)
		}
		return f, fi, nil
	}
	// No such path for a descriptor here (or the platform has none): compare it
	// with where the name resolves again.
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
