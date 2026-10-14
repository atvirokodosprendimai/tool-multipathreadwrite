package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
		ok, err := matches(dir, m)
		if err != nil {
			return v, err
		}
		if !ok {
			v.Missing = append(v.Missing, fmt.Sprintf("%s: no match for %s", m.File, m.Match))
		}
	}
	for _, m := range t.Forbid {
		ok, err := matches(dir, m)
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
	v.Solved = len(v.Missing)+len(v.Outside)+len(v.Structure) == 0
	return v, nil
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

func matches(dir string, m Match) (bool, error) {
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(m.File)))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return regexp.MustCompile(m.Match).Match(b), nil
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
