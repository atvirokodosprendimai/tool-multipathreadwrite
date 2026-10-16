package read

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc"
)

// ErrAstGrepMissing is the missing-binary path: the flag exists, the CLI does
// not. Callers print this at exit 2. A present binary with zero hits is a
// different error and must not wrap this sentinel.
var ErrAstGrepMissing = errors.New("ast-grep: not found on PATH")

// ErrAstGrepTimeout is a present binary that did not return within
// astGrepTimeout. Callers print this at exit 2. It must not wrap
// ErrAstGrepMissing, and it must not look like zero hits.
var ErrAstGrepTimeout = errors.New("ast-grep: timed out")

// errAstGrepInterrupted is ast-grep stopped because mrw was sent an interrupt,
// a terminate or a hangup while it ran (ADR-074).
var errAstGrepInterrupted = errors.New("ast-grep: interrupted")

const astGrepTimeout = 2 * time.Second

// maxAstGrepBytes is the largest ast-grep answer mrw reads (ADR-104): the answer
// was read whole, so a pattern matching a huge tree could hold it all in memory.
// A variable so a test can lower it.
var maxAstGrepBytes int64 = 256 << 20

// astGrepHit is the slice of ast-grep --json this mapper needs. Lines are
// 0-based, the way the CLI documents them. Text, Replacement and ByteOffset
// are what --rewrite adds to the object (ADR-161): pointers, because a field
// ast-grep left out is not an empty one.
type astGrepHit struct {
	File        string  `json:"file"`
	Path        string  `json:"path"`
	Text        *string `json:"text"`
	Replacement *string `json:"replacement"`
	Range       struct {
		Start struct {
			Line int `json:"line"`
		} `json:"start"`
		End struct {
			Line int `json:"line"`
		} `json:"end"`
		ByteOffset *struct {
			Start *int `json:"start"`
			End   *int `json:"end"`
		} `json:"byteOffset"`
	} `json:"range"`
}

func (h astGrepHit) name() string {
	if h.File != "" {
		return h.File
	}
	return h.Path
}

// AstGrepOptions are the walk options an ast-grep read shares with --grep
// (ADR-122): NoIgnore turns off every ignore source, mrw's and ast-grep's
// both; Skipped, when set, receives what mrw's own rules dropped. A non-nil
// Rewrites asks for ast-grep's own rewrite to Rewrite (ADR-161, and "" is a
// real one): AstGrep then reads each file with a hit once, and keeps its bytes
// and its matches there instead of leaving them to Run.
type AstGrepOptions struct {
	NoIgnore bool
	Skipped  *WalkSkipped
	Rewrite  string
	Rewrites map[string]*RewriteFile
}

// AstGrep shells out to ast-grep on PATH and maps hits to Specs read.Run
// already serves. It records NOTHING: its process is finding, Run's read is
// the only one that observes (ADR-007). It is a separate primitive from Walk
// on purpose — Walk's go/no-go forbade a fifth matching rule. Its hits pass
// the walk's own rules (ADR-122), so the two finders serve the same files.
func AstGrep(root string, paths []string, pattern string, exclude []string, opt AstGrepOptions) ([]Spec, []Problem, error) {
	if _, err := exec.LookPath("ast-grep"); err != nil {
		return nil, nil, ErrAstGrepMissing
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	// The root as Walk builds it: rooted.Real follows a Windows junction,
	// which filepath.EvalSymlinks has not since Go 1.23 (ADR-071).
	absRoot = rooted.Real(absRoot)
	// ADR-096 decision 4: every named path is judged as the walk judges it,
	// and opened, before the binary starts. A refused one is a Problem and
	// never reaches ast-grep, so a path outside the root is not searched. What
	// reaches it is the path mrw judged — absolute and cleaned, or "." for the
	// root — so no ".." after a link and no leading "-" makes ast-grep read
	// another file or take the path for an option. With paths named and none
	// accepted, ast-grep is not started: its "." fallback is for a caller who
	// named nothing.
	var problems []Problem
	var accepted []string
	for _, p := range paths {
		n, prob := judgeNamed(root, absRoot, p)
		if prob == nil {
			// ast-grep's own failure to read a path can come back as no
			// hits, where the walk reports it; mrw asks first, without
			// blocking on a FIFO swapped in after judgeNamed (ADR-109).
			if f, _, err := regular.Open(n.full); err != nil {
				prob = &Problem{Path: n.rel, Reason: err.Error()}
			} else {
				_ = f.Close() // opened to learn it can be; nothing was read
			}
		}
		if prob != nil {
			problems = append(problems, *prob)
			continue
		}
		if n.dir && n.full == absRoot {
			accepted = append(accepted, ".")
		} else {
			accepted = append(accepted, n.full)
		}
	}
	// ADR-122: ast-grep keeps the ignore files mrw also reads (.gitignore,
	// info/exclude), so it never opens what both skip; the sources only
	// ast-grep reads are turned off, and under NoIgnore every one is. The
	// flags go before --json: what follows it is the paths.
	args := []string{"-p", pattern}
	for _, src := range astGrepIgnoresOff(opt.NoIgnore) {
		args = append(args, "--no-ignore", src)
	}
	// --rewrite=R as one token: a replacement that starts with a dash is not
	// taken for an option. --json=stream, because the array form carries the
	// same fields but only after ast-grep has held every hit.
	if opt.Rewrites != nil {
		args = append(args, "--rewrite="+opt.Rewrite, "--json=stream")
	} else {
		args = append(args, "--json")
	}
	switch {
	case len(paths) == 0:
		args = append(args, ".")
	case len(accepted) == 0:
		return nil, problems, nil
	default:
		args = append(args, accepted...)
	}
	// ADR-074: through subproc, so the 2 s bound kills a wrapper's grandchild
	// too and does not wait on a pipe the grandchild still holds. ast-grep's
	// process group no longer hears the terminal's ^C, so mrw listens for it
	// and kills the group instead.
	sctx, stopSignals := subproc.Interruptible(context.Background())
	defer stopSignals()
	ctx, cancel := context.WithTimeout(sctx, astGrepTimeout)
	defer cancel()
	cmd := subproc.Command(ctx, "ast-grep", args...)
	cmd.Dir = absRoot
	// A rewrite plans from every hit, so an ast-grep that reports a path it could
	// not read (valid hits, exit 0, an ERROR line on stderr) is not a complete
	// answer: its diagnostics are kept, bounded as stdout is, and named below.
	var diag *boundedBuffer
	if opt.Rewrites != nil {
		diag = &boundedBuffer{limit: maxAstGrepBytes}
		cmd.Stderr = diag
	}
	out, cmdErr := subproc.Output(cmd, maxAstGrepBytes)
	if tooLarge := (*subproc.ErrOutputTooLarge)(nil); errors.As(cmdErr, &tooLarge) {
		return nil, nil, fmt.Errorf("ast-grep %w: narrow the pattern or name fewer paths", tooLarge)
	}
	if diag != nil && diag.over {
		return nil, nil, fmt.Errorf("ast-grep answered more than %d bytes on its error stream: narrow the pattern or name fewer paths", maxAstGrepBytes)
	}
	// ADR-120: on Windows a child that could not be put in its job was killed
	// before it ran, so its empty output is no answer. Without this the read
	// would report no matches, silently (the review of #325).
	if errors.Is(cmdErr, subproc.ErrNotContained) && ctx.Err() == nil {
		return nil, nil, fmt.Errorf("ast-grep %w", cmdErr)
	}
	// Only a run that ended badly is read for why: one that exited cleanly a
	// moment before a deadline or a signal answered, and its output stands
	// (review of #232; no test can reach that window).
	if cmdErr != nil {
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return nil, nil, ErrAstGrepTimeout
		case context.Canceled:
			return nil, nil, errAstGrepInterrupted
		}
	}
	// A rewrite plans from every hit, so a run that ended badly is no answer even
	// when its output parses: a killed run leaves a valid prefix of the stream,
	// whose missing files no completeness check can see (the Codex review of T2).
	// ast-grep 0.45.3's no-match is exit 1 with nothing on either stream.
	if opt.Rewrites != nil && cmdErr != nil {
		var ee *exec.ExitError
		noMatch := errors.As(cmdErr, &ee) && ee.ExitCode() == 1 && len(bytes.TrimSpace(out)) == 0 && (diag == nil || diag.buf.Len() == 0)
		if !noMatch {
			return nil, nil, fmt.Errorf("ast-grep did not finish its scan (%w), so its matches may be a part: nothing is planned", cmdErr)
		}
	}
	parse := parseAstGrepJSON
	if opt.Rewrites != nil {
		parse = parseAstGrepStream
	}
	hits, parseErr := parse(out)
	if parseErr != nil {
		if cmdErr != nil {
			return nil, nil, fmt.Errorf("ast-grep: %w", cmdErr)
		}
		return nil, nil, parseErr
	}
	// Every line ast-grep printed on stderr is a path or a rule it could not
	// serve; its hits are still valid, so the answer is partial and each line is
	// named. Any line counts: the cost of a harmless one is a refused rewrite.
	if diag != nil {
		for _, l := range strings.Split(diag.buf.String(), "\n") {
			if l = strings.TrimSpace(l); l != "" {
				problems = append(problems, Problem{Path: "ast-grep", Reason: l})
			}
		}
	}
	// ADR-122: each hit passes the walk's own judgement — the ignore rules of
	// the checkout it is in, a binary file, a .git directory — unless its file
	// was named, as --grep serves a named file whatever the rules say. What is
	// dropped is counted, as the walk counts what it skips.
	var judge *hitJudge
	if !opt.NoIgnore {
		judge = newHitJudge(root, absRoot)
	}
	dropped := map[string]bool{}
	grouped := map[string][]Range{}
	order := []string{}
	named, starts := astGrepStarts(absRoot, accepted)
	if judge != nil {
		// A named directory is entered, so an ignored one at or above it is
		// not counted, as the walk counts (skipCounts).
		for _, s := range starts {
			if s != "." && s != "" {
				judge.w.starts = append(judge.w.starts, s)
			}
		}
	}
	res := rooted.NewResolver(absRoot) // this call's hits only (ADR-131)
	crOnly := map[string]bool{}
	for _, h := range hits {
		rel, ok := astGrepRel(absRoot, h.name())
		if !ok {
			problems = append(problems, Problem{Path: h.name(), Reason: "is outside the root " + absRoot})
			continue
		}
		// Every hit passes the boundary before anything here opens it: the
		// CR-only probe below read a file the boundary refuses (ADR-081, the
		// reviews of #243; ADR-077 dropped only a hit in mrw's own state). A
		// discovered hit it refuses is dropped, as the walk drops one (ADR-007
		// rule 2); one the caller named is left for Run, which reports it.
		_, resolveErr := res.Resolve(filepath.FromSlash(rel))
		refused := resolveErr != nil
		if refused && !named[rel] {
			continue
		}
		// A rewrite has no Run to report a named path the boundary refuses:
		// said here, once per file, and the file is left out of the plan.
		if refused && opt.Rewrites != nil {
			if !crOnly[rel] {
				problems = append(problems, Problem{Path: rel, Reason: resolveErr.Error()})
				crOnly[rel] = true
			}
			continue
		}
		if dropped[rel] {
			continue
		}
		// --exclude first, as the walk drops an excluded path before it asks
		// the ignore rules: an excluded file is not counted as skipped.
		if astGrepExcluded(rel, exclude, named, starts) {
			continue
		}
		if judge != nil && !named[rel] && judge.skip(rel, startDepth(starts, rel)) {
			dropped[rel] = true
			continue
		}
		// The Codex review of v1.42.0..v1.47.0: --no-ignore turns off the
		// ignore rules and the binary skip, never .git, which the walk prunes
		// whatever the flag says.
		if judge == nil && !named[rel] && underGit(rel, startDepth(starts, rel)) {
			dropped[rel] = true
			continue
		}
		// ADR-065: ast-grep numbers rows by "\n". On a CR-only file that is
		// not mrw's numbering, so its row would be served as a different line;
		// the file is reported once instead of served wrong.
		cr, known := crOnly[rel]
		if !known && !refused {
			b, err := readCapped(filepath.Join(absRoot, filepath.FromSlash(rel)))
			if over := (errOverFileCap{}); errors.As(err, &over) {
				// ADR-104: the probe reads the file whole, so it is bounded as
				// read is; the hit is reported once and not served.
				problems = append(problems, Problem{Path: rel, Reason: over.Error()})
				crOnly[rel] = true
				continue
			}
			// A rewrite plans from the bytes read here, so a file it cannot read is
			// reported here, once, where a read leaves that to Run.
			if opt.Rewrites != nil && err != nil {
				problems = append(problems, Problem{Path: rel, Reason: err.Error()})
				crOnly[rel] = true
				continue
			}
			if judge != nil && !named[rel] && err == nil && lines.Unsplittable(b) != "" {
				judge.binary(rel)
				dropped[rel] = true
				continue
			}
			_, eol, _ := lines.Split(string(b))
			cr = err == nil && eol == "\r"
			crOnly[rel] = cr
			if cr {
				problems = append(problems, Problem{Path: rel, Reason: "ends its lines with \\r alone, and ast-grep numbers rows by \\n, so its hit would name a different line"})
			} else if opt.Rewrites != nil {
				opt.Rewrites[rel] = &RewriteFile{Data: b}
			}
		}
		if cr {
			continue
		}
		start := h.Range.Start.Line + 1
		end := h.Range.End.Line + 1
		if start < 1 {
			start = 1
		}
		if end < start {
			end = start
		}
		text := strconv.Itoa(start)
		if end != start {
			text = fmt.Sprintf("%d-%d", start, end)
		}
		if _, seen := grouped[rel]; !seen {
			order = append(order, rel)
		}
		grouped[rel] = append(grouped[rel], Range{Start: start, End: end, Text: text})
		if rf := opt.Rewrites[rel]; rf != nil {
			rh := RewriteHit{Text: h.Text, Replacement: h.Replacement}
			if bo := h.Range.ByteOffset; bo != nil {
				rh.Start, rh.End = bo.Start, bo.End
			}
			rf.Hits = append(rf.Hits, rh)
		}
		if judge != nil {
			judge.served(rel)
		}
	}
	sort.Strings(order)
	specs := make([]Spec, 0, len(order))
	for _, rel := range order {
		specs = append(specs, Spec{Path: rel, Raw: rel, Ranges: grouped[rel]})
	}
	if judge != nil && opt.Skipped != nil {
		*opt.Skipped = judge.counts()
	}
	return specs, problems, nil
}

func parseAstGrepJSON(out []byte) ([]astGrepHit, error) {
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	var hits []astGrepHit
	if err := json.Unmarshal([]byte(s), &hits); err == nil {
		return hits, nil
	}
	return nil, fmt.Errorf("ast-grep: output is not a JSON array of hits")
}

// parseAstGrepStream reads --json=stream: one JSON object per hit, in the
// order ast-grep found them.
func parseAstGrepStream(out []byte) ([]astGrepHit, error) {
	dec := json.NewDecoder(bytes.NewReader(out))
	var hits []astGrepHit
	for {
		var h astGrepHit
		err := dec.Decode(&h)
		if err == io.EOF {
			return hits, nil
		}
		if err != nil {
			return nil, fmt.Errorf("ast-grep: output is not a stream of JSON hits")
		}
		hits = append(hits, h)
	}
}

func astGrepRel(absRoot, file string) (string, bool) {
	if file == "" {
		return "", false
	}
	full := file
	if !rooted.IsRooted(file) {
		full = filepath.Join(absRoot, file)
	}
	cleaned := filepath.Clean(full)
	cleaned = rooted.Real(cleaned) // through junctions too, as the root was (ADR-103)
	rel, err := filepath.Rel(absRoot, cleaned)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

func pathExcluded(rel string, globs []string) bool {
	base := path.Base(rel)
	for _, g := range globs {
		if ok, err := path.Match(g, rel); err == nil && ok {
			return true
		}
		if ok, err := path.Match(g, base); err == nil && ok {
			return true
		}
	}
	return false
}

// astGrepStarts splits the caller's paths the way read.Walk treats them
// (ADR-007:206-210, ADR-064): a named regular file is exempt from --exclude,
// and a named directory is a walk start that is never itself tested. "." names
// the root. Both are normalised through astGrepRel, so a named path and a hit
// compare in one root-relative form.
func astGrepStarts(absRoot string, paths []string) (map[string]bool, []string) {
	named := map[string]bool{}
	var starts []string
	for _, p := range paths {
		rel, ok := astGrepRel(absRoot, p)
		if !ok {
			continue
		}
		st, err := os.Stat(filepath.Join(absRoot, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		if st.Mode().IsRegular() {
			named[rel] = true
		} else if st.IsDir() {
			starts = append(starts, rel)
		}
	}
	return named, starts
}

// astGrepExcluded reports whether --exclude drops a hit, by read.Walk's rule.
// A named file never. Otherwise the hit, and every ancestor directory strictly
// below a start that contains it, meet the glob — and the hit is kept if ANY
// containing start admits it, so the order paths were named in cannot matter.
// A hit no named directory contains is tested from the root, as a walk with
// nothing named would test it.
func astGrepExcluded(rel string, exclude []string, named map[string]bool, starts []string) bool {
	if len(exclude) == 0 || named[rel] {
		return false
	}
	if pathExcluded(rel, exclude) {
		return true
	}
	var containing []string
	for _, s := range starts {
		if s == "." || strings.HasPrefix(rel, s+"/") {
			containing = append(containing, s)
		}
	}
	if len(containing) == 0 {
		containing = []string{"."}
	}
	for _, s := range containing {
		if !ancestorExcluded(rel, s, exclude) {
			return false
		}
	}
	return true
}

// ancestorExcluded reports whether a directory strictly between start and rel
// matches a glob: the directories a walk from start would meet on its way down.
func ancestorExcluded(rel, start string, exclude []string) bool {
	for dir := path.Dir(rel); dir != "." && dir != start; dir = path.Dir(dir) {
		if pathExcluded(dir, exclude) {
			return true
		}
	}
	return false
}

// astGrepIgnoresOff names the ast-grep ignore sources to turn off (ADR-122).
// mrw's walk reads neither .ignore files nor core.excludesFile and serves
// hidden files, so those are always off; .gitignore, its parents and
// info/exclude stay on unless the caller asked for every file.
func astGrepIgnoresOff(noIgnore bool) []string {
	off := []string{"hidden", "dot", "global"}
	if noIgnore {
		off = append(off, "exclude", "parent", "vcs")
	}
	return off
}

// startDepth is how many components the named directory a hit was found
// under has: the deepest start rel is below, or 0 under the root.
func startDepth(starts []string, rel string) int {
	best := 0
	for _, s := range starts {
		if s == "." || s == "" {
			continue
		}
		if strings.HasPrefix(rel, s+"/") {
			if n := strings.Count(s, "/") + 1; n > best {
				best = n
			}
		}
	}
	return best
}
