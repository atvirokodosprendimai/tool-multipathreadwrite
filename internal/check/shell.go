package check

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Shell returns the POSIX shell a check runs under, and whether one was found:
// sh from PATH, or on Windows the sh.exe Git for Windows ships beside git.exe.
// A plain PowerShell PATH holds Git's cmd directory and not its usr\bin, so the
// check could not start there at all, on a platform mrw is used on as much as
// any other (ADR-082). Not found, it returns "sh", so the start fails with the
// usual exec error and Run reports it with noShellAdvice.
func Shell() (string, bool) {
	if p, err := exec.LookPath("sh"); err == nil {
		return p, true
	}
	if runtime.GOOS != "windows" {
		return "sh", false
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return "sh", false
	}
	if sh := gitShell(git); sh != "" {
		return sh, true
	}
	return "sh", false
}

// gitShell finds Git for Windows' sh.exe from a git.exe: git.exe lives in
// <Git>\cmd, <Git>\bin or <Git>\mingw64\bin, and the shell in <Git>\usr\bin or
// <Git>\bin. It walks up at most three directories and takes only a regular
// file, returning "" when there is none.
func gitShell(git string) string {
	dir := filepath.Dir(git)
	for i := 0; i < 3; i++ {
		for _, rel := range []string{filepath.Join("usr", "bin", "sh.exe"), filepath.Join("bin", "sh.exe")} {
			p := filepath.Join(dir, rel)
			if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
				return p
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// noShellAdvice ends a could-not-start report when no shell was found, naming
// what would give the check one (ADR-082).
func noShellAdvice() string {
	if runtime.GOOS == "windows" {
		return " — no sh on PATH and no Git for Windows beside git: install Git for Windows, or put an sh on PATH"
	}
	return " — no sh on PATH"
}
