package read

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
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

// astGrepHit is the slice of ast-grep --json this mapper needs. Lines are
// 0-based, the way the CLI documents them.
type astGrepHit struct {
	File  string `json:"file"`
	Path  string `json:"path"`
	Range struct {
		Start struct {
			Line int `json:"line"`
		} `json:"start"`
		End struct {
			Line int `json:"line"`
		} `json:"end"`
	} `json:"range"`
}

func (h astGrepHit) name() string {
	if h.File != "" {
		return h.File
	}
	return h.Path
}

// AstGrep shells out to ast-grep on PATH and maps hits to Specs read.Run
// already serves. It records NOTHING: its process is finding, Run's read is
// the only one that observes (ADR-007). It is a separate primitive from Walk
// on purpose — Walk's go/no-go forbade a fifth matching rule.
func AstGrep(root string, paths []string, pattern string, exclude []string) ([]Spec, []Problem, error) {
	if _, err := exec.LookPath("ast-grep"); err != nil {
		return nil, nil, ErrAstGrepMissing
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	if real, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = real
	}
	args := []string{"-p", pattern, "--json"}
	if len(paths) == 0 {
		args = append(args, ".")
	} else {
		args = append(args, paths...)
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
	out, cmdErr := subproc.Output(cmd)
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
	hits, parseErr := parseAstGrepJSON(out)
	if parseErr != nil {
		if cmdErr != nil {
			return nil, nil, fmt.Errorf("ast-grep: %v", cmdErr)
		}
		return nil, nil, parseErr
	}
	grouped := map[string][]Range{}
	order := []string{}
	var problems []Problem
	named, starts := astGrepStarts(absRoot, paths)
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
		_, resolveErr := rooted.Resolve(absRoot, filepath.FromSlash(rel))
		refused := resolveErr != nil
		if refused && !named[rel] {
			continue
		}
		if astGrepExcluded(rel, exclude, named, starts) {
			continue
		}
		// ADR-065: ast-grep numbers rows by "\n". On a CR-only file that is
		// not mrw's numbering, so its row would be served as a different line;
		// the file is reported once instead of served wrong.
		cr, known := crOnly[rel]
		if !known && !refused {
			b, err := os.ReadFile(filepath.Join(absRoot, filepath.FromSlash(rel)))
			_, eol, _ := lines.Split(string(b))
			cr = err == nil && eol == "\r"
			crOnly[rel] = cr
			if cr {
				problems = append(problems, Problem{Path: rel, Reason: "ends its lines with \\r alone, and ast-grep numbers rows by \\n, so its hit would name a different line"})
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
	}
	sort.Strings(order)
	specs := make([]Spec, 0, len(order))
	for _, rel := range order {
		specs = append(specs, Spec{Path: rel, Raw: rel, Ranges: grouped[rel]})
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

func astGrepRel(absRoot, file string) (string, bool) {
	if file == "" {
		return "", false
	}
	full := file
	if !rooted.IsRooted(file) {
		full = filepath.Join(absRoot, file)
	}
	cleaned := filepath.Clean(full)
	if real, err := filepath.EvalSymlinks(cleaned); err == nil {
		cleaned = real
	}
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
