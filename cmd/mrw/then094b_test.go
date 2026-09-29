package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// afterHead is the line that follows the first line of out containing head,
// or "" when there is none.
func afterHead(t *testing.T, out, head string) string {
	t.Helper()
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		if strings.Contains(l, head) {
			if i+1 < len(lines) {
				return lines[i+1]
			}
			return ""
		}
	}
	t.Fatalf("no line holds %q:\n%s", head, out)
	return ""
}

// keptLog is the log a pointer line names, removed when the test ends.
func keptLog(t *testing.T, line string) string {
	t.Helper()
	i := strings.Index(line, " in ")
	if i < 0 {
		t.Fatalf("not a pointer line: %q", line)
	}
	log := strings.TrimSpace(line[i+len(" in "):])
	t.Cleanup(func() { _ = os.Remove(log) })
	return log
}

// ADR-094 T2. A passing step shows the last non-empty line of its output under
// its PASS head, on write and on check, and that line is the last non-empty
// entry of the JSON tail. When its output ran past tail_lines and the log was
// kept, the pointer above the shown line counts every line above it — the
// second fixture's trailing blank is what tells that count from the tail's
// length less one.
func TestAPassingStepShowsItsLastLine(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": "{\"check\":\"exit 0\"}",
	})
	plan := primed(t, root)
	step := "echo first; echo LASTLINE; echo"
	out, code := writeIn(t, root, "--no-check", "--then-sh", step, plan)
	if got := afterHead(t, out, "— PASS"); code != 0 || got != "  | LASTLINE" {
		t.Errorf("write: exit %d, the line under the head is %q, want %q:\n%s", code, got, "  | LASTLINE", out)
	}
	out, code = runIn(t, root, "check", "--full", "--then-sh", step)
	if got := afterHead(t, out, "then 1/1"); code != 0 || got != "  | LASTLINE" {
		t.Errorf("check: exit %d, the line under the head is %q:\n%s", code, got, out)
	}
	out, code = writeIn(t, root, "--no-check", "--json", "--then-sh", step, plan)
	var r struct {
		Then struct {
			Steps []struct {
				Tail []string `json:"tail"`
			} `json:"steps"`
		} `json:"then"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil || code != 0 || len(r.Then.Steps) != 1 {
		t.Fatalf("--json: exit %d, %v:\n%s", code, err, out)
	}
	tail := r.Then.Steps[0].Tail
	last := ""
	for _, l := range tail {
		if strings.TrimSpace(l) != "" {
			last = l
		}
	}
	if last != "LASTLINE" {
		t.Errorf("the JSON tail's last non-empty entry is %q, want LASTLINE: %q", last, tail)
	}

	t.Run("a kept log counts every line above the shown one", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := grepTree(t, map[string]string{
			"a.go":                  "package a\nfunc A() {}\n",
			".quality-harness.json": "{\"check\":\"exit 0\",\"tail_lines\":2}",
		})
		plan := primed(t, root)
		for _, step := range []string{"seq 1 50", "seq 1 50; echo"} {
			out, code := writeIn(t, root, "--no-check", "--then-sh", step, plan)
			pointer := afterHead(t, out, "— PASS")
			if code != 0 || !strings.HasPrefix(pointer, "... 49 earlier line(s) in ") {
				t.Errorf("%s: exit %d, the pointer is %q, want 49 earlier lines:\n%s", step, code, pointer, out)
				continue
			}
			keptLog(t, pointer)
			if got := afterHead(t, out, pointer); got != "  | 50" {
				t.Errorf("%s: the line under the pointer is %q, want %q:\n%s", step, got, "  | 50", out)
			}
		}
	})
}

// ADR-094 T2. A passing step with nothing to show prints its head alone: no
// output, a masked failure that printed nothing, and — with the log kept — a
// tail of blank lines, whose pointer still names the log. then last: marks a
// failure only.
func TestASilentPassingStepPrintsOnlyItsHead(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": "{\"check\":\"exit 0\",\"tail_lines\":2}",
	})
	plan := primed(t, root)
	for _, step := range []string{"true", "false || true"} {
		out, code := writeIn(t, root, "--no-check", "--then-sh", step, plan)
		if got := afterHead(t, out, "— PASS"); code != 0 || strings.HasPrefix(got, "  | ") || strings.HasPrefix(got, "...") {
			t.Errorf("%s: exit %d, the line under the head is %q, want none of the step's:\n%s", step, code, got, out)
		}
		if strings.Contains(out, "then last:") {
			t.Errorf("%s: a passing step printed then last:\n%s", step, out)
		}
	}
	out, code := writeIn(t, root, "--no-check", "--then-sh", "printf '\\n\\n\\n\\n\\n'", plan)
	pointer := afterHead(t, out, "— PASS")
	if code != 0 || !strings.HasPrefix(pointer, "... 3 earlier line(s) in ") {
		t.Fatalf("five blank lines: exit %d, the line under the head is %q, want the pointer to 3 earlier lines:\n%s", code, pointer, out)
	}
	if _, err := os.Stat(keptLog(t, pointer)); err != nil {
		t.Errorf("the named log: %v", err)
	}
	if got := afterHead(t, out, pointer); strings.HasPrefix(got, "  | ") || strings.Contains(out, "then last:") {
		t.Errorf("five blank lines printed a tail line %q:\n%s", got, out)
	}
	out, code = writeIn(t, root, "--no-check", "--then-sh", "echo X; false", plan)
	if code != exitCheckFailed || !strings.Contains(out, "then last: X") {
		t.Errorf("a failing step: exit %d, want %d with then last: X:\n%s", code, exitCheckFailed, out)
	}
}

// ADR-094 T2. --quiet drops the ok hunk row and keeps every step verdict and
// its last line, and its usage says so.
func TestQuietKeepsEveryStepVerdictAndItsLastLine(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": "{\"check\":\"exit 0\"}",
	})
	out, code := writeIn(t, root, "--quiet", "--no-check", "--then-sh", "echo LASTLINE", primed(t, root))
	if code != 0 || strings.Contains(out, "ok   a.go") {
		t.Errorf("--quiet: exit %d, or the ok row is printed:\n%s", code, out)
	}
	if got := afterHead(t, out, "then 1/1 --then-sh: echo LASTLINE — PASS"); got != "  | LASTLINE" {
		t.Errorf("--quiet: the line under the step's head is %q, want %q:\n%s", got, "  | LASTLINE", out)
	}
	help, _ := runIn(t, root, "write", "--help")
	if !strings.Contains(help, "the check's report and every step verdict still print") {
		t.Errorf("--quiet's usage does not say the verdicts still print:\n%s", help)
	}
}
