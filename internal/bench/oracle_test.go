package bench

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tasksDir is the corpus, two levels above this package.
func tasksDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.Abs(filepath.Join("..", "..", "bench", "tasks"))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func needGit(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bench runs on the operator's macOS or Linux machine; Windows is out of ADR-156's scope")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
}

// TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit proves, for every
// task and without an agent, the three properties ADR-156 Decision 3 needs of
// an oracle: it fails on the base, passes with the reference fix, and fails
// when the reference fix is joined by one change outside the allowed regions.
// A task whose repository is not on this machine at its pinned commit is
// skipped by name; the fixture tasks have no such dependency and run anywhere.
func TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit(t *testing.T) {
	needGit(t)
	tasks, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	ran := 0
	for _, task := range tasks {
		t.Run(task.ID, func(t *testing.T) {
			tree := t.TempDir()
			if _, err := Materialize(task, tree, task.GradedFiles()); err != nil {
				if errors.Is(err, errUnreachable) {
					t.Skipf("task %s: %v", task.ID, err)
				}
				t.Fatal(err)
			}
			v, err := Oracle(task, tree)
			if err != nil {
				t.Fatal(err)
			}
			if v.Solved || len(v.Missing) == 0 {
				t.Fatalf("the oracle does not fail on the base: %+v", v)
			}
			if err := ApplyPatch(tree, filepath.Join(task.Dir, "reference.patch")); err != nil {
				t.Fatal(err)
			}
			v, err = Oracle(task, tree)
			if err != nil {
				t.Fatal(err)
			}
			if !v.Solved {
				t.Fatalf("the oracle does not pass the reference fix: %+v", v)
			}
			// Deleting the line just before the fix's first change: the line
			// the task says to keep, which slack must not license (the
			// review of #411).
			if f, base, ok := task.neighbour(); ok && task.TestPackage == "" {
				nb := filepath.Join(tree, f)
				keep, err := os.ReadFile(nb)
				if err != nil {
					t.Fatal(err)
				}
				lines := strings.SplitAfter(string(keep), "\n")
				fixed := task.FixedLine(f, base)
				cut := append(append([]string(nil), lines[:fixed-1]...), lines[fixed:]...)
				if err := os.WriteFile(nb, []byte(strings.Join(cut, "")), 0o644); err != nil {
					t.Fatal(err)
				}
				if v, err := Oracle(task, tree); err != nil || v.Solved || len(v.Outside) == 0 {
					t.Fatalf("the oracle accepts the reference fix with %s:%d deleted: %+v %v", f, base, v, err)
				}
				if err := os.WriteFile(nb, keep, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			file, line := strayLine(t, task, tree)
			if task.TestPackage != "" { // inside the package its tests judge; leaving it is the stray edit
				file, line = "go.mod", 1
			}
			b, err := os.ReadFile(filepath.Join(tree, file))
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.SplitAfter(string(b), "\n")
			lines[line-1] = strings.TrimRight(lines[line-1], "\n") + " stray\n"
			if err := os.WriteFile(filepath.Join(tree, file), []byte(strings.Join(lines, "")), 0o644); err != nil {
				t.Fatal(err)
			}
			v, err = Oracle(task, tree)
			if err != nil {
				t.Fatal(err)
			}
			if v.Solved || len(v.Outside) == 0 {
				t.Fatalf("the oracle accepts the reference fix plus a change at %s:%d: %+v", file, line, v)
			}
			ran++
		})
	}
	if ran == 0 {
		t.Fatal("no task ran: the fixtures alone must run everywhere")
	}
}

// strayLine picks a line of a touched file that no allowed region of the task
// covers, counted in the fixed tree; the reference fix leaves lines outside its
// regions where they were, so the first such line is outside by construction.
func strayLine(t *testing.T, task Task, tree string) (string, int) {
	t.Helper()
	for _, f := range task.Files() {
		b, err := os.ReadFile(filepath.Join(tree, f))
		if err != nil {
			continue
		}
		n := strings.Count(string(b), "\n")
		for line := 1; line <= n; line++ {
			if fixed := task.FixedLine(f, line); !task.CoversBaseLine(f, line) && fixed <= n {
				return f, fixed
			}
		}
	}
	t.Fatalf("task %s: every line of every touched file is inside an allowed region", task.ID)
	return "", 0
}

// TestTheCorpusHasTheRegisteredShapeCounts holds the corpus to ADR-156's
// Decision 3 and ADR-158's Decision 3: among the tasks not superseded, 8
// multi-site, 6 structural, 6 repeated-anchor, 4 large-file, 6 single-line and
// 10 varied-multi-site tasks, at least one CI-runnable fixture; and for every
// task, superseded or not, instructions that name no tool and no reference fix
// that touches an instruction file the runner strips.
func TestTheCorpusHasTheRegisteredShapeCounts(t *testing.T) {
	tasks, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, task := range tasks {
		if task.SupersededBy == "" {
			got[task.Shape]++
		}
		low := strings.ToLower(task.Instruction)
		for _, tool := range []string{"mrw", "sed ", "the edit tool", "the read tool", "awk", "perl"} {
			if strings.Contains(low, tool) {
				t.Errorf("task %s: the instruction names a tool (%q)", task.ID, tool)
			}
		}
		for _, f := range task.Files() {
			if IsInstructionPath(f) {
				t.Errorf("task %s: the reference fix touches %s, which the runner strips", task.ID, f)
			}
		}
	}
	// wide-rename and check-coupled hold Stage 4's sample (10 and 8) and the
	// four calibration tasks of each that ADR-160 T3 runs on A alone and
	// bench/stage-4-tasks.txt never lists.
	want := map[string]int{ShapeMultiSite: 8, ShapeStructural: 6, ShapeRepeatedAnchor: 6, ShapeLargeFile: 4, ShapeSingleLine: 6, ShapeVariedMultiSite: 10, ShapeWideRename: 14, ShapeCheckCoupled: 12}
	for shape, n := range want {
		if got[shape] != n {
			t.Errorf("shape %s: %d tasks, want %d", shape, got[shape], n)
		}
	}
	if got[ShapeFixture] == 0 {
		t.Error("no fixture task: CI would run no oracle at all")
	}
	for shape := range got {
		if _, ok := want[shape]; !ok && shape != ShapeFixture {
			t.Errorf("unregistered shape %q", shape)
		}
	}
}

// TestThePreparedTreeRedactsInstructionsAndStripsConfiguration is the half of
// the shape test that needs a tree (ADR-156 Decision 2, as Zy amended it on
// 2026-10-14 after the pilot): at any depth, a CLAUDE.md, CLAUDE.local.md,
// AGENTS.md or .cursorrules keeps every line but those naming mrw, byte for
// byte, so a repository's tests that read its own AGENTS.md still find it, and so
// does every file one of them imports with @path, as Claude Code loads those
// too; one that is a link into the tree stays a link, its target redacted, and
// one that leads out of the tree is removed. The configuration a host would
// load and run (.claude/, .cursor/, .codex/, and .mcp.json at the root) is
// gone; a .mcp.json below the root, which no host reads as configuration and a
// plugin may ship and test (agentsmemory's clients/claude-code), is kept; every
// other file is untouched, mrw in it or not. A letter beside mrw in any script
// makes it another word (the review of #413).
func TestThePreparedTreeRedactsInstructionsAndStripsConfiguration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bench runs on the operator's macOS or Linux machine; Windows is out of ADR-156's scope")
	}
	root := t.TempDir()
	outside := t.TempDir()
	md := "# Rules\nUse mrw for every edit.\r\nKeep tests green.\r\nmrw_read is the MCP tool.\nRun MRW first.\nsmrw, mrwx and émrw are other words.\n@docs/rules.md\nlast line, no newline"
	redacted := "# Rules\nKeep tests green.\r\nsmrw, mrwx and émrw are other words.\n@docs/rules.md\nlast line, no newline"
	files := map[string]string{
		"CLAUDE.md": md, "AGENTS.md": md, "CLAUDE.local.md": md, ".cursorrules": md, "sub/AGENTS.md": md, "shared.md": md,
		"docs/rules.md": "Imported.\nmrw is the tool.\n", "keep.go": "// mrw\n", "sub/keep.md": "mrw\n", "sub/.mcp.json": "{}\n",
		".mcp.json": "x\n", ".claude/settings.json": "x\n", ".claude/CLAUDE.md": md, ".cursor/rules/a.mdc": "x\n", ".codex/config.toml": "x\n", "sub/.claude/x": "x\n",
		// An @path in a code block or a code span is an example, not an import: its
		// target is no instruction (the re-review of #413).
		"code/CLAUDE.md": "Example:\n```\n@ex/a.md\n```\nInline `@ex/b.md` too.\n", "code/ex/a.md": "mrw\n", "code/ex/b.md": "mrw\n",
	}
	for p, s := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "x.md"), []byte("mrw\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A link into a directory that preparation removes leads out of the tree once
	// it is gone, so it goes too (the re-review of #413).
	for link, to := range map[string]string{"linked/CLAUDE.md": filepath.Join("..", "shared.md"), "away/AGENTS.md": filepath.Join(outside, "x.md"), "cfg/CLAUDE.md": filepath.Join("..", ".claude", "CLAUDE.md")} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, link)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(to, filepath.Join(root, link)); err != nil {
			t.Fatal(err)
		}
	}
	if err := PrepareInstructions(root); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"CLAUDE.md": redacted, "AGENTS.md": redacted, "CLAUDE.local.md": redacted, ".cursorrules": redacted, "sub/AGENTS.md": redacted,
		"shared.md": redacted, "linked/CLAUDE.md": redacted, "docs/rules.md": "Imported.\n",
		"keep.go": "// mrw\n", "sub/keep.md": "mrw\n", "sub/.mcp.json": "{}\n",
		"code/CLAUDE.md": "Example:\n```\n@ex/a.md\n```\nInline `@ex/b.md` too.\n", "code/ex/a.md": "mrw\n", "code/ex/b.md": "mrw\n",
	}
	got := map[string]string{}
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, p)
			b, _ := os.ReadFile(p)
			got[filepath.ToSlash(rel)] = string(b)
		}
		return nil
	})
	if len(got) != len(want) {
		t.Errorf("files after preparing: %v, want %v", got, want)
	}
	for p, s := range want {
		if got[p] != s {
			t.Errorf("%s after preparing: %q, want %q", p, got[p], s)
		}
	}
	if info, err := os.Lstat(filepath.Join(root, "linked", "CLAUDE.md")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("an instruction file that is a link into the tree is no longer a link: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(outside, "x.md")); err != nil || string(b) != "mrw\n" {
		t.Errorf("a file outside the tree was changed through a link: %q %v", b, err)
	}
}

// taskFrom writes a fixture task from its parts and loads it.
func taskFrom(t *testing.T, id, taskJSON, patch string, base map[string]string) Task {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id)
	for p, s := range base {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, "base", p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "base", p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "task.json"), []byte(taskJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reference.patch"), []byte(patch), 0o644); err != nil {
		t.Fatal(err)
	}
	task, err := LoadTask(dir)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

// TestTheOracleRefusesWhatTheReviewFound holds the oracle to the holes the
// Codex review of #411 traced: a change git calls binary, a mode-only change
// and a deletion are outside the regions, not a setup failure; slack moves an
// insertion but never licenses deleting the line beside it; and an exact task
// fails when a line its fix adds is commented out though every token survives.
func TestTheOracleRefusesWhatTheReviewFound(t *testing.T) {
	needGit(t)
	notes := taskFrom(t, "notes", `{"id":"notes","shape":"fixture","repo":"fixture","instruction":"x","require":[{"file":"notes.txt","match":"(?m)^gamma$"}]}`,
		"diff --git a/notes.txt b/notes.txt\n--- a/notes.txt\n+++ b/notes.txt\n@@ -2 +2 @@\n-beta\n+gamma\n",
		map[string]string{"notes.txt": "alpha\nbeta\ndelta\nepsilon\n", "other.txt": "keep\n"})
	insert := taskFrom(t, "insert", `{"id":"insert","shape":"fixture","repo":"fixture","instruction":"x","slack":1,"require":[{"file":"s.yml","match":"(?m)^  x: 1$"}]}`,
		"diff --git a/s.yml b/s.yml\n--- a/s.yml\n+++ b/s.yml\n@@ -2,0 +3 @@\n+  x: 1\n",
		map[string]string{"s.yml": "a:\n  b: 1\n  c: 1\n  d: 1\n"})
	exact := taskFrom(t, "exact", `{"id":"exact","shape":"fixture","repo":"fixture","instruction":"x","exact":true,"require":[{"file":"r.go","match":"\\bsendMessage\\b"}],"forbid":[{"file":"r.go","match":"\\bpostSend\\b"}]}`,
		"diff --git a/r.go b/r.go\n--- a/r.go\n+++ b/r.go\n@@ -4 +4 @@\n-\tregister(postSend)\n+\tregister(sendMessage)\n",
		map[string]string{"r.go": "package r\n\nfunc init() {\n\tregister(postSend)\n}\n"})
	cases := []struct {
		name   string
		task   Task
		mangle func(tree string) error
		want   string // "outside" or "missing"
	}{
		// Each on the file the fix touches, so the whole-file rule — not the
		// untouched-file rule — is what must catch it.
		{"a binary change", notes, func(tree string) error {
			return os.WriteFile(filepath.Join(tree, "notes.txt"), []byte("alpha\ngamma\ndelta\nepsilon\x00\n"), 0o644)
		}, "outside"},
		{"a mode-only change", notes, func(tree string) error {
			return os.Chmod(filepath.Join(tree, "notes.txt"), 0o755)
		}, "outside"},
		{"a deleted file", notes, func(tree string) error {
			return os.Remove(filepath.Join(tree, "notes.txt"))
		}, "outside"},
		{"slack deleting the neighbour", insert, func(tree string) error {
			return os.WriteFile(filepath.Join(tree, "s.yml"), []byte("a:\n  b: 1\n  x: 1\n  d: 1\n"), 0o644)
		}, "outside"},
		{"a fix line kept inside a block comment", exact, func(tree string) error {
			return os.WriteFile(filepath.Join(tree, "r.go"), []byte("package r\n\nfunc init() {\n/*\n\tregister(sendMessage)\n*/\n}\n"), 0o644)
		}, "missing"},
		{"a fix line commented out", exact, func(tree string) error {
			return os.WriteFile(filepath.Join(tree, "r.go"), []byte("package r\n\nfunc init() {\n\t// sendMessage\n}\n"), 0o644)
		}, "missing"},
	}
	for _, c := range cases {
		tree := t.TempDir()
		if _, err := Materialize(c.task, tree, nil); err != nil {
			t.Fatal(err)
		}
		if err := ApplyPatch(tree, filepath.Join(c.task.Dir, "reference.patch")); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if v, err := Oracle(c.task, tree); err != nil || !v.Solved {
			t.Fatalf("%s: the reference fix alone is not solved: %+v %v", c.name, v, err)
		}
		if err := c.mangle(tree); err != nil {
			t.Fatal(err)
		}
		v, err := Oracle(c.task, tree)
		if err != nil {
			t.Fatalf("%s: the oracle failed instead of grading: %v", c.name, err)
		}
		if v.Solved || (c.want == "outside" && len(v.Outside) == 0) || (c.want == "missing" && len(v.Missing) == 0) {
			t.Errorf("%s: %+v, want it %s", c.name, v, c.want)
		}
	}
}

// TestADeclaredSectionIsDroppedAndAMissingOneRefuses: a section a repository
// declares (ADR-158 Decision 2) goes from its heading to the next heading of
// the same or a higher level, a "#" line inside a code fence not being a
// heading; a declared file absent from the tree is skipped, since the oracle
// test materializes only the graded files; a heading found zero or two times
// is an error naming the file and the heading, never a silent no-op.
func TestADeclaredSectionIsDroppedAndAMissingOneRefuses(t *testing.T) {
	root := t.TempDir()
	// The ```` block holds a ``` example and a heading: neither closes the
	// fence, nor ends the section (the review of #417).
	in := "# Protocol\n\nintro\n\n## Gate\n\nstop here\n#not-a-heading, no space after the marks\n\n```sh\n# a comment, not a heading\n## nor this\n```\n\n````md\n```sh\n## an example heading\n```\n````\n\n### Inside the gate\n\nstill gate\n\n## Kept\n\nkept text\n\n## Absent\n\nask six times\n\n# Next top\n\ntail\n"
	want := "# Protocol\n\nintro\n\n## Kept\n\nkept text\n\n# Next top\n\ntail\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	drops := []Section{{File: "AGENTS.md", Heading: "## Gate"}, {File: "AGENTS.md", Heading: "## Absent"}, {File: "CLAUDE.md", Heading: "## Gate"}}
	if err := dropSections(root, drops); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != want {
		t.Errorf("after the drop:\n%s\nwant:\n%s", b, want)
	}
	for _, c := range []struct{ name, text string }{
		{"missing", "# Protocol\n\n## Kept\n"},
		{"twice", "# Protocol\n\n## Gate\n\na\n\n## Gate\n\nb\n"},
	} {
		if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(c.text), 0o644); err != nil {
			t.Fatal(err)
		}
		err := dropSections(root, drops[:1])
		if err == nil || !strings.Contains(err.Error(), "AGENTS.md") || !strings.Contains(err.Error(), "## Gate") {
			t.Errorf("%s: a heading not found exactly once was not refused by name: %v", c.name, err)
		}
	}
}

// TestAgentsmemorysTreeLosesBothDeclaredSections: through Materialize, at the
// commit every agentsmemory task pins, AGENTS.md loses the "Gate" section and
// the "When the tools are absent" section it sends a session to, and keeps the
// rest in order (ADR-158 Decision 2). In Stage 1, 11 attempts obeyed them and
// stopped after one turn, because the bench withholds the tools they ask for.
func TestAgentsmemorysTreeLosesBothDeclaredSections(t *testing.T) {
	needGit(t)
	tasks, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	var task Task
	for _, tk := range tasks {
		if tk.Repo == "agentsmemory" {
			task = tk
			break
		}
	}
	if task.ID == "" {
		t.Fatal("no agentsmemory task in the corpus")
	}
	src, err := repoPath(task)
	if err != nil {
		if errors.Is(err, errUnreachable) {
			t.Skipf("task %s: %v", task.ID, err)
		}
		t.Fatal(err)
	}
	orig, err := exec.Command("git", "-C", src, "show", task.Commit+":AGENTS.md").Output()
	if err != nil {
		t.Fatal(err)
	}
	tree := t.TempDir()
	if _, err := Materialize(task, tree, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(tree, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, gone := range []string{"\n## Gate — verify the `am_*` tools before you do anything else\n", "\n## When the tools are absent\n", "ask six times"} {
		if strings.Contains(got, gone) {
			t.Errorf("AGENTS.md still holds %q", strings.TrimSpace(gone))
		}
	}
	for _, kept := range []string{"\n## The working loop — what a session does WITHOUT being told\n", "\n## Reachability — the defect this repo keeps shipping\n"} {
		if !strings.Contains(got, kept) {
			t.Errorf("AGENTS.md lost %q, which no declaration names", strings.TrimSpace(kept))
		}
	}
	// What is left is the original's lines in their order: nothing added.
	rest := strings.Split(string(orig), "\n")
	i := 0
	for _, line := range strings.Split(got, "\n") {
		for i < len(rest) && rest[i] != line {
			i++
		}
		if i == len(rest) {
			t.Fatalf("AGENTS.md holds a line the original does not, or out of order: %q", line)
		}
		i++
	}
}

// TestEveryVariedMultiSiteTaskHasARepeatedAnchorAcrossTwoFiles holds every
// varied-multi-site task to its definition (ADR-158 Decision 4): its reference
// fix changes at least three places in at least two files, and at least one
// line it removes has text that also occurs, left unchanged by the fix, in a
// file the fix touches, so substituting that text everywhere is a mis-edit.
func TestEveryVariedMultiSiteTaskHasARepeatedAnchorAcrossTwoFiles(t *testing.T) {
	needGit(t)
	tasks, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, task := range tasks {
		if task.Shape != ShapeVariedMultiSite {
			continue
		}
		n++
		t.Run(task.ID, func(t *testing.T) {
			hunks := 0
			for _, hs := range task.regions {
				hunks += len(hs)
			}
			if len(task.regions) < 2 || hunks < 3 {
				t.Fatalf("%d hunk(s) in %d file(s), want at least 3 in at least 2", hunks, len(task.regions))
			}
			src, err := repoPath(task)
			if err != nil {
				if errors.Is(err, errUnreachable) {
					t.Skipf("task %s: %v", task.ID, err)
				}
				t.Fatal(err)
			}
			patch, err := os.ReadFile(filepath.Join(task.Dir, "reference.patch"))
			if err != nil {
				t.Fatal(err)
			}
			removed := map[string]int{}
			for _, line := range strings.Split(string(patch), "\n") {
				if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					if s := strings.TrimSpace(line[1:]); s != "" {
						removed[s]++
					}
				}
			}
			for text, times := range removed {
				seen := 0
				for file := range task.regions {
					b, err := exec.Command("git", "-C", src, "show", task.Commit+":"+file).Output()
					if err != nil {
						continue
					}
					for _, l := range strings.Split(string(b), "\n") {
						if strings.Contains(l, text) {
							seen++
						}
					}
				}
				if seen > times {
					return
				}
			}
			t.Errorf("no removed line's text occurs elsewhere, unchanged, in the files the fix touches: one substitution everywhere makes this change")
		})
	}
	if n == 0 {
		t.Fatal("no varied-multi-site task in the corpus")
	}
}

// TestTheVariedOraclesGradeWhatTheReviewOf417Found: on the reference fix of
// two ADR-158 tasks, edits the first oracles graded wrongly (Codex, the
// review of #417). A reranker timeout moved out of its healthcheck is valid
// to the structure check and must fail; a quoted duration means the same and
// must pass; a 3.13 line inserted above the 3.12 it was to replace leaves
// 3.12 in force and must fail; and a 3.13 is bare or quoted on both sides,
// never on one (the re-review).
func TestTheVariedOraclesGradeWhatTheReviewOf417Found(t *testing.T) {
	needGit(t)
	tasks, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Task{}
	for _, task := range tasks {
		byID[task.ID] = task
	}
	for _, c := range []struct {
		name, task, file, from, to string
		solved                     bool
	}{
		{"the reranker timeout indented out of its healthcheck", "am-vms-compose-health", "docker-compose.full.yml", "\n      timeout: 10s\n", "\n    timeout: 10s\n", false},
		{"the reranker timeout quoted", "am-vms-compose-health", "docker-compose.full.yml", "\n      timeout: 10s\n", "\n      timeout: \"10s\"\n", true},
		{"3.13 inserted above the 3.12 it replaces", "qh-vms-python-313", ".github/workflows/selftest.yml", "          python-version: '3.13'\n", "          python-version: '3.13'\n          python-version: '3.12'\n", false},
		{"3.13 with a closing quote and no opening one", "qh-vms-python-313", ".github/workflows/selftest.yml", "          python-version: '3.13'\n", "          python-version: 3.13'\n", false},
		{"3.13 bare", "qh-vms-python-313", ".github/workflows/selftest.yml", "          python-version: '3.13'\n", "          python-version: 3.13\n", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			task, ok := byID[c.task]
			if !ok {
				t.Fatalf("no task %s", c.task)
			}
			tree := t.TempDir()
			if _, err := Materialize(task, tree, task.GradedFiles()); err != nil {
				if errors.Is(err, errUnreachable) {
					t.Skipf("task %s: %v", task.ID, err)
				}
				t.Fatal(err)
			}
			if err := ApplyPatch(tree, filepath.Join(task.Dir, "reference.patch")); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(tree, c.file)
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(b), c.from) {
				t.Fatalf("%s does not hold %q after the reference fix", c.file, c.from)
			}
			if err := os.WriteFile(p, []byte(strings.Replace(string(b), c.from, c.to, 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			v, err := Oracle(task, tree)
			if err != nil {
				t.Fatal(err)
			}
			if v.Solved != c.solved {
				t.Errorf("solved %v, want %v: %+v", v.Solved, c.solved, v)
			}
		})
	}
}

// TestThePostSendRenameLeavesTheTestHelper: cac-rename-post-send-2 asks for
// the handler method only, so its oracle passes the fix that renames the
// method and its references, and refuses the old task's fix, which renamed the
// unrelated test helper of the same name too (ADR-159).
func TestThePostSendRenameLeavesTheTestHelper(t *testing.T) {
	needGit(t)
	tasks, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	var task Task
	for _, x := range tasks {
		if x.ID == "cac-rename-post-send-2" {
			task = x
		}
	}
	if task.ID == "" {
		t.Fatal("no task cac-rename-post-send-2")
	}
	for _, c := range []struct {
		patch  string
		solved bool
	}{
		{filepath.Join(task.Dir, "reference.patch"), true},
		{filepath.Join(filepath.Dir(task.Dir), "cac-rename-post-send", "reference.patch"), false},
	} {
		tree := t.TempDir()
		if _, err := Materialize(task, tree, task.GradedFiles()); err != nil {
			if errors.Is(err, errUnreachable) {
				t.Skipf("task %s: %v", task.ID, err)
			}
			t.Fatal(err)
		}
		if err := ApplyPatch(tree, c.patch); err != nil {
			t.Fatal(err)
		}
		v, err := Oracle(task, tree)
		if err != nil {
			t.Fatal(err)
		}
		if v.Solved != c.solved {
			t.Errorf("%s: solved %v, want %v: %+v", c.patch, v.Solved, c.solved, v)
		}
	}
}

// TestACheckCoupledTaskIsGradedByItsPackageTests holds the grading Zy chose
// after T3's pilot ("Grade by the tests", ADR-160 Decision 1): a check-coupled
// attempt is solved when the source holds the new literal and not the old,
// its package's tests pass, and nothing outside the package changed — so the
// answers exact refused in the pilot are solved, and a source-only change that
// breaks the test is not.
func TestACheckCoupledTaskIsGradedByItsPackageTests(t *testing.T) {
	needGit(t)
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not on PATH")
	}
	repo := t.TempDir()
	files := map[string]string{
		"go.mod":      "module example.com/m\n\ngo 1.21\n",
		"p/p.go":      "package p\n\n// Status is the state a new subscription takes.\nfunc Status() string { return \"active\" }\n",
		"p/p_test.go": "package p\n\nimport \"testing\"\n\nfunc TestStatus(t *testing.T) {\n\tif Status() != \"active\" {\n\t\tt.Fatalf(\"Status() = %q, want \\\"active\\\"\", Status())\n\t}\n}\n",
		"q/q.go":      "package q\n\n// X is unrelated.\nconst X = 1\n",
	}
	write := func(root string, fs map[string]string) {
		t.Helper()
		for name, body := range fs {
			p := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	git := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=b", "-c", "user.email=b@b"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write(repo, files)
	git(repo, "init", "-q")
	git(repo, "add", "-A")
	git(repo, "commit", "-qm", "base")
	sha := git(repo, "rev-parse", "HEAD")
	fixed := map[string]string{
		"p/p.go":      strings.Replace(files["p/p.go"], `"active"`, `"Active"`, 1),
		"p/p_test.go": strings.Replace(files["p/p_test.go"], `Status() != "active"`, `Status() != "Active"`, 1),
	}
	write(repo, fixed)
	patch := git(repo, "diff") + "\n"
	git(repo, "checkout", "-q", "--", ".")

	taskDir := filepath.Join(t.TempDir(), "am-coupled-p-active")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "reference.patch"), []byte(patch), 0o644); err != nil {
		t.Fatal(err)
	}
	taskJSON := func(extra string) {
		t.Helper()
		body := `{"id": "am-coupled-p-active", "shape": "check-coupled", "repo": "m", "commit": "` + sha + `",
  "instruction": "In p/p.go, change the string \"active\" to \"Active\", and keep the tests passing.",
  "require": [{"file": "p/p.go", "match": "\"Active\""}], "forbid": [{"file": "p/p.go", "match": "\"active\""}]` + extra + `}`
		if err := os.WriteFile(filepath.Join(taskDir, "task.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, bad := range []string{`, "test_package": "p", "exact": true`, `, "test_package": "../p"`, `, "test_package": "/p"`} {
		taskJSON(bad)
		if _, err := LoadTask(taskDir); err == nil {
			t.Errorf("the loader accepted %s", bad)
		}
	}
	taskJSON(`, "test_package": "p"`)
	task, err := LoadTask(taskDir)
	if err != nil {
		t.Fatal(err)
	}
	grade := func(name string, change map[string]string) Verdict {
		t.Helper()
		tree := t.TempDir()
		git(tree, "clone", "-q", repo, ".")
		write(tree, change)
		v, err := Oracle(task, tree)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return v
	}
	if v := grade("base", nil); v.Solved {
		t.Errorf("the base is solved: %+v", v)
	}
	if v := grade("the reference", fixed); !v.Solved {
		t.Errorf("the reference is unsolved: %+v", v)
	}
	if v := grade("the source alone", map[string]string{"p/p.go": fixed["p/p.go"]}); v.Solved || !strings.Contains(strings.Join(v.Missing, " "), "go test") {
		t.Errorf("a source-only change that breaks the package's test is not unsolved by go test: %+v", v)
	}
	both := "package p\n\nimport \"strings\"\n\n// Status is the state a new subscription takes.\nfunc Status() string { return strings.ToLower(\"Active\") }\n"
	if v := grade("the code accepting both", map[string]string{"p/p.go": both}); !v.Solved {
		t.Errorf("an answer that keeps the tests passing another way is unsolved: %+v", v)
	}
	message := map[string]string{"p/p.go": fixed["p/p.go"], "p/p_test.go": strings.ReplaceAll(fixed["p/p_test.go"], `want \"active\"`, `want \"Active\"`)}
	if v := grade("the test message too", message); !v.Solved {
		t.Errorf("updating the test's message as well is unsolved: %+v", v)
	}
	outside := map[string]string{"p/p.go": fixed["p/p.go"], "p/p_test.go": fixed["p/p_test.go"], "q/q.go": files["q/q.go"] + "\n// stray\n"}
	if v := grade("a change outside the package", outside); v.Solved || len(v.Outside) == 0 {
		t.Errorf("a change outside the package is not outside: %+v", v)
	}
}
