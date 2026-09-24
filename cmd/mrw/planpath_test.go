package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
)

// A plan file is a SHELL argument, not a path inside the tree being edited, so
// it resolves against the working directory while `-C` moves only the paths
// INSIDE the plan. That split is defensible and it is invisible: `-C repo write
// plan.mrw` fails with a bare "no such file or directory" naming a path the
// caller believes they are standing in.
//
// The refusal has to say which directory it looked in, and that -C did not
// apply — otherwise the caller's next move is to doubt the plan, not the path.
func TestAMissingPlanFileSaysWhereItLooked(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "plan.mrw"), []byte("@@ a.go 1 delete\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Stand somewhere the plan is NOT, and point --root at where it is.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(elsewhere); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer, cmd.ErrWriter = &buf, &buf

	err = cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--dry-run", "plan.mrw"})
	if err == nil {
		t.Fatal("a plan file that is not in the working directory was somehow read")
	}
	msg := err.Error() + buf.String()
	if !strings.Contains(msg, elsewhere) {
		t.Errorf("the error does not name the directory it looked in (%s):\n%s", elsewhere, msg)
	}
	if !strings.Contains(msg, "--root") && !strings.Contains(msg, "-C") {
		t.Errorf("the error does not say that -C does not apply to the plan file:\n%s", msg)
	}
}

// And the ordinary case still works: a plan beside the caller, editing a tree
// somewhere else entirely.
func TestAPlanIsReadRelativeToTheWorkingDirectory(t *testing.T) {
	// The ledger lives outside the tree; point it somewhere disposable, and use
	// --force so this test is about the PLAN PATH and nothing else.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	here := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(here, "plan.mrw"), []byte("@@ a.txt 1 replace lines=1\nONE\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(here); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer, cmd.ErrWriter = &buf, &buf

	// Not a dry run: the file at --root is the evidence that the plan beside the
	// caller was read AND that its paths resolved into the tree, which is the
	// whole of the split this pair of tests documents. mrw prints its receipt on
	// stdout rather than through cmd.Writer, so the tree is the thing to assert.
	if err := cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--force", "plan.mrw"}); err != nil {
		t.Fatalf("a plan in the working directory was not read: %v\n%s", err, buf.String())
	}
	got, err := os.ReadFile(filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ONE\n" {
		t.Errorf("the plan did not edit the tree at --root: %q", got)
	}
}

// The explanation WRAPS the original, so a caller can still ask what went
// wrong rather than matching on prose. Flagged in review: every other failure
// in main.go is an error, and this one was a string only because cli.Exit
// happened to accept both.
func TestThePlanErrorWrapsTheCause(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got := planOpenError("plan.mrw", filepath.Join(wd, "elsewhere"), fs.ErrNotExist)

	if !errors.Is(got, fs.ErrNotExist) {
		t.Errorf("the cause was flattened into prose: %v", got)
	}
	if !strings.Contains(got.Error(), "working directory") {
		t.Errorf("the explanation was lost: %v", got)
	}
	// And the branches that have nothing to explain hand the cause straight
	// back, so an ordinary `mrw write plan.mrw` gains no paragraph.
	if plain := planOpenError("/abs/plan.mrw", wd, fs.ErrNotExist); plain != fs.ErrNotExist {
		t.Errorf("an absolute path was given an explanation it does not need: %v", plain)
	}
}

// ---------------------------------------------------------------------------
// ADR-007 T3: --grep, --exclude and --files-from at the command line.
//
// `read` writes to os.Stdout directly rather than to cmd.Writer, so these
// capture the real thing. That is deliberate: the flags are a public surface
// and what a caller sees is what a pipe sees.

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	_ = r.Close()
	return string(b), runErr
}

// readIn runs `mrw -C root read <args...>` and returns everything printed.
func readIn(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	// Only when the test has not already pinned one. A test that runs two
	// commands sharing state — `iter add` then `read` — needs them to see the
	// SAME state directory, and re-setting it here silently gave the second
	// command an empty working set.
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}
	argv := append([]string{"mrw", "-C", root, "read"}, args...)
	return captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), argv)
	})
}

func grepTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// --grep with no paths walks --root. This is the ADR's headline: the caller
// stops running a searcher first and composing one spec per hit.
func TestGrepWithNoPathsWalksTheRoot(t *testing.T) {
	root := grepTree(t, map[string]string{
		"a.go":     "package a\nWANTED here\n",
		"sub/b.go": "package b\nalso WANTED\n",
		"c.go":     "package c\nnothing\n",
	})
	out, err := readIn(t, root, "--grep", "WANTED")
	if err != nil {
		t.Fatalf("walk failed: %v\n%s", err, out)
	}
	for _, want := range []string{"a.go", "sub/b.go"} {
		if !strings.Contains(out, want) {
			t.Errorf("%s was not served:\n%s", want, out)
		}
	}
	if strings.Contains(out, "c.go") {
		t.Errorf("c.go has no match and was served anyway:\n%s", out)
	}
}

// The no-argument behaviour is untouched: without --grep it is still the
// working set, which is what `mrw read` has always meant.
func TestNoArgumentsWithoutGrepStillReadsTheWorkingSet(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nWANTED\n"})
	out, err := readIn(t, root)
	if err == nil {
		t.Fatalf("an empty working set with no arguments should be a usage error:\n%s", out)
	}
	if !strings.Contains(err.Error(), "working set") {
		t.Errorf("the refusal does not mention the working set: %v", err)
	}
}

// Silence is the one output this project refuses: a pattern that matched no
// file says so, and names the pattern.
func TestGrepReportsAPatternThatMatchedNoFile(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	out, err := readIn(t, root, "--grep", "zzz-absent")
	if err == nil {
		t.Fatalf("a pattern matching nothing exited 0:\n%s", out)
	}
	if !strings.Contains(err.Error(), "zzz-absent") {
		t.Errorf("the report does not name the pattern: %v", err)
	}
}

// Rule 5: one bad path never costs the caller the answers about the good ones,
// and is never silent about itself either.
func TestGrepReportsARefusedPathAndServesTheRest(t *testing.T) {
	root := grepTree(t, map[string]string{"good.go": "package good\nWANTED\n"})
	out, _ := readIn(t, root, "--grep", "WANTED", "good.go", "absent.go")
	if !strings.Contains(out, "good.go") {
		t.Errorf("the good path was not served:\n%s", out)
	}
	if !strings.Contains(out, "absent.go") || !strings.Contains(out, "REFUSED") {
		t.Errorf("the refused path was not reported:\n%s", out)
	}
}

// The runner-up: any searcher composes with mrw in one call, and the specs
// arrive on a pipe rather than through argv quoting.
func TestFilesFromReadsSpecsFromStdin(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc Target() {}\n"})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, _ = w.WriteString("# provenance line, skipped\n\na.go:/func Target/\n")
		_ = w.Close()
	}()
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })

	out, err := readIn(t, root, "--files-from", "-")
	if err != nil {
		t.Fatalf("--files-from failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "func Target") {
		t.Errorf("the spec from stdin was not served:\n%s", out)
	}
}

// The exclusion algorithm at the CLI: a glob drops a matching file, and naming
// a directory prunes it and everything under it.
func TestExcludeDropsAMatchingFileAndPrunesADirectory(t *testing.T) {
	root := grepTree(t, map[string]string{
		"keep.txt":       "WANTED\n",
		"drop.go":        "WANTED\n",
		"vendor/deep.md": "WANTED\n",
	})
	out, err := readIn(t, root, "--grep", "WANTED", "--exclude", "*.go", "--exclude", "vendor")
	if err != nil {
		t.Fatalf("exclude failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "keep.txt") {
		t.Errorf("keep.txt was excluded and should not have been:\n%s", out)
	}
	if strings.Contains(out, "drop.go") {
		t.Errorf("--exclude '*.go' did not drop drop.go — the basename half of the match is what makes it work:\n%s", out)
	}
	if strings.Contains(out, "deep.md") {
		t.Errorf("--exclude vendor did not prune the directory:\n%s", out)
	}
}

// ADR-007:208 on the --grep path: a file the caller names is never pruned by a
// glob that matches it. The walker always did this; nothing tested it.
func TestANamedFileIsServedThroughGrepDespiteExclude(t *testing.T) {
	root := grepTree(t, map[string]string{"drop.go": "WANTED\n", "keep.txt": "WANTED\n"})
	out, err := readIn(t, root, "--grep", "WANTED", "--exclude", "*.go", "drop.go")
	if err != nil {
		t.Fatalf("a named excluded file must be served: %v\n%s", err, out)
	}
	if !strings.Contains(out, "drop.go") {
		t.Fatalf("the named file was pruned:\n%s", out)
	}
}

// Every row of the ADR's precedence table that says "usage error" is one. This
// test fails if any of them stops being an error — the precedence table is
// where a caller's mental model breaks, so it is pinned rather than described.
func TestTheDocumentedUsageErrorsAreErrors(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	for _, tc := range []struct {
		why  string
		args []string
	}{
		{"--exclude without --grep", []string{"--exclude", "*.go"}},
		{"--grep with --ast-grep", []string{"--grep", "X", "--ast-grep", "X"}},
		{"--grep with --files-from", []string{"--grep", "X", "--files-from", "-"}},
		{"--files-from with positional paths", []string{"--files-from", "-", "a.go"}},
		{"--grep with a positional spec carrying a range", []string{"--grep", "X", "a.go:1-2"}},
		{"a glob path.Match rejects", []string{"--grep", "X", "--exclude", "["}},
		{"a pattern regexp rejects", []string{"--grep", "("}},
	} {
		out, err := readIn(t, root, tc.args...)
		if err == nil {
			t.Errorf("%s: exited 0, want a usage error\n%s", tc.why, out)
		}
	}
}

// ── Findings from an independent Codex review of v0.0.15..main, 2026-09-03 ──

// The ADR's precedence table documents `mrw read --grep P @1 @2` as how a
// caller asks for the walk AND their working set. `@1` was walked as a literal
// filename instead: refused as missing, or — worse — matching a real file
// actually named `@1`.
func TestGrepResolvesWorkingSetPointers(t *testing.T) {
	root := grepTree(t, map[string]string{"sub/a.go": "package a\nWANTED\n", "b.go": "package b\nWANTED\n"})
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := readIn(t, root, "--grep", "WANTED"); err != nil {
		t.Fatalf("baseline walk failed: %v", err)
	}
	// Put one file in the working set, then name it by pointer under --grep.
	cmd := rootCommand()
	var sink bytes.Buffer
	cmd.Writer, cmd.ErrWriter = &sink, &sink
	if err := cmd.Run(context.Background(), []string{"mrw", "-C", root, "iter", "add", "sub/a.go"}); err != nil {
		t.Skipf("iter add unavailable: %v", err)
	}
	out, err := readIn(t, root, "--grep", "WANTED", "@1")
	if err != nil {
		t.Fatalf("--grep with a working-set pointer failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "sub/a.go") {
		t.Errorf("@1 did not resolve to the working-set entry:\n%s", out)
	}
	if strings.Contains(out, "@1") {
		t.Errorf("@1 was walked as a literal filename:\n%s", out)
	}
}

// An absolute path inside the root is honoured on the ordinary read path, so
// --grep must honour it too. It was joined onto the root instead, so
// `--grep P /repo/sub` looked for /repo/repo/sub and refused a directory that
// plain `read` serves. One surface honoured the convention, its sibling did not.
func TestGrepHonoursAnAbsolutePathInsideTheRoot(t *testing.T) {
	root := grepTree(t, map[string]string{"sub/a.go": "package a\nWANTED\n"})
	out, err := readIn(t, root, "--grep", "WANTED", filepath.Join(root, "sub"))
	if err != nil {
		t.Fatalf("an absolute path inside the root was refused under --grep: %v\n%s", err, out)
	}
	if !strings.Contains(out, "sub/a.go") {
		t.Errorf("nothing served:\n%s", out)
	}
}

// Precedence must depend on whether a flag was SUPPLIED, not on whether its
// value is non-empty. Testing the value let three documented usage errors
// through, and made `--grep ""` read the working set instead of walking --root.
func TestPrecedenceUsesFlagPresenceNotEmptiness(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nX\n"})
	list := filepath.Join(t.TempDir(), "l.txt")
	if err := os.WriteFile(list, []byte("a.go:/X/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		why  string
		args []string
	}{
		{`--grep "" with --files-from is still two sources`, []string{"--grep", "", "--files-from", list}},
		{`--files-from "" with a positional path is still two sources`, []string{"--files-from", "", "a.go"}},
		{`--exclude with --grep "" is still --exclude with a grep`, []string{"--exclude", "*.go", "--files-from", list}},
	} {
		if out, err := readIn(t, root, tc.args...); err == nil {
			t.Errorf("%s: exited 0\n%s", tc.why, out)
		}
	}
	// And an empty pattern is a real request — every line of every file — so it
	// WALKS rather than falling back to the working set.
	out, err := readIn(t, root, "--grep", "")
	if err != nil {
		t.Fatalf(`--grep "" should walk the root: %v\n%s`, err, out)
	}
	if !strings.Contains(out, "a.go") {
		t.Errorf(`--grep "" did not walk the root:\n%s`, out)
	}
}

// ── ADR-009 T2: `mrw stats` ────────────────────────────────────────────────

// statsIn runs `mrw -C root stats <args...>` and returns everything printed.
func statsIn(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}
	argv := append([]string{"mrw", "-C", root, "stats"}, args...)
	return captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), argv)
	})
}

// A zero tally means nothing has been RECORDED, not that nothing has failed.
// An empty measurement that reads like a good result is the silent-success
// shape this project refuses.
func TestStatsOnAFreshCheckoutSaysNothingIsRecordedYet(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	out, err := statsIn(t, root)
	if err != nil {
		t.Fatalf("stats on a fresh checkout failed: %v\n%s", err, out)
	}
	low := strings.ToLower(out)
	if !strings.Contains(low, "no plans recorded") && !strings.Contains(low, "nothing recorded") {
		t.Errorf("a zero tally must SAY it is empty rather than print zeros:\n%s", out)
	}
}

// A rate is never printed without its denominator. ADR-009's criterion is valid
// for the population that produced it, and a bare percentage is the form that
// gets quoted out of that context.
func TestStatsPrintsTheDenominatorBesideTheRate(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	writePlan(t, root, "@@ a.go 2 replace\nfunc A() { _ = 1 }\n")

	out, err := statsIn(t, root)
	if err != nil {
		t.Fatalf("stats failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "of 1 plan") {
		t.Errorf("no denominator in the output — a rate without its sample size must not ship:\n%s", out)
	}
}

func TestStatsCountsARecordedWrite(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	writePlan(t, root, "@@ a.go 2 replace\nfunc A() { _ = 1 }\n")

	out, err := statsIn(t, root)
	if err != nil {
		t.Fatalf("stats failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "applied") {
		t.Errorf("the applied plan was not counted:\n%s", out)
	}
}

// A reset that reports nothing is indistinguishable from a reset that did
// nothing.
func TestStatsResetEmptiesAndReportsWhatItDiscarded(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	writePlan(t, root, "@@ a.go 2 replace\nfunc A() { _ = 1 }\n")

	out, err := statsIn(t, root, "--reset")
	if err != nil {
		t.Fatalf("--reset failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "1") {
		t.Errorf("--reset did not say how many records it discarded:\n%s", out)
	}
	after, err := statsIn(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(after), "no plans recorded") {
		t.Errorf("the tally survived --reset:\n%s", after)
	}
}

func TestStatsJSONParses(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	writePlan(t, root, "@@ a.go 2 replace\nfunc A() { _ = 1 }\n")

	out, err := statsIn(t, root, "--json")
	if err != nil {
		t.Fatalf("--json failed: %v\n%s", err, out)
	}
	var got struct {
		Plans  int            `json:"plans"`
		Counts map[string]int `json:"counts"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("--json did not parse: %v\n%s", err, out)
	}
	if got.Plans != 1 || got.Counts["applied"] != 1 {
		t.Errorf("json = %+v, want plans 1 and applied 1", got)
	}
}

// writePlan applies a plan through the real command, so the tally is written by
// the same path a caller uses.
func writePlan(t *testing.T, root, doc string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "p.mrw")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", p})
	})
}

// ── ADR-054 T3: stats prints the whole vocabulary, and names its denominator ──

// Names() iterates only the keys present in the tally, so a counter that never
// incremented has no key and could not print. Zeus read three names and no
// failed_check on a checkout where three applied writes had broken the tree.
// Every vocabulary name prints, zero included.
func TestStatsPrintsFailedCheckEvenWhenZero(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	writePlan(t, root, "@@ a.go 2 replace\nfunc A() { _ = 1 }\n")

	out, err := statsIn(t, root)
	if err != nil {
		t.Fatalf("stats failed: %v\n%s", err, out)
	}
	for _, name := range []string{"applied", "refused_parse", "refused_apply", "check_not_run", "failed_check"} {
		if !strings.Contains(out, name) {
			t.Errorf("stats omits %q at zero — an all-green tally hides the column:\n%s", name, out)
		}
	}
	if !strings.Contains(out, "failed_check      0 of 1") && !strings.Contains(out, "failed_check    0 of 1") && !strings.Contains(out, "failed_check 0 of 1") {
		if !strings.Contains(out, "failed_check") || !strings.Contains(out, "0 of 1") {
			t.Errorf("failed_check is not printed as zero of the denominator:\n%s", out)
		}
	}
}

// The derived line names its denominator: landed = applied + failed_check +
// check_not_run, because check_not_run is "written, but no check could run"
// (exit 2) — the tree changed. Landed is not "wrote and was checked". The
// tally here is 2 applied (one --no-check, one prose), 1 failed_check and
// 1 check_not_run.
func TestStatsLandedLineUsesAppliedPlusFailedCheckPlusCheckNotRun(t *testing.T) {
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\nfunc B() {}\n",
		"notes.md":              "# n\nline\n",
		".quality-harness.json": `{"check":"exit 3"}`,
	})
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := readIn(t, root, "a.go", "notes.md"); err != nil {
		t.Fatal(err)
	}
	writeIn(t, root, "--no-check", planFile(t, "@@ a.go 2 replace\nfunc A() { _ = 1 }\n")) // applied
	writeIn(t, root, planFile(t, "@@ notes.md 2 replace\nline 2\n"))                       // applied (prose skip)
	writeIn(t, root, planFile(t, "@@ a.go 3 replace\nfunc B() { _ = 1 }\n"))               // failed_check
	// A written tree whose check could not run (exit 2). Recorded directly:
	// the fixture declares a check, so no write here can reach that exit,
	// and without this row a denominator that drops check_not_run passes.
	if err := authoring.Record(root, authoring.CheckNotRun); err != nil {
		t.Fatal(err)
	}

	out, err := statsIn(t, root)
	if err != nil {
		t.Fatalf("stats failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "landed writes: 4") || !strings.Contains(out, "failed_check 1 of those") {
		t.Errorf("no landed line naming 4 landed (2 applied + 1 failed_check + 1 check_not_run) and 1 failed_check:\n%s", out)
	}

	js, err := statsIn(t, root, "--json")
	if err != nil {
		t.Fatalf("stats --json failed: %v\n%s", err, js)
	}
	var got struct {
		Plans  int            `json:"plans"`
		Counts map[string]int `json:"counts"`
		Landed int            `json:"landed"`
		Failed int            `json:"failed_check_of_landed"`
	}
	if err := json.Unmarshal([]byte(js), &got); err != nil {
		t.Fatalf("stats --json is not JSON: %v\n%s", err, js)
	}
	if got.Landed != 4 || got.Failed != 1 || got.Plans != 4 {
		t.Errorf("json landed=%d failed_check_of_landed=%d plans=%d, want 4/1/4:\n%s", got.Landed, got.Failed, got.Plans, js)
	}
	for _, name := range []string{"applied", "refused_parse", "refused_apply", "check_not_run", "failed_check"} {
		if _, ok := got.Counts[name]; !ok {
			t.Errorf("json counts omits %q — the five keys are always present:\n%s", name, js)
		}
	}
}

// ADR-056 T2: four real writes and the block stats prints for them. A --no-check
// delta (unchecked); a checked delta in a tree whose check fails (broke); a
// clean single-line replace (candidate, not refused); a --strict-balance delta
// (refused: not landed, not priced). Then the block and its JSON.
func TestStatsPricesStrictBalance(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"f.go":                  "func A() {\n\treturn\n}\n",
		".quality-harness.json": `{"check":"exit 3"}`,
	})
	reset := func() {
		if err := os.WriteFile(filepath.Join(root, "f.go"), []byte("func A() {\n\treturn\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := readIn(t, root, "f.go"); err != nil {
			t.Fatal(err)
		}
	}
	const delta = "@@ f.go 1 replace anchor=\"func A\"\nfunc A() { return }\n"
	const clean = "@@ f.go 1 replace anchor=\"func A\"\nfunc B() {\n"
	steps := []struct {
		args []string
		plan string
		exit int
	}{
		{[]string{"--no-check"}, delta, 0},
		{nil, delta, 3},
		{[]string{"--no-check"}, clean, 0},
		{[]string{"--no-check", "--strict-balance"}, delta, 1},
	}
	for i, s := range steps {
		reset()
		out, code := writeIn(t, root, append(s.args, planFile(t, s.plan))...)
		if code != s.exit {
			t.Fatalf("write %d: exit %d, want %d:\n%s", i+1, code, s.exit, out)
		}
	}

	st, err := statsIn(t, root)
	if err != nil {
		t.Fatalf("stats: %v\n%s", err, st)
	}
	for _, want := range []string{
		"strict-balance pricing: 3 landed writes with a single-line code replace; the flag would have refused 2 — broke 1, held 0, unchecked 1",
		"false positives 0 of 1 checked (0.0%)",
		"pre-registered bar",
	} {
		if !strings.Contains(st, want) {
			t.Errorf("stats lacks %q:\n%s", want, st)
		}
	}
	js, err := statsIn(t, root, "--json")
	if err != nil {
		t.Fatalf("stats --json: %v\n%s", err, js)
	}
	var got struct {
		Pricing *struct {
			Candidates  int `json:"strict_candidates"`
			WouldRefuse int `json:"strict_would_refuse"`
			Broke       int `json:"strict_would_refuse_broke"`
			Held        int `json:"strict_would_refuse_held"`
			Unchecked   int `json:"strict_would_refuse_unchecked"`
		} `json:"pricing"`
	}
	if err := json.Unmarshal([]byte(js), &got); err != nil {
		t.Fatalf("stats --json is not JSON: %v\n%s", err, js)
	}
	if got.Pricing == nil {
		t.Fatalf("stats --json has no pricing:\n%s", js)
	}
	if p := *got.Pricing; p.Candidates != 3 || p.WouldRefuse != 2 || p.Broke != 1 || p.Held != 0 || p.Unchecked != 1 {
		t.Errorf("pricing json = %+v, want 3/2/1/0/1", p)
	}
}
