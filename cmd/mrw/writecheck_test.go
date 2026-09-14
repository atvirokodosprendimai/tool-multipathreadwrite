package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-054 T1: a CLI write runs the project's check by default when a written
// path is not prose. The fixture declares a check that always fails (`exit 3`),
// so a write that ran it exits 3 and a write that did not exits 0 — the two
// outcomes are one exit code apart and nothing else in the run can produce
// either.

// checkTree is a root with one Go file, one markdown file, and a declared
// check that fails with a distinctive status. `exit 3` rather than `false`
// so a real check failure (3) cannot be confused with a usage refusal (2).
func checkTree(t *testing.T) string {
	t.Helper()
	return grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		"notes.md":              "# notes\nline two\n",
		".quality-harness.json": `{"check":"exit 3"}`,
	})
}

// writeIn runs `mrw -C root write <args...>` and returns stdout and the exit
// code the binary would have used.
func writeIn(t *testing.T, root string, args ...string) (string, int) {
	t.Helper()
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}
	argv := append([]string{"mrw", "-C", root, "write"}, args...)
	code := 0
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), argv)
	})
	if err != nil {
		code = exitCode(err)
	}
	return out, code
}

func planFile(t *testing.T, doc string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "p.mrw")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const goPlan = "@@ a.go 2 replace anchor=\"func A\"\nfunc A() { _ = 1 }\n"
const mdPlan = "@@ notes.md 2 replace anchor=\"line two\"\nline 2\n"

// TestWriteRunsTheCheckByDefault is the record's Enforced-by: a write to a
// .go file with no --check flag runs the declared check, and a failing check
// is exit 3 — the tree is changed and unverified (ADR-003).
func TestWriteRunsTheCheckByDefault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, planFile(t, goPlan))
	if code != exitCheckFailed {
		t.Fatalf("a .go write without --check exited %d, want %d (the check ran and failed):\n%s", code, exitCheckFailed, out)
	}
	if !strings.Contains(out, "check") || !strings.Contains(out, "FAIL") {
		t.Errorf("the receipt does not show the failing check:\n%s", out)
	}
}

// TestWriteOfProseDoesNotRunTheDefaultCheck: a markdown-only plan in a
// harnessed tree is Applied, exit 0, and never spawns the check. The
// motivating checkout wrote 259 plans, most of them markdown, against a
// full-workspace cargo command.
func TestWriteOfProseDoesNotRunTheDefaultCheck(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "notes.md"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, planFile(t, mdPlan))
	if code != 0 {
		t.Fatalf("a markdown-only write exited %d, want 0 (no check on prose):\n%s", code, out)
	}
	if strings.Contains(out, "check") {
		t.Errorf("a markdown-only write spawned or mentioned the check:\n%s", out)
	}
}

// TestExplicitCheckStillRunsOnProse: --check is a demand, and a demand is
// honoured on any path.
func TestExplicitCheckStillRunsOnProse(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "notes.md"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--check", planFile(t, mdPlan))
	if code != exitCheckFailed {
		t.Fatalf("--check on markdown exited %d, want %d:\n%s", code, exitCheckFailed, out)
	}
}

// TestNoCheckOptsOut: --no-check on a .go write exits 0 even though the
// declared check would fail. The opt-out is visible in stats because
// failed_check is no longer omitted at zero (T3).
func TestNoCheckOptsOut(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--no-check", planFile(t, goPlan))
	if code != 0 {
		t.Fatalf("--no-check exited %d, want 0:\n%s", code, out)
	}
	if strings.Contains(out, "check") {
		t.Errorf("--no-check still ran or mentioned the check:\n%s", out)
	}
}

// TestWriteWithoutACheckCommandStillApplies: no harness, no go.mod — a
// Desktop tree (ADR-019). The default does not invent exit 2; the write is
// Applied and exits 0. Explicit --check would still be 2 (ADR-003), and that
// path is already asserted by the check package.
func TestWriteWithoutACheckCommandStillApplies(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, planFile(t, goPlan))
	if code != 0 {
		t.Fatalf("a write with no check command exited %d, want 0:\n%s", code, out)
	}
	if strings.Contains(out, "SKIPPED") {
		t.Errorf("the default reported a skipped check where none was demanded:\n%s", out)
	}
}

// TestCheckAndNoCheckTogetherIsUsage: the two flags contradict, and a
// contradiction is settled before the plan is read.
func TestCheckAndNoCheckTogetherIsUsage(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	out, code := writeIn(t, root, "--check", "--no-check", planFile(t, goPlan))
	if code != exitUsage {
		t.Fatalf("--check --no-check exited %d, want %d:\n%s", code, exitUsage, out)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "func A() {}") {
		t.Errorf("a usage refusal wrote the file:\n%s", got)
	}
}

// ADR-060 T5: FAIL prints check last: above full output:; PASS does not.
func TestFailedCheckPrintsTheLastErrorLine(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Run("fail", func(t *testing.T) {
		root := grepTree(t, map[string]string{
			"a.go":                  "package a\nfunc A() {}\n",
			".quality-harness.json": `{"check":"sh -c 'echo unique-last-error-line; exit 1'"}`,
		})
		if _, err := readIn(t, root, "a.go"); err != nil {
			t.Fatal(err)
		}
		out, code := writeIn(t, root, planFile(t, goPlan))
		if code != exitCheckFailed {
			t.Fatalf("exited %d, want %d:\n%s", code, exitCheckFailed, out)
		}
		last := strings.Index(out, "check last: unique-last-error-line")
		full := strings.Index(out, "full output:")
		if last < 0 || full < 0 || last > full {
			t.Errorf("check last: must sit above full output:\n%s", out)
		}
	})
	t.Run("pass", func(t *testing.T) {
		root := grepTree(t, map[string]string{
			"a.go":                  "package a\nfunc A() {}\n",
			".quality-harness.json": `{"check":"true"}`,
		})
		if _, err := readIn(t, root, "a.go"); err != nil {
			t.Fatal(err)
		}
		out, code := writeIn(t, root, planFile(t, goPlan))
		if code != 0 {
			t.Fatalf("passing check exited %d:\n%s", code, out)
		}
		if strings.Contains(out, "check last:") {
			t.Errorf("PASS printed check last:\n%s", out)
		}
	})
}
