package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// stateHint names a filesystem failure in mrw's own state directory as that,
// with the variable that moves it. Bare, a sandbox's error read "mrw: mkdir
// /nonexistent: read-only file system", and the agent there could not tell what
// mrw was making or that XDG_STATE_HOME moves it (field report, 2026-10-20). A
// path under the state base is mrw's; so is a mkdir of one of the base's
// parents, which is where MkdirAll stops when the home itself cannot be made.
// The exit code is err's: the hint only adds words.
func stateHint(err error) error {
	base, berr := state.Base()
	if berr != nil {
		return err
	}
	op, path, ok := failedPath(err)
	if !ok || !within(path, base) && (op != "mkdir" || !within(base, path)) {
		return err
	}
	where := "XDG_STATE_HOME is not set, so it is under ~/.local/state"
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		where = "it is under XDG_STATE_HOME=" + xdg
	}
	return fmt.Errorf("%w — mrw's state directory, where it keeps the read ledger outside the checkout; %s: set XDG_STATE_HOME to a writable absolute directory", err, where)
}

// pathError is the text of an *fs.PathError, "<op> <path>: <reason>", for when
// errors.As cannot reach one: most commands return cli.Exit(err, …), and
// urfave/cli's exit error has no Unwrap.
var pathError = regexp.MustCompile(`^(mkdir|open) (.+?): `)

// failedPath is the op and path of the filesystem call err reports.
func failedPath(err error) (op, path string, ok bool) {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Op, pe.Path, true
	}
	m := pathError.FindStringSubmatch(err.Error())
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// within reports whether p is dir or a path below it.
func within(p, dir string) bool {
	p, dir = filepath.Clean(p), filepath.Clean(dir)
	return p == dir || strings.HasPrefix(p, dir+string(filepath.Separator))
}
