package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const baselineRule = `the 4 ids with the lowest sha256("adr-163:" + instance_id)`

// baselineRig is a stage6 rig whose tasks are fixtures with a swe image, each
// carrying a `status` file the declared check `sh status` runs: its body is
// the check's behaviour. A claude stand-in on PATH marks any model start.
type baselineRig struct {
	*stage6Rig
	opts     BaselineOptions
	selected []string
	model    string
}

func newBaselineRig(t *testing.T, statuses ...[2]string) *baselineRig {
	t.Helper()
	g := newStage6Rig(t)
	dir := t.TempDir()
	put(t, filepath.Join(dir, "checks.json"), `[{"repo":"fixture","command":"sh status","source":"setup.cfg:1@`+strings.Repeat("a", 40)+`"}]`+"\n", 0o644)
	model := filepath.Join(dir, "model-started")
	put(t, filepath.Join(dir, "bin", "claude"), "#!/bin/sh\ntouch "+shellQuote(model)+"\n", 0o755)
	t.Setenv("PATH", filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := &baselineRig{stage6Rig: g, model: model}
	var ids []string
	for _, s := range statuses {
		id, body := s[0], s[1]
		put(t, filepath.Join(dir, "tasks", id, "base", "status"), body+"\n", 0o644)
		put(t, filepath.Join(dir, "tasks", id, "base", ".gitignore"), "build/\n", 0o644)
		r.opts.Tasks = append(r.opts.Tasks, Task{ID: id, Shape: ShapeSWEBench, Repo: "fixture", Dir: filepath.Join(dir, "tasks", id), swe: &sweInstance{image: fakeImage}})
		ids = append(ids, id)
	}
	sel, err := json.Marshal(map[string]any{"rule": baselineRule, "dataset": "SWE-bench/SWE-bench_Verified", "revision": "78f4", "split": "test", "selected": ids})
	if err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(dir, "selection.json"), string(sel), 0o644)
	r.selected = ids
	r.opts.Stage, r.opts.Out, r.opts.Scratch = "base", filepath.Join(dir, "out"), filepath.Join(dir, "scratch")
	r.opts.Docker, r.opts.Checks, r.opts.Budget = g.opts.Docker, filepath.Join(dir, "checks.json"), DefaultBudget
	r.opts.Selection, r.opts.Surviving = filepath.Join(dir, "selection.json"), filepath.Join(dir, "surviving.json")
	r.opts.SWE = &SWEGrader{Docker: g.opts.Docker}
	return r
}

func (r *baselineRig) rows(t *testing.T) []BaselineRow {
	t.Helper()
	var f struct{ Instances []BaselineRow }
	if err := readJSON(filepath.Join(r.opts.Out, "baseline.json"), &f); err != nil {
		t.Fatal(err)
	}
	return f.Instances
}

// TestTheBaselineRecordsGreenRedAndTime: a declared check exiting 0 on the
// unmodified tree is green, one exiting non-zero is red with its exit code, a
// container that cannot start is red with its reason; the wall time is
// recorded; each instance runs once and no model is started (ADR-163
// Decision 4).
func TestTheBaselineRecordsGreenRedAndTime(t *testing.T) {
	r := newBaselineRig(t, [2]string{"i-green", "exit 0"}, [2]string{"i-red", "exit 3"}, [2]string{"i-slow", "sleep 0.4; exit 0"})
	if err := Baseline(r.opts); err != nil {
		t.Fatal(err)
	}
	rows := r.rows(t)
	if len(rows) != 3 {
		t.Fatalf("%d rows, want 3: %+v", len(rows), rows)
	}
	if g := rows[0]; g.Instance != "i-green" || !g.Green || g.ExitCode != 0 || g.Reason != "" {
		t.Errorf("a check exiting 0: %+v", g)
	}
	if g := rows[1]; g.Instance != "i-red" || g.Green || g.ExitCode != 3 {
		t.Errorf("a check exiting 3: %+v", g)
	}
	if g := rows[2]; !g.Green || g.WallMS < 350 || g.WallMS > 30000 {
		t.Errorf("a check that sleeps 0.4 s: %+v, want its wall time in wall_ms", g)
	}
	for _, id := range r.selected {
		if n := len(r.callsMatching("run -d", "bench-baseline."+id+"-")); n != 1 {
			t.Errorf("%s: %d containers started, want 1", id, n)
		}
	}
	if n := len(r.callsMatching("exec", "sh status")); n != 3 {
		t.Errorf("the check ran %d times for 3 instances", n)
	}
	if n := len(r.callsMatching("image rm")); n != 3 {
		t.Errorf("%d images dropped, want one per instance", n)
	}
	if n := len(r.callsMatching("pull")); n != 0 {
		t.Errorf("a present image was pulled %d times", n)
	}
	if _, err := os.Stat(r.model); err == nil {
		t.Error("a model was started by a baseline")
	}
	// A second baseline into the same directory is refused before anything runs:
	// one run decides. Any error is not enough — a later failure would also be one.
	if err := Baseline(r.opts); err == nil || !strings.Contains(err.Error(), "holds a baseline already") {
		t.Errorf("a second baseline into the same directory: %v, want the refusal naming the baseline", err)
	}
	if n := len(r.callsMatching("exec", "sh status")); n != 3 {
		t.Errorf("a refused second baseline ran the check: %d runs, want 3", n)
	}

	// A container that cannot start is red with its reason, and the stage goes on.
	f := newBaselineRig(t, [2]string{"i-nostart", "exit 0"})
	put(t, filepath.Join(f.root, "fail-run"), "", 0o644)
	if err := Baseline(f.opts); err != nil {
		t.Fatal(err)
	}
	if g := f.rows(t)[0]; g.Green || g.ExitCode != -1 || !strings.Contains(g.Reason, "boom") {
		t.Errorf("a container that cannot start: %+v, want red, exit -1 and the reason", g)
	}
}

// TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder: with reds interleaved,
// the surviving file holds exactly the greens in the selection's order, the
// rule, the source's sha and the red count, and no red id; block K of it is
// instances 40(K-1)+1 to 40K and a last block of fewer is accepted (ADR-163
// Decision 4).
func TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder(t *testing.T) {
	r := newBaselineRig(t,
		[2]string{"zeta-1", "exit 0"}, [2]string{"alpha-2", "exit 1"}, [2]string{"mid-3", "exit 0"},
		[2]string{"beta-4", "exit 0"}, [2]string{"omega-5", "exit 7"}, [2]string{"gamma-6", "exit 0"})
	if err := Baseline(r.opts); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(r.opts.Surviving)
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Rule, Dataset, Revision, Split string
		Selected                       []string
		SurvivingRule                  string `json:"surviving_rule"`
		SourceSHA256                   string `json:"source_selection_sha256"`
		RedCount                       int    `json:"red_count"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(s.Selected, ","); got != "zeta-1,mid-3,beta-4,gamma-6" {
		t.Errorf("the surviving pool is %s, want the greens in pool order", got)
	}
	src, err := os.ReadFile(r.opts.Selection)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(src)
	if s.RedCount != 2 || s.SourceSHA256 != hex.EncodeToString(sum[:]) || s.Rule != baselineRule || s.Dataset == "" || s.Revision == "" || s.Split == "" || s.SurvivingRule == "" {
		t.Errorf("the surviving file lacks its rule, source sha or red count: %s", raw)
	}
	for _, red := range []string{"alpha-2", "omega-5"} {
		if strings.Contains(string(raw), red) {
			t.Errorf("the red instance %s is named in the surviving file", red)
		}
	}
	// A surviving file naming other instances than the selection is refused.
	r.opts.Out, r.opts.Surviving = filepath.Join(filepath.Dir(r.opts.Out), "out2"), filepath.Join(filepath.Dir(r.opts.Out), "other.json")
	r.opts.Tasks = r.opts.Tasks[:2]
	if err := Baseline(r.opts); err == nil {
		t.Error("tasks that are not the selection's instances were accepted")
	}

	var pool []Task
	for i := 1; i <= 85; i++ {
		pool = append(pool, Task{ID: fmt.Sprintf("p%02d", i)})
	}
	ids := func(ts []Task) string { return ts[0].ID + ".." + ts[len(ts)-1].ID + fmt.Sprint(" ", len(ts)) }
	for k, want := range map[int]string{1: "p01..p40 40", 2: "p41..p80 40", 3: "p81..p85 5"} {
		got, err := SurvivingBlock(pool, k)
		if err != nil || ids(got) != want {
			t.Errorf("block %d = %v, %v, want %s", k, got, err, want)
		}
	}
	for _, k := range []int{0, 4} {
		if got, err := SurvivingBlock(pool, k); err == nil {
			t.Errorf("block %d of 85 instances was %d tasks, want a refusal", k, len(got))
		}
	}
	if got, err := SurvivingBlock(append(pool, make([]Task, 130)...), 6); err != nil || len(got) != 15 {
		t.Errorf("block 6 of 215: %d tasks, %v: the blocks are the pool's, not five", len(got), err)
	}
}

// exportRig writes a stage directory whose run.json and receipts are those of
// a model's blocks under profile, with one cap-hit and one clean W/P attempt.
func exportRig(t *testing.T, profile, arm string) (dir string, o ExportOptions) {
	t.Helper()
	dir = t.TempDir()
	write := func(name string, v any) {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		put(t, filepath.Join(dir, name), string(b), 0o644)
	}
	yes := true
	stage := profile
	write("run.json", plan{Stage: stage, Tasks: []string{"t1", "t3"}, Models: []string{"m1"}, Configs: []string{"A", arm}, Repeats: 1, Seed: 1,
		CLIVersion: "9.9.9", Containment: "bypass", Profile: profile, Plugin: &PluginRef{Tag: "v3.8.39", Commit: strings.Repeat("a", 40)},
		StatusLineSHA256: strings.Repeat("b", 64), PromptSHA256: strings.Repeat("d", 64), BudgetSeconds: 1200})
	receipt := func(task, config string, over, timed bool) {
		id := task + "." + config + ".m1.1"
		write(id+".receipt.json", Receipt{Attempt: id, Stage: stage, Task: task, Model: "m1", Config: config, Repeat: 1, Status: "done",
			Oracle: Verdict{HarnessResolved: &yes}, OverBudget: over, TimedOut: timed})
	}
	receipt("t1", "A", false, false)
	receipt("t1", arm, true, false) // past the turn cap, claiming done, resolved
	receipt("t3", "A", false, false)
	receipt("t3", arm, false, true) // past the wall time
	write("canary.json", map[string]any{"pass": true})
	return dir, ExportOptions{Dirs: []string{dir}, X: "A", Y: arm, Model: "m1", ScorerSHA: strings.Repeat("c", 40), Canary: filepath.Join(dir, "canary.json")}
}

func exported(t *testing.T, o ExportOptions) (config map[string]any, rows map[string]pairedRow) {
	t.Helper()
	var out bytes.Buffer
	if err := ExportPaired(o, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var head struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &head); err != nil {
		t.Fatal(err)
	}
	rows = map[string]pairedRow{}
	for _, l := range lines[1:] {
		var r pairedRow
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatal(err)
		}
		rows[r.Instance+"."+r.Arm] = r
	}
	return head.Config, rows
}

// TestExportPairedTakesAWAgainstTheSurvivingPool: --pair A,W writes arms A and
// W, and the config line's instance_list_sha256 is the sha256 of the
// surviving file's selected ids joined by a newline with no trailing newline,
// its instance_count their number, not ADR-162's 191 (ADR-163 Decision 4).
func TestExportPairedTakesAWAgainstTheSurvivingPool(t *testing.T) {
	r := newBaselineRig(t, [2]string{"t1", "exit 0"}, [2]string{"t2", "exit 1"}, [2]string{"t3", "exit 0"})
	if err := Baseline(r.opts); err != nil {
		t.Fatal(err)
	}
	_, o := exportRig(t, ProfileStage6, "W")
	o.Selection = r.opts.Surviving
	config, rows := exported(t, o)
	sum := sha256.Sum256([]byte("t1\nt3"))
	if config["instance_list_sha256"] != hex.EncodeToString(sum[:]) || config["instance_count"] != float64(2) {
		t.Errorf("the config line names %v instances with digest %v, want the surviving pool's 2 and %x", config["instance_count"], config["instance_list_sha256"], sum)
	}
	if len(rows) != 4 || rows["t1.A"].Arm != "A" || rows["t1.W"].Arm != "W" || rows["t3.W"].Arm != "W" {
		t.Errorf("rows for A and W: %+v", rows)
	}
	if config["selection_salt"] != "adr-163:" {
		t.Errorf("selection_salt = %v: the surviving file keeps the source's rule", config["selection_salt"])
	}
	o.Selection = realSelection
	if config, _ := exported(t, o); config["instance_count"] != float64(191) {
		t.Errorf("ADR-162's selection counts %v, want its 191", config["instance_count"])
	}
}

// TestACappedRunExportsUnresolvedWithNoDoneClaim: under stage6 a receipt over
// the turn cap or the wall time, however it claimed and the harness said,
// exports `not done`, unresolved, limit true; under stage5 the row is today's
// (ADR-163 Decision 3).
func TestACappedRunExportsUnresolvedWithNoDoneClaim(t *testing.T) {
	sel := filepath.Join(t.TempDir(), "selection.json")
	put(t, sel, `{"rule":"`+strings.ReplaceAll(baselineRule, `"`, `\"`)+`","dataset":"d","revision":"r","split":"test","selected":["t1","t3"]}`, 0o644)
	_, o := exportRig(t, ProfileStage6, "W")
	o.Selection = sel
	_, rows := exported(t, o)
	want := pairedRow{Status: "not done", Resolved: false, Limit: true}
	for _, k := range []string{"t1.W", "t3.W"} {
		if got := rows[k]; got.Status != want.Status || got.Resolved != want.Resolved || got.Limit != want.Limit {
			t.Errorf("%s capped: %+v, want not done, unresolved, limit", k, got)
		}
	}
	if got := rows["t1.A"]; got.Status != "done" || !got.Resolved || got.Limit {
		t.Errorf("an uncapped arm A row changed: %+v", got)
	}
	_, o = exportRig(t, ProfileStage5, "P")
	o.Selection = sel
	_, rows = exported(t, o)
	for _, k := range []string{"t1.P", "t3.P"} {
		if got := rows[k]; got.Status != "done" || !got.Resolved || !got.Limit {
			t.Errorf("stage5 %s: %+v, want today's row: done, resolved, limit", k, got)
		}
	}
}
