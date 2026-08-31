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

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// Name is the ledger's filename inside the state directory. The DIRECTORY is
// resolved by internal/state and is deliberately outside the working tree —
// mrw used to write this beside your source, where nothing ignored it and one
// copy was committed by accident. See ADR-004.
const Name = "seen"

// Ledger maps a path to the SHA-256 mrw last observed it to hold.
type Ledger map[string]string

// Load reads the ledger. A missing file is an empty ledger, not an error: no
// observations yet is the normal starting state.
func Load(root string) (Ledger, error) {
	l := Ledger{}
	path, err := ReadPath(root)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
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

	path, err := writePath(root)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
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
	return os.WriteFile(path, []byte(b.String()), 0o600)
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
	path, err := writePath(root)
	if err != nil {
		return 0, err
	}
	dir := filepath.Dir(path)
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
	return n, os.WriteFile(path, []byte(b.String()), 0o600)
}

// SHA is the ledger's hash of a byte slice, and the one every other package
// must use — two hashes of the same bytes disagreeing would make every write
// look stale.
func SHA(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ReadPath is where this root's ledger is READ from: the state directory,
// falling back to a legacy in-tree file when the state directory holds none, so
// a caller who has not run a migrating command still sees data they already
// have. Writes never use this — see writePath.
func ReadPath(root string) (string, error) {
	p, err := state.Path(root, Name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	if legacy := state.LegacyPath(root, Name); fileExists(legacy) {
		return legacy, nil
	}
	return p, nil
}

// writePath is where the ledger is WRITTEN: always the state directory, never
// the legacy in-tree file. Reading a file a caller already has is compatibility;
// writing it again would be re-creating the bug ADR-004 fixes.
func writePath(root string) (string, error) {
	return state.Path(root, Name)
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
