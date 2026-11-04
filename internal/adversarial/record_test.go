package adversarial

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestEveryTestNamedByARecordExists is a repo-owned guard against one specific
// kind of rot: a task file's Tests table naming a test that no longer exists.
//
// It is NOT adr-lint. adr-lint is a plugin installed on a developer's machine
// and is not available to CI, so nothing in this repository's pipeline reads
// the decision records at all — which is how THREE records sat failing for days
// with every pull request green:
//
//   - ADR-002-T1 named `TestForget`, deleted with `seen.Forget` in #50.
//   - ADR-004-T2 named `TestLegacyInTreeLedgerIsStillRead`, deleted in #19 when
//     the v2 ledger header made its fixture invalid — while the behaviour it
//     covered kept shipping, untested, for weeks.
//   - ADR-007-T1 named four `Descendable` tests, deleted with the function.
//
// Two of those are bookkeeping and one was a real coverage hole. This test
// cannot tell them apart, and does not try: it says the record and the code
// disagree, which is the point at which a person should look.
//
// A row struck through with ~~ is exempt. That is how this corpus records a
// test that was removed on purpose, and the strikethrough is a deliberate act
// rather than an omission.
//
// A task its tasks/README.md lists as pending, partial or blocked names tests
// that do not exist yet: its Tests table is its work list, written before the
// red test (.claude/rules/lifecycle.md §3). Such a miss is counted, not failed;
// a task whose status cannot be read is checked (ADR-194, ADR-195).
func TestEveryTestNamedByARecordExists(t *testing.T) {
	repo := repoRoot(t)
	tasks, err := filepath.Glob(filepath.Join(repo, "docs", "adr", "*", "tasks", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) == 0 {
		t.Fatal("no task files found; this guard would pass by looking at nothing")
	}

	// | `TestName` | `path/to/file_test.go` | ... — both backticked, and the
	// row not struck through.
	// A struck NAME: | ~~`TestX`~~ | ... — the first cell only.
	struckName := regexp.MustCompile(`^\|\s*~~`)

	row := regexp.MustCompile(`^\|\s*` + "`" + `(Test\w+)` + "`" + `\s*\|\s*` + "`" + `([^` + "`" + `]+_test\.go)` + "`")

	// A floor, not a presence check. `checked > 0` passed while 128 of 129 rows
	// drifted to an unrecognised syntax — a guard that reports "checked 1" and
	// goes green is the vacuity this repository keeps rediscovering. The floor
	// is deliberately well under the current count so ordinary editing does not
	// trip it, and far above the number a syntax drift would leave.
	const floor = 100
	checked, notYet := 0, 0
	for _, task := range tasks {
		b, err := os.ReadFile(task)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(repo, task)
		for i, line := range strings.Split(string(b), "\n") {
			// The strikethrough exemption is FIRST-CELL ONLY. `strings.Contains`
			// over the whole line exempted any row whose description cell struck
			// a word, with its test name unchecked — no such row exists today,
			// which is exactly when to close it.
			if struckName.MatchString(line) {
				continue // the NAME is struck: removed on purpose, and said so
			}
			m := row.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			name, file := m[1], m[2]
			checked++
			src, err := os.ReadFile(filepath.Join(repo, file))
			if (err != nil || !strings.Contains(string(src), "func "+name+"(")) && unfinished(taskStatus(task)) {
				notYet++
				continue
			}
			if err != nil {
				t.Errorf("%s:%d names %s in %s, which cannot be read: %v", rel, i+1, name, file, err)
				continue
			}
			if !strings.Contains(string(src), "func "+name+"(") {
				t.Errorf("%s:%d names %s in %s, and that file defines no such test.\n"+
					"Either the test was renamed or deleted and the table was not reconciled, or the\n"+
					"behaviour lost its coverage. Strike the row with ~~ and say why if it went on purpose.",
					rel, i+1, name, file)
			}
		}
	}
	if checked < floor {
		t.Errorf("matched only %d Tests rows, below the floor of %d — the table syntax has drifted "+
			"and this guard is reading almost nothing", checked, floor)
	}
	t.Logf("checked %d test names across %d task files; %d in tasks not yet done do not exist yet", checked, len(tasks), notYet)
}

func unfinished(status string) bool {
	return status == "pending" || status == "partial" || status == "blocked"
}

// taskStatus is the cell under the Status column of the table in the task's
// tasks/README.md whose first cell is the task's id, or "" when there is no
// such row or that table names no Status column. The column is the one the
// table's header names, never the first status-like word in the row: a goal
// that says "pending" is not a status (the Codex review of #618).
func taskStatus(task string) string {
	id, _, _ := strings.Cut(strings.TrimSuffix(filepath.Base(task), ".md"), "-")
	b, err := os.ReadFile(filepath.Join(filepath.Dir(task), "README.md"))
	if err != nil {
		return ""
	}
	col := -2 // no table open
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			col = -2
			continue
		}
		var cells []string
		for _, c := range strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|") {
			cells = append(cells, strings.Trim(strings.TrimSpace(c), "`"))
		}
		if col == -2 {
			col = -1
			for i, c := range cells {
				if strings.EqualFold(c, "status") {
					col = i
				}
			}
			continue
		}
		if len(cells) > 0 && cells[0] == id && col >= 0 && col < len(cells) {
			return cells[col]
		}
	}
	return ""
}

// The status a record's guard reads is the Status column's, whatever the
// other cells say, and a task its README does not list reads as unknown.
func TestATaskStatusIsReadFromItsTablesStatusColumn(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tasks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	readme := "| Task | Goal | Depends-on | Status |\n|---|---|---|---|\n| T1 | pending | none | done |\n| T2 | x | none | pending |\n\n" +
		"| Task | Produces | Consumers |\n|---|---|---|\n| T3 | blocked | T1 |\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	for task, want := range map[string]string{"T1-a.md": "done", "T2-b.md": "pending", "T3-c.md": "", "T9-d.md": ""} {
		if got := taskStatus(filepath.Join(dir, task)); got != want {
			t.Errorf("%s: status %q, want %q", task, got, want)
		}
	}
}

// repoRoot walks up from the test's working directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the working directory")
		}
		dir = parent
	}
}
