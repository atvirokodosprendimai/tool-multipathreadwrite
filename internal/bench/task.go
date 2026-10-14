// Package bench runs an agent CLI over a fixed task corpus and grades what it
// leaves behind, to measure whether mrw pays for itself (ADR-156). It never
// calls a model: it starts the CLI as a subprocess, reads what the CLI prints,
// and judges the tree with an oracle that knows the answer by construction.
package bench

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The shapes ADR-156 Decision 3 registers, and the CI-runnable fixtures that
// are never measured.
const (
	ShapeMultiSite      = "multi-site"
	ShapeStructural     = "structural"
	ShapeRepeatedAnchor = "repeated-anchor"
	ShapeLargeFile      = "large-file"
	ShapeSingleLine     = "single-line"
	ShapeFixture        = "fixture"
)

// Match is one regular expression the oracle checks against one file of the
// finished tree.
type Match struct {
	File  string `json:"file"`
	Match string `json:"match"`
}

// Task is one item of the corpus: what the agent is told, where its tree comes
// from, and how the oracle grades the result. Its allowed regions are not
// written by hand: they are the reference fix's hunks, widened by Slack lines.
type Task struct {
	ID          string  `json:"id"`
	Shape       string  `json:"shape"`
	Repo        string  `json:"repo"`
	Commit      string  `json:"commit,omitempty"`
	Instruction string  `json:"instruction"`
	Slack       int     `json:"slack,omitempty"`
	Exact       bool    `json:"exact,omitempty"`
	Require     []Match `json:"require"`
	Forbid      []Match `json:"forbid,omitempty"`

	Dir     string            `json:"-"`
	regions map[string][]hunk // base-numbered hunks of the reference fix, by file
	created map[string]bool   // files the reference fix creates
}

// hunk is one change in base numbering: Count lines from Start are replaced,
// or, when Count is 0, lines are inserted after line Start. Added is how many
// lines the change leaves in their place, and Lines, for the reference fix, are
// those lines.
type hunk struct {
	Start, Count, Added int
	Lines               []string
}

// LoadTask reads DIR/task.json and DIR/reference.patch.
func LoadTask(dir string) (Task, error) {
	var t Task
	b, err := os.ReadFile(filepath.Join(dir, "task.json"))
	if err != nil {
		return t, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return t, fmt.Errorf("%s: %w", dir, err)
	}
	t.Dir = dir
	if t.ID != filepath.Base(dir) {
		return t, fmt.Errorf("%s: id %q is not the directory's name", dir, t.ID)
	}
	if t.Repo == "" || t.Instruction == "" || len(t.Require)+len(t.Forbid) == 0 {
		return t, fmt.Errorf("%s: a task needs repo, instruction and at least one require or forbid", dir)
	}
	if (t.Repo == "fixture") != (t.Shape == ShapeFixture) {
		return t, fmt.Errorf("%s: a fixture task, and only one, uses repo \"fixture\"", dir)
	}
	if t.Repo != "fixture" && !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(t.Commit) {
		return t, fmt.Errorf("%s: commit must be a full sha", dir)
	}
	for _, m := range append(append([]Match(nil), t.Require...), t.Forbid...) {
		if _, err := regexp.Compile(m.Match); err != nil {
			return t, fmt.Errorf("%s: %s: %w", dir, m.File, err)
		}
	}
	patch, err := os.ReadFile(filepath.Join(dir, "reference.patch"))
	if err != nil {
		return t, err
	}
	t.regions, t.created, err = parsePatch(patch)
	if err != nil {
		return t, fmt.Errorf("%s: reference.patch: %w", dir, err)
	}
	if len(t.regions) == 0 {
		return t, fmt.Errorf("%s: reference.patch changes nothing", dir)
	}
	return t, nil
}

// LoadTasks loads every task directory under root, sorted by id.
func LoadTasks(root string) ([]Task, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var ts []Task
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		t, err := LoadTask(filepath.Join(root, e.Name()))
		if err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].ID < ts[j].ID })
	return ts, nil
}

// CoversBaseLine says whether replacing or deleting base line n of file is
// inside an allowed region: a line the reference fix itself replaced. Slack
// never widens this: a line the fix left alone is one the task says to keep,
// so changing it is outside however close it sits (the review of #411).
func (t Task) CoversBaseLine(file string, n int) bool {
	if t.created[file] {
		return true
	}
	for _, h := range t.regions[file] {
		if h.Count > 0 && n >= h.Start && n <= h.Start+h.Count-1 {
			return true
		}
	}
	return false
}

// coversInsertion says whether lines inserted after base line n are inside an
// allowed region: at the reference fix's own insertion point, or anywhere
// from just before to the end of a stretch it replaced, widened by Slack. Slack
// moves where new lines may go; it never licenses changing an existing one.
func (t Task) coversInsertion(file string, n int) bool {
	if t.created[file] {
		return true
	}
	for _, h := range t.regions[file] {
		lo, hi := h.Start-1-t.Slack, h.Start+h.Count-1+t.Slack
		if h.Count == 0 {
			lo, hi = h.Start-t.Slack, h.Start+t.Slack
		}
		if n >= lo && n <= hi {
			return true
		}
	}
	return false
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+\d+(?:,(\d+))? @@`)

// parsePatch reads a unified diff as git writes it and returns each file's
// hunks in base numbering, with the lines each adds, and the files it creates.
// A hunk's body is consumed by its counts, so a changed line that itself begins
// "--- " is not taken for a file header.
func parsePatch(b []byte) (map[string][]hunk, map[string]bool, error) {
	regions := map[string][]hunk{}
	created := map[string]bool{}
	file, fromNull := "", false
	remOld, remNew := 0, 0
	// A run is a stretch of - and + lines between context lines; it is one
	// region, in base numbering, so a fix's context lines widen nothing.
	oldLine, runStart, runDel, runAdd := 0, 0, 0, 0
	var runLines []string
	flush := func() {
		switch {
		case runDel > 0:
			regions[file] = append(regions[file], hunk{Start: runStart, Count: runDel, Added: runAdd, Lines: runLines})
		case runAdd > 0:
			regions[file] = append(regions[file], hunk{Start: runStart - 1, Count: 0, Added: runAdd, Lines: runLines})
		}
		runDel, runAdd, runLines = 0, 0, nil
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		l := sc.Text()
		if remOld > 0 || remNew > 0 {
			switch {
			case strings.HasPrefix(l, " "):
				flush()
				remOld--
				remNew--
				oldLine++
			case strings.HasPrefix(l, "-"):
				if runDel+runAdd == 0 {
					runStart = oldLine
				}
				runDel++
				remOld--
				oldLine++
			case strings.HasPrefix(l, "+"):
				if runDel+runAdd == 0 {
					runStart = oldLine
				}
				runAdd++
				remNew--
				runLines = append(runLines, l[1:])
			case strings.HasPrefix(l, `\`):
			default:
				return nil, nil, fmt.Errorf("hunk body line %q", l)
			}
			if remOld == 0 && remNew == 0 {
				flush()
			}
			continue
		}
		switch {
		case strings.HasPrefix(l, "diff --git "):
			file, fromNull = "", false
		case strings.HasPrefix(l, "--- "):
			fromNull = l == "--- /dev/null"
		case strings.HasPrefix(l, "+++ "):
			if l == "+++ /dev/null" {
				return nil, nil, fmt.Errorf("a reference fix may not delete a file")
			}
			file = strings.TrimPrefix(strings.TrimPrefix(l, "+++ "), "b/")
			created[file] = fromNull
			if _, ok := regions[file]; !ok {
				regions[file] = nil
			}
		case strings.HasPrefix(l, "@@ "):
			m := hunkHeader.FindStringSubmatch(l)
			if m == nil || file == "" {
				return nil, nil, fmt.Errorf("unreadable hunk header %q", l)
			}
			start, _ := strconv.Atoi(m[1])
			count, added := 1, 1
			if m[2] != "" {
				count, _ = strconv.Atoi(m[2])
			}
			if m[3] != "" {
				added, _ = strconv.Atoi(m[3])
			}
			// "-5,0" inserts after line 5: the first line it would number is 6.
			oldLine = start
			if count == 0 {
				oldLine = start + 1
			}
			remOld, remNew = count, added
		}
	}
	if remOld > 0 || remNew > 0 {
		return nil, nil, fmt.Errorf("the last hunk is cut short")
	}
	return regions, created, sc.Err()
}

// instructionNames are the files and directories an agent host reads as
// instructions; the runner strips them from every attempt's tree.
var instructionNames = map[string]bool{
	"CLAUDE.md": true, "CLAUDE.local.md": true, "AGENTS.md": true, ".mcp.json": true,
	".claude": true, ".cursor": true, ".codex": true, ".cursorrules": true,
}

// StripInstructions removes every instruction file and directory under root,
// at any depth.
func StripInstructions(root string) error {
	var doomed []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if p != root && instructionNames[info.Name()] {
			doomed = append(doomed, p)
			if info.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, p := range doomed {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}
