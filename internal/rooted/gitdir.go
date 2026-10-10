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
	full := filepath.Join(absRoot, path)
	if inGit(full) {
		return fmt.Errorf("%s is inside a .git directory; %s", path, gitAdvice)
	}
	if inGit(RealAsFarAsItExists(full)) {
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
// exist (short, a Windows build) — its 8.3 name git~N.
func isGitName(c string, short bool) bool {
	if strings.EqualFold(c, ".git") {
		return true
	}
	if !short || len(c) < 5 || !strings.EqualFold(c[:4], "git~") {
		return false
	}
	for _, r := range c[4:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
