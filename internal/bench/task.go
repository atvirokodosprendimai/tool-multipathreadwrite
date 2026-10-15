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
	"path"
	"path/filepath"
	"regexp"
	"slices"
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
	// ShapeVariedMultiSite is a change at three or more places in two or more
	// files, one of whose old texts also occurs where it must stay, so a
	// global substitution gets it wrong (ADR-158 Decision 4).
	ShapeVariedMultiSite = "varied-multi-site"
	// ShapeWideRename and ShapeCheckCoupled are ADR-160's own-task categories:
	// a Go identifier renamed across 15 or more places in 8 or more files, and
	// a string literal changed where a test asserts it (ADR-160 Decision 1).
	ShapeWideRename   = "wide-rename"
	ShapeCheckCoupled = "check-coupled"
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
	// CheckRedAtBase says why the repository's own declared check fails at
	// the pinned commit in the tree bench builds, so configuration B's
	// automatic check (ADR-054) is red there whatever the agent does: a
	// measured fact, empty for every other task (ADR-156 T1 S6).
	CheckRedAtBase string `json:"check_red_at_base,omitempty"`
	// SupersededBy names the task that replaced this one after it was found
	// graded wrongly; the task itself is never edited, since receipts name it
	// (ADR-158 Decision 3).
	SupersededBy string `json:"superseded_by,omitempty"`
	// TestPackage is the directory, relative to the tree, whose go test grades
	// a check-coupled attempt in place of exact and the reference's regions:
	// solved when the require and forbid hold, nothing outside the directory
	// changed and its tests pass (Zy, "Grade by the tests", ADR-160 Decision 1).
	TestPackage string `json:"test_package,omitempty"`

	Dir     string            `json:"-"`
	regions map[string][]hunk // base-numbered hunks of the reference fix, by file
	created map[string]bool   // files the reference fix creates
	swe     *sweInstance      // set for a task of the SWE-bench slice (LoadSWEBench)
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
	if t.TestPackage != "" && (t.Shape != ShapeCheckCoupled || t.Exact || path.IsAbs(t.TestPackage) ||
		path.Clean(t.TestPackage) != t.TestPackage || t.TestPackage == ".." || strings.HasPrefix(t.TestPackage, "../")) {
		return t, fmt.Errorf("%s: test_package is a clean relative directory, on a check-coupled task only, never with exact", dir)
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
	ids := map[string]bool{}
	for _, t := range ts {
		ids[t.ID] = true
	}
	for _, t := range ts {
		if t.SupersededBy != "" && !ids[t.SupersededBy] {
			return nil, fmt.Errorf("%s: superseded_by names %q, which is not a task", t.Dir, t.SupersededBy)
		}
	}
	return ts, nil
}

// SelectTasks is the stage's corpus: the tasks named, or with none named every
// measured task — never a fixture, and never a superseded task (ADR-158
// Decision 3). Every task named must exist.
func SelectTasks(all []Task, only []string) ([]Task, error) {
	var out []Task
	for _, t := range all {
		if t.Shape == ShapeFixture {
			continue
		}
		if len(only) == 0 && t.SupersededBy == "" || slices.Contains(only, t.ID) {
			out = append(out, t)
		}
	}
	if len(only) > 0 && len(out) != len(only) {
		return nil, fmt.Errorf("--task names %d task(s), %d found", len(only), len(out))
	}
	return out, nil
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
// instructions or loads as configuration (ADR-156 Decision 2).
var instructionNames = map[string]bool{
	"CLAUDE.md": true, "CLAUDE.local.md": true, "AGENTS.md": true, ".mcp.json": true,
	".claude": true, ".cursor": true, ".codex": true, ".cursorrules": true,
}

// redactedNames are the instruction files that are plain text the tests of a
// repository may read: they are kept, without the lines that name mrw (Zy,
// 2026-10-14, after the pilot: the agentsmemory tests read its AGENTS.md, so a
// stripped tree failed the check of configuration B for a reason the harness
// made). The rest of instructionNames is configuration a host would load and
// run, such as settings, hooks, skills and MCP servers, and is removed; a
// .mcp.json only at the root, where a host reads one.
var redactedNames = map[string]bool{"CLAUDE.md": true, "CLAUDE.local.md": true, "AGENTS.md": true, ".cursorrules": true}

// namesMrw matches a line that names mrw: the three letters in any case with no
// letter or digit of any script on either side, so mrw_read and mrw-bench count
// and smrw or émrw do not. It is wider than the canary check, whose \bmrw\b an
// underscore would not end.
var namesMrw = regexp.MustCompile(`(?i)(^|[^\p{L}\p{N}])mrw([^\p{L}\p{N}]|\z)`)

// importRef is an @path import in an instruction file, which Claude Code loads
// beside the file that names it, up to five imports deep.
var importRef = regexp.MustCompile(`(?:^|\s)@([^\s` + "`" + `]+)`)

// PrepareInstructions readies the tree at root for an attempt: at any depth, an
// instruction file named in redactedNames loses every line that names mrw, and
// so does each file in the tree it imports; one that is a link into the tree
// stays a link and its target is redacted, one that leads elsewhere is
// removed. Every other instruction file or directory is removed, except a
// .mcp.json below the root.
func PrepareInstructions(root string) error {
	var doomed, keep []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if p == root || !instructionNames[info.Name()] {
			return nil
		}
		// A host reads project MCP configuration from the root of the tree only, and
		// --strict-mcp-config ignores even that: a .mcp.json deeper down is a file
		// of the project, such as a plugin manifest its tests read (agentsmemory's
		// clients/claude-code, measured 2026-10-14).
		if info.Name() == ".mcp.json" && filepath.Dir(p) != root {
			return nil
		}
		if redactedNames[info.Name()] {
			if _, ok := inTreeFile(root, p); ok {
				keep = append(keep, p)
				return nil
			}
		}
		doomed = append(doomed, p)
		if info.IsDir() {
			return filepath.SkipDir
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
	// An instruction file is resolved again once configuration is gone: one that
	// led into a removed directory leads nowhere now and goes too (the re-review
	// of #413).
	var redact []string
	for _, p := range keep {
		target, ok := inTreeFile(root, p)
		if !ok {
			if err := os.Remove(p); err != nil {
				return err
			}
			continue
		}
		redact = append(redact, target)
	}
	done := map[string]bool{}
	for depth := 0; depth <= 5 && len(redact) > 0; depth++ {
		var next []string
		for _, p := range redact {
			if done[p] {
				continue
			}
			done[p] = true
			imports, err := redactMrw(p)
			if err != nil {
				return err
			}
			for _, ref := range imports {
				if !filepath.IsAbs(ref) {
					ref = filepath.Join(filepath.Dir(p), ref)
				}
				if target, ok := inTreeFile(root, ref); ok {
					next = append(next, target)
				}
			}
		}
		redact = next
	}
	return nil
}

// inTreeFile returns the regular file p leads to, through any links, when that
// file is inside root.
func inTreeFile(root, p string) (string, bool) {
	target, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(target)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	top, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(top, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return target, true
}

// redactMrw rewrites p without the lines that name mrw, keeping every other
// byte, and returns the @path imports the kept lines name outside a code block
// or a code span, which Claude Code reads as examples rather than imports.
func redactMrw(p string) ([]string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var kept strings.Builder
	var imports []string
	fence := ""
	for _, line := range strings.SplitAfter(string(b), "\n") {
		if namesMrw.MatchString(line) {
			continue
		}
		kept.WriteString(line)
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			continue
		}
		var prose []string
		for i, part := range strings.Split(line, "`") {
			if i%2 == 0 {
				prose = append(prose, part)
			}
		}
		for _, m := range importRef.FindAllStringSubmatch(strings.Join(prose, " "), -1) {
			imports = append(imports, m[1])
		}
	}
	return imports, os.WriteFile(p, []byte(kept.String()), 0o644)
}
