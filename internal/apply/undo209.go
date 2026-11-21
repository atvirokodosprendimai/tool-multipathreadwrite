package apply

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"unicode"
)

// contentFile is one content file the commit renamed into place, with what its
// undo needs (ADR-209): the staged temp as it was when renamed, which the path
// must still match for the undo to remove it, and the aside holding the previous
// version when there is one.
type contentFile struct {
	path     string      // the plan's path
	target   string      // the resolved path the rename went onto
	created  bool        // a create: nothing to put back, only to remove
	aside    string      // the hard link to the previous version, "" when none could be made
	asideErr error       // why there is none, for a file that existed
	tmp      fs.FileInfo // the staged temp when it was renamed in, its file ID loaded
	dirs     []string    // directories staging made for it
	listed   string      // the target as the directory lists it, to link the aside back under the spelling it had
	tmpErr   error       // why tmp is nil
	seen     fs.FileInfo // the file as the plan loaded it, which the aside must still be
}

// asideIs says whether the entry at the aside's name is still the file the plan loaded
// (os.SameFile with the snapshot taken then), the one instant between the check made
// before the first rename and the link back or the removal that the commit cannot
// close: another process may put another entry at a name in the shared directory. A
// file with no snapshot cannot be checked and is taken as it is.
func asideIs(tr *tree, aside string, seen fs.FileInfo) bool {
	if aside == "" || seen == nil {
		return true
	}
	ai, err := tr.lstat(aside)
	return err == nil && sameFileFn(seen, ai)
}

// removeAside removes an aside that is still the file loaded; any other entry at that
// name is not mrw's to remove, and is named.
func removeAside(tr *tree, res *Result, aside string, seen fs.FileInfo) {
	if aside == "" {
		return
	}
	if !asideIs(tr, aside, seen) {
		res.leave(tr, aside)
		return
	}
	res.cleanUp(tr, aside)
}

// restoreTo is the path the aside is linked back to: the spelling the directory listed
// before the commit. On a volume that folds case a plan may name a.txt as A.TXT, and
// the restored file must keep the name it had.
func (c contentFile) restoreTo() string {
	if c.listed != "" {
		return c.listed
	}
	return c.target
}

// listedName is target spelled as its directory lists the file, and whether that is
// known. The entry is chosen by identity: the one name of the directory that is the
// same file as target (os.SameFile), every name compared, so another name of the
// file is never skipped. The plan's own spelling, when the directory lists it, is the
// answer without looking further; when several other names are the same file and none
// is the plan's, the answer is not known and false is returned, so no aside is made
// and the file is reported as one the undo cannot put back (nothing is guessed).
//
// The directory is listed only when nothing cheaper answers, because a rooted listing
// stats every entry (Go 1.27.4, newUnixDirent): none is made on Windows, where staging
// resolves the path through EvalSymlinks, which returns the stored case (sameSpelling,
// apply.go), so the staged spelling is the listed one; none for a name with no letter
// to fold, which is its own case fold; none when the same name with its case reversed
// finds nothing (a volume that keeps case apart lists the name the path spells,
// whether the name is ASCII or not); and on darwin none where the open descriptor's
// path names the plan's spelling, or the file has one name and that name differs from
// the plan's only in case. What is left is a volume that folds case and a file with
// several names, or a name with no ASCII letter on such a volume. cache holds each
// directory's entries, listed once.
func listedName(tr *tree, target string, cache map[string][]fs.DirEntry) (string, bool) {
	if runtime.GOOS == "windows" {
		return target, true
	}
	dir, base := filepath.Dir(target), filepath.Base(target)
	fi, err := tr.lstat(target)
	if err != nil {
		return target, true
	}
	if swapped := swapCase(base); swapped != base {
		if si, err := tr.lstat(filepath.Join(dir, swapped)); err != nil || !os.SameFile(fi, si) {
			return target, true
		}
	} else if isASCII(base) {
		return target, true // no letter to fold: the name is its own case fold
	}
	if stored, single, ok := spelledByDescriptorFn(tr, target); ok {
		switch {
		case stored == base:
			return target, true
		case single && strings.EqualFold(stored, base):
			return filepath.Join(dir, stored), true
		}
	}
	ents, ok := cache[dir]
	if !ok {
		ents, _ = tr.readDir(dir)
		cache[dir] = ents
	}
	for _, e := range ents {
		if e.Name() == base {
			return target, true
		}
	}
	var same []string
	for _, e := range ents {
		if info, err := e.Info(); err == nil && os.SameFile(fi, info) {
			same = append(same, e.Name())
		}
	}
	switch len(same) {
	case 0:
		return target, true
	case 1:
		return filepath.Join(dir, same[0]), true
	}
	return "", false
}

// spelledByDescriptorFn is the seam of the descriptor's path, which a test turns off to
// reach the listing.
var spelledByDescriptorFn = spelledByDescriptor

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// swapCase is s with the case of each letter reversed, which a volume that folds
// case reads as the same name.
func swapCase(s string) string {
	r := []rune(s)
	for i, c := range r {
		switch {
		case unicode.IsUpper(c):
			r[i] = unicode.ToLower(c)
		case unicode.IsLower(c):
			r[i] = unicode.ToUpper(c)
		}
	}
	return string(r)
}

// makeAside links target, which exists, under a fresh .mrw-aside-* name in its
// own directory and returns the name. A link never replaces: a taken name is
// another name. Whatever else refuses the link is returned, and the file then
// commits as before, with nothing to put back (ADR-209 Decision 2).
func makeAside(tr *tree, target string) (string, error) {
	var err error
	for range 100 {
		var b [8]byte
		if _, rerr := rand.Read(b[:]); rerr != nil {
			return "", rerr
		}
		name := filepath.Join(filepath.Dir(target), ".mrw-aside-"+hex.EncodeToString(b[:]))
		if err = asideLinkFn(tr, target, name); err == nil {
			return name, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	return "", err
}

// undoContent puts the content files a failed commit renamed in back, newest
// first (ADR-209 Decisions 3 and 4). A path is removed only when it is still the
// regular file mrw renamed there, by identity, size and modification time; a file
// that existed is then linked back from its aside, which never replaces, and the
// aside removed. What could not be put back stays written, is marked in its
// record, and is named in the text this returns, which is empty when the tree is
// back. The records of the files put back are dropped from res.Files.
func undoContent(tr *tree, res *Result, done []contentFile) string {
	var left []string
	drop := map[int]bool{}
	var dirs []string
	mark := func(c contentFile, why string) {
		for i := range res.Files {
			if res.Files[i].Path == c.path && !res.Files[i].Removed {
				res.Files[i].NotHeld, res.Files[i].NotHeldWhy = true, why
			}
		}
	}
	for i := len(done) - 1; i >= 0; i-- {
		c := done[i]
		name := asPlan(c.path)
		if c.tmp == nil {
			// The temp's identity could not be taken before the rename: the undo cannot tell
			// the file it renamed in from another, so it leaves the path, and says so. The
			// path is not marked: nothing says it holds anything but mrw's bytes.
			left = append(left, fmt.Sprintf("%s: the file renamed in could not be identified (%v), so it is left as it is", name, c.tmpErr))
			if c.aside != "" {
				res.noteIfLeft(tr, c.aside)
			}
			continue
		}
		cur, lerr := tr.lstat(c.target)
		held := lerr == nil && cur.Mode().IsRegular() && entryDrift(c.tmp, cur) == ""
		switch {
		case !c.created && c.aside == "":
			left = append(left, fmt.Sprintf("%s was written with no aside to put it back from (%v)", name, c.asideErr))
			if !held {
				mark(c, fmt.Sprintf("%s changed after mrw wrote it", name))
			}
			continue
		case !held:
			why := fmt.Sprintf("%s is missing or changed since mrw wrote it", name)
			switch {
			case c.created:
				left = append(left, fmt.Sprintf("%s was created by mrw and changed since: left as it is", name))
			case errors.Is(lerr, fs.ErrNotExist):
				left = append(left, fmt.Sprintf("%s is no longer at its path (a rename of this plan moved it, or another process did): its previous version is kept in %s", name, res.rel(c.aside)))
				res.noteIfLeft(tr, c.aside)
			default:
				left = append(left, fmt.Sprintf("%s changed since mrw wrote it: its previous version is kept in %s", name, res.rel(c.aside)))
				res.noteIfLeft(tr, c.aside)
			}
			mark(c, why)
			continue
		}
		v, err := removeContentFn(tr, c.target, c.tmp)
		if err != nil {
			switch {
			case errors.Is(err, fs.ErrNotExist):
				// It vanished between the look and the move: there is nothing to move, and
				// the path is free for the aside, or the dirs of a created file are taken back.
			case errors.Is(err, errNotTheFile):
				// Another entry stood there by then, and was put back untouched, or could not
				// be and sits in a private directory, which is named whatever it holds.
				if c.created {
					left = append(left, fmt.Sprintf("%s was created by mrw and changed since: left as it is", name))
				} else {
					left = append(left, fmt.Sprintf("%s changed since mrw wrote it: its previous version is kept in %s", name, res.rel(c.aside)))
				}
				res.noteExtraName(tr, err)
				if c.aside != "" {
					res.noteIfLeft(tr, c.aside)
				}
				mark(c, fmt.Sprintf("%s is missing or changed since mrw wrote it", name))
				continue
			default:
				left = append(left, fmt.Sprintf("could not remove %s: %v", name, err))
				res.noteExtraName(tr, err)
				if c.aside != "" {
					res.noteIfLeft(tr, c.aside)
				}
				continue
			}
		}
		// The path is vacant. The previous version goes back first; the staged file in its
		// private directory is deleted last, so a deletion that fails is a named leftover
		// and not an incomplete undo. Where the previous version cannot go back, the staged
		// file is moved back to the path, so a file stands there and the receipt says which.
		restored, keepPrivate := true, false
		putBack := func() bool {
			if v == nil {
				return false
			}
			if err := tr.noReplace().rename(v.entry, c.target); err != nil {
				keepPrivate = true
				res.leave(tr, v.dir)
				return false
			}
			if err := tr.remove(v.dir); err != nil {
				res.leave(tr, v.dir)
			}
			v = nil
			return true
		}
		switch {
		case c.created:
			dirs = append(dirs, c.dirs...)
		case !asideIs(tr, c.aside, c.seen):
			// Another entry stands at the aside's name now: it is not linked back, not
			// removed, and named; the file mrw wrote stays at the path.
			restored = false
			left = append(left, fmt.Sprintf("%s: %s is not the file mrw loaded (another entry stands at that name), so it is left as it is and the file mrw wrote stays at the path", name, res.rel(c.aside)))
			res.leave(tr, c.aside)
			if !putBack() {
				mark(c, fmt.Sprintf("%s was removed and its aside is not its previous version", name))
			}
		default:
			if err := asideLinkFn(tr, c.aside, c.restoreTo()); err != nil {
				restored = false
				left = append(left, fmt.Sprintf("could not put %s back from %s: %v", name, res.rel(c.aside), err))
				res.noteIfLeft(tr, c.aside)
				if errors.Is(err, fs.ErrExist) || !putBack() {
					mark(c, fmt.Sprintf("%s was removed and could not be put back: %v", name, err))
				}
			} else {
				removeAside(tr, res, c.aside, c.seen)
			}
		}
		if v != nil && !keepPrivate {
			if derr := discardVacatedFn(tr, v); derr != nil {
				res.leave(tr, v.dir)
			}
		}
		if !restored {
			continue
		}
		for j := range res.Files {
			if res.Files[j].Path == c.path && !res.Files[j].Removed {
				drop[j] = true
				break
			}
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range slices.Compact(dirs) {
		res.cleanUp(tr, d)
	}
	kept := make([]FileResult, 0, len(res.Files))
	for i, f := range res.Files {
		if !drop[i] {
			kept = append(kept, f)
		}
	}
	res.Files = kept
	return strings.Join(left, "; ")
}
