package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-092 Decision 5: `then` is present whenever a step was asked for and the
// command got as far as a receipt. A write that landed and then could not save
// its ledger prints a receipt, and dropped the steps from it — no `then` under
// --json, no NOT RUN line in human form (the 2026-09-29 gap survey, C3). The
// tally stays ADR-083's: a --no-check landing whose ledger failed is applied,
// steps or not. The ledger file is made read-only so its save fails after the
// commit.
func TestALedgerFailureStillNamesTheStepsNotRun(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file does not stop this user from writing it")
	}
	needShell(t)
	for _, asJSON := range []bool{true, false} {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := stepsTree(t, "exit 0")
		plan := primed(t, root)
		ledger, err := seen.ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ledger, 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
		args := []string{"--no-check", "--then", "a", "--then-sh", "echo x >> log", plan}
		if asJSON {
			args = append([]string{"--json"}, args...)
		}
		out, code := writeIn(t, root, args...)
		if code != exitUsage {
			t.Fatalf("json %v: exit %d, want %d for a landing whose ledger could not be written:\n%s", asJSON, code, exitUsage, out)
		}
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); !strings.Contains(string(b), "_ = 1") {
			t.Fatalf("json %v: the write did not land, so the ledger was never reached:\n%s", asJSON, b)
		}
		if asJSON {
			r := parseThen(t, out)
			if got := thenStatuses(r); !reflect.DeepEqual(got, []string{"not_run", "not_run"}) || r.Error == "" {
				t.Errorf("statuses %v, error %q, want both not_run beside the error:\n%s", got, r.Error, out)
			}
		} else if strings.Count(out, "— NOT RUN") != 2 {
			t.Errorf("the human receipt does not name both steps not run:\n%s", out)
		}
		if got := logOf(t, root); got != "" {
			t.Errorf("json %v: a step ran: %q", asJSON, got)
		}
		stats, code := runIn(t, root, "stats", "--json")
		if code != 0 {
			t.Fatalf("stats exited %d:\n%s", code, stats)
		}
		var s struct {
			Counts map[string]int `json:"counts"`
			Landed int            `json:"landed"`
		}
		if err := json.Unmarshal([]byte(stats), &s); err != nil {
			t.Fatalf("%v\n%s", err, stats)
		}
		if s.Counts["applied"] != 1 || s.Counts["check_not_run"] != 0 || s.Landed != 1 {
			t.Errorf("json %v: counts %v landed %d, want applied 1 (ADR-083), check_not_run 0, landed 1", asJSON, s.Counts, s.Landed)
		}
	}
}

// ADR-092 Decision 5, "the check could not start": such a check still names
// every step asked for, not_run, in one document. With no temp directory the
// check cannot create its log, and `check --json` printed nothing at all and
// dropped the steps (the 2026-09-29 gap survey, C2). Two refusals stay as they
// were: one with no step asked, and a refused scope (ADR-092 T4, Out of
// Scope). The pair: the same check with its temp directory back runs the steps.
func TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	gone := filepath.Join(t.TempDir(), "gone")
	t.Setenv("TMPDIR", gone)
	if runtime.GOOS == "windows" {
		t.Setenv("TMP", gone)
		t.Setenv("TEMP", gone)
	}
	out, code := runIn(t, root, "check", "--full", "--json", "--then", "a", "--then-sh", "echo x >> log")
	if code != exitUsage {
		t.Fatalf("exit %d, want %d:\n%s", code, exitUsage, out)
	}
	end := strings.LastIndex(out, "}")
	if end < 0 {
		t.Fatalf("no JSON document on stdout:\n%s", out)
	}
	r := parseThen(t, out[:end+1])
	if got := thenStatuses(r); !reflect.DeepEqual(got, []string{"not_run", "not_run"}) || r.Error == "" {
		t.Errorf("statuses %v, error %q, want both not_run beside the error:\n%s", got, r.Error, out)
	}
	if out, code := runIn(t, root, "check", "--full", "--then", "a"); code != exitUsage || !strings.Contains(out, "— NOT RUN") {
		t.Errorf("the human report does not name the step not run: exit %d:\n%s", code, out)
	}
	if got := logOf(t, root); got != "" {
		t.Errorf("a step ran: %q", got)
	}
	if out, code := runIn(t, root, "check", "--full", "--json"); code != exitUsage || strings.Contains(out, `"error"`) {
		t.Errorf("no step asked: exit %d, want %d and no JSON document, as before:\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, root, "check", "--json", "--then", "a", "nosuchdir"); code != exitUsage || strings.Contains(out, `"then"`) {
		t.Errorf("a refused scope: exit %d, want %d and no then block, as before:\n%s", code, exitUsage, out)
	}
	back := t.TempDir()
	t.Setenv("TMPDIR", back)
	if runtime.GOOS == "windows" {
		t.Setenv("TMP", back)
		t.Setenv("TEMP", back)
	}
	out, code = runIn(t, root, "check", "--full", "--json", "--then", "a")
	if code != 0 {
		t.Fatalf("with its temp directory back: exit %d:\n%s", code, out)
	}
	if got := thenStatuses(parseThen(t, out)); !reflect.DeepEqual(got, []string{"pass"}) || logOf(t, root) != "a\n" {
		t.Errorf("with its temp directory back: statuses %v, log %q:\n%s", got, logOf(t, root), out)
	}
}
