package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// stateHint names a filesystem failure in mrw's own state directory as that,
// with the variable that moves it. Bare, a sandbox's error read "mrw: mkdir
// /nonexistent: read-only file system", and the agent there could not tell what
// mrw was making or that XDG_STATE_HOME moves it (field report, 2026-10-20).
// It judges the error's type, never its text: a refusal can echo what a caller
// typed, and text naming the state directory is not a failure there (Codex,
// review of #460). The exit code is err's: the hint only adds words.
func stateHint(err error) error {
	base, berr := state.Base()
	if berr != nil || !aboutState(err, base) {
		return err
	}
	where := "XDG_STATE_HOME is not set, so it is under ~/.local/state"
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		where = "it is under XDG_STATE_HOME=" + xdg
	}
	return fmt.Errorf("%w — mrw's state directory, where it keeps the read ledger outside the checkout; %s: set XDG_STATE_HOME to a writable absolute directory", err, where)
}

// aboutState reports whether err is a filesystem failure on mrw's state
// directory: a path at or below base, or a mkdir or open of one of base's
// parents, which is where MkdirAll stops and where prune opens the home.
func aboutState(err error, base string) bool {
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

// exitWith is cli.Exit with exit 2 (exitUsage) for an error that stays unwrappable.
// urfave/cli's exit error has no Unwrap, so errors.As could not reach the
// *fs.PathError under a command that failed in the state directory, and
// stateHint could not see it. Every site it replaced exited 2.
func exitWith(err error) cli.ExitCoder {
	return &exitError{err: err}
}

// exitSaying is exitWith with the words a site already printed, so a failure
// keeps its text and its type: the landing's exitTwoLine points at the FAIL
// line instead of repeating the reason, and still unwraps to the error.
func exitSaying(msg string, err error) cli.ExitCoder {
	return &exitError{err: err, msg: msg}
}

// exitError is a usage-or-filesystem failure (exit 2), unwrapping to its error.
type exitError struct {
	err error
	msg string // what to print, when it is not err's own text
}

func (e *exitError) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return e.err.Error()
}
func (e *exitError) ExitCode() int { return exitUsage }
func (e *exitError) Unwrap() error { return e.err }
