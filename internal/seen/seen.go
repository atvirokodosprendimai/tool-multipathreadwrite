// Package seen is mrw's record of what it last observed each file to be.
//
// It exists to make one guarantee: mrw will not edit a file whose current
// contents it has never seen. That is the property the harness's own Write tool
// has — it refuses to overwrite a file you have not Read — and the reason is
// that an edit is written against a picture of the file, so a picture that is
// out of date silently writes the wrong thing into the wrong place.
//
// A range edit needs it more than a whole-file write does, not less: an address
// like "replace 42-58" means nothing without the version of the file those line
// numbers were counted in.
//
// The ledger is updated on READ and on WRITE, recording what the file is after
// each. So a chain of edits flows without re-reading — mrw already knows what it
// just produced — while anything that changed the file BEHIND mrw's back leaves
// the recorded sha and the real one disagreeing, and the next write is refused.
package seen

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// File is the ledger's location, relative to the repository root. Per-developer
// state, like the working set beside it — not project configuration.
const File = ".mrw/seen"

// Ledger maps a path to the SHA-256 mrw last observed it to hold.
type Ledger map[string]string

// Load reads the ledger. A missing file is an empty ledger, not an error: no
// observations yet is the normal starting state.
func Load(root string) (Ledger, error) {
	l := Ledger{}
	f, err := os.Open(filepath.Join(root, File))
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		sha, path, ok := strings.Cut(strings.TrimSpace(sc.Text()), "  ")
		if ok && sha != "" && path != "" {
			l[path] = sha
		}
	}
	return l, sc.Err()
}

// Record merges observations into the ledger on disk and saves it. Paths absent
// from obs keep whatever was recorded for them: one command observing two files
// must not erase what another observed about a third.
func Record(root string, obs map[string]string) error {
	if len(obs) == 0 {
		return nil
	}
	l, err := Load(root)
	if err != nil {
		return err
	}
	for path, sha := range obs {
		l[path] = sha
	}

	dir := filepath.Join(root, filepath.Dir(File))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	paths := make([]string, 0, len(l))
	for p := range l {
		paths = append(paths, p)
	}
	// Sorted so the file diffs cleanly and two runs produce the same bytes.
	sort.Strings(paths)

	var b strings.Builder
	for _, p := range paths {
		fmt.Fprintf(&b, "%s  %s\n", l[p], p)
	}
	return os.WriteFile(filepath.Join(root, File), []byte(b.String()), 0o644)
}

// Forget drops paths from the ledger, so the next write must observe them
// afresh. Used by `mrw forget` when a caller knows their picture is stale.
func Forget(root string, paths []string) (int, error) {
	l, err := Load(root)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, p := range paths {
		if _, ok := l[p]; ok {
			delete(l, p)
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	// Record cannot express a deletion, so rewrite the whole file here.
	dir := filepath.Join(root, filepath.Dir(File))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	keys := make([]string, 0, len(l))
	for p := range l {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, p := range keys {
		fmt.Fprintf(&b, "%s  %s\n", l[p], p)
	}
	return n, os.WriteFile(filepath.Join(root, File), []byte(b.String()), 0o644)
}

// SHA is the ledger's hash of a byte slice, and the one every other package
// must use — two hashes of the same bytes disagreeing would make every write
// look stale.
func SHA(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
