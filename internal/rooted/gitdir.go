package rooted

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GitDir refuses a write whose path lands in a .git directory (ADR-143): a hook,
// a config or a ref changed there changes what git does next, without git being
// asked. The path is judged as spelled and by where it really lands, links
// followed as far as it exists, so `alias/config` where alias leads to .git is
// refused too; the root's own components count, so a root inside .git refuses
// every write. A read of .git is not this function's business.
func GitDir(root, path string) error {
	absRoot, err := Abs(root)
	if err != nil {
		return err
	}
	// The root as spelled counts as well as where it leads: a .git that is a
	// link to somewhere else is still the .git the caller pointed at (the Codex
	// review of #382).
	spelled, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	full := filepath.Join(absRoot, path)
	if inGit(full) || inGit(filepath.Join(spelled, path)) {
		return fmt.Errorf("%s is inside a .git directory; %s", path, gitAdvice)
	}
	// unlink and rename act on the entry, not on what a link entry leads to, so
	// the resolved directory plus the literal leaf is judged as well as the
	// fully followed path.
	entry := filepath.Join(RealAsFarAsItExists(filepath.Dir(full)), filepath.Base(full))
	if inGit(entry) || inGit(RealAsFarAsItExists(full)) {
		return fmt.Errorf("%s leads into a .git directory through a link; %s", path, gitAdvice)
	}
	return nil
}

const gitAdvice = "mrw does not write there, since a hook, a config or a ref changed behind git's back changes what git does next — use git for it (mrw read still reads it)"

// inGit reports whether a component of the absolute path p is a .git.
func inGit(p string) bool {
	for _, c := range strings.Split(filepath.ToSlash(p), "/") {
		if isGitName(c, followLinks) {
			return true
		}
	}
	return false
}

// isGitName is the component test: .git in any case, and — where short names
// exist (short, a Windows build) — git~1, the 8.3 name NTFS gives it, the one
// spelling git's own protection matches.
func isGitName(c string, short bool) bool {
	return strings.EqualFold(c, ".git") || (short && strings.EqualFold(c, "git~1"))
}
