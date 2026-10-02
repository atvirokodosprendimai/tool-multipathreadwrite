package read

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
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
	// NoIgnore walks every regular file, as ADR-007 did. Without it, a walk
	// inside a checkout skips what its .gitignore files and .git/info/exclude
	// ignore, and every walk skips a discovered binary (ADR-116).
	NoIgnore bool
	// Skipped, when set, receives the counts of what the walk skipped, so a
	// caller can say it: nothing is skipped silently (ADR-116).
	Skipped *WalkSkipped
}

// WalkSkipped counts what a walk skipped under ADR-116's rules. A path the
// caller named is never skipped.
type WalkSkipped struct {
	// Ignored is the files the ignore rules named.
	Ignored int `json:"ignored"`
	// IgnoredDirs is the directories the ignore rules named, pruned unread.
	IgnoredDirs int `json:"ignored_dirs"`
	// Binary is the discovered files that would have matched and are binary
	// (lines.Unsplittable: a UTF-16/32 byte-order mark or a NUL in the first
	// 8 KiB).
	Binary int `json:"binary"`
}

// SkipNote is the sentence a surface prints for what a walk skipped
// (ADR-116), naming the flag that walks it; "" when nothing was skipped.
func SkipNote(sk WalkSkipped, flag string) string {
	if sk == (WalkSkipped{}) {
		return ""
	}
	var parts []string
	if sk.Ignored > 0 {
		parts = append(parts, fmt.Sprintf("%d file(s) .gitignore ignores", sk.Ignored))
	}
	if sk.IgnoredDirs > 0 {
		parts = append(parts, fmt.Sprintf("%d director(ies) .gitignore ignores, not entered", sk.IgnoredDirs))
	}
	if sk.Binary > 0 {
		parts = append(parts, fmt.Sprintf("%d binary file(s) that matched", sk.Binary))
	}
	return "-- skipped: " + strings.Join(parts, ", ") + "; " + flag + " walks them"
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
	// rooted.Real, not filepath.EvalSymlinks: since Go 1.23 EvalSymlinks
	// leaves a Windows junction as written, so a root reached through one was
	// walked from the junction, which WalkDir Lstats as no directory, and a
	// walk from the root served nothing (ADR-071; review of ADR-096).
	absRoot = rooted.Real(absRoot)
	if len(paths) == 0 {
		paths = []string{"."}
	}

	w := walker{root: root, absRoot: absRoot, opt: opt, seen: map[string]bool{}, nested: map[string]*ignorer{},
		skipFiles: map[string]bool{}, skipDirs: map[string]bool{}, skipBin: map[string]bool{}}
	if !opt.NoIgnore {
		w.ign = newIgnorer(absRoot)
	}
	for _, p := range paths {
		w.consider(p)
	}
	if opt.Skipped != nil {
		*opt.Skipped = w.skipCounts()
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
	ign      *ignorer // nil outside a checkout or under NoIgnore
	// nested holds each checkout found below the root, by its root-relative
	// "/"-joined directory: its own rules apply beneath it and the outer
	// rules do not, as git keeps a nested repository's files its own.
	nested map[string]*ignorer
	// What the ignore rules and the binary test skipped, by root-relative
	// path, so a path two walks meet counts once, and one served after all
	// counts not at all (skipCounts). starts are the named directories walked.
	skipFiles, skipDirs, skipBin map[string]bool
	starts                       []string
}

// consider handles one path the caller named: judgeNamed decides what it is,
// and consider serves a file or walks a directory. Something found by walking
// was not asked for and is skipped in silence; something named is reported.
func (w *walker) consider(p string) {
	n, prob := judgeNamed(w.root, w.absRoot, p)
	if prob != nil {
		w.problems = append(w.problems, *prob)
		return
	}
	if n.dir {
		w.walkDir(n.rel, n.full)
		return
	}
	w.offer(n.rel, n.full, false)
}

// namedPath is a path the caller named that judgeNamed accepted: rel is the
// root-relative spelling it is served and reported under, full the path to
// read or the directory to walk from, and dir says which.
type namedPath struct {
	rel  string
	full string
	dir  bool
}

// judgeNamed judges one path the caller named, before anything is read: it is
// refused (the Problem says why), a regular file to serve, or a directory to
// walk. It is a function of the root and the path alone, so every finder asks
// it the same question (ADR-096 decision 1). absRoot is the root with its links
// resolved, as Walk computes it.
func judgeNamed(root, absRoot, p string) (namedPath, *Problem) {
	asWritten := p
	// An ABSOLUTE path is honoured, not joined — the same command-line
	// convention read.Run already applies (see its argPath block). Without
	// this, `mrw read --grep P /repo/sub` refused a directory that plain
	// `mrw read /repo/sub/f.go` serves, because Resolve joined it onto the
	// root and looked for /repo/repo/sub. One surface honoured the convention
	// and its sibling did not, which is the asymmetry nobody expects.
	//
	// Found by an independent review, 2026-09-03.
	if rooted.IsRooted(p) {
		abs, absErr := rooted.Abs(root)
		if absErr != nil {
			return namedPath{}, &Problem{Path: p, Reason: absErr.Error()}
		}
		cleaned := rooted.Real(p)
		if !rooted.Contains(abs, cleaned) {
			// Named, so it is reported rather than skipped: rule 5.
			return namedPath{}, &Problem{
				Path:   p,
				Reason: "is outside the root " + abs + ": walk it with --root pointed where you mean",
			}
		}
		if rel, relErr := filepath.Rel(abs, cleaned); relErr == nil {
			// ADR-076: Real cleaned the separator away; it is kept, so the
			// spelling is judged below as the caller wrote it (Codex review
			// of #237).
			if rooted.SpelledAsDirectory(p) {
				rel += string(filepath.Separator)
			}
			p = rel
		}
	}
	full, err := rooted.Resolve(root, p)
	if err != nil {
		return namedPath{}, &Problem{Path: p, Reason: err.Error()}
	}
	fi, err := os.Stat(full)
	if err != nil {
		return namedPath{}, &Problem{Path: p, Reason: err.Error()}
	}
	if fi.IsDir() {
		// ADR-096 decision 3: a path that resolves to the root IS the root,
		// walked from its resolved path. Walked from a link's own path,
		// WalkDir would Lstat its start, see no directory, and drop it.
		realRoot := rooted.Real(absRoot)
		target := rooted.Real(full)
		if target == realRoot {
			return namedPath{rel: p, full: absRoot, dir: true}, nil
		}
		// ADR-096 decision 2: the walk follows no link to a directory, so
		// one the caller names is refused with the directory to name. full
		// is joined and cleaned, so `dlink/` and `dlink/.` are asked about
		// the link itself. IsDir rather than ModeSymlink, so a Windows
		// junction (ModeIrregular since Go 1.23, ADR-071) meets the same
		// test. An absolute spelling was resolved by Real above and names
		// the directory itself here, so it is walked, as v1.31.0 walks it.
		if lfi, err := os.Lstat(full); err == nil && !lfi.IsDir() {
			t := target
			if rel, relErr := filepath.Rel(realRoot, target); relErr == nil {
				t = filepath.ToSlash(rel)
			}
			return namedPath{}, &Problem{
				Path:   asWritten,
				Reason: "is a link to the directory " + t + ", and a walk does not follow a link: name " + t,
			}
		}
		return namedPath{rel: p, full: full, dir: true}, nil
	}
	if !fi.Mode().IsRegular() {
		return namedPath{}, &Problem{Path: p, Reason: lines.NotRegular}
	}
	return namedPath{rel: p, full: full}, nil
}

// walkDir descends. A symlinked directory is never entered, and the walk does
// not have to check: filepath.WalkDir does not follow symlinks, so one arrives
// as a non-directory entry and is refused by the regular-file test below —
// ADR-007 rule 3, enforced by the walk itself rather than by a guard.
func (w *walker) walkDir(named string, full string) {
	// How many components of the path a caller named: those directories do not
	// prune what is below them (ADR-116, Ignored).
	from := 0
	if r := filepath.ToSlash(w.rel(full)); r != "" && r != "." {
		from = strings.Count(r, "/") + 1
	}
	// A named start inside a nested checkout takes that checkout's rules.
	if r := filepath.ToSlash(w.rel(full)); r != "" && r != "." {
		parts := strings.Split(r, "/")
		for i := 1; i <= len(parts); i++ {
			d := strings.Join(parts[:i], "/")
			w.noteNested(d, filepath.Join(w.absRoot, filepath.FromSlash(d)))
		}
		w.starts = append(w.starts, r)
	}
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
			// ADR-116: an ignored directory is pruned, not entered.
			r := filepath.ToSlash(rel)
			if w.ignored(r, true, from) {
				w.skipDirs[r] = true
				return fs.SkipDir
			}
			w.noteNested(r, filepath.Join(w.absRoot, rel))
			return nil
		}
		// THE BOUNDARY, on the discovered path. `consider` resolves a NAMED
		// path, so an explicit `../outside` was always refused — but an entry
		// found by WALKING reached os.Stat and os.ReadFile with the path the
		// walk built, never through rooted.Resolve. A symlink inside the root
		// pointing at a file outside it was therefore READ to match against,
		// and although read.Run refused to SERVE it afterwards, the two
		// outcomes differ: a pattern that matched printed a REFUSED line
		// naming the resolved out-of-root path, one that did not printed "no
		// file matched". That is a pattern oracle over files outside the root,
		// paid for with a real read of their bytes.
		//
		// ADR-007 rule 3 already said every candidate passes rooted.Resolve;
		// this is that sentence being true on both paths. Found by an
		// independent review, 2026-09-03 — the existing boundary tests covered
		// an explicit ../ and an IN-root symlink, so nothing failed.
		full, err := rooted.Resolve(w.root, rel)
		if err != nil {
			// Discovered, not named: skipped in silence, per rule 2. Reporting
			// it would re-create the oracle in the problem list.
			return nil //nolint:nilerr // discovered, not named: skipped in silence (rule 2, above)
		}
		// Resolve, THEN ask what it is: a symlink to an in-root regular file
		// is a candidate, a FIFO or device is not.
		if st, err := os.Stat(full); err != nil || !st.Mode().IsRegular() {
			return nil //nolint:nilerr // a discovered non-file is skipped in silence
		}
		if w.excluded(rel) {
			return nil
		}
		if r := filepath.ToSlash(rel); w.ignored(r, false, from) {
			w.skipFiles[r] = true
			return nil
		}
		w.offer(rel, full, true)
		return nil
	})
	if err != nil {
		w.problems = append(w.problems, Problem{Path: named, Reason: err.Error()})
	}
}

// offer turns one candidate into a spec if it matches and is not already there.
// A discovered binary that would have matched is skipped and counted; a named
// one is served, as a path the caller names always is (ADR-116).
func (w *walker) offer(p, full string, discovered bool) {
	key := filepath.ToSlash(filepath.Clean(p))
	if w.seen[key] {
		return
	}
	b, err := readCapped(full)
	if err != nil {
		w.problems = append(w.problems, Problem{Path: p, Reason: err.Error()})
		return
	}
	binary := discovered && !w.opt.NoIgnore && lines.Unsplittable(b) != ""
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
	if binary {
		w.skipBin[key] = true
		return
	}
	w.seen[key] = true
	w.specs = append(w.specs, Spec{
		Path:   key,
		Raw:    key,
		Ranges: []Range{{Re: w.opt.Pattern, Text: "/" + w.opt.Pattern.String() + "/"}},
	})
}

// ignored judges rel, root-relative and "/"-joined, by the rules of the
// checkout it is in: the deepest nested one above it, else the root's.
func (w *walker) ignored(rel string, isDir bool, from int) bool {
	ig, sub, best := w.ign, rel, ""
	for k := range w.nested {
		if strings.HasPrefix(rel, k+"/") && len(k) > len(best) {
			best = k
		}
	}
	if best != "" {
		ig, sub = w.nested[best], rel[len(best)+1:]
		from = max(0, from-(strings.Count(best, "/")+1))
	}
	return ig != nil && ig.Ignored(sub, isDir, from)
}

// noteNested records dir (root-relative, "/"-joined; abs its path) as a
// nested checkout when it holds a .git, so its own rules apply beneath it.
func (w *walker) noteNested(dir, abs string) {
	if w.opt.NoIgnore || w.nested[dir] != nil {
		return
	}
	if _, err := os.Lstat(filepath.Join(abs, ".git")); err != nil {
		return
	}
	if ig := newIgnorer(abs); ig != nil {
		w.nested[dir] = ig
	}
}

// skipCounts is what the walk skipped and did not serve after all: a file
// another path served is not counted, nor a directory a named path entered.
func (w *walker) skipCounts() WalkSkipped {
	var sk WalkSkipped
	for p := range w.skipFiles {
		if !w.seen[p] {
			sk.Ignored++
		}
	}
	for p := range w.skipBin {
		if !w.seen[p] {
			sk.Binary++
		}
	}
	// A directory counts when nothing entered it after all: no named start at
	// or below it, and no served path below it. One set of every start and
	// every served path's directories keeps this linear in what was served.
	entered := map[string]bool{}
	for _, s := range w.starts {
		entered[s] = true
	}
	for p := range w.seen {
		for d := path.Dir(p); d != "." && d != "/" && !entered[d]; d = path.Dir(d) {
			entered[d] = true
		}
	}
	for d := range w.skipDirs {
		if !entered[d] && !startBelow(w.starts, d) {
			sk.IgnoredDirs++
		}
	}
	return sk
}

// startBelow says whether a named start lies below dir.
func startBelow(starts []string, dir string) bool {
	for _, s := range starts {
		if strings.HasPrefix(s, dir+"/") {
			return true
		}
	}
	return false
}

// excluded matches a glob against the cleaned root-relative path AND the
// basename, with the one matcher every finder shares (pathExcluded). A glob that can
// never match is refused before a walk starts (CheckExclude), not here.
func (w *walker) excluded(rel string) bool {
	return pathExcluded(rel, w.opt.Exclude)
}

// CheckExclude refuses an --exclude glob that can never match: one path.Match
// rejects as malformed, and one spelled so no root-relative path or base name
// (ADR-007) can match it — rooted (a leading /, or on Windows a drive, which is
// what MSYS makes of /vendor), `./`-prefixed, or ending in /, since the walk
// compares cleaned paths. The CLI and MCP both call it; over MCP
// `exclude: ["["]` was ignored while the CLI refused `--exclude '['` (ADR-078).
func CheckExclude(globs []string) error {
	const why = "a glob matches root-relative paths and base names, so one "
	for _, g := range globs {
		if _, err := path.Match(g, "x"); err != nil {
			return fmt.Errorf("%q: %w", g, err)
		}
		switch {
		case rooted.IsRooted(g):
			return fmt.Errorf("%q: %sthat is rooted never matches; name it relative to --root", g, why)
		case strings.HasPrefix(g, "./"):
			return fmt.Errorf("%q: %sthat starts with ./ never matches; drop the ./", g, why)
		case strings.HasSuffix(g, "/"):
			return fmt.Errorf("%q: %sthat ends in / never matches; drop the trailing / to exclude the directory", g, why)
		}
	}
	return nil
}

func (w *walker) rel(full string) string {
	r, err := filepath.Rel(w.absRoot, full)
	if err != nil {
		return full
	}
	return filepath.ToSlash(r)
}
