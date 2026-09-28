package check

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// statuses lists each step's status, in order.
func statuses(r StepsResult) []string {
	var s []string
	for _, x := range r.Steps {
		s = append(s, x.Status)
	}
	return s
}

// ADR-092. Three steps each append to one file; the second exits 1. The file
// shows the third never ran, and the verdicts say so with the real exit code.
// An old check log is pruned once, after the sequence, and counted.
func TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass(t *testing.T) {
	needShell(t)
	dir := tempDirForLogs(t)
	old := time.Now().Add(-LogRetention - time.Hour)
	oldLog := filepath.Join(dir, "mrw-check-old.log")
	if err := os.WriteFile(oldLog, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(oldLog, old, old); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	res := RunSteps(context.Background(), root, Config{}, []Step{
		{Name: "one", Command: "echo 1 >> order"},
		{Name: "two", Command: "echo 2 >> order; exit 1", AdHoc: true},
		{Name: "three", Command: "echo 3 >> order"},
	})
	b, _ := os.ReadFile(filepath.Join(root, "order"))
	if got := strings.ReplaceAll(string(b), "\r", ""); got != "1\n2\n" {
		t.Errorf("the steps wrote %q, want steps 1 and 2 only", got)
	}
	if got, want := statuses(res), []string{StepPass, StepFail, StepNotRun}; !reflect.DeepEqual(got, want) {
		t.Fatalf("statuses %v, want %v", got, want)
	}
	if two := res.Steps[1]; !two.Ran || two.ExitCode != 1 || two.Name != "two" || !two.AdHoc || two.Command != "echo 2 >> order; exit 1" {
		t.Errorf("the failed step does not carry its own verdict: %+v", two)
	}
	if res.Steps[2].Ran {
		t.Errorf("a step after the failure says it ran: %+v", res.Steps[2])
	}
	if res.Pruned != 1 {
		t.Errorf("pruned %d, want 1", res.Pruned)
	}
}

// ADR-092. A step whose shell cannot start gave no verdict, so the sequence
// stops there: it is could_not_start, and the next step never runs.
func TestAStepThatCouldNotStartStopsTheSequence(t *testing.T) {
	tempDirForLogs(t)
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	res := RunSteps(context.Background(), root, Config{}, []Step{
		{Name: "a", Command: "exit 0"},
		{Name: "b", Command: "exit 0"},
	})
	if got, want := statuses(res), []string{StepCouldNotStart, StepNotRun}; !reflect.DeepEqual(got, want) {
		t.Fatalf("statuses %v, want %v", got, want)
	}
	if res.Steps[0].Ran {
		t.Errorf("a step that could not start says it ran: %+v", res.Steps[0])
	}
}

// ADR-092. A sequence whose context is already done runs nothing: the first
// step is interrupted before its process started, and the rest are not_run.
func TestACancelledSequenceReportsEveryStepNotRun(t *testing.T) {
	needShell(t)
	tempDirForLogs(t)
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := RunSteps(ctx, root, Config{}, []Step{
		{Name: "a", Command: "touch m1"},
		{Name: "b", Command: "touch m2"},
	})
	if got, want := statuses(res), []string{StepInterrupted, StepNotRun}; !reflect.DeepEqual(got, want) {
		t.Fatalf("statuses %v, want %v", got, want)
	}
	if res.Steps[0].Ran {
		t.Errorf("an interrupt that came first says the step ran: %+v", res.Steps[0])
	}
	for _, m := range []string{"m1", "m2"} {
		if _, err := os.Stat(filepath.Join(root, m)); err == nil {
			t.Errorf("%s was written by a cancelled sequence", m)
		}
	}
}

// ADR-092, amended by T5. A step with no command, a step with no name and a
// name holding a space are refused when a step is asked for, naming the step;
// a sound step beside them resolves. Load itself refuses none of them (T5).
func TestAStepWithNoCommandIsRefusedWhenAsked(t *testing.T) {
	for doc, name := range map[string]string{
		`{"steps":{"vet":""}}`:     `"vet"`,
		`{"steps":{"vet":"   "}}`:  `"vet"`,
		`{"steps":{"":"go vet"}}`:  `""`,
		`{"steps":{"go vet":"x"}}`: `"go vet"`,
	} {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(root)
		if err == nil {
			_, err = cfg.StepCommands()
		}
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: want a refusal naming %s, got %v", doc, name, err)
		}
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte(`{"steps":{"vet":"go vet ./..."}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	var cmds map[string]string
	if err == nil {
		cmds, err = cfg.StepCommands()
	}
	if err != nil || cmds["vet"] != "go vet ./..." {
		t.Errorf("a sound step did not resolve: %v %+v", err, cmds)
	}
}
