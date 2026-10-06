package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// captured runs render against a temp file and returns what it wrote.
func captured(t *testing.T, render func(*os.File)) string {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	render(f)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ADR-132 Decision 3. ADR-091 decided a receipt spells a root-relative path
// with "/", and only mrw_write did: the CLI's --json, its text and its drift
// carried `\` on Windows, and its created line printed the separator. Driven
// with `\` as the separator, so it runs on every platform.
func TestTheCLIReceiptSpellsPathsWithSlash(t *testing.T) {
	prev := receiptSep
	receiptSep = '\\'
	t.Cleanup(func() { receiptSep = prev })
	res := shown(apply.Result{
		Applied:     true,
		Hunks:       []apply.HunkResult{{Path: `d\f.txt`, Addr: "1", Op: "replace", Status: apply.StatusOK}},
		Files:       []apply.FileResult{{Path: `d\f.txt`, Written: true, RenamedTo: `d\g.txt`}},
		DirsCreated: []string{`d\e`},
		LeftBehind:  []string{`d\.mrw-x`},
	})
	b, err := json.Marshal(receipt{Result: res, Drift: shownPaths([]string{`d\f.txt`})})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `\\`) {
		t.Errorf("the --json receipt spells a path with a backslash: %s", b)
	}
	text := captured(t, func(f *os.File) { report(f, res, false) })
	if strings.Contains(text, `\`) || !strings.Contains(text, "created d/e/") || !strings.Contains(text, "d/f.txt") {
		t.Errorf("the text receipt spells a path with a backslash, or no created d/e/ line:\n%s", text)
	}
}

// ADR-132 Decision 4. The exit-2 line printed the whole error under a receipt
// whose FAIL row already carried its reason — 1.5 KB again at a long path. It
// names the row in the reason's place and keeps every other cause.
func TestAnExitTwoLineNamesTheFailLineInsteadOfRepeatingIt(t *testing.T) {
	reason := `b.txt cannot be renamed: a long system error at C:\a\very\long\path\b.txt`
	res := apply.Result{Hunks: []apply.HunkResult{{Path: "a.txt", Status: apply.StatusSkipped}, {Path: "b.txt", Status: apply.StatusFailed, Reason: reason}}}
	got := exitTwoLine(res, fmt.Errorf("b.txt: %s", reason))
	if strings.Contains(got, "a long system error") || !strings.Contains(got, "see its FAIL line") {
		t.Errorf("the exit-2 line repeats the FAIL row, or does not point at it: %q", got)
	}
	withLedger := fmt.Errorf("b.txt: %s (ALREADY WRITTEN: a.txt); and the ledger could not record what landed: disk gone", reason)
	got = exitTwoLine(res, withLedger)
	if strings.Contains(got, "a long system error") || !strings.Contains(got, "ALREADY WRITTEN: a.txt") || !strings.Contains(got, "the ledger could not record what landed: disk gone") {
		t.Errorf("the exit-2 line lost a cause the error carries beside the FAIL row: %q", got)
	}
	plain := fmt.Errorf("the root cannot be opened: gone")
	if got := exitTwoLine(apply.Result{}, plain); got != plain.Error() {
		t.Errorf("an error no FAIL row carries was changed: %q", got)
	}
	whole := fmt.Errorf("%s", "d/x.txt: openat d/.mrw-aside-1: permission denied (ALREADY WRITTEN: a.go)")
	res = apply.Result{Hunks: []apply.HunkResult{{Path: "d/x.txt", Status: apply.StatusFailed, Reason: whole.Error()}}}
	if got := exitTwoLine(res, whole); got != "d/x.txt: see its FAIL line" {
		t.Errorf("a commit error that is its FAIL row's whole reason lost its path: %q", got)
	}
}

// ADR-132 Decision 6. A check stopped by an interrupt or its deadline was
// headed FAIL while no process gave a verdict.
func TestAStoppedCheckIsHeadedByWhatStoppedIt(t *testing.T) {
	for _, tc := range []struct {
		r    check.Result
		want string
	}{
		{check.Result{Ran: true, Command: "sleep 30", Skipped: check.Interrupted, ExitCode: -1}, "check INTERRUPTED"},
		{check.Result{Ran: true, Command: "sleep 30", Skipped: "timed out after 5s", ExitCode: -1}, "check TIMED OUT"},
		{check.Result{Ran: false, Command: "sleep 30", Skipped: check.TimedOutBeforeStart, ExitCode: -1}, "check TIMED OUT"},
		{check.Result{Ran: false, Command: "sleep 30", Skipped: check.Interrupted, ExitCode: -1}, "check INTERRUPTED"},
	} {
		got := captured(t, func(f *os.File) { reportCheck(f, &tc.r) })
		if !strings.Contains(got, tc.want) || strings.Contains(got, "check FAIL") || strings.Contains(got, "check SKIPPED") {
			t.Errorf("%q: want %q, got:\n%s", tc.r.Skipped, tc.want, got)
		}
	}
	failed := captured(t, func(f *os.File) { reportCheck(f, &check.Result{Ran: true, Command: "false", ExitCode: 1}) })
	if !strings.Contains(failed, "check FAIL") {
		t.Errorf("the pair: a check that failed is no longer headed FAIL:\n%s", failed)
	}
}

// ADR-132 Decision 3, on the wire: the CLI write builds its receipt through the
// converter. With the separator seam set to '_', a_b.txt is shown as a/b.txt —
// observable on any platform, where a real backslash exists only on Windows.
func TestTheCLIWriteBuildsItsReceiptThroughTheConverter(t *testing.T) {
	prev := receiptSep
	receiptSep = '_'
	t.Cleanup(func() { receiptSep = prev })
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a_b.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err, out := runMrw(t, root, "read", "a_b.txt"); err != nil {
		t.Fatalf("read: %v %s", err, out)
	}
	plan := filepath.Join(t.TempDir(), "plan")
	if err := os.WriteFile(plan, []byte("@@ a_b.txt 1 replace\ny\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err, out := runMrw(t, root, "write", "--no-check", "--json", plan)
	if err != nil || !strings.Contains(out, "\"path\": \"a/b.txt\"") {
		t.Fatalf("the CLI write's receipt was not built through the converter: %v\n%s", err, out)
	}
}
