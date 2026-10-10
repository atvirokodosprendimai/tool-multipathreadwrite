// Package rooted turns a caller-supplied path into the file it names inside a
// root, and refuses one that leaves.
//
// It exists because the boundary was implemented once, on the write path, and
// therefore held on exactly one of the two ways into the tree: `mrw -C repo
// read ../outside.txt` served the file at exit 0, and a symlink out of the tree
// was followed, while the identical path in a write plan was refused by name.
//
// A boundary that holds on one path and not the other is not a boundary, it is
// a coincidence of which function you happened to call — so there is one
// implementation and both callers use it.
package rooted

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/links"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// Resolve returns the absolute file that path names under root, and an error if
// it resolves outside it.
//
// Symlinks are resolved before the check, because following one out of the tree
// is the same escape wearing a different hat. A path that does not exist yet is
// checked lexically — `create` is entitled to name a file that is not there,
// and filepath.Join has already cleaned it.

// Abs is the root as the boundary compares it: absolute, and with symlinks
// resolved so a root reached through one — /var on macOS — is the same string
// as the paths it is compared against.
//
// Exported because every caller that pre-screens a path needs the identical
// root, and three hand-rolled copies of these four lines is how the two
// spellings drift apart.
func Abs(root string) (string, error) {
	if followLinks {
		if c, _ := win32Alias(root); c != "" {
			return "", fmt.Errorf("root %s: Windows does not keep %q as written: %s", root, c, aliasCause(c))
		}
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	// ADR-071: a root reached through a junction must compare as its target,
	// or every path under it would resolve outside it.
	if followLinks {
		if absRoot, err = throughLinks(absRoot, osLinks); err != nil {
			return "", err
		}
	}
	// ADR-076: a root that is not there was judged by its spelling, so every
	// path under it "resolved outside the root" — to the root's parent — and a
	// create under it made the root. It is named for what it is.
	real, err := filepath.EvalSymlinks(absRoot)
	switch {
	case err == nil:
		absRoot = real
	case errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("the root %s does not exist", root)
	}
	if fi, err := os.Stat(absRoot); err == nil && !fi.IsDir() {
		return "", fmt.Errorf("the root %s is not a directory", root)
	}
	return absRoot, nil
}
func Resolve(root, path string) (string, error) {
	if err := aliasRefusal(path); err != nil {
		return "", err
	}
	absRoot, err := Abs(root)
	if err != nil {
		return "", err
	}
	return resolveIn(absRoot, path)
}

// aliasRefusal refuses a path Windows would not keep as written (ADR-071).
func aliasRefusal(path string) error {
	if followLinks {
		if c, _ := win32Alias(path); c != "" {
			return aliasError(fmt.Sprintf("%s: Windows does not keep %q as written: %s", path, c, aliasCause(c)))
		}
	}
	return nil
}

// aliasCause says which of win32Alias's causes applies to comp, so a refusal
// names the one that does and not all three (the Windows retest of v1.60.0).
func aliasCause(comp string) string {
	switch {
	case strings.Contains(comp, ":"):
		return "Windows reads ':' as an NTFS stream, not as part of a name"
	case strings.TrimRight(comp, ". ") != comp:
		return `Windows drops a trailing dot or space from a name, so it opens another file; a file that really has this name can only be reached through a \\?\ path, which mrw does not use`
	default:
		return "Windows turns a byte that is not valid UTF-8 into U+FFFD, so it opens another file"
	}
}

// resolveIn is Resolve under a root Abs already resolved.
func resolveIn(absRoot, path string) (string, error) {
	var err error
	full := filepath.Join(absRoot, path)
	// ADR-076: Win32 opens CON, NUL, COM1 and the rest as devices — before
	// Windows 11 with any extension too — so `mrw read NUL` served an empty
	// file and a plan could write to a device at exit 0. The path is judged as
	// it lands under the root, cleaned, since `NUL/.` opens NUL too (Codex
	// review of #237), and never by the root's own components.
	// ADR-081: refused by name on every build. v1.27.0 asked GetFullPathName,
	// which on Windows 11 no longer maps `con` or `nul.txt` in a directory, so
	// mrw created them — files its own unlink and PowerShell 5 could not reach.
	if followLinks {
		if err := deviceName(path, absRoot, full); err != nil {
			return "", err
		}
	}
	// ADR-071: on Windows a junction is followed here, because EvalSymlinks
	// no longer does. Elsewhere target is full and nothing changes.
	target := full
	if followLinks {
		if target, err = throughLinks(full, osLinks); err != nil {
			return "", err
		}
	}
	// ADR-081: a link inside the root that leads to a reserved name reaches it
	// as surely as the name itself (the reviews of #243).
	if followLinks && target != full {
		if err := deviceName(path, absRoot, target); err != nil {
			return "", err
		}
	}
	check := target
	leafReal := false
	if real, err := filepath.EvalSymlinks(target); err == nil {
		check, leafReal = real, true
	} else {
		// A missing leaf is checked through the deepest existing ancestor.
		// EvalSymlinks(full) fails for create/rename dests that are not there
		// yet; walking up is what stops `link/new` from being judged lexical
		// when `link` is a symlink out of the root.
		for p := filepath.Dir(target); ; p = filepath.Dir(p) {
			if real, err := filepath.EvalSymlinks(p); err == nil {
				check = real
				break
			}
			parent := filepath.Dir(p)
			if parent == p {
				break
			}
		}
	}
	if !Contains(absRoot, check) {
		return "", fmt.Errorf("%s resolves to %s, which is outside the root %s", path, check, absRoot)
	}
	// ADR-076: a trailing separator names a directory — the OS refuses
	// open("a.txt/"), and "a.txt/." — and the Join above cleaned it away, so
	// `mrw read a.txt/` served a.txt. A name that does not exist is left to the
	// caller, which reports it missing in its own words.
	if SpelledAsDirectory(path) {
		if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
			return "", fmt.Errorf("%s %w, but %s is a file", path, ErrNotADirectory, filepath.Clean(path))
		}
	}
	// ADR-077: mrw's own state — the ledger that licenses a write, the ack
	// store, the tally — is not the caller's file. Under a root that holds it
	// (XDG_STATE_HOME inside the checkout, or --root "$HOME" with
	// ~/.local/state), a read served pending.json, whose checkpoint ids could
	// then be acked without the lines ever being read (ADR-031), and a plan
	// could edit the ledger.
	// ADR-123: the real path just computed is reused rather than resolved a
	// second time inside InState; a missing leaf is resolved as InState did.
	q := check
	if !leafReal {
		q = RealAsFarAsItExists(target)
	}
	if inState(q) {
		return "", fmt.Errorf("%s is inside mrw's own state directory; mrw does not serve or edit its own ledger", path)
	}
	// ADR-134: a second name of a state file is the state file.
	if err := hardLinkRefusal(absRoot, q, path); err != nil {
		return "", err
	}
	return full, nil
}

// Real is p as the boundary compares it: cleaned, and with its links resolved
// the way Abs resolves a root — on Windows through junctions as well (ADR-071)
// — so an absolute argument and the root it is checked against are spelled the
// same way. Resolving only with EvalSymlinks, which stops at a junction,
// refused an absolute path inside a root reached through one (review of #228).
// A path that cannot be resolved comes back cleaned; Resolve still judges it.
func Real(p string) string { return links.Real(p) }

// Contains reports whether p is absRoot itself or something beneath it. The
// separator matters: without it, "/repo-backup" counts as inside "/repo". A root
// that already ends in one — the filesystem root "/", a volume root "C:\" — is
// its own prefix: appending another asked for "//" and refused every child
// (ADR-103).
func Contains(absRoot, p string) bool {
	if p == absRoot {
		return true
	}
	prefix := absRoot
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(p, prefix)
}

// ErrNotADirectory is what Resolve wraps, and plan validation reports, when a
// path is spelled as a directory but does not name one (ADR-076).
var ErrNotADirectory = errors.New("names a directory, ending in a separator, `.` or `..`")

// ErrDeviceName is Resolve's refusal of a Windows device name (ADR-081). It is
// its own error so a caller does not append advice that fits an escape from the
// root — no --root makes `con` a file (the review of #243).
var ErrDeviceName = errors.New("a Windows device name")

// ErrWin32Alias is Resolve's refusal of a name Windows would not keep as written
// (ADR-071): a trailing dot or space, a stream, a byte that is not UTF-8.
var ErrWin32Alias = errors.New("a name Windows does not keep as written")

// aliasError is aliasRefusal's refusal: its words are the message, ErrWin32Alias
// is what errors.Is finds.
type aliasError string

func (e aliasError) Error() string { return string(e) }

func (e aliasError) Is(target error) bool { return target == ErrWin32Alias }

// UnkeepableName reports whether err is Resolve's refusal of a name Windows will
// not keep, as opposed to its refusals of where a path leads (ADR-135). Only a
// refusal made by the path's own spelling may be counted by a walk: a device
// name reached through a link says where the link leads, not what the discovered
// file is called (the Codex review of #363).
func UnkeepableName(err error) bool {
	return (errors.Is(err, ErrDeviceName) && !errors.Is(err, errViaLink)) || errors.Is(err, ErrWin32Alias)
}

// RefusedByName reports whether err is Resolve's refusal of a name's own
// spelling — a device name or a name Windows does not keep as written — whether
// or not a link led there. Advice that fits an escape from the root ("point
// --root where you mean") sends the caller nowhere for one of these.
func RefusedByName(err error) bool {
	return errors.Is(err, ErrDeviceName) || errors.Is(err, ErrWin32Alias)
}

// errViaLink marks a device-name refusal reached through a link.
var errViaLink = errors.New("through a link")

// viaLinkError is deviceName's refusal for a link whose target holds a device
// name: the same words and still an ErrDeviceName, marked errViaLink.
type viaLinkError struct{ error }

func (e viaLinkError) Unwrap() error { return e.error }

func (e viaLinkError) Is(target error) bool { return target == errViaLink }

// deviceName refuses p when the part of at that lies under absRoot holds a
// reserved device name in any component. The root's own components are not
// judged: a checkout under a directory named aux is the caller's to keep.
func deviceName(p, absRoot, at string) error {
	rel, err := filepath.Rel(absRoot, at)
	if err != nil {
		return nil //nolint:nilerr // no relative path means nothing under the root to judge
	}
	if d := win32Device(rel); d != "" {
		via := ""
		if at != filepath.Join(absRoot, p) {
			via = " through a link"
		}
		err := fmt.Errorf("%s leads%s to %q, which is %w: some Windows APIs still open it as a device on every build, so mrw neither creates, reads nor edits it", p, via, d, ErrDeviceName)
		if via != "" {
			return viaLinkError{err}
		}
		return err
	}
	return nil
}

// SpelledAsDirectory reports whether p, as written, names a directory: it ends
// in a path separator of this platform, or its last element is "." or "..".
// Cleaning drops the first and folds the others away, so a file spelled
// `a.txt/` or `a.txt/.` reached a.txt; the OS refuses both (ADR-076).
func SpelledAsDirectory(p string) bool {
	if p == "" {
		return false
	}
	if os.IsPathSeparator(p[len(p)-1]) {
		return true
	}
	last := p
	for i := len(p) - 1; i >= 0; i-- {
		if os.IsPathSeparator(p[i]) {
			last = p[i+1:]
			break
		}
	}
	return last == "." || last == ".."
}

// RealAsFarAsItExists is p with its links resolved as far as p exists — on
// Windows through junctions too, as Real does — and the rest appended as
// written. A file a create is about to make does not exist, so resolving it
// whole failed and a create through a linked directory named no target; its
// deepest existing ancestor is what the link decides (ADR-076).
func RealAsFarAsItExists(p string) string {
	p = filepath.Clean(p)
	if followLinks {
		if t, err := throughLinks(p, osLinks); err == nil {
			p = t
		}
	}
	var rest []string
	for q := p; ; {
		if real, err := filepath.EvalSymlinks(q); err == nil {
			return filepath.Join(append([]string{real}, rest...)...)
		}
		parent := filepath.Dir(q)
		if parent == q {
			return p
		}
		rest = append([]string{filepath.Base(q)}, rest...)
		q = parent
	}
}

// IsRooted reports whether p names a location of its own, rather than one to be
// resolved relative to a root.
//
// filepath.IsAbs is not enough, and the gap is Windows-only. There, `\etc\hosts`
// and `/etc/hosts` are ROOTED — they name the root of the current drive — but
// IsAbs is FALSE for both, because neither carries a volume. `C:etc` is
// drive-relative and equally not root-relative, and IsAbs is false for that too.
// So every caller asking "did the caller hand me a path that is not relative to
// my root?" answered no on Windows and joined it on.
//
// In read and apply that only misdirects a message. In check it is a silent
// PASS: the joined path places no package, the run falls back to the full check
// and the verdict is green, which is the failure class this project exists to
// refuse. Found by the windows CI job on its first run, 2026-09-03.
//
// A leading backslash is rooted on Windows and an ordinary filename character on
// POSIX, so it counts only where it means something.
func IsRooted(p string) bool { return links.IsRooted(p) }

// InState reports whether p lies inside mrw's state base (state.Base). Both
// sides are resolved as far as they exist (RealAsFarAsItExists, through
// junctions on Windows), so /var and /private/var on macOS, or a base not made
// yet, compare as the one place they are (ADR-077). Where the strings differ,
// each existing ancestor of p is compared with the base as a FILE: on a
// filesystem that folds case `.st/MRW` and `.ST/mrw` are the base, and a
// firmlink root (/System/Volumes/Data on macOS) is another spelling of it — the
// comparison by string let a read of pending.json through (the reviews of
// #238). A base mrw cannot name, or that does not exist, holds nothing a
// second spelling could reach.
func InState(p string) bool {
	return inState(RealAsFarAsItExists(p))
}

// inState is InState for q already resolved as far as it exists (ADR-123).
func inState(q string) bool {
	base, err := state.Base()
	if err != nil {
		return false
	}
	b, bi := resolvedBase(base)
	return inStateAt(b, bi, q)
}

// inStateAt is inState against the base b, resolved, and bi its FileInfo, nil
// when it does not exist.
func inStateAt(b string, bi os.FileInfo, q string) bool {
	if Contains(b, q) {
		return true
	}
	if bi == nil {
		return false
	}
	for {
		if qi, err := os.Stat(q); err == nil && os.SameFile(bi, qi) {
			return true
		}
		parent := filepath.Dir(q)
		if parent == q {
			return false
		}
		q = parent
	}
}

// stateBase caches the state base resolved, while it exists (ADR-123). A walk
// resolves every file it serves, and resolving the base again for each one —
// two passes over its components — was much of the per-file cost on Windows,
// where each component is a syscall (10-14 ms a file, a peer's measurement of
// 2026-10-02). The entry is checked with one Stat on every use, so a base
// removed or replaced during a long session is resolved again.
var stateBase struct {
	mu   sync.Mutex
	base string
	real string
	fi   os.FileInfo
}

// resolvedBase is base's real path and, when it exists, its FileInfo.
func resolvedBase(base string) (string, os.FileInfo) {
	stateBase.mu.Lock()
	defer stateBase.mu.Unlock()
	// The entry stands only while the base, followed NOW, is the directory it
	// was made for: a symlink above the base re-pointed during a long session
	// leaves the old directory in place, and checking the old real path alone
	// served the new base (the review of #330).
	if stateBase.fi != nil && stateBase.base == base {
		if fi, err := os.Stat(stateBase.real); err == nil && os.SameFile(fi, stateBase.fi) && sameAs(base, fi) {
			return stateBase.real, fi
		}
	}
	b := RealAsFarAsItExists(base)
	fi, err := os.Stat(b)
	if err != nil {
		stateBase.fi = nil
		return b, nil
	}
	stateBase.base, stateBase.real, stateBase.fi = base, b, fi
	return b, fi
}

// sameAs reports whether p, followed now, is the directory fi describes.
func sameAs(p string, fi os.FileInfo) bool {
	pi, err := os.Stat(p)
	return err == nil && os.SameFile(pi, fi)
}

// Resolver resolves the paths one walk discovers, answering each exactly as
// Resolve does, with the work their directories share done once (ADR-131).
//
// On Windows every path component is a syscall, and Resolve walked them all
// for every file: the root (Abs), the links (throughLinks), the real path
// (EvalSymlinks) and each ancestor's identity against the state base (inState)
// — 85% of a --grep walk's CPU, 3.1 ms a file, a peer's profile of 2026-10-06.
// Every file in a directory shares all of that but its own name. A Resolver
// resolves the root and the state base once, and each directory once — its
// links, its real path, its ancestors' verdict — and then judges a regular
// file by one Lstat of its own name. Anything else, a link, a directory, a
// name that is not there or a state base that is not a directory, is resolved
// whole, by Resolve's own code.
//
// What it caches is stale for as long as the Resolver lives, and a walk owns
// one and drops it when it returns: a directory swapped for a link, or made
// mrw's state base, during a walk is seen by the next one. That bound is
// acceptable because a walk only chooses what to match: read.Run resolves every
// path it serves afresh, with Resolve, before it reads it to serve it. A
// Resolver is not safe for concurrent use.
type Resolver struct {
	absRoot string
	err     error // Abs's refusal of the root, returned for every path
	based   bool  // the state base below has been resolved
	none    bool  // there is no state base to compare with (state.Base failed)
	b       string
	bi      os.FileInfo
	links   stateLinks // judges a file with a second name against the state base (ADR-134)
	dirs    map[string]resolvedDir
}

// resolvedDir is one directory as Resolve would resolve it: target through its
// links, real its real path, in whether it or an ancestor is the state base.
// ok is false when it could not be resolved, so its files are resolved whole.
type resolvedDir struct {
	target, real string
	in, ok       bool
}

// NewResolver returns a Resolver for paths under root.
func NewResolver(root string) *Resolver {
	r := &Resolver{dirs: map[string]resolvedDir{}}
	r.absRoot, r.err = Abs(root)
	return r
}

// Resolve is rooted.Resolve(root, path) for the root the Resolver was made for.
func (r *Resolver) Resolve(path string) (string, error) {
	if err := aliasRefusal(path); err != nil {
		return "", err
	}
	if r.err != nil {
		return "", r.err
	}
	full := filepath.Join(r.absRoot, path)
	fi, err := os.Lstat(full)
	if err != nil || !fi.Mode().IsRegular() || SpelledAsDirectory(path) || !r.base() {
		return resolveIn(r.absRoot, path)
	}
	d := r.dir(filepath.Dir(full))
	if !d.ok {
		return resolveIn(r.absRoot, path)
	}
	if followLinks {
		if err := deviceName(path, r.absRoot, full); err != nil {
			return "", err
		}
	}
	leaf := filepath.Base(full)
	if target := filepath.Join(d.target, leaf); followLinks && target != full {
		if err := deviceName(path, r.absRoot, target); err != nil {
			return "", err
		}
	}
	// A regular file is not a link, so its real path is its directory's
	// joined with its name; and it is not a directory, so it is the state
	// base's identity only if it is the base's own path.
	check := filepath.Join(d.real, leaf)
	if !Contains(r.absRoot, check) {
		// Resolve names the real path in its refusal, and on Windows
		// EvalSymlinks spells the name as it is on disk, where check keeps
		// the caller's spelling (the Codex review of #353). A refusal is
		// rare; it is worded by Resolve's own code.
		return resolveIn(r.absRoot, path)
	}
	if d.in || (!r.none && Contains(r.b, check)) {
		return "", fmt.Errorf("%s is inside mrw's own state directory; mrw does not serve or edit its own ledger", path)
	}
	// ADR-134: the same file as one under the state base, by another name.
	if !r.none {
		if err := r.links.refusal(fi, full, path); err != nil {
			return "", err
		}
	}
	return full, nil
}

// base resolves the state base once, and reports whether a regular file can be
// judged against it by its directory alone: there is no base, or the base is a
// directory, which no regular file can be the same file as.
func (r *Resolver) base() bool {
	if !r.based {
		r.based = true
		base, err := state.Base()
		if err != nil {
			r.none = true
		} else {
			r.b, r.bi = resolvedBase(base)
			r.links = stateLinks{root: r.absRoot, b: r.b, bi: r.bi}
		}
	}
	return r.none || r.bi == nil || r.bi.IsDir()
}

// dir resolves the directory dir as Resolve resolves a path's parent, once.
func (r *Resolver) dir(dir string) resolvedDir {
	if d, ok := r.dirs[dir]; ok {
		return d
	}
	d := resolvedDir{target: dir}
	var err error
	if followLinks {
		d.target, err = throughLinks(dir, osLinks)
	}
	if err == nil {
		if d.real, err = filepath.EvalSymlinks(d.target); err == nil {
			d.ok = true
			d.in = !r.none && inStateAt(r.b, r.bi, d.real)
		}
	}
	r.dirs[dir] = d
	return d
}
