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

// TestStrippingRemovesEveryInstructionFile is the half of the shape test that
// needs a tree: whatever depth they sit at, the files a Claude Code, Codex or
// Cursor session would read as instructions are gone after Strip.
func TestStrippingRemovesEveryInstructionFile(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"CLAUDE.md", "AGENTS.md", "CLAUDE.local.md", ".mcp.json", ".claude/settings.json", ".cursor/rules/a.mdc", ".codex/config.toml", "sub/CLAUDE.md", "sub/AGENTS.md", "sub/.claude/x", "keep.go", "sub/keep.md"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, p), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := StripInstructions(root); err != nil {
		t.Fatal(err)
	}
	var left []string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, p)
			left = append(left, filepath.ToSlash(rel))
		}
		return nil
	})
	if strings.Join(left, ",") != "keep.go,sub/keep.md" {
		t.Errorf("left after stripping: %v", left)
	}
}
