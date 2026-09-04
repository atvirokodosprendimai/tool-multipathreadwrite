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
	row := regexp.MustCompile(`^\|\s*` + "`" + `(Test\w+)` + "`" + `\s*\|\s*` + "`" + `([^` + "`" + `]+_test\.go)` + "`")

	checked := 0
	for _, task := range tasks {
		b, err := os.ReadFile(task)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(repo, task)
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "~~") {
				continue // struck through: removed on purpose, and said so
			}
			m := row.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			name, file := m[1], m[2]
			checked++
			src, err := os.ReadFile(filepath.Join(repo, file))
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
	if checked == 0 {
		t.Error("matched no Tests rows at all; the pattern has drifted from the table format")
	}
	t.Logf("checked %d test names across %d task files", checked, len(tasks))
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
