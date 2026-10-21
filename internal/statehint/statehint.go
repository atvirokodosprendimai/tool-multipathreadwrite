// Package statehint names a filesystem failure in mrw's own state directory as
// that, for every surface that can meet one: the CLI's main and the MCP server's
// routes (ADR-164 Decision 3).
package statehint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// Hint names a filesystem failure in mrw's own state directory as that, with
// the variable that moves it. Bare, a sandbox's error read "mrw: mkdir
// /nonexistent: read-only file system", and the agent there could not tell what
// mrw was making or that XDG_STATE_HOME moves it (field report, 2026-10-20).
// It judges the error's type, never its text: a refusal can echo what a caller
// typed, and text naming the state directory is not a failure there (Codex,
// review of #460). An error that is not about the state directory comes back
// as it was; one that is wraps err with %w, so its type and exit code stay.
func Hint(err error) error {
	if !About(err) {
		return err
	}
	where := "XDG_STATE_HOME is not set, so it is under ~/.local/state"
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		where = "it is under XDG_STATE_HOME=" + xdg
	}
	return fmt.Errorf("%w — mrw's state directory, where it keeps the read ledger outside the checkout; %s: set XDG_STATE_HOME to a writable absolute directory", err, where)
}

// About reports whether err is a filesystem failure in mrw's own state
// directory, the errors Hint names; a caller that answers such a failure
// differently asks this rather than comparing Hint's result with err.
func About(err error) bool {
	base, berr := state.Base()
	return berr == nil && about(err, base)
}

// about reports whether err is a filesystem failure on mrw's state directory: a
// path at or below base, or a mkdir or open of one of base's parents, which is
// where MkdirAll stops and where prune opens the home.
func about(err error, base string) bool {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return within(pe.Path, base) || (pe.Op == "mkdir" || pe.Op == "open" || pe.Op == "openat") && within(base, pe.Path)
	}
	var le *os.LinkError
	return errors.As(err, &le) && (within(le.Old, base) || within(le.New, base))
}

// within reports whether p is dir or a path below it.
func within(p, dir string) bool {
	p, dir = filepath.Clean(p), filepath.Clean(dir)
	if p == dir {
		return true
	}
	if !strings.HasSuffix(dir, string(filepath.Separator)) {
		dir += string(filepath.Separator)
	}
	return strings.HasPrefix(p, dir)
}
