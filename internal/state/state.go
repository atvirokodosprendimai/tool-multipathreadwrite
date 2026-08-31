// Package state resolves where mrw keeps its per-checkout data, which is
// deliberately NOT in the checkout.
//
// mrw used to write `.mrw/` into the working tree. Nothing in a fresh
// repository ignores that, nothing told you it had appeared, and on 2026-08-31
// a ledger was committed by accident under `git add -A` — noticed only because
// someone happened to ask about the tool. Per-checkout tool state belongs
// beside the user's other state, not beside their source.
//
// The location is $XDG_STATE_HOME/mrw/<key>/, keyed by a hash of the absolute
// root. Two alternatives were weighed and are recorded in ADR-004:
//
//   - `.git/mrw/` reads like the obvious answer and is not: in a linked
//     worktree `.git` is a FILE, so resolving it needs `git rev-parse
//     --git-dir` — a subprocess per invocation and a git dependency in a tool
//     that has none — and mrw does not require git at all, so a fallback would
//     be needed anyway.
//   - `.mrw/` with a self-ignoring `.gitignore` fixes the reported bug cleanly
//     and stays visible. It is the runner-up, and the thing to revert to if a
//     hidden state directory proves more annoying than a tracked one.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LegacyDir is the in-tree directory mrw used before ADR-004. It is still READ
// when the state directory holds nothing, and is never written or deleted.
const LegacyDir = ".mrw"

// migratable are the files carried across from a legacy in-tree directory.
var migratable = []string{"seen", "iteration"}

// Dir returns the state directory for root, creating it if needed, and writes a
// `root` marker naming the checkout it belongs to — so an orphan left behind by
// a moved or deleted repository is identifiable rather than an anonymous hash.
func Dir(root string) (string, error) {
	base, err := stateHome()
	if err != nil {
		return "", err
	}
	abs, err := absReal(root)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(base, "mrw", key(abs))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// Best effort: a state directory that works but cannot describe itself is
	// still usable, and failing the whole command over a marker would be worse
	// than the orphan it prevents.
	marker := filepath.Join(dir, "root")
	if existing, err := os.ReadFile(marker); err != nil || strings.TrimSpace(string(existing)) != abs {
		_ = os.WriteFile(marker, []byte(abs+"\n"), 0o600)
	}
	return dir, nil
}

// Path returns the full path of one state file for root.
func Path(root, name string) (string, error) {
	dir, err := Dir(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// LegacyPath returns where a pre-ADR-004 mrw would have kept name. Callers read
// it as a fallback; nothing writes it.
func LegacyPath(root, name string) string {
	return filepath.Join(root, LegacyDir, name)
}

// Migrate copies any legacy in-tree state into the state directory and returns
// the names it copied. It NEVER deletes and never overwrites: a destination
// that already holds a file is live state, and clobbering it with something
// older would lose the newer observations. Removing the legacy files is left to
// the caller — deleting something they may have committed is a worse bug than
// the one this fixes.
func Migrate(root string) ([]string, error) {
	if _, err := os.Stat(filepath.Join(root, LegacyDir)); err != nil {
		return nil, nil //nolint:nilerr // no legacy directory is the normal case
	}
	dir, err := Dir(root)
	if err != nil {
		return nil, err
	}

	var moved []string
	for _, name := range migratable {
		from := LegacyPath(root, name)
		b, err := os.ReadFile(from)
		if err != nil {
			continue
		}
		to := filepath.Join(dir, name)
		if _, err := os.Stat(to); err == nil {
			continue // live state wins over a legacy copy
		} else if !isNotExist(err) {
			return moved, err
		}
		if err := os.WriteFile(to, b, 0o600); err != nil {
			return moved, err
		}
		moved = append(moved, name)
	}
	return moved, nil
}

// stateHome resolves the XDG state base directory.
func stateHome() (string, error) {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		// Refused rather than normalised. The spec requires an absolute path,
		// and resolving a relative one against the working directory is
		// precisely how state lands back inside a checkout.
		if !filepath.IsAbs(xdg) {
			return "", fmt.Errorf("XDG_STATE_HOME is %q, which is not absolute; mrw will not resolve it "+
				"against the working directory because that is how state ends up inside your repository", xdg)
		}
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("no XDG_STATE_HOME and no home directory: %w", err)
	}
	return filepath.Join(home, ".local", "state"), nil
}

// absReal makes root absolute and resolves symlinks, so two spellings of one
// checkout share state rather than silently keeping two ledgers.
func absReal(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real, nil
	}
	// A root that does not exist yet still deserves a stable key.
	return abs, nil
}

// key is a short, stable directory name for a checkout. Truncated because a
// 64-character directory name is unreadable and 64 bits is ample for telling
// one checkout from another on one machine.
func key(abs string) string {
	sum := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(sum[:])[:16]
}

func isNotExist(err error) bool {
	return err != nil && (os.IsNotExist(err) || err == fs.ErrNotExist)
}
