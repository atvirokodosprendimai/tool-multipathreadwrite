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
	// SyncsDirs is whether directories are synced at all: not on Windows.
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

// openDir opens the directory p beneath the root for a sync.
func (t *tree) openDir(p string) (*os.File, error) {
	r, err := t.rel(p)
	if err != nil {
		return nil, err
	}
	return t.r.Open(r)
}

// drainDevices runs drainFn once on one of paths for each device they are on. The
// paths are directories already synced or holding files already synced; a drain
// is a barrier for what was synced before it on the same device, whichever
// handle of that device asks. EVERY device is attempted: the first failure is
// returned with its path, and the others are still drained, so one device's EIO
// does not leave another's data unbarriered.
func drainDevices(tr *tree, paths []string) (string, error) {
	if !drains {
		return "", nil
	}
	done := map[uint64]bool{}
	var firstPath string
	var first error
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
		if err != nil {
			note(p, err)
		}
	}
	return firstPath, first
}

// syncStaged drains every device the staged temps are on, once, after each temp
// was synced through the handle it was written with and before the first rename.
// The drain asks through each temp's DIRECTORY: a temp takes its target's mode,
// which may leave the owner unable to open it for reading (0244), and a drain is
// the device's barrier whichever handle issues it.
func syncStaged(tr *tree, staged []staged) (string, error) {
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

// changedDirs is every directory whose entries the write changed, each once,
// sorted: a content target's directory, a directory staging made and its parent,
// a path op's source directory and its rename's destination directory.
func changedDirs(staged []staged, pathOps []pending) []string {
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
	dirs := make([]string, 0, len(set))
	for d := range set {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs
}

// syncChanged syncs each changed directory once, after the last change, then
// drains once per device. Outside the platforms that sync directories it does
// nothing (ADR-210 Decision 3). A directory that fails does not stop the others:
// each is tried, the drain still covers those that synced, and the first failure
// is the *SyncError returned, naming the path. A changed directory that cannot be
// opened is such a failure (Decision 4: a name that cannot be made stable is not
// quietly passed over).
func syncChanged(tr *tree, dirs []string) error {
	if !SyncsDirs {
		return nil
	}
	var first error
	fail := func(d string, err error) {
		if first == nil {
			first = &SyncError{Path: tr.shown(d), Err: err}
		}
	}
	var synced []string
	var opened []os.FileInfo
	for _, d := range dirs {
		f, err := tr.openDir(d)
		if err != nil {
			fail(d, err)
			continue
		}
		// One directory under two spellings (d and D on a case-insensitive
		// filesystem) is synced once: identity, not the spelling, decides.
		if fi, serr := f.Stat(); serr == nil {
			dup := false
			for _, o := range opened {
				dup = dup || os.SameFile(o, fi)
			}
			if dup {
				_ = f.Close()
				continue
			}
			opened = append(opened, fi)
		}
		err = SyncDirFn(f)
		_ = f.Close()
		if err != nil {
			fail(d, err)
			continue
		}
		synced = append(synced, d)
	}
	if p, err := drainDevices(tr, synced); err != nil {
		fail(p, err)
	}
	return first
}
