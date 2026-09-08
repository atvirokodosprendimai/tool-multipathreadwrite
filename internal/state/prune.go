package state

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Entry is one per-root state directory under the state base, described well
// enough for a caller to decide what to do about it — and to SAY what it did,
// which ADR-008 requires of anything mrw removes.
type Entry struct {
	// Dir is the absolute path of the state directory. It is what a caller
	// PRINTS; it is never what a removal is addressed by, because a path is
	// re-resolved by whoever walks it and the walk that found this entry has
	// already finished.
	Dir string
	// Name is the single path component under the state base. It is what a
	// removal is addressed by, through the handle the base was enumerated on.
	Name string
	// Root is the checkout its `root` marker names, or empty when the marker
	// could not be read or did not name an absolute path.
	Root string
	// Identified is whether Root came from a marker mrw could believe. An
	// entry that is not identified is never removed: unknown provenance is the
	// one mistake here that re-reading a file cannot undo.
	Identified bool
	// Dead is whether Root is known NOT to be there any more — it does not
	// exist, or it exists and is not a directory. Anything else is
	// INDETERMINATE and leaves this false, which is why the field is spelled
	// this way round: the zero value is "keep". A permission error, an
	// unmounted volume or a network timeout are all answers about the stat,
	// not about the checkout, and a prune that reads them as "gone" deletes
	// state for a checkout that is still there.
	Dead bool
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
	base, dir, err := openBase()
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, nil // no base yet
	}
	defer base.Close()

	names, err := children(base)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(names))
	for _, name := range names {
		out = append(out, describe(base, dir, name))
	}
	return out, nil
}

// Prune removes the entries whose `root` marker names a path that is no longer
// a directory, and returns the ones it removed — or, when dryRun is true, the
// ones it WOULD remove, having touched nothing. The two lists are the same by
// construction, which is what makes a dry run a preview rather than a different
// question.
//
// ⚠ IT RE-OPENS THE BASE AND RE-ENUMERATES IT, and removes only a child NAME it
// can still see under that handle. The entries it is given describe what a
// previous walk found; they are a filter, never an address. Removing by the
// absolute path in Entry.Dir would re-resolve every component after the walk
// that produced it, so a base — or a component above it — swapped for a symlink
// in between would carry os.RemoveAll straight out of the directory mrw owns.
// A handle cannot be re-pointed that way: it names the object, not the route to
// it.
//
// The cost is one extra walk on the removing path, and the alternative was
// handing the caller a live *os.Root to pass back in. That widens the API to
// save a directory read, and this record already carries a finding about giving
// a boundary away by widening a signature.
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
	self, err := selfNames(root)
	if err != nil {
		return nil, err
	}

	base, _, err := openBase()
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, nil
	}
	defer base.Close()

	// What is under the base NOW, not what the caller's walk once saw.
	present := map[string]bool{}
	names, err := children(base)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		present[name] = true
	}

	var removed []Entry
	for _, e := range entries {
		if !e.Identified || !e.Dead || self[e.Name] {
			continue
		}
		// ⚠ ONLY WHAT THIS WALK JUST SAW. The entries are a filter, not an
		// address: the caller's slice may name something that has since gone,
		// and os.Root.RemoveAll succeeds silently on a name that is not there.
		// Reporting that as a removal would break ADR-008 — a delete says what
		// it removed, and nothing was removed.
		//
		// ⚠ There is deliberately NO separate "is this a single component"
		// check. It would be unkillable: `present` is built from this walk, so
		// a crafted name cannot be in it, and os.Root refuses a traversal on
		// its own ("path escapes from parent"). A guard no mutant can kill is
		// this repository's named defect, so containment is left where it is
		// actually enforced and this clause carries only what it can prove.
		if !present[e.Name] {
			continue
		}
		if !dryRun {
			if err := base.RemoveAll(e.Name); err != nil {
				e.Err = err
			}
		}
		removed = append(removed, e)
	}
	return removed, nil
}

// openBase opens the state base as a HANDLE, refusing one that is a symlink.
//
// The order is the whole point. Checking the path and then opening it leaves a
// window in which `<state>/mrw` is replaced between the two calls, which moves
// the race rather than closing it. So the parent is opened first, the child is
// interrogated THROUGH that handle, and the base is opened through it too:
// every step after the first names an object rather than a path, and the thing
// that was checked is the thing that was opened.
//
// It returns a nil handle and no error when there is nothing there yet, which
// is what a machine that has never run mrw looks like. The second return is the
// base's path, for entries to report themselves by.
func openBase() (*os.Root, string, error) {
	home, err := stateHome()
	if err != nil {
		return nil, "", err
	}
	dir := filepath.Join(home, "mrw")

	parent, err := os.OpenRoot(home)
	if err != nil {
		if isNotExist(err) {
			return nil, dir, nil
		}
		return nil, "", err
	}
	defer parent.Close()

	fi, err := parent.Lstat("mrw")
	if err != nil {
		if isNotExist(err) {
			return nil, dir, nil
		}
		return nil, "", err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, "", fmt.Errorf("the state base %s is a symlink; mrw will not enumerate or remove "+
			"through it, because it did not create it and cannot tell what is on the other side", dir)
	}
	if !fi.IsDir() {
		return nil, "", fmt.Errorf("the state base %s is not a directory", dir)
	}

	base, err := parent.OpenRoot("mrw")
	if err != nil {
		if isNotExist(err) {
			return nil, dir, nil
		}
		return nil, "", err
	}
	return base, dir, nil
}

// children is the entry names directly under the base: real directories only.
//
// ⚠ Lstat, NOT Stat, and it is the same reason the old os.DirEntry.IsDir was
// right: a symlink planted under the base has ModeSymlink and must be skipped
// rather than followed out of the directory mrw owns. A plain file is skipped
// for the same reason.
func children(base *os.Root) ([]string, error) {
	f, err := base.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	items, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		fi, err := base.Lstat(it.Name())
		if err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			continue
		}
		out = append(out, it.Name())
	}
	return out, nil
}

// selfNames is every entry name the caller's own root could be keyed under.
//
// ⚠ THREE SPELLINGS, BECAUSE absReal GIVES A DIFFERENT ANSWER ONCE THE ROOT IS
// GONE. It runs EvalSymlinks and falls back to filepath.Abs when that fails, so
// a checkout reached through a symlinked component — /var on macOS, every day —
// is keyed under its RESOLVED path while it exists and under its LITERAL path
// once it does not. Asking only the live spelling means the self guard stops
// matching at exactly the moment it is needed: the run whose checkout was
// deleted underneath it.
//
// ⚠ AND THE TWO OBVIOUS SPELLINGS ARE NOT ENOUGH, which cost a red test to
// learn. Once the leaf is gone, absReal and filepath.Abs return the SAME
// literal path — the resolved one cannot be recovered from either, because
// EvalSymlinks fails on the whole path and gives back nothing. The spelling the
// entry was actually created under is rebuilt from the ancestors that are still
// there: resolve the deepest one that exists, then re-append what was removed.
// A name that is not an entry matches nothing, so extra candidates cost only
// the lookup.
func selfNames(root string) (map[string]bool, error) {
	out := map[string]bool{}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	out[key(abs)] = true
	if real, err := absReal(root); err == nil {
		out[key(real)] = true
	}
	if real, ok := resolveThroughAncestors(abs); ok {
		out[key(real)] = true
	}
	return out, nil
}

// resolveThroughAncestors spells abs the way absReal did while the path still
// existed: EvalSymlinks the deepest ancestor that IS there, then re-append the
// components that are not. It reports false only when nothing on the way up
// resolves, which means there is no symlink to account for and the literal
// candidate already covers it.
func resolveThroughAncestors(abs string) (string, bool) {
	var missing []string
	for cur := abs; ; {
		if real, err := filepath.EvalSymlinks(cur); err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				real = filepath.Join(real, missing[i])
			}
			return real, true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false // reached the volume root and nothing resolved
		}
		missing = append(missing, filepath.Base(cur))
		cur = parent
	}
}

// describe reads one entry's marker and measures it, through the base handle.
func describe(base *os.Root, baseDir, name string) Entry {
	e := Entry{Dir: filepath.Join(baseDir, name), Name: name}

	sub, err := base.OpenRoot(name)
	if err != nil {
		return e
	}
	defer sub.Close()
	e.Bytes = sizeOf(sub)

	b, err := readAll(sub, "root")
	if err != nil {
		return e // no marker, or unreadable: provenance unknown, so keep
	}
	// ⚠ EXACTLY THE ONE NEWLINE Dir WROTE, never TrimSpace. A checkout whose
	// directory name ends in a space is legal, and trimming it yields a path
	// that is not the one the marker recorded — which then usually does not
	// exist, so a live checkout is classified dead and its state removed.
	path := strings.TrimSuffix(string(b), "\n")
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
	e.Dead = isDead(path)
	return e
}

// isDead is whether a marker's path is known not to be a directory any more.
//
// ⚠ ONLY ErrNotExist ANSWERS THE QUESTION. Every other stat error describes the
// LOOKUP, not the checkout: a denied parent, an unmounted volume, a network
// filesystem that timed out. Reading those as "gone" is how a prune removes the
// state of a checkout that is still there, and the entry is unrecoverable
// afterwards while the wrong answer costs nothing to repeat.
func isDead(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return errors.Is(err, fs.ErrNotExist)
	}
	return !fi.IsDir()
}

// readAll reads one file inside an entry through its handle.
func readAll(sub *os.Root, name string) ([]byte, error) {
	f, err := sub.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// sizeOf sums the files directly inside one entry. Entries are flat — a ledger,
// an iteration marker, a tally, the root marker — so this does not recurse, and
// a size it cannot read is reported as the zero it can prove.
func sizeOf(sub *os.Root) int64 {
	f, err := sub.Open(".")
	if err != nil {
		return 0
	}
	defer f.Close()
	items, err := f.ReadDir(-1)
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
