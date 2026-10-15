package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc"
)

// Verdict is what the oracle found. Solved holds only when every list is empty.
type Verdict struct {
	Solved    bool     `json:"solved"`
	Missing   []string `json:"missing,omitempty"`   // a require not met, or a forbid matched
	Outside   []string `json:"outside,omitempty"`   // a change outside the allowed regions
	Structure []string `json:"structure,omitempty"` // a changed file its structural check refuses
}

// Oracle grades the tree at dir, a git work tree whose HEAD is the task's
// base. It consults no model and no rater (ADR-156 Decision 3): the required
// change is a regular expression over the finished file, and anything changed
// outside the reference fix's regions, widened by the task's slack, is named.
func Oracle(t Task, dir string) (Verdict, error) {
	var v Verdict
	changed, err := changes(dir)
	if err != nil {
		return v, err
	}
	for _, c := range changed {
		if t.TestPackage != "" { // its tests judge the package; leaving it, or deleting a file, is outside
			switch {
			case path.Dir(c.file) != t.TestPackage:
				v.Outside = append(v.Outside, c.file+": a file outside the package whose tests grade the task")
			case c.whole:
				v.Outside = append(v.Outside, c.file+": deleted, made binary or changed in mode, which no task asks for")
			}
			continue
		}
		if c.untracked {
			if !t.created[c.file] {
				v.Outside = append(v.Outside, c.file+": a file the task does not create")
			}
			continue
		}
		if _, ok := t.regions[c.file]; !ok {
			v.Outside = append(v.Outside, c.file+": a file the task does not touch")
			continue
		}
		if c.whole {
			v.Outside = append(v.Outside, c.file+": deleted, made binary or changed in mode, which no task asks for")
			continue
		}
		for _, h := range c.hunks {
			if !t.allows(c.file, h) {
				v.Outside = append(v.Outside, fmt.Sprintf("%s: base lines %d-%d", c.file, h.Start, h.Start+max(h.Count, 1)-1))
			}
		}
	}
	// An exact task's touched files must be the reference result itself: a
	// line it adds kept inside a block comment, or beside a deleted call site,
	// still has every token and is not the fix (the reviews of #411).
	if t.Exact {
		for _, f := range sortedKeys(t.regions) {
			base, err := gitOut(dir, "show", "HEAD:"+f)
			if err != nil {
				return v, err
			}
			got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
			if err != nil || string(got) != reference(base, t.regions[f]) {
				v.Missing = append(v.Missing, f+": is not the reference result")
			}
		}
	}
	for _, m := range t.Require {
		ok, err := matches(dir, m, t.TestPackage != "")
		if err != nil {
			return v, err
		}
		if !ok {
			v.Missing = append(v.Missing, fmt.Sprintf("%s: no match for %s", m.File, m.Match))
		}
	}
	for _, m := range t.Forbid {
		ok, err := matches(dir, m, t.TestPackage != "")
		if err != nil {
			return v, err
		}
		if ok {
			v.Missing = append(v.Missing, fmt.Sprintf("%s: still matches %s", m.File, m.Match))
		}
	}
	for _, c := range changed {
		if msg := structural(filepath.Join(dir, c.file)); msg != "" {
			v.Structure = append(v.Structure, c.file+": "+msg)
		}
	}
	if t.TestPackage != "" && len(v.Missing)+len(v.Outside)+len(v.Structure) == 0 {
		msg, err := packageTests(dir, t.TestPackage, t.AssertingTests)
		if err != nil {
			return v, err
		}
		if msg != "" {
			v.Missing = append(v.Missing, msg)
		}
	}
	v.Solved = len(v.Missing)+len(v.Outside)+len(v.Structure) == 0
	return v, nil
}

// testBudget bounds a check-coupled attempt's go test: a hung test is
// unsolved, never a stage held.
const testBudget = 10 * time.Minute

// packageTests runs go test of pkg in the tree at dir and returns "" when it
// passes and every asserting test ran and passed, or what the oracle reports
// missing: the package and a test that did not, or the last line the run
// printed. Deleting or skipping the test that asserts the literal is not a
// fix (the review of #434). The run's test flags are the oracle's, never the
// host's (gradingEnv): GOFLAGS=-run=^$ would run no test. A signal to bench stops it and
// grading reports errGradingInterrupted, as the SWE-bench harness's does
// (#425); the signal context is the deadline's parent, so the two are told
// apart.
func packageTests(dir, pkg string, asserting []string) (string, error) {
	sigCtx, stopSignals := subproc.Interruptible(context.Background())
	defer stopSignals()
	ctx, cancel := context.WithTimeout(sigCtx, testBudget)
	defer cancel()
	sink := &testSink{passed: map[string]bool{}}
	cmd := subproc.Command(ctx, "go", "test", "-json", "-count=1", "./"+pkg)
	cmd.Dir, cmd.Stdout, cmd.Stderr = dir, sink, sink
	env, err := gradingEnv(ctx, dir)
	if err != nil {
		if sigCtx.Err() != nil {
			return "", errGradingInterrupted
		}
		return "", err
	}
	cmd.Env = env
	err = subproc.Run(cmd)
	switch {
	case sigCtx.Err() != nil:
		return "", errGradingInterrupted
	case ctx.Err() == context.DeadlineExceeded:
		return fmt.Sprintf("go test ./%s: ran past %s", pkg, testBudget), nil
	}
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		return "", fmt.Errorf("go test ./%s: %w", pkg, err)
	}
	sink.flush()
	if err != nil {
		return fmt.Sprintf("go test ./%s fails: %s", pkg, sink.last), nil
	}
	for _, name := range asserting {
		if !sink.passed[name] {
			return fmt.Sprintf("go test ./%s: %s did not run and pass", pkg, name), nil
		}
	}
	return "", nil
}

// gradingEnv is the host's environment for grading at dir, with what would
// change which tests run taken out: GOWORK off, and GOFLAGS cut to the -mod
// flags of its effective value, from the environment or the go env file, in
// their order (the last one wins, as go reads them). The go env file stays,
// for its proxy and private-module settings, so GOFLAGS is never left empty —
// an empty value falls back to that file — and starts with -modcacherw, which
// changes nothing graded. -modfile is dropped: it names a host file a graded
// tree should not depend on. The lookup runs under the grading context with
// GOWORK off, and a failure is a grading error, never module flags silently
// dropped (the reviews of #434).
func gradingEnv(ctx context.Context, dir string) ([]string, error) {
	env := append(os.Environ(), "GOWORK=off")
	var out bytes.Buffer
	cmd := subproc.Command(ctx, "go", "env", "GOFLAGS")
	cmd.Dir, cmd.Env, cmd.Stdout = dir, env, &out
	if err := subproc.Run(cmd); err != nil {
		return nil, fmt.Errorf("go env GOFLAGS: %w", err)
	}
	fields, err := splitGoFlags(strings.TrimSpace(out.String()))
	if err != nil {
		return nil, fmt.Errorf("go env GOFLAGS: %w", err)
	}
	keep := []string{"-modcacherw"}
	for _, f := range fields {
		if strings.HasPrefix(f, "--") {
			f = f[1:]
		}
		// Only go's three module modes: a quoted field such as
		// '-mod=mod -run=x' would come apart again when joined (the fourth
		// review of #434).
		if f == "-mod=mod" || f == "-mod=readonly" || f == "-mod=vendor" {
			keep = append(keep, f)
		}
	}
	return append(env, "GOFLAGS="+strings.Join(keep, " ")), nil
}

// splitGoFlags splits GOFLAGS as go does (cmd/internal/quoted): fields
// separated by spaces, a field wholly quoted with ' or " taken as one, its
// quotes removed, so text inside one flag's quoted value is never a flag of
// its own (the third review of #434).
func splitGoFlags(s string) ([]string, error) {
	var f []string
	for {
		s = strings.TrimLeft(s, " \t\n\r")
		if s == "" {
			return f, nil
		}
		if q := s[0]; q == '"' || q == '\'' {
			end := strings.IndexByte(s[1:], q)
			if end < 0 {
				return nil, fmt.Errorf("unterminated %c string", q)
			}
			f, s = append(f, s[1:1+end]), s[2+end:]
			continue
		}
		end := strings.IndexAny(s, " \t\n\r")
		if end < 0 {
			end = len(s)
		}
		f, s = append(f, s[:end]), s[end:]
	}
}

// maxTestLine bounds what the oracle keeps of one line go test prints.
const maxTestLine = 64 << 10

// testSink reads `go test -json` as it arrives, a line at a time, keeping
// which top-level tests passed and the last line of output, each bounded, so
// a noisy test cannot fill memory before its deadline (the review of #434).
type testSink struct {
	partial []byte
	passed  map[string]bool
	last    string
}

func (s *testSink) Write(p []byte) (int, error) {
	for _, b := range p {
		if b != '\n' {
			if len(s.partial) < maxTestLine {
				s.partial = append(s.partial, b)
			}
			continue
		}
		s.flush()
	}
	return len(p), nil
}

// flush reads the line held so far, an event or plain output.
func (s *testSink) flush() {
	line := s.partial
	s.partial = s.partial[:0]
	var e struct{ Action, Test, Output string }
	text := string(line)
	if json.Unmarshal(line, &e) == nil {
		if e.Test != "" && !strings.Contains(e.Test, "/") {
			switch e.Action {
			case "pass":
				s.passed[e.Test] = true
			case "fail", "skip":
				s.passed[e.Test] = false
			}
		}
		text = e.Output
	}
	if t := strings.TrimSpace(text); t != "" {
		if len(t) > 300 {
			t = t[:300] + "…"
		}
		s.last = t
	}
}

// allows says whether a change the agent made lies inside the task's regions.
func (t Task) allows(file string, h hunk) bool {
	if h.Count == 0 {
		return t.coversInsertion(file, h.Start)
	}
	for n := h.Start; n < h.Start+h.Count; n++ {
		if !t.CoversBaseLine(file, n) {
			return false
		}
	}
	return true
}

type change struct {
	file      string
	untracked bool
	whole     bool // deleted, binary or a mode change: no line of it lies in a region
	hunks     []hunk
}

// changes lists what differs from HEAD in the tree: each tracked file git
// names as changed, with its hunks in base numbering, and the untracked files
// git does not ignore. A file deleted, made binary or changed only in mode has
// no hunks to judge and is marked whole.
func changes(dir string) ([]change, error) {
	names, err := gitOut(dir, "diff", "--name-status", "-z", "--no-renames", "HEAD")
	if err != nil {
		return nil, err
	}
	var cs []change
	fields := strings.Split(strings.TrimSuffix(names, "\x00"), "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		status, file := fields[i], fields[i+1]
		c := change{file: file, whole: status == "D" || status == "T"}
		if !c.whole {
			out, err := gitOut(dir, "diff", "-U0", "--no-color", "--no-ext-diff", "--no-renames", "HEAD", "--", file)
			if err != nil {
				return nil, err
			}
			for _, l := range strings.Split(out, "\n") {
				if strings.HasPrefix(l, "Binary files ") || strings.HasPrefix(l, "old mode ") || strings.HasPrefix(l, "new mode ") {
					c.whole = true
				}
				if strings.HasPrefix(l, "@@ ") {
					m := hunkHeader.FindStringSubmatch(l)
					if m == nil {
						return nil, fmt.Errorf("unreadable hunk header %q", l)
					}
					start, _ := strconv.Atoi(m[1])
					count := 1
					if m[2] != "" {
						count, _ = strconv.Atoi(m[2])
					}
					c.hunks = append(c.hunks, hunk{Start: start, Count: count})
				}
			}
			if len(c.hunks) == 0 {
				c.whole = true
			}
		}
		cs = append(cs, c)
	}
	others, err := gitOut(dir, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, f := range strings.Split(strings.TrimSuffix(others, "\x00"), "\x00") {
		if f != "" {
			cs = append(cs, change{file: f, untracked: true})
		}
	}
	return cs, nil
}

// reference is base with the reference fix's hunks applied: each replaces its
// lines, or is inserted after its line, in base numbering.
func reference(base string, hs []hunk) string {
	lines := strings.SplitAfter(base, "\n")
	var out []string
	next := 1 // the next base line to copy
	for _, h := range hs {
		end := h.Start // copy through this line, then the change
		if h.Count > 0 {
			end = h.Start - 1
		}
		for ; next <= end && next <= len(lines); next++ {
			out = append(out, lines[next-1])
		}
		for _, l := range h.Lines {
			out = append(out, l+"\n")
		}
		next += h.Count
	}
	for ; next <= len(lines); next++ {
		out = append(out, lines[next-1])
	}
	return strings.Join(out, "")
}

func sortedKeys(m map[string][]hunk) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func gitOut(dir string, args ...string) (string, error) {
	var out, stderr bytes.Buffer
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "core.quotepath=off", "-c", "core.autocrlf=false"}, args...)...)
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return out.String(), nil
}

// matches says whether m's pattern matches its file. With code set, a Go
// file's comments are blanked first, so a comment quoting the value is not
// the change (the review of #434).
func matches(dir string, m Match, code bool) (bool, error) {
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(m.File)))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if code && strings.HasSuffix(m.File, ".go") {
		b = codeOnly(b)
	}
	return regexp.MustCompile(m.Match).Match(b), nil
}

// codeOnly returns src with every comment's bytes but its newlines turned to
// spaces, as go/scanner finds them.
func codeOnly(src []byte) []byte {
	out := append([]byte(nil), src...)
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, scanner.ScanComments)
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			return out
		}
		if tok != token.COMMENT {
			continue
		}
		start := file.Offset(pos)
		end := start + 2
		if src[start+1] == '/' { // a line comment runs to the newline
			for end < len(src) && src[end] != '\n' {
				end++
			}
		} else if i := bytes.Index(src[end:], []byte("*/")); i >= 0 {
			end += i + 2
		} else {
			end = len(src)
		}
		for i := start; i < end; i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}
}

// structural is the check a changed file must pass for its kind, or "".
func structural(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "" // deleted; the regions judge that
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		if _, err := parser.ParseFile(token.NewFileSet(), path, b, parser.AllErrors); err != nil {
			return "does not parse as Go: " + err.Error()
		}
	case ".json":
		if !json.Valid(b) {
			return "is not valid JSON"
		}
	case ".yml", ".yaml":
		for i, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimLeft(l, " "), "\t") {
				return fmt.Sprintf("line %d is indented with a tab", i+1)
			}
		}
	case ".md":
		fences := 0
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimLeft(l, " "), "```") {
				fences++
			}
		}
		if fences%2 != 0 {
			return "has an unclosed code fence"
		}
	}
	return ""
}
