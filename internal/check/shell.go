package check

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// NoShell is the phrase a could-not-start report carries when no POSIX shell
// was found, so a caller can tell a missing shell from a missing check and not
// advise declaring one (ADR-082).
const NoShell = "no sh on PATH"

// Shell returns the POSIX shell a check runs under, and whether one was found:
// sh from PATH, or on Windows the sh.exe of the Git for Windows installation
// git.exe on PATH belongs to. A plain PowerShell PATH holds Git's cmd directory
// and not its usr\bin, so the check could not start there at all, on a platform
// mrw is used on as much as any other (ADR-082). Not found, it returns "sh", so
// the start fails with the usual exec error and Run reports it.
func Shell() (string, bool) {
	sh, _, ok := shellEnv()
	return sh, ok
}

// shellEnv is Shell plus the directory to put first on the check's PATH: Git's
// usr\bin when its shell was taken, so a nested sh, cat or grep in the declared
// command resolves as it does in Git Bash; "" when sh came from PATH already
// (the reviews of #247).
func shellEnv() (sh, pathDir string, ok bool) {
	if p, err := exec.LookPath("sh"); err == nil {
		return p, "", true
	}
	if runtime.GOOS != "windows" {
		return "sh", "", false
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return "sh", "", false
	}
	if p := gitShell(git); p != "" {
		return p, filepath.Dir(p), true
	}
	return "sh", "", false
}

// gitShell is <root>\usr\bin\sh.exe of the Git for Windows installation git
// belongs to, when that is a regular file, and "" otherwise.
func gitShell(git string) string {
	root := gitRoot(git)
	if root == "" {
		return ""
	}
	p := filepath.Join(root, "usr", "bin", "sh.exe")
	if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
		return p
	}
	return ""
}

// gitRoot is the Git for Windows installation a git.exe belongs to, recognised
// only by Git's own layouts — <root>\cmd, <root>\bin or <root>\mingw64\bin —
// and "" for any other place, a shim's directory included. Walking up from an
// unrecognised git.exe took an unrelated sh.exe: a scoop shim in
// C:\ProgramData\scoop\shims led to C:\ProgramData\bin\sh.exe (the reviews
// of #247).
func gitRoot(git string) string {
	dir := filepath.Dir(git)
	switch strings.ToLower(filepath.Base(dir)) {
	case "cmd":
		return filepath.Dir(dir)
	case "bin":
		parent := filepath.Dir(dir)
		if strings.EqualFold(filepath.Base(parent), "mingw64") {
			return filepath.Dir(parent)
		}
		return parent
	}
	return ""
}

// noShellAdvice ends a could-not-start report when no shell was found, naming
// what would give the check one (ADR-082).
func noShellAdvice() string {
	if runtime.GOOS == "windows" {
		return " — " + NoShell + ", and no Git for Windows installation found from git on PATH: install Git for Windows, or put an sh on PATH"
	}
	return " — " + NoShell
}
