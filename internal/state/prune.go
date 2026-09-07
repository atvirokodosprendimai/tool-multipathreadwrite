package state

import (
	"os"
	"path/filepath"
	"strings"
)

// Entry is one per-root state directory under the state base, described well
// enough for a caller to decide what to do about it — and to SAY what it did,
// which ADR-008 requires of anything mrw removes.
type Entry struct {
	// Dir is the absolute path of the state directory.
	Dir string
	// Root is the checkout its `root` marker names, or empty when the marker
	// could not be read or did not name an absolute path.
	Root string
	// Identified is whether Root came from a marker mrw could believe. An
	// entry that is not identified is never removed: unknown provenance is the
	// one mistake here that re-reading a file cannot undo.
	Identified bool
	// Live is whether Root still resolves to a directory. Meaningless, and
	// always false, when Identified is false.
	Live bool
	// Bytes is the size of the files this entry holds.
	Bytes int64
	// Err is why a removal failed, on an entry Prune tried to remove. One
	// unremovable directory does not strand the rest, so the failure travels
	// with the entry rather than aborting the run.
	Err error
}

// Entries lists every per-root state directory under the state base.
//
// It judges nothing and removes nothing: it is what `mrw seen` counts and what
// Prune selects from. A base that does not exist yet is not an error — it is
// what a machine that has never run mrw looks like.
func Entries() ([]Entry, error) {
	dir, err := entriesRoot()
	if err != nil {
		return nil, err
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		if isNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]Entry, 0, len(items))
	for _, it := range items {
		// ⚠ os.DirEntry.IsDir REPORTS THE LINK, NOT ITS TARGET, which is
		// exactly what is wanted: a symlink planted under the base has
		// ModeSymlink and IsDir false, so it is skipped rather than followed
		// out of the directory mrw owns. A plain file is skipped for the same
		// reason. Using os.Stat here instead would follow the link and put a
		// directory nobody gave mrw inside the prune's reach.
		if !it.IsDir() {
			continue
		}
		out = append(out, describe(filepath.Join(dir, it.Name())))
	}
	return out, nil
}

// Prune removes the entries whose `root` marker names a path that is no longer
// a directory, and returns the ones it removed — or, when dryRun is true, the
// ones it WOULD remove, having touched nothing. The two lists are the same by
// construction, which is what makes a dry run a preview rather than a different
// question.
//
// It selects from the entries it is GIVEN rather than walking again, so a
// caller that also wants to report what was KEPT — the unidentifiable ones,
// the live ones — pays for one walk instead of two. On a base holding tens of
// thousands of entries that is a difference the caller notices.
//
// root is the checkout the caller is working in. Its own entry is never
// removed, even when that checkout has been deleted underneath the process: a
// tool pruning the state of the run in progress is not cleaning up, it is
// breaking the run.
//
// ⚠ NOTHING CALLS THIS ON ITS OWN. ADR-004 deferred a prune and gave the reason
// a reaper would be wrong — "deciding a directory is dead means deciding a path
// will never come back, and a tool should not decide that". An absent root is
// also a volume that is not mounted, and mrw cannot tell those apart. So this
// runs only when an operator asks for it, and the operator is the one who can.
func Prune(root string, entries []Entry, dryRun bool) ([]Entry, error) {
	self, err := selfDir(root)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	var removed []Entry
	for _, e := range entries {
		if e.Dir == self || !e.Identified || e.Live {
			continue
		}
		if !dryRun {
			if err := os.RemoveAll(e.Dir); err != nil {
				e.Err = err
			}
		}
		removed = append(removed, e)
	}
	return removed, nil
}

// entriesRoot is the directory holding one entry per checkout.
func entriesRoot() (string, error) {
	base, err := stateHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "mrw"), nil
}

// selfDir is where root's state lives, WITHOUT creating it. Dir makes the
// directory as a side effect, which a prune must not do while deciding what to
// keep — it would bring an entry into existence in order to spare it.
func selfDir(root string) (string, error) {
	dir, err := entriesRoot()
	if err != nil {
		return "", err
	}
	abs, err := absReal(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, key(abs)), nil
}

// describe reads one entry's marker and measures it.
func describe(dir string) Entry {
	e := Entry{Dir: dir, Bytes: sizeOf(dir)}

	b, err := os.ReadFile(filepath.Join(dir, "root"))
	if err != nil {
		return e // no marker, or unreadable: provenance unknown, so keep
	}
	path := strings.TrimSpace(string(b))
	// ⚠ filepath.IsAbs IS THE RIGHT QUESTION HERE, unlike in the root guards
	// where rooted.IsRooted is (a Windows path such as `\etc` is rooted and not
	// absolute). This asks something narrower: did MRW write this marker?
	// absReal runs filepath.Abs, so every marker mrw has ever written is
	// absolute. Anything else came from somewhere mrw does not know about, and
	// a wrong answer here can only be "keep", which is the safe direction.
	if path == "" || !filepath.IsAbs(path) {
		return e
	}
	e.Root, e.Identified = path, true
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		e.Live = true
	}
	return e
}

// sizeOf sums the files directly inside one entry. Entries are flat — a ledger,
// an iteration marker, a tally, the root marker — so this does not recurse, and
// a size it cannot read is reported as the zero it can prove.
func sizeOf(dir string) int64 {
	items, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var n int64
	for _, it := range items {
		info, err := it.Info()
		if err != nil || info.IsDir() {
			continue
		}
		n += info.Size()
	}
	return n
}
