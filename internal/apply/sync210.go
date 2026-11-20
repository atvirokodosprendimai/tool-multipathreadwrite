package apply

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// SyncError is a sync that failed AFTER the plan landed (ADR-210): every file is
// where the plan put it and the receipt says so, but the directory entries, or
// the data a drain was to make stable, are not known to be on stable storage. The
// writer turns it into the error a caller sees; Path is what could not be synced.
type SyncError struct {
	Path string
	Err  error
}

func (e *SyncError) Error() string { return fmt.Sprintf("could not sync %s: %v", e.Path, e.Err) }

// Unwrap returns the filesystem's own error.
func (e *SyncError) Unwrap() error { return e.Err }

// ErrStagingSync is what errors.Is finds in a sync that failed BEFORE anything was
// renamed: a staged temp, or the drain that makes the temps stable. It is the
// environment's failure, not the plan's, so a caller that scores plans (the
// curve) must not read it as a refusal of the plan.
var ErrStagingSync = errors.New("a staged file could not be made stable")

// tempSyncError is a staged temp, or the drain after the temps, that could not be
// made durable. Nothing is renamed yet, so it stops the plan, as any staging
// failure that is not the target's does.
type tempSyncError struct{ Err error }

func (e *tempSyncError) Error() string {
	return "the staged file could not be synced: " + e.Err.Error()
}

// Unwrap returns the filesystem's own error.
func (e *tempSyncError) Unwrap() error { return e.Err }

// Is makes errors.Is(err, ErrStagingSync) true.
func (e *tempSyncError) Is(target error) bool { return target == ErrStagingSync }

// The seams ADR-210's syncs go through, so a test can see each one, its order
// against the renames, and make one fail. The platform files set the defaults.
var (
	// SyncTempFn makes one staged temp's data durable as far as the platform's
	// per-file sync goes (a plain fsync on macOS, where File.Sync drains the device).
	SyncTempFn = platformSyncTemp
	// drainFn is the one drain per device on a platform whose per-file sync is not
	// one (macOS: F_FULLFSYNC on a handle of a file already synced).
	drainFn = platformDrain
	// SyncDirFn syncs one open directory. Exported, with SyncsDirs, for the tests
	// of cmd/mrw and internal/mcp, which must make the sync after landing fail on
	// every platform, Windows included. An export_test.go cannot serve them: a
	// package's _test files are compiled only into its own tests, so another
	// package's test never sees them; an environment variable would put a test
	// switch in the shipped binary. A value only a test sets, behind a name that
	// says so, is the smaller surface.
	SyncDirFn = platformSyncDir
	// SyncsDirs is whether directories are synced at all (every platform, since
	// ADR-210's Windows amendment).
	SyncsDirs = syncsDirs
	// deviceOfFn names the device a file is on, so the drain runs once per device.
	deviceOfFn = platformDevice
)

// shown spells p relative to the root, with slashes, for a message.
func (t *tree) shown(p string) string {
	if r, err := t.rel(p); err == nil {
		return filepath.ToSlash(r)
	}
	return p
}

// openDir opens the directory p beneath the root for a sync or a drain, through
// the held root, without blocking and only if it is a directory: a path swapped
// for a FIFO after the renames must not hang the write under its lock, and one
// swapped for a regular file must not be synced as if it were the directory.
func (t *tree) openDir(p string) (*os.File, error) {
	r, err := t.rel(p)
	if err != nil {
		return nil, err
	}
	f, err := t.r.OpenFile(r, os.O_RDONLY|dirOpenFlags, 0)
	if err != nil {
		return nil, err
	}
	if fi, serr := f.Stat(); serr != nil || !fi.IsDir() {
		_ = f.Close()
		if serr == nil {
			serr = errors.New("not a directory")
		}
		return nil, serr
	}
	return f, nil
}

// ErrNoBarrier is what a drain or a directory flush answers when the volume gives
// no device barrier or does not support the flush (macOS: F_FULLFSYNC answered
// ENOTSUP, then a plain fsync of the same directory succeeded; Windows:
// ERROR_INVALID_FUNCTION from FlushFileBuffers on a directory, as over SMB). It is
// not a failure: the write proceeds with the weaker guarantee and the receipt says
// so (SyncNote). Exported with the other seams, for the tests of other packages.
var ErrNoBarrier = errors.New("this volume gives no device barrier")

// NoBarrierNote is the receipt's SyncNote.
const NoBarrierNote = "this volume gives no device barrier for the sync (the drain or the directory flush is not supported here): the files are written and synced as far as the volume allows, but a power loss can still lose them"

// drainDevices runs drainFn once on one of paths for each device they are on. The
// paths are directories already synced or holding files already synced; a drain
// is a barrier for what was synced before it on the same device, whichever
// handle of that device asks. EVERY device is attempted: the first failure is
// returned with its path, and the others are still drained, so one device's EIO
// does not leave another's data unbarriered. A device whose volume gives no
// barrier (ErrNoBarrier) is not a failure: noBarrier says so.
func drainDevices(tr *tree, paths []string) (firstPath string, noBarrier bool, first error) {
	if !drains {
		return "", false, nil
	}
	done := map[uint64]bool{}
	note := func(p string, err error) {
		if first == nil {
			firstPath, first = p, err
		}
	}
	for _, p := range paths {
		f, err := tr.openDir(p)
		if err != nil {
			note(p, err)
			continue
		}
		fi, err := f.Stat()
		if err != nil {
			_ = f.Close()
			note(p, err)
			continue
		}
		dev := deviceOfFn(fi)
		if done[dev] {
			_ = f.Close()
			continue
		}
		done[dev] = true
		err = drainFn(f)
		_ = f.Close()
		switch {
		case errors.Is(err, ErrNoBarrier):
			noBarrier = true
		case err != nil:
			note(p, err)
		}
	}
	return firstPath, noBarrier, first
}

// syncStaged drains every device the staged temps are on, once, after each temp
// was synced through the handle it was written with and before the first rename.
// The drain asks through each temp's DIRECTORY: a temp takes its target's mode,
// which may leave the owner unable to open it for reading (0244), and a drain is
// the device's barrier whichever handle issues it.
func syncStaged(tr *tree, staged []staged) (string, bool, error) {
	seenDir := map[string]bool{}
	var dirs []string
	for _, sf := range staged {
		if sf.tmp == "" {
			continue
		}
		if d := filepath.Dir(sf.tmp); !seenDir[d] {
			seenDir[d] = true
			dirs = append(dirs, d)
		}
	}
	return drainDevices(tr, dirs)
}

// dirRef is a directory whose entries the write changes: the path staging or the
// commit resolved for it, and its identity at that moment. The sync opens THAT
// path and refuses what it finds if it is no longer that directory, so a link
// retargeted, or a directory replaced by a file or a FIFO, after the renames
// cannot send the sync to another object.
type dirRef struct {
	path string
	id   os.FileInfo // nil when the directory could not be examined
}

// recordDirs names every directory the write is about to change, each once and
// sorted by path: a content target's directory, a directory staging made and its
// parent, a path op's source directory and its rename's destination directory.
// It runs after staging and before the first rename, when every one of them
// exists and every path is already resolved through links; one directory under
// two spellings (d and D on a case-insensitive filesystem) is recorded once,
// by identity.
func recordDirs(tr *tree, staged []staged, pathOps []pending) []dirRef {
	set := map[string]bool{}
	for _, sf := range staged {
		if sf.target != "" {
			set[filepath.Dir(sf.target)] = true
		}
		for _, d := range sf.dirs {
			set[d] = true
			set[filepath.Dir(d)] = true
		}
	}
	for _, w := range pathOps {
		set[filepath.Dir(resolvedAt(w.full))] = true
		if w.renameTo != "" {
			set[filepath.Dir(w.renameTo)] = true
		}
	}
	paths := make([]string, 0, len(set))
	for d := range set {
		paths = append(paths, d)
	}
	sort.Strings(paths)
	var refs []dirRef
	for _, d := range paths {
		fi, err := tr.stat(d)
		if err != nil {
			fi = nil
		}
		dup := false
		for _, r := range refs {
			dup = dup || (fi != nil && r.id != nil && os.SameFile(r.id, fi))
		}
		if !dup {
			refs = append(refs, dirRef{path: d, id: fi})
		}
	}
	return refs
}

// beforeDirSync runs when the directory syncs begin, after the last rename: a
// seam for the tests that swap a directory for something else at that moment.
var beforeDirSync = func() {}

// syncChanged syncs each recorded directory once, after the last change, then
// drains once per device. A directory that fails does not stop the others: each
// is tried, the drain still covers those that synced, and the first failure is
// the *SyncError returned, naming the path. A directory that cannot be opened, is
// no longer a directory, or is no longer the one recorded is such a failure
// (Decision 4: a name that cannot be made stable is not quietly passed over). A
// directory the volume cannot flush (ErrNoBarrier) is not: it is skipped, and
// noBarrier says so.
func syncChanged(tr *tree, refs []dirRef) (noBarrier bool, first error) {
	if !SyncsDirs {
		return false, nil
	}
	beforeDirSync()
	fail := func(d string, err error) {
		if first == nil {
			first = &SyncError{Path: tr.shown(d), Err: err}
		}
	}
	var synced []string
	for _, r := range refs {
		f, err := tr.openDirSync(r.path)
		if err != nil {
			fail(r.path, err)
			continue
		}
		if fi, serr := f.Stat(); serr != nil {
			_ = f.Close()
			fail(r.path, serr)
			continue
		} else if r.id != nil && !os.SameFile(r.id, fi) {
			_ = f.Close()
			fail(r.path, errors.New("the directory is not the one the write changed: it was replaced after the renames"))
			continue
		}
		err = SyncDirFn(f)
		_ = f.Close()
		switch {
		case errors.Is(err, ErrNoBarrier):
			noBarrier = true
		case err != nil:
			fail(r.path, err)
			continue
		}
		synced = append(synced, r.path)
	}
	p, nb, err := drainDevices(tr, synced)
	if err != nil {
		fail(p, err)
	}
	return noBarrier || nb, first
}
