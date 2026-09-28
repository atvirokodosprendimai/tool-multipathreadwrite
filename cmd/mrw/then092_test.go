package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ADR-092. stepsTree holds a Go file, a markdown file and a harness declaring
// check and three steps; each step appends its name to `log`, and `bad` fails.
func stepsTree(t *testing.T, check string) string {
	t.Helper()
	return grepTree(t, map[string]string{
		"a.go":     "package a\nfunc A() {}\n",
		"notes.md": "# notes\nline two\n",
		".quality-harness.json": `{"check":"` + check + `","steps":{"a":"echo a >> log","b":"echo b >> log",` +
			`"bad":"echo bad >> log; exit 1"}}`,
	})
}

// logOf is what the steps appended, CRs dropped for Git's sh on Windows.
func logOf(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "log"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r", "")
}

// thenReceipt is the part of a --json receipt ADR-092 adds, and its error.
type thenReceipt struct {
	Then *struct {
		Steps []struct {
			Name     string `json:"name"`
			Command  string `json:"command"`
			AdHoc    bool   `json:"adhoc"`
			Status   string `json:"status"`
			Ran      bool   `json:"ran"`
			ExitCode int    `json:"exit_code"`
		} `json:"steps"`
		Pruned *int `json:"pruned_logs"`
	} `json:"then"`
	Error string `json:"error"`
}

func parseThen(t *testing.T, out string) thenReceipt {
	t.Helper()
	var r thenReceipt
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("not one JSON document: %v\n%s", err, out)
	}
	return r
}

func thenStatuses(r thenReceipt) []string {
	var s []string
	if r.Then != nil {
		for _, x := range r.Then.Steps {
			s = append(s, x.Status)
		}
	}
	return s
}

// primed reads a.go so a plan may edit it, and returns the Go plan's path.
func primed(t *testing.T, root string) string {
	t.Helper()
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	return planFile(t, goPlan)
}

// ADR-092. Declared steps run in the order the caller names them, not the
// file's; the receipt lists both and carries the prune count.
func TestThenRunsDeclaredStepsInTheOrderGiven(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	out, code := writeIn(t, root, "--no-check", "--json", "--then", "b", "--then", "a", primed(t, root))
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if got := logOf(t, root); got != "b\na\n" {
		t.Errorf("steps ran as %q, want b then a", got)
	}
	r := parseThen(t, out)
	if got := thenStatuses(r); !reflect.DeepEqual(got, []string{"pass", "pass"}) {
		t.Fatalf("statuses %v:\n%s", got, out)
	}
	if r.Then.Steps[0].Name != "b" || r.Then.Steps[0].Command != "echo b >> log" || r.Then.Pruned == nil {
		t.Errorf("the receipt does not name the step and the prune count:\n%s", out)
	}
}

// ADR-092. --then and --then-sh share one list in command-line order, and only
// the ad-hoc step is marked adhoc.
func TestThenShAndThenKeepTheirCommandLineOrder(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	out, code := writeIn(t, root, "--no-check", "--json", "--then", "a", "--then-sh", "echo x >> log", "--then", "b", primed(t, root))
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if got := logOf(t, root); got != "a\nx\nb\n" {
		t.Errorf("steps ran as %q, want a, x, b", got)
	}
	r := parseThen(t, out)
	var adhoc []bool
	for _, s := range r.Then.Steps {
		adhoc = append(adhoc, s.AdHoc)
	}
	if !reflect.DeepEqual(adhoc, []bool{false, true, false}) {
		t.Errorf("adhoc %v, want only the --then-sh step:\n%s", adhoc, out)
	}
}

// ADR-092. A --then naming no declared step, and a --then-sh with no command,
// are refused before anything is written, naming the declared steps; under
// --json the refusal is one document.
func TestAnUnknownStepIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, args := range [][]string{{"--then", "nope"}, {"--then-sh", "   "}} {
		root := stepsTree(t, "exit 0")
		plan := primed(t, root)
		before, _ := os.ReadFile(filepath.Join(root, "a.go"))
		out, code := writeIn(t, root, append(args, plan)...)
		after, _ := os.ReadFile(filepath.Join(root, "a.go"))
		if code != exitUsage || string(after) != string(before) {
			t.Fatalf("%v: exit %d, file changed %v, want 2 and untouched:\n%s", args, code, string(after) != string(before), out)
		}
		out, code = runIn(t, root, append(append([]string{"write", "--json"}, args...), plan)...)
		if code != exitUsage {
			t.Fatalf("%v --json: exit %d:\n%s", args, code, out)
		}
		if args[0] == "--then" && !strings.Contains(out, "a, b, bad") {
			t.Errorf("the refusal does not name the declared steps:\n%s", out)
		}
		doc := out[:strings.LastIndex(out, "}")+1]
		if r := parseThen(t, doc); r.Error == "" {
			t.Errorf("%v: the --json refusal carries no error:\n%s", args, out)
		}
	}
}

// ADR-092. The second step fails: exit 3, the third never ran, the receipt
// names it not run, and the landing is counted failed_check.
func TestAFailedStepExitsThreeAndNamesTheStepsNotRun(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	out, code := writeIn(t, root, "--no-check", "--then", "a", "--then", "bad", "--then", "b", primed(t, root))
	if code != exitCheckFailed {
		t.Fatalf("exit %d, want %d:\n%s", code, exitCheckFailed, out)
	}
	if got := logOf(t, root); got != "a\nbad\n" {
		t.Errorf("steps ran as %q, want a then bad", got)
	}
	if !strings.Contains(out, "NOT RUN") || !strings.Contains(out, "FAIL") {
		t.Errorf("the human receipt does not show the failure and the step not run:\n%s", out)
	}

	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root = stepsTree(t, "exit 0")
	out, code = writeIn(t, root, "--no-check", "--json", "--then", "a", "--then", "bad", "--then", "b", primed(t, root))
	if code != exitCheckFailed {
		t.Fatalf("--json: exit %d:\n%s", code, out)
	}
	r := parseThen(t, out)
	if got := thenStatuses(r); !reflect.DeepEqual(got, []string{"pass", "fail", "not_run"}) {
		t.Fatalf("statuses %v:\n%s", got, out)
	}
	if bad := r.Then.Steps[1]; !bad.Ran || bad.ExitCode != 1 {
		t.Errorf("the failed step's verdict: %+v", bad)
	}
	st, sc := runIn(t, root, "stats", "--json")
	var s struct {
		Counts map[string]int `json:"counts"`
	}
	if err := json.Unmarshal([]byte(st), &s); sc != 0 || err != nil || s.Counts["failed_check"] != 1 || s.Counts["applied"] != 0 {
		t.Errorf("the landing is not counted failed_check: %v\n%s", err, st)
	}
}

// ADR-092. A plan that did not land runs no step: a failed hunk (exit 1) and a
// dry run (exit 0) both report every step not_run.
func TestARefusedWriteRunsNoStep(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	primed(t, root)
	for _, c := range []struct {
		args []string
		exit int
	}{
		{[]string{planFile(t, "@@ a.go 2 replace anchor=\"not there\"\nx\n")}, exitNotApplied},
		{[]string{"--dry-run", planFile(t, goPlan)}, 0},
	} {
		out, code := writeIn(t, root, append([]string{"--json", "--then", "a", "--then-sh", "echo x >> log"}, c.args...)...)
		if code != c.exit {
			t.Fatalf("%v: exit %d, want %d:\n%s", c.args, code, c.exit, out)
		}
		if got := logOf(t, root); got != "" {
			t.Errorf("%v: a step ran: %q", c.args, got)
		}
		if got := thenStatuses(parseThen(t, out)); !reflect.DeepEqual(got, []string{"not_run", "not_run"}) {
			t.Errorf("%v: statuses %v, want every step not_run:\n%s", c.args, got, out)
		}
	}
}

// ADR-092. The steps follow a PASSING check: a demanded check on a prose plan
// that fails runs no step.
func TestAFailedCheckRunsNoStep(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 3")
	if _, err := readIn(t, root, "notes.md"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--check", "--json", "--then", "a", planFile(t, mdPlan))
	if code != exitCheckFailed {
		t.Fatalf("exit %d, want %d:\n%s", code, exitCheckFailed, out)
	}
	if got := logOf(t, root); got != "" {
		t.Errorf("a step ran after a failed check: %q", got)
	}
	if got := thenStatuses(parseThen(t, out)); !reflect.DeepEqual(got, []string{"not_run"}) {
		t.Errorf("statuses %v:\n%s", got, out)
	}
}

// ADR-092. `mrw check --then` runs the steps after a passing check; its JSON
// keeps the flat check fields and adds `then` beside them. A failing step is 3.
func TestCheckThenRunsStepsAfterAPassingCheck(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	out, code := runIn(t, root, "check", "--full", "--json", "--then", "a")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	var flat struct {
		Ran     bool   `json:"ran"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(out), &flat); err != nil || !flat.Ran || flat.Command != "exit 0" {
		t.Errorf("the flat check fields changed: %v\n%s", err, out)
	}
	if got := thenStatuses(parseThen(t, out)); !reflect.DeepEqual(got, []string{"pass"}) || logOf(t, root) != "a\n" {
		t.Errorf("statuses %v, log %q:\n%s", got, logOf(t, root), out)
	}
	if out, code := runIn(t, root, "check", "--full", "--then", "bad"); code != exitCheckFailed {
		t.Errorf("a failing step after check: exit %d, want %d:\n%s", code, exitCheckFailed, out)
	}
}

// ADR-092 Decision 7. strict-balance pricing reads the check alone: a wrap-tail
// write whose check passed and whose step failed is priced held, not broke.
func TestAStepFailureIsPricedByTheCheckAlone(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"f.go":                  "func A() {\n\treturn\n}\n",
		".quality-harness.json": `{"check":"exit 0"}`,
	})
	if _, err := readIn(t, root, "f.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--then-sh", "exit 1", planFile(t, "@@ f.go 1 replace anchor=\"func A\"\nfunc A() { return }\n"))
	if code != exitCheckFailed {
		t.Fatalf("exit %d, want %d:\n%s", code, exitCheckFailed, out)
	}
	js, err := statsIn(t, root, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Pricing *struct {
			Broke int `json:"strict_would_refuse_broke"`
			Held  int `json:"strict_would_refuse_held"`
		} `json:"pricing"`
	}
	if err := json.Unmarshal([]byte(js), &got); err != nil || got.Pricing == nil || got.Pricing.Held != 1 || got.Pricing.Broke != 0 {
		t.Errorf("pricing moved with the step: %v\n%s", err, js)
	}
}

// ADR-092. A step's value is one step whatever it holds: a comma does not split it.
func TestAValueWithACommaIsOneStep(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	out, code := writeIn(t, root, "--no-check", "--json", "--then-sh", "printf a,b > m", primed(t, root))
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "m")); string(b) != "a,b" {
		t.Errorf("m holds %q, want a,b", b)
	}
	if r := parseThen(t, out); r.Then == nil || len(r.Then.Steps) != 1 {
		t.Errorf("want one step:\n%s", out)
	}
}

// ADR-092 with ADR-069. An attached --then-sh value ending in whitespace is
// refused as every flag's is; the separate form is kept as typed and runs.
func TestAPaddedThenValueFollowsTheArgumentGuard(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	plan := primed(t, root)
	if err := refusePaddedFlagValues(rootCommand(), []string{"-C", root, "write", "--then-sh=echo p >> log ", plan}); err == nil {
		t.Error("an attached --then-sh value ending in whitespace was not refused")
	}
	if err := refusePaddedFlagValues(rootCommand(), []string{"-C", root, "write", "--then-sh", "echo p >> log ", plan}); err != nil {
		t.Errorf("the separate form was refused: %v", err)
	}
	if out, code := writeIn(t, root, "--no-check", "--then-sh", "echo p >> log ", plan); code != 0 || logOf(t, root) != "p\n" {
		t.Errorf("the separate form did not run as typed: exit %d, log %q:\n%s", code, logOf(t, root), out)
	}
}
