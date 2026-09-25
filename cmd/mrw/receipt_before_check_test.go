package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-072 T2. A write killed while its check ran printed nothing: the receipt
// was rendered after check.Run returned, and the tally was written after it
// too, so `mrw stats` never counted the landing. Here the check kills mrw
// itself (kill -9 $PPID: the check's shell is mrw's child), so whatever mrw
// printed before the check started is all there is — and it must be the
// receipt. A real binary, because an in-process run cannot survive the kill.
func TestTheReceiptIsOnStdoutBeforeTheCheckStarts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line that kills its parent")
	}
	bin := filepath.Join(t.TempDir(), "mrw")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	state := t.TempDir()
	root := t.TempDir()
	for n, b := range map[string]string{
		"go.mod":                "module a\n\ngo 1.26\n",
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"kill -9 $PPID"}`,
	} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) (string, error) {
		c := exec.Command(bin, append([]string{"-C", root}, args...)...)
		c.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
		out, err := c.Output()
		return string(out), err
	}
	if _, err := run("read", "a.go"); err != nil {
		t.Fatal(err)
	}
	out, err := run("write", planFile(t, goPlan))
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != -1 {
		t.Fatalf("want mrw killed by its own check, got %v:\n%s", err, out)
	}
	if !strings.Contains(out, "— applied") || !strings.Contains(out, "a.go") {
		t.Errorf("a write killed during its check printed no receipt:\n%q", out)
	}
	stats, err := run("stats")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stats, "landed writes: 1;") {
		t.Errorf("stats does not count the killed landing:\n%s", stats)
	}
}

// The pair: a check that ran and failed is counted once, as failed_check,
// and the provisional applied count is moved, not left beside it.
func TestAFailedCheckIsCountedOnceInStats(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	if out, code := writeIn(t, root, planFile(t, goPlan)); code != exitCheckFailed {
		t.Fatalf("exit %d, want %d:\n%s", code, exitCheckFailed, out)
	}
	out, code := runIn(t, root, "stats", "--json")
	if code != 0 {
		t.Fatalf("stats exited %d:\n%s", code, out)
	}
	var s struct {
		Counts map[string]int `json:"counts"`
		Landed int            `json:"landed"`
	}
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if s.Counts["failed_check"] != 1 || s.Counts["applied"] != 0 || s.Landed != 1 {
		t.Fatalf("counts %v landed %d, want failed_check 1, applied 0, landed 1", s.Counts, s.Landed)
	}
}

// A write that landed and then could not record itself in the ledger exited 2
// with nothing printed in human form, and the landing went uncounted: the tree
// changed and neither the receipt nor stats said so (review of #229). The
// ledger file is made read-only so its save fails after the commit.
func TestALandingWhoseLedgerCannotBeWrittenStillPrintsItsReceipt(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file does not stop this user from writing it")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ledger, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
	out, code := writeIn(t, root, "--no-check", planFile(t, goPlan))
	if code != exitUsage || !strings.Contains(out, "— applied") {
		t.Fatalf("exit %d, want %d with the receipt printed:\n%s", code, exitUsage, out)
	}
	stats, code := runIn(t, root, "stats")
	if code != 0 || !strings.Contains(stats, "landed writes: 1;") {
		t.Fatalf("stats does not count the landing:\n%s", stats)
	}
}

// A check that could not run after the write landed (its log file could not
// be created) was counted as applied: the error return skipped the move to
// the check's verdict. It is check_not_run (review of #229).
func TestACheckThatCannotRunIsCountedAsCheckNotRun(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	plan := planFile(t, goPlan)
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	if out, code := writeIn(t, root, plan); code != exitUsage {
		t.Fatalf("exit %d, want %d for a check that could not run:\n%s", code, exitUsage, out)
	}
	out, code := runIn(t, root, "stats", "--json")
	if code != 0 {
		t.Fatalf("stats exited %d:\n%s", code, out)
	}
	var s struct {
		Counts map[string]int `json:"counts"`
		Landed int            `json:"landed"`
	}
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if s.Counts["check_not_run"] != 1 || s.Counts["applied"] != 0 || s.Landed != 1 {
		t.Fatalf("counts %v landed %d, want check_not_run 1, applied 0, landed 1", s.Counts, s.Landed)
	}
}
