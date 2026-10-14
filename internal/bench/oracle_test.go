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
			if f, base, ok := task.neighbour(); ok {
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

// TestTheCorpusHasTheRegisteredShapeCounts holds the corpus to Decision 3:
// 8 multi-site, 6 structural, 6 repeated-anchor, 4 large-file and 6
// single-line tasks, at least one CI-runnable fixture, instructions that name
// no tool, and no reference fix that touches an instruction file the runner
// strips.
func TestTheCorpusHasTheRegisteredShapeCounts(t *testing.T) {
	tasks, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, task := range tasks {
		got[task.Shape]++
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
	want := map[string]int{ShapeMultiSite: 8, ShapeStructural: 6, ShapeRepeatedAnchor: 6, ShapeLargeFile: 4, ShapeSingleLine: 6}
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
		".mcp.json": "x\n", ".claude/settings.json": "x\n", ".cursor/rules/a.mdc": "x\n", ".codex/config.toml": "x\n", "sub/.claude/x": "x\n",
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
	for link, to := range map[string]string{"linked/CLAUDE.md": filepath.Join("..", "shared.md"), "away/AGENTS.md": filepath.Join(outside, "x.md")} {
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
