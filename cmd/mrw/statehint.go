package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// stateHint names a filesystem failure in mrw's own state directory as that,
// with the variable that moves it. Bare, a sandbox's error read "mrw: mkdir
// /nonexistent: read-only file system", and the agent there could not tell what
// mrw was making or that XDG_STATE_HOME moves it (field report, 2026-10-20).
// The exit code is err's: the hint only adds words.
func stateHint(err error) error {
	base, berr := state.Base()
	if berr != nil || !aboutState(err.Error(), base) {
		return err
	}
	where := "XDG_STATE_HOME is not set, so it is under ~/.local/state"
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		where = "it is under XDG_STATE_HOME=" + xdg
	}
	return fmt.Errorf("%w — mrw's state directory, where it keeps the read ledger outside the checkout; %s: set XDG_STATE_HOME to a writable absolute directory", err, where)
}

// aboutState reports whether msg names a filesystem call on mrw's state
// directory: an op on base or a path below it, or a mkdir or open of one of
// base's parents, which is where MkdirAll stops and where prune opens the home.
// It reads text because most commands return cli.Exit(err, …), and urfave/cli's
// exit error has no Unwrap for errors.As to follow; and it searches for the
// paths it knows rather than parsing one out, so a path holding ": " is found,
// and a prefix such as the ledger's "could not be updated" does not hide it.
func aboutState(msg, base string) bool {
	for _, op := range []string{"mkdir ", "open ", "openat ", "rename ", "remove ", "chmod ", "stat ", "lstat "} {
		if atCause(msg, op+base+string(filepath.Separator)) || atCause(msg, op+base+": ") {
			return true
		}
	}
	for dir := filepath.Dir(base); ; {
		if atCause(msg, "mkdir "+dir+": ") || atCause(msg, "open "+dir+": ") || atCause(msg, "openat "+dir+": ") {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// atCause reports whether needle starts msg or follows a ": ", where Go joins a
// wrapped error to its context. Anywhere else it is text the caller typed and a
// refusal echoed, such as --format='open <base>/x' (Codex, re-check of #460).
func atCause(msg, needle string) bool {
	for i := 0; ; {
		j := strings.Index(msg[i:], needle)
		if j < 0 {
			return false
		}
		if at := i + j; at == 0 || strings.HasSuffix(msg[:at], ": ") {
			return true
		}
		i += j + 1
	}
}
