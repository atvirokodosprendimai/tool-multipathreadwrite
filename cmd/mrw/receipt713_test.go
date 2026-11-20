package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// Issue #713. A human write receipt that cannot be written (a closed standard
// output) was dropped by the buffered flush and the write exited 0, so a caller
// learned of neither the landing nor the lost receipt. The write has landed by
// then, so the exit is 2 and one line says so; the null device accepts the
// write and keeps exiting 0 (ADR-133).
func TestAHumanReceiptThatCannotBeWrittenExitsTwoSayingTheWriteLanded(t *testing.T) {
	run := func(t *testing.T, stdout *os.File) (error, string) {
		t.Helper()
		root := t.TempDir()
		plan := filepath.Join(t.TempDir(), "plan")
		if err := os.WriteFile(plan, []byte("@@ a.txt 0 create\nx\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		saved := os.Stdout
		os.Stdout = stdout
		defer func() { os.Stdout = saved }()
		err := rootCommand().Run(context.Background(), []string{"mrw", "-C", root, "write", "--no-check", plan})
		if _, serr := os.Stat(filepath.Join(root, "a.txt")); serr != nil {
			t.Errorf("the plan did not land: %v", serr)
		}
		if err == nil {
			return nil, ""
		}
		return err, err.Error()
	}

	t.Run("a closed standard output exits 2 and says the write landed", func(t *testing.T) {
		closed, err := os.CreateTemp(t.TempDir(), "closed")
		if err != nil {
			t.Fatal(err)
		}
		_ = closed.Close()
		err, msg := run(t, closed)
		if err == nil || exitCode(err) != 2 {
			t.Fatalf("exit of %v, want 2", err)
		}
		if !strings.Contains(msg, "receipt could not be written") || !strings.Contains(msg, "landed") {
			t.Errorf("the line %q does not say the receipt was lost and the write landed", msg)
		}
	})

	t.Run("the null device accepts the receipt and the write exits 0", func(t *testing.T) {
		null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = null.Close() }()
		if err, msg := run(t, null); err != nil {
			t.Errorf("a write to the null device exited %v (%s), want 0", err, msg)
		}
	})
}

// A lost receipt does not end the write early: the plan's own verdict keeps its
// exit code (a refusal 1, a failing check 3) and the check runs, so only a
// verdict that was clean becomes exit 2 (review of #722).
func TestALostReceiptKeepsTheWritesOwnExitCode(t *testing.T) {
	const bad = "@@ a.txt 9 replace\ny\n"
	const good = "@@ b.txt 0 create\nx\n"
	for _, tc := range []struct {
		name   string
		plan   string
		flags  []string
		check  string // the project's check command; "" declares none
		want   int
		marker bool // the check must have run
		says   string
	}{
		{"a refused plan", bad, nil, "", 1, false, "nothing was written"},
		{"a failed hunk in a dry run", bad, []string{"--dry-run"}, "", 1, false, "nothing was written"},
		{"a clean dry run", good, []string{"--dry-run"}, "", 2, false, "dry run, nothing written"},
		{"a failing check", good, []string{"--check"}, "echo ran >> marker; false", 3, true, "the write landed"},
		{"a passing check", good, []string{"--check"}, "echo ran >> marker", 2, true, "the write landed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.check != "" {
				cfg := `{"check":"` + tc.check + `"}`
				if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte(cfg), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			plan := filepath.Join(t.TempDir(), "plan")
			if err := os.WriteFile(plan, []byte(tc.plan), 0o644); err != nil {
				t.Fatal(err)
			}
			ro := filepath.Join(t.TempDir(), "ro")
			if err := os.WriteFile(ro, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			stdout, err := os.Open(ro) // read-only: every write to it fails
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = stdout.Close() }()
			saved := os.Stdout
			os.Stdout = stdout
			args := append([]string{"mrw", "-C", root, "write"}, tc.flags...)
			runErr := rootCommand().Run(context.Background(), append(args, plan))
			os.Stdout = saved
			if runErr == nil || exitCode(runErr) != tc.want {
				t.Fatalf("exit of %v, want %d", runErr, tc.want)
			}
			if !strings.Contains(runErr.Error(), "receipt could not be written") || !strings.Contains(runErr.Error(), tc.says) {
				t.Errorf("the line %q does not name the lost receipt and %q", runErr.Error(), tc.says)
			}
			_, merr := os.Stat(filepath.Join(root, "marker"))
			if ran := merr == nil; ran != tc.marker {
				t.Errorf("the check ran = %v, want %v", ran, tc.marker)
			}
		})
	}
}

// sinkAfter takes ok writes and fails every one after, as a standard output
// that goes away partway through a write does.
type sinkAfter struct{ ok int }

func (s *sinkAfter) Write(p []byte) (int, error) {
	if s.ok > 0 {
		s.ok--
		return len(p), nil
	}
	return 0, errors.New("sink closed")
}

// Review of #722, P2-2 and P2-4. A report written after the receipt (the check's
// or a step's) that cannot be written must not leave the write exiting 0, and
// every refusal after a lost receipt names it, with the commit's own words and
// not a pointer to a FAIL line that never reached standard output.
func TestALostCheckOrStepReportOrRefusalStillNamesTheLostReceipt(t *testing.T) {
	const cfg = `{"check":"true","steps":{"s":"true"}}`
	run := func(t *testing.T, ok int, plan string, env map[string]string, args ...string) (error, string) {
		t.Helper()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
		pf := filepath.Join(t.TempDir(), "plan")
		if err := os.WriteFile(pf, []byte(plan), 0o644); err != nil {
			t.Fatal(err)
		}
		for k, v := range env {
			t.Setenv(k, v)
		}
		receiptSink = &sinkAfter{ok: ok}
		defer func() { receiptSink = nil }()
		err := rootCommand().Run(context.Background(), append(append([]string{"mrw", "-C", root, "write"}, args...), pf))
		if err == nil {
			return nil, ""
		}
		return err, err.Error()
	}
	const create = "@@ b.txt 0 create\nx\n"
	for _, tc := range []struct {
		name string
		ok   int
		args []string
	}{
		{"the check report", 1, []string{"--check"}},
		{"the step report", 2, []string{"--check", "--then", "s"}},
	} {
		t.Run(tc.name+" cannot be written", func(t *testing.T) {
			err, msg := run(t, tc.ok, create, nil, tc.args...)
			if err == nil || exitCode(err) != 2 || !strings.Contains(msg, "receipt could not be written") {
				t.Errorf("exit of %v, want 2 naming the lost receipt", err)
			}
		})
	}
	t.Run("a check that could not run still names the lost receipt", func(t *testing.T) {
		gone := filepath.Join(t.TempDir(), "gone")
		err, msg := run(t, 0, create, map[string]string{"TMPDIR": gone, "TEMP": gone, "TMP": gone}, "--check")
		if err == nil || exitCode(err) != 2 || !strings.Contains(msg, "receipt could not be written") || !strings.Contains(msg, "mrw-check-") {
			t.Errorf("exit of %v with %q, want 2 with the check log's own error and the lost receipt", err, msg)
		}
	})
	// A write that stops at exit 2 goes through the branch a commit failure
	// takes. The stop was once two creates of q.txt and q.txt/x, which #701 now
	// refuses at validation (exit 1); a directory named in a plan stops at load
	// with its verdict (#703), on the same branch.
	t.Run("a write that stops names the lost receipt and the error, not a FAIL line", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
			t.Fatal(err)
		}
		pf := filepath.Join(t.TempDir(), "plan")
		if err := os.WriteFile(pf, []byte("@@ d 1 replace\nz\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		receiptSink = &sinkAfter{ok: 0}
		defer func() { receiptSink = nil }()
		err := rootCommand().Run(context.Background(), []string{"mrw", "-C", root, "write", "--no-check", pf})
		if err == nil || exitCode(err) != 2 {
			t.Fatalf("the write did not stop with exit 2: %v", err)
		}
		if msg := err.Error(); !strings.Contains(msg, "receipt could not be written") || strings.Contains(msg, "see its FAIL line") {
			t.Errorf("the line %q does not name the lost receipt in full", msg)
		}
	})
	t.Run("the pair: every report written leaves a passing write at exit 0", func(t *testing.T) {
		if err, msg := run(t, 9, create, nil, "--check", "--then", "s"); err != nil {
			t.Errorf("a write whose reports were all written exited %v (%s), want 0", err, msg)
		}
	})
}

// Final review of #722. The sentence says what became of the write by
// writer.MutationOf: a commit that failed after one file landed is partial, not
// "nothing was written".
func TestAReceiptLostSentenceSaysWhatBecameOfThePlan(t *testing.T) {
	lost := errors.New("sink closed")
	for _, tc := range []struct {
		name string
		res  apply.Result
		want string
	}{
		{"a refusal", apply.Result{}, "nothing was written"},
		{"a dry run", apply.Result{DryRun: true}, "dry run, nothing written"},
		{"a complete write", apply.Result{Applied: true, Files: []apply.FileResult{{Written: true}}}, "the write landed"},
		{"a commit that failed after one file landed", apply.Result{Failed: 1, Files: []apply.FileResult{{Written: true}, {}}}, "PARTIALLY landed"},
	} {
		if got := receiptLost(tc.res, lost).Error(); !strings.Contains(got, tc.want) {
			t.Errorf("%s: %q does not say %q", tc.name, got, tc.want)
		}
	}
}

// Final review of #722. The repeat-pattern advisory is part of the human output:
// a write whose pattern line cannot be written exits 2 naming the lost receipt,
// and one whose every line was written exits 0.
func TestALostPatternLineIsNotDropped(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("x := f(1)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	write := func(body string, sink *sinkAfter) error {
		if out, code := runIn(t, root, "read", "a.go"); code != 0 {
			t.Fatalf("read: %d %s", code, out)
		}
		pf := filepath.Join(t.TempDir(), "plan")
		if err := os.WriteFile(pf, []byte("@@ a.go 1 replace\n"+body+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if sink != nil {
			receiptSink = sink
			defer func() { receiptSink = nil }()
		}
		return rootCommand().Run(context.Background(), []string{"mrw", "-C", root, "write", "--no-check", pf})
	}
	// Each write moves the parens' balance, so each carries the advisory; the
	// third makes the pattern line due.
	for _, body := range []string{"x := f(1,", "x := f(1)", "x := f(1,"} {
		if err := write(body, nil); err != nil {
			t.Fatalf("priming write: %v", err)
		}
	}
	if err := write("x := f(1)", &sinkAfter{ok: 1}); err == nil || exitCode(err) != 2 || !strings.Contains(err.Error(), "receipt could not be written") {
		t.Errorf("a lost pattern line exited %v, want 2 naming the lost receipt", err)
	}
	if err := write("x := f(1,", &sinkAfter{ok: 9}); err != nil {
		t.Errorf("a write whose lines were all written exited %v, want 0", err)
	}
}
