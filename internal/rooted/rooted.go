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
			return "", fmt.Errorf("root %s: Windows does not keep %q as written (it drops a trailing dot or space from a name and reads ':' as a stream); name it as it is on disk", root, c)
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
	if followLinks {
		if c, _ := win32Alias(path); c != "" {
			return "", fmt.Errorf("%s: Windows does not keep %q as written (it drops a trailing dot or space from a name and reads ':' as a stream); name the file as it is on disk", path, c)
		}
	}
	absRoot, err := Abs(root)
	if err != nil {
		return "", err
	}

	full := filepath.Join(absRoot, path)
	// ADR-076: Win32 opens CON, NUL, COM1 and the rest as devices — before
	// Windows 11 with any extension too — so `mrw read NUL` served an empty
	// file and a plan could write to a device at exit 0. The cleaned name picks
	// the candidate, since `NUL/.` opens NUL too (Codex review of #237).
	// ADR-081: refused by name on every build. v1.27.0 asked GetFullPathName,
	// which on Windows 11 no longer maps `con` or `nul.txt` in a directory, so
	// mrw created them — files its own unlink and PowerShell 5 could not reach.
	if followLinks {
		if d := win32Device(filepath.Clean(path)); d != "" {
			return "", fmt.Errorf("%s: %q is a Windows device name, which some Windows APIs open as a device on every build; mrw reads and writes files", path, d)
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
	check := target
	if real, err := filepath.EvalSymlinks(target); err == nil {
		check = real
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
	if InState(target) {
		return "", fmt.Errorf("%s is inside mrw's own state directory; mrw does not serve or edit its own ledger", path)
	}
	return full, nil
}

// Real is p as the boundary compares it: cleaned, and with its links resolved
// the way Abs resolves a root — on Windows through junctions as well (ADR-071)
// — so an absolute argument and the root it is checked against are spelled the
// same way. Resolving only with EvalSymlinks, which stops at a junction,
// refused an absolute path inside a root reached through one (review of #228).
// A path that cannot be resolved comes back cleaned; Resolve still judges it.
func Real(p string) string {
	p = filepath.Clean(p)
	if followLinks {
		if t, err := throughLinks(p, osLinks); err == nil {
			p = t
		}
	}
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return p
}

// Contains reports whether p is absRoot itself or something beneath it. The
// separator matters: without it, "/repo-backup" counts as inside "/repo".
func Contains(absRoot, p string) bool {
	return p == absRoot || strings.HasPrefix(p, absRoot+string(filepath.Separator))
}

// ErrNotADirectory is what Resolve wraps, and plan validation reports, when a
// path is spelled as a directory but does not name one (ADR-076).
var ErrNotADirectory = errors.New("names a directory, ending in a separator, `.` or `..`")

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
func IsRooted(p string) bool {
	if p == "" {
		return false
	}
	if filepath.IsAbs(p) || filepath.VolumeName(p) != "" {
		return true
	}
	return p[0] == '/' || (filepath.Separator == '\\' && p[0] == '\\')
}

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
	base, err := state.Base()
	if err != nil {
		return false
	}
	b, q := RealAsFarAsItExists(base), RealAsFarAsItExists(p)
	if Contains(b, q) {
		return true
	}
	bi, err := os.Stat(b)
	if err != nil {
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
