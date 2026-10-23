package read

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// Guard is one file and the sha prefix the caller holds for it (ADR-169).
type Guard struct{ Path, SHA string }

var shaPrefix = regexp.MustCompile(`^[0-9a-fA-F]{8,64}$`)

// ParseGuard splits PATH=SHA at its last "=", so a PATH holding one is kept,
// and refuses a SHA that is not 8 to 64 hex digits.
func ParseGuard(v string) (Guard, error) {
	i := strings.LastIndex(v, "=")
	if i <= 0 {
		return Guard{}, fmt.Errorf("%q: want PATH=SHA", v)
	}
	return NewGuard(v[:i], v[i+1:])
}

// NewGuard is ParseGuard for a caller that holds the path and the sha apart.
func NewGuard(path, sha string) (Guard, error) {
	if path == "" {
		return Guard{}, fmt.Errorf("%q: a guard names a file", path+"="+sha)
	}
	if !shaPrefix.MatchString(sha) {
		return Guard{}, fmt.Errorf("%q: SHA must be 8 to 64 hex digits, a prefix of the file's sha256 (the 8 a header prints are enough)", path+"="+sha)
	}
	return Guard{Path: path, SHA: sha}, nil
}

// ResolveGuards keys each guard by the cleaned path of every spec that names
// its file, however spelled (ADR-007), which is how Options.IfChanged is
// looked up. A guard no spec names is refused, unless the specs came from a
// walk: the file may have stopped matching because it changed, so its path is
// returned in unused to be named, not refused (ADR-169 Decision 1). Two
// guards on one file are refused.
func ResolveGuards(root string, guards []Guard, specs []Spec, walked bool) (keyed map[string]string, unused []string, err error) {
	// Each path is resolved and stated once: a walk can name hundreds of
	// files, and every guard is compared with each of them.
	infos := map[string]os.FileInfo{}
	info := func(p string) os.FileInfo {
		if fi, ok := infos[p]; ok {
			return fi
		}
		infos[p] = guardInfo(root, p)
		return infos[p]
	}
	same := func(a, b string) bool {
		if filepath.Clean(a) == filepath.Clean(b) {
			return true
		}
		fa, fb := info(a), info(b)
		return fa != nil && fb != nil && os.SameFile(fa, fb)
	}
	for i, g := range guards {
		for _, h := range guards[:i] {
			if same(g.Path, h.Path) {
				return nil, nil, fmt.Errorf("two guards on one file: %s and %s", h.Path, g.Path)
			}
		}
	}
	keyed = map[string]string{}
	for _, g := range guards {
		named := false
		for _, sp := range specs {
			if sp.Data == nil && same(g.Path, sp.Path) {
				keyed[filepath.Clean(sp.Path)] = g.SHA
				named = true
			}
		}
		switch {
		case named:
		case walked:
			unused = append(unused, g.Path)
		default:
			return nil, nil, fmt.Errorf("%s=%s: no spec of this read serves that file: name it, or drop the guard", g.Path, g.SHA)
		}
	}
	return keyed, unused, nil
}

// guardInfo is the file p names inside root, or nil when mrw cannot find it.
func guardInfo(root, p string) os.FileInfo {
	full := rooted.Real(p)
	if !rooted.IsRooted(p) {
		var err error
		if full, err = rooted.Resolve(root, p); err != nil {
			return nil
		}
	}
	fi, err := os.Stat(full)
	if err != nil {
		return nil
	}
	return fi
}
