package mcp

import (
	"os"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// projectDirEnv is the variable a host sets to name the project it is working
// in. Claude Code sets it in every MCP server's environment for exactly this
// purpose, because the working directory a server inherits is not guaranteed to
// be the project — measured 2026-09-03, `mrw mcp` launched with cwd /tmp served
// /private/tmp while this variable named the repository.
const projectDirEnv = "CLAUDE_PROJECT_DIR"

// Source says where a resolved root came from. It is a named type rather than a
// bool because there are three answers, and the startup announcement must not
// claim an environment origin for a fallback — a bool would make that mistake
// representable.
type Source string

const (
	// SourceFlag: the caller passed --root, which always wins.
	SourceFlag Source = "flag"
	// SourceProjectDir: the host named the project in the environment.
	SourceProjectDir Source = "project-dir"
	// SourceWorkingDir: nothing named a root, so the working directory stands.
	SourceWorkingDir Source = "working-dir"
)

// String describes the source in the words the startup line uses.
func (s Source) String() string {
	switch s {
	case SourceFlag:
		return "--root"
	case SourceProjectDir:
		return projectDirEnv
	case SourceWorkingDir:
		return "the working directory"
	}
	return string(s)
}

// ResolveRoot decides which checkout the server serves, and reports which rule
// decided it.
//
// Precedence: an explicit --root, then the host's project directory, then the
// working directory. An explicit root is never second-guessed, because a user
// passing one is overriding the host on purpose.
//
// It takes a lookup function rather than reading the environment itself so the
// rules are testable without setting a real variable — an environment set in one
// test is an environment leaked into its siblings.
//
// A root that is not a usable directory falls back rather than failing. A host
// that sets the variable to something stale should get a working server bound
// somewhere honest, not a dead one; the startup line is what makes the choice
// visible either way.
func ResolveRoot(explicit string, lookup func(string) (string, bool)) (string, Source) {
	// "." is the flag's default value, which means nobody asked. Treating it as
	// a choice would make the environment unreachable through the documented
	// config block, which passes no --root at all.
	if explicit != "" && explicit != "." {
		return explicit, SourceFlag
	}
	if lookup != nil {
		if v, ok := lookup(projectDirEnv); ok {
			if dir := strings.TrimSpace(v); isUsableDir(dir) {
				return dir, SourceProjectDir
			}
		}
	}
	return ".", SourceWorkingDir
}

// isUsableDir reports whether the path names an existing directory by an
// absolute path. Relative is refused: the value is resolved against a working
// directory this process does not control, so a relative answer would be a
// second guess at the very thing the variable exists to settle.
//
// Absoluteness is rooted.IsRooted rather than filepath.IsAbs, because IsAbs is
// FALSE for "/etc/hosts" on Windows and this repository has already shipped one
// guard that silently passed for that reason.
func isUsableDir(p string) bool {
	if !rooted.IsRooted(p) {
		return false
	}
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
