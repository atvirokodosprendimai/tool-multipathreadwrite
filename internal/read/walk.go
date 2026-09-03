package read

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// Problem is one path the walk could not serve, and why. A walk reports every
// one and keeps going: this is read.Run's existing batch behaviour — "a missing
// file is reported in the output rather than aborting the batch" — applied to
// discovery, because the other N-1 answers are still worth having.
type Problem struct {
	Path   string
	Reason string
}

// WalkOptions is what the caller asked for.
type WalkOptions struct {
	// Pattern selects files. A file with no match is not served and not
	// observed: mrw showed the caller nothing of it.
	Pattern *regexp.Regexp
	// Exclude prunes. Each glob is matched with path.Match against BOTH the
	// cleaned root-relative path AND the basename, and a match on either
	// excludes. The basename half is not a convenience: path.Match's * does not
	// cross a separator and ** is not a token, so "*_test.go" against the full
	// path alone matches no test file anywhere below the root.
	Exclude []string
}

// Walk turns the caller's paths into the Specs read.Run already knows how to
// serve: one per file that matches, addressed by the pattern.
//
// It records NOTHING. Its reads are for matching only — Run's read is the
// authoritative one and the only one that observes (ADR-005), so a file that
// stops matching between the walk and the serve prints nothing and observes
// nothing, which is the honest answer.
//
// The error return is for a root that cannot be resolved, and nothing else.
// Every other failure is a Problem, so one bad path never costs the caller the
// answers about the good ones.
func Walk(root string, paths []string, opt WalkOptions) ([]Spec, []Problem, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	if real, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = real
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}

	w := walker{root: root, absRoot: absRoot, opt: opt, seen: map[string]bool{}}
	for _, p := range paths {
		w.consider(p, true)
	}
	sort.Slice(w.specs, func(i, j int) bool { return w.specs[i].Path < w.specs[j].Path })
	return w.specs, w.problems, nil
}

type walker struct {
	root     string
	absRoot  string
	opt      WalkOptions
	seen     map[string]bool // cleaned root-relative paths already turned into specs
	specs    []Spec
	problems []Problem
}

// consider handles one path. named says the CALLER wrote it, which is the whole
// difference between silence and a Problem: something found by walking was not
// asked for, and something named was.
func (w *walker) consider(p string, named bool) {
	full, err := rooted.Resolve(w.root, p)
	if err != nil {
		w.problems = append(w.problems, Problem{Path: p, Reason: err.Error()})
		return
	}
	fi, err := os.Stat(full)
	if err != nil {
		w.problems = append(w.problems, Problem{Path: p, Reason: err.Error()})
		return
	}
	if fi.IsDir() {
		w.walkDir(p, full)
		return
	}
	if !fi.Mode().IsRegular() {
		if named {
			w.problems = append(w.problems, Problem{
				Path:   p,
				Reason: "not a regular file: mrw would block on a pipe or stream a device without end",
			})
		}
		return
	}
	w.offer(p, full)
}

// walkDir descends. A symlinked directory is never entered, and the walk does
// not have to check: filepath.WalkDir does not follow symlinks, so one arrives
// as a non-directory entry and is refused by the regular-file test below —
// ADR-007 rule 3, enforced by the walk itself rather than by a guard.
func (w *walker) walkDir(named string, full string) {
	err := filepath.WalkDir(full, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == full {
				return err
			}
			w.problems = append(w.problems, Problem{Path: w.rel(p), Reason: err.Error()})
			return fs.SkipDir
		}
		rel := w.rel(p)
		if d.IsDir() {
			if p == full {
				return nil
			}
			if d.Name() == ".git" || w.excluded(rel) {
				return fs.SkipDir
			}
			return nil
		}
		// A symlink reports itself here, so ask about what it resolves to
		// rather than about the link: rule 2 is "resolve, then ask".
		if st, err := os.Stat(p); err != nil || !st.Mode().IsRegular() {
			return nil // a discovered non-file is skipped in silence
		}
		if w.excluded(rel) {
			return nil
		}
		w.offer(rel, p)
		return nil
	})
	if err != nil {
		w.problems = append(w.problems, Problem{Path: named, Reason: err.Error()})
	}
}

// offer turns one candidate into a spec if it matches and is not already there.
func (w *walker) offer(p, full string) {
	key := filepath.ToSlash(filepath.Clean(p))
	if w.seen[key] {
		return
	}
	b, err := os.ReadFile(full)
	if err != nil {
		w.problems = append(w.problems, Problem{Path: p, Reason: err.Error()})
		return
	}
	lines, _ := split(b)
	matched := false
	for _, l := range lines {
		if w.opt.Pattern.MatchString(l) {
			matched = true
			break
		}
	}
	if !matched {
		return
	}
	w.seen[key] = true
	w.specs = append(w.specs, Spec{
		Path:   key,
		Raw:    key,
		Ranges: []Range{{Re: w.opt.Pattern, Text: "/" + w.opt.Pattern.String() + "/"}},
	})
}

// excluded matches a glob against the cleaned root-relative path AND the
// basename. A glob path.Match rejects is a usage error the caller sees at parse
// time, not a pattern that silently matches nothing, so a bad one is reported
// rather than ignored here.
func (w *walker) excluded(rel string) bool {
	base := path.Base(rel)
	for _, g := range w.opt.Exclude {
		if ok, err := path.Match(g, rel); err == nil && ok {
			return true
		}
		if ok, err := path.Match(g, base); err == nil && ok {
			return true
		}
	}
	return false
}

func (w *walker) rel(full string) string {
	r, err := filepath.Rel(w.absRoot, full)
	if err != nil {
		return full
	}
	return filepath.ToSlash(r)
}
