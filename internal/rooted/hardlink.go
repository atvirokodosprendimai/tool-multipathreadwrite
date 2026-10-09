package rooted

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-134. ADR-077 compared a served path, and each ancestor, with the state
// base, and never the served FILE with the files inside it: a hard link in the
// root to the ledger was read whole and matched by --grep. A file is mrw's
// state if it is the same file as one under the base, whatever it is called.

// fileKey is a file's identity: the volume and the file on it.
type fileKey struct{ vol, idx uint64 }

// errHardLinked is the refusal for a second name of a state file.
func errHardLinked(path string) error {
	return fmt.Errorf("%s is a hard link to a file in mrw's own state directory; mrw does not serve or edit its own ledger", path)
}

// errNotComparable is the refusal when a file has a second name and the state
// base cannot be examined to the end: refused, since "nothing was found" from a
// comparison that did not finish would serve the ledger by an alias.
func errNotComparable(path string) error {
	return fmt.Errorf("%s has more than one name, and mrw cannot tell whether one is a file in its own state directory; mrw does not serve or edit its own ledger", path)
}

// stateLinks answers whether a file is a second name of a file in THIS
// checkout's state directory — the ledger, the ack store, the locks, the files
// that license a write or hold a checkpoint. The identities of those that HAVE
// a second name are read once, when a candidate first needs them, and kept for
// the life of the value: one walk (Resolver), or one Resolve call. A link made
// to one of them after the read is seen by the next walk, which is the
// staleness ADR-131 accepts for a directory swapped mid-walk; read.Run
// resolves afresh before it serves. The scan reads one directory of a handful
// of files, not the whole base, because read.Run resolves every served path
// afresh and a scan of every checkout's directory made a grep matching 6,000
// hard-linked files take 70 s instead of 0.5 s (the Codex review of #361).
type stateLinks struct {
	root     string      // the checkout, resolved
	b        string      // the state base, resolved
	bi       os.FileInfo // its FileInfo, nil when it could not be examined
	prepared bool
	absent   bool // there is no base: nothing to be a second name of
	bad      bool // the state directory could not be examined to the end
	scanned  bool
	ids      map[fileKey]struct{} // state files with a second name
}

// prepare settles, once, whether there is a base at all.
func (s *stateLinks) prepare() {
	if s.prepared {
		return
	}
	s.prepared = true
	if s.bi != nil {
		return
	}
	_, err := os.Stat(s.b)
	s.absent = errors.Is(err, fs.ErrNotExist)
	s.bad = !s.absent // present but not examined, or not there and then is
}

// scan reads the identity of every file in this checkout's state directory
// whose own link count is above one: only those can have a second name. A name
// that vanishes while the directory is listed (a ledger saved by rename) is
// passed over; any other failure makes the comparison incomplete. A checkout
// with no state directory has no state file to be a second name of.
func (s *stateLinks) scan() {
	s.scanned = true
	s.ids = map[fileKey]struct{}{}
	dir, err := state.DirPath(s.root)
	if err != nil {
		s.bad = true
		return
	}
	// The directory may itself be a link (a moved state directory, a Windows
	// junction): WalkDir lists a root that is one as a single non-file entry,
	// and an empty listing would read as "no state file" (the Codex review of
	// #361, second pass). It is listed where it really is.
	dir = Real(dir)
	if _, err := os.Stat(dir); err != nil {
		s.bad = !errors.Is(err, fs.ErrNotExist)
		return
	}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			err = s.noteLink(p)
		case d.Type().IsRegular():
			var fi os.FileInfo
			if fi, err = d.Info(); err == nil {
				err = s.note(fi, p)
			}
		}
		if errors.Is(err, fs.ErrNotExist) {
			return nil // a name that vanished while the directory was listed
		}
		return err
	})
	s.bad = s.bad || err != nil
}

// note keeps the identity of fi, the regular file at p, when it has a second
// name.
func (s *stateLinks) note(fi os.FileInfo, p string) error {
	key, n, err := identity(fi, p)
	if err == nil && n > 1 {
		s.ids[key] = struct{}{}
	}
	return err
}

// noteLink does the same for the file a link in the state directory leads to:
// the ledger moved elsewhere and left a link behind is still the ledger, and
// the Codex review of #361 (third pass) served it by an alias. A link to a
// directory is not examined, which refuses a file with a second name; a link
// that leads nowhere has nothing to be a name of.
func (s *stateLinks) noteLink(p string) error {
	t := Real(p)
	fi, err := os.Lstat(t)
	switch {
	case err != nil:
		return err
	case fi.Mode().IsRegular():
		return s.note(fi, t)
	case fi.IsDir():
		return fmt.Errorf("%s is a link to a directory", p)
	}
	return nil
}

// refusal judges fi, the regular file at path (shown to the caller as shown).
// A file with one name is never compared, which is nearly every file: a pnpm
// store or a build cache is full of link counts above one, and only identity
// with a state file refuses them.
func (s *stateLinks) refusal(fi os.FileInfo, path, shown string) error {
	s.prepare()
	if s.absent {
		return nil
	}
	key, n, err := identity(fi, path)
	switch {
	case err != nil:
		return errNotComparable(shown)
	case n <= 1:
		return nil
	}
	if !s.scanned {
		s.scan()
	}
	if s.bad {
		return errNotComparable(shown)
	}
	if _, hit := s.ids[key]; hit {
		return errHardLinked(shown)
	}
	return nil
}

// hardLinkRefusal is resolveIn's judgement of q, a path already resolved as
// far as it exists: nil unless it is a regular file that is a second name of a
// state file.
func hardLinkRefusal(root, q, shown string) error {
	base, err := state.Base()
	if err != nil {
		return nil //nolint:nilerr // no state home: inState answers false the same way, and there is no state to be a name of
	}
	b, bi := resolvedBase(base)
	s := stateLinks{root: root, b: b, bi: bi}
	s.prepare()
	if s.absent {
		return nil
	}
	fi, err := os.Lstat(q)
	if err != nil || !fi.Mode().IsRegular() {
		return nil //nolint:nilerr // a path that cannot be examined, or is not a regular file, is not a candidate; the caller opens it and says why not
	}
	return s.refusal(fi, q, shown)
}
