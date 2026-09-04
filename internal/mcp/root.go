package mcp

import (
	"fmt"
	"os"
	"path/filepath"
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
// The caller passes "" when --root was NOT given. It must not decide that by
// comparing against the flag's default: `--root .` typed deliberately and
// `--root` omitted both arrive as "." from the flag parser, and an earlier
// version of this treated the pair as identical — so `--root .` beside a
// CLAUDE_PROJECT_DIR served the environment's tree instead of the working
// directory, silently, which is the opposite of what the user asked for.
// cmd/mrw already learned this once; see the IsSet comment in main.go.
func ResolveRoot(explicit string, lookup func(string) (string, bool)) (string, Source) {
	if explicit != "" {
		return explicit, SourceFlag
	}
	if lookup != nil {
		if v, ok := lookup(projectDirEnv); ok {
			// Trimmed only to decide whether anything was said. The path itself
			// is used UNTRIMMED: a directory whose name really ends in a space
			// is legal, and silently trimming it would resolve to a different
			// directory than the host named.
			if strings.TrimSpace(v) != "" && isUsableDir(v) {
				return v, SourceProjectDir
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

// CheckRoot refuses a root that NOBODY NAMED and that cannot plausibly be a
// project. It returns nil for everything else.
//
// ⚠ THE GUARD IS ON THE SOURCE, NOT ON THE PATH. Issue #81 reports a server
// bound to `/`, but `/` is the symptom: the defect is that with no --root and
// no CLAUDE_PROJECT_DIR the server serves whichever directory the host
// happened to launch it in, and a fallback onto some arbitrary directory is
// equally unintended. Confinement still works and every refusal is still
// correct — about a tree nobody asked about, on a surface that also WRITES.
//
// EXPLICITNESS IS THE LICENCE. `--root /` and a host-set CLAUDE_PROJECT_DIR
// are statements of intent and are honoured whatever they name; only the
// fallback is second-guessed. That is what keeps this from becoming a list of
// paths somebody finds distasteful — a rule this repository would then have to
// defend, and which would break the Desktop population whose documents really
// do live under the home directory.
//
// The two refused cases are the ones that cannot be a project by construction:
// a filesystem root, and the home directory itself.
func CheckRoot(root string, src Source) error {
	if src != SourceWorkingDir {
		return nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		// Undecidable, so not refused: a guard that fires when it cannot tell
		// would stop a server for a reason it cannot name.
		return nil
	}
	abs = resolved(abs)

	// filepath.Dir of a filesystem root is itself, which is true for "/" and
	// for a Windows volume root like `C:\` without naming either.
	if filepath.Dir(abs) == abs {
		return fmt.Errorf("refusing to serve %s: nothing named a project, so this is the directory this process happened to start in, and it is a filesystem root.\n"+
			"Every read and write would be scoped to the whole filesystem. Name the tree you mean:\n"+
			"  mrw --root /path/to/repo mcp\n"+
			"or set CLAUDE_PROJECT_DIR. An explicit --root is always honoured, including this one", abs)
	}
	if home, err := os.UserHomeDir(); err == nil && resolved(home) == abs {
		return fmt.Errorf("refusing to serve %s: nothing named a project, so this is the directory this process happened to start in, and it is your home directory.\n"+
			"Name the tree you mean:\n"+
			"  mrw --root /path/to/repo mcp\n"+
			"or set CLAUDE_PROJECT_DIR. An explicit --root is always honoured, including this one", abs)
	}
	return nil
}

// resolved follows symlinks when it can, so /tmp and /private/tmp on macOS are
// not two different answers about one directory. A path that cannot be
// resolved is returned as given: this is a comparison aid, not a check.
func resolved(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return p
}
