package bench

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// receiptsFor builds paired receipts: for each task, one A and one B attempt
// per repeat with the given costs and solved flags.
func receiptsFor(model, source string, costA, costB []float64, solvedA, solvedB []bool) []Receipt {
	var rs []Receipt
	for i := range costA {
		a, b := costA[i], costB[i]
		rs = append(rs,
			Receipt{Task: taskName(i), Source: source, Model: model, Config: "A", Repeat: 1, CostUSD: &a, Oracle: Verdict{Solved: solvedA[i]}},
			Receipt{Task: taskName(i), Source: source, Model: model, Config: "B", Repeat: 1, CostUSD: &b, Oracle: Verdict{Solved: solvedB[i]}})
	}
	return rs
}

func taskName(i int) string { return "t" + strings.Repeat("x", i) }

func repeat[T any](v T, n int) []T {
	s := make([]T, n)
	for i := range s {
		s[i] = v
	}
	return s
}

// TestTheAnalysisPairsByTaskAndKeepsEveryAttempt: verdicts are per model and
// per task source, never pooled; every attempt is counted, and a task with
// attempts in one configuration only is named unpaired rather than dropped.
func TestTheAnalysisPairsByTaskAndKeepsEveryAttempt(t *testing.T) {
	rs := append(receiptsFor("sonnet", "own", repeat(1.0, 10), repeat(0.5, 10), repeat(true, 10), repeat(true, 10)),
		receiptsFor("opus", "own", repeat(1.0, 10), repeat(2.0, 10), repeat(true, 10), repeat(true, 10))...)
	lone := 1.0
	rs = append(rs, Receipt{Task: "only-a", Source: "own", Model: "sonnet", Config: "A", Repeat: 1, CostUSD: &lone})
	got, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("verdicts: %d, want one per model and source (2)", len(got))
	}
	attempts := 0
	for _, v := range got {
		attempts += v.Attempts
		if v.Model == "sonnet" && (v.Tasks != 10 || strings.Join(v.Unpaired, ",") != "only-a" || v.Verdict != "PAYS") {
			t.Errorf("sonnet: %+v", v)
		}
		if v.Model == "opus" && (v.Tasks != 10 || v.Verdict != "LOSES") {
			t.Errorf("opus: %+v", v)
		}
	}
	if attempts != len(rs) {
		t.Errorf("attempts counted: %d, want every receipt (%d)", attempts, len(rs))
	}
}

// TestAPilotIsNeverCounted: a pilot's receipts sit beside the stage's (T3 keeps
// pilot/ and its rerun pilot-2/ under the stage's directory), and the
// registration counts no pilot. The rerun was counted once because only a
// directory named exactly pilot was skipped (the review of #416).
func TestAPilotIsNeverCounted(t *testing.T) {
	rs := receiptsFor("sonnet", "own", repeat(1.0, 10), repeat(1.0, 10), repeat(true, 10), repeat(true, 10))
	counted := len(rs)
	for _, p := range receiptsFor("sonnet", "own", repeat(1.0, 3), repeat(9.0, 3), repeat(true, 3), repeat(false, 3)) {
		p.Stage = "pilot"
		rs = append(rs, p)
	}
	got, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Attempts != counted {
		t.Errorf("a pilot's attempts were counted: %+v, want %d attempts", got, counted)
	}
}

// TestOnlyTheRegisteredExclusionsAreAdmitted: a start error and a setup failure
// are left out of the pairing and counted; an unsolved attempt that ran past
// its budget is not excluded; any other exclusion is refused outright.
func TestOnlyTheRegisteredExclusionsAreAdmitted(t *testing.T) {
	rs := receiptsFor("sonnet", "own", repeat(1.0, 5), repeat(1.0, 5), repeat(true, 5), repeat(true, 5))
	c := 9.0
	rs = append(rs,
		Receipt{Task: taskName(0), Source: "own", Model: "sonnet", Config: "B", Repeat: 2, CostUSD: &c, Exclusion: ExclusionStart},
		Receipt{Task: taskName(1), Source: "own", Model: "sonnet", Config: "B", Repeat: 2, CostUSD: &c, Exclusion: ExclusionSetup},
		Receipt{Task: taskName(2), Source: "own", Model: "sonnet", Config: "B", Repeat: 2, CostUSD: &c, OverBudget: true},
		// An agent that erred after its first tool call is unsolved, its
		// cost kept, though the oracle found the tree right (the review of #411).
		Receipt{Task: taskName(3), Source: "own", Model: "sonnet", Config: "B", Repeat: 2, CostUSD: &c, IsError: true, Oracle: Verdict{Solved: true}})
	got, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Excluded != 2 || got[0].Attempts != len(rs) || got[0].SolvedDiff > -0.19 {
		t.Fatalf("%+v", got)
	}
	// The over-budget attempt counts, unsolved, with its cost: B is dearer and
	// solves less for task 2 than if it had been dropped.
	if got[0].RelCost <= 0 || got[0].SolvedDiff >= 0 {
		t.Errorf("the over-budget attempt was not counted: %+v", got[0])
	}
	rs = append(rs, Receipt{Task: taskName(3), Source: "own", Model: "sonnet", Config: "A", Repeat: 2, CostUSD: &c, Exclusion: "flaky"})
	if _, err := Analyze(rs); err == nil || !strings.Contains(err.Error(), "flaky") {
		t.Errorf("an unregistered exclusion was admitted: %v", err)
	}
}

// TestTheVerdictFollowsTheRegisteredBar: constructed receipts that must give
// PAYS, LOSES and INCONCLUSIVE, and the boundaries of the bar.
func TestTheVerdictFollowsTheRegisteredBar(t *testing.T) {
	ten := repeat(true, 10)
	cases := map[string]struct {
		a, b   []float64
		sa, sb []bool
		want   string
	}{
		"half the cost, same solved": {repeat(1.0, 10), repeat(0.5, 10), ten, ten, "PAYS"},
		"exactly -10%":               {repeat(1.0, 10), repeat(0.9, 10), ten, ten, "PAYS"},
		"-9% is not enough":          {repeat(1.0, 10), repeat(0.91, 10), ten, ten, "INCONCLUSIVE"},
		"dearer":                     {repeat(1.0, 10), repeat(1.2, 10), ten, ten, "LOSES"},
		"cheaper but solves less":    {repeat(1.0, 10), repeat(0.5, 10), ten, repeat(false, 10), "LOSES"},
		"noise either way":           {[]float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, []float64{0.5, 1.5, 0.5, 1.5, 0.5, 1.5, 0.5, 1.5, 0.5, 1.5}, ten, ten, "INCONCLUSIVE"},
	}
	for name, c := range cases {
		got, err := Analyze(receiptsFor("m", "own", c.a, c.b, c.sa, c.sb))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(got) != 1 || got[0].Verdict != c.want {
			t.Errorf("%s: %+v, want %s", name, got, c.want)
		}
	}
	// The registration overlaps here: B is half the cost and every task's
	// solved rate is exactly 5 points lower, so the solved interval's lower
	// bound meets PAYS (at -5 pp) while the interval lies wholly below 0, which
	// is LOSES. Zy decided on 2026-10-14, before any stage ran: LOSES wins.
	var overlap []Receipt
	for i := range 10 {
		for r := 1; r <= 20; r++ {
			a, b := 1.0, 0.5
			overlap = append(overlap,
				Receipt{Task: taskName(i), Source: "own", Model: "m", Config: "A", Repeat: r, CostUSD: &a, Oracle: Verdict{Solved: true}},
				Receipt{Task: taskName(i), Source: "own", Model: "m", Config: "B", Repeat: r, CostUSD: &b, Oracle: Verdict{Solved: r != 20}})
		}
	}
	if got, err := Analyze(overlap); err != nil || got[0].Verdict != "LOSES" || math.Abs(got[0].SolvedLo+0.05) > 1e-9 {
		t.Errorf("the overlap: %+v %v, want LOSES with the solved lower bound at -0.05", got, err)
	}
	// Silent mis-edits are counted per configuration beside every verdict.
	rs := receiptsFor("m", "own", repeat(1.0, 3), repeat(0.5, 3), repeat(true, 3), repeat(true, 3))
	rs[1].SilentMisEdit = true
	// B's counted shell reads are reported beside the verdict (Zy, 2026-10-14);
	// A has no hook, and a count on an A receipt is not B's.
	rs[1].HookReads, rs[3].HookReads, rs[0].HookReads = 3, 2, 7
	// And B's slipped writes, the backstop for a write the hook missed.
	rs[1].SlippedWrites, rs[3].SlippedWrites, rs[0].SlippedWrites = []string{"a.go", "b.go"}, []string{"c.go"}, []string{"x.go"}
	got, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].SilentA != 0 || got[0].SilentB != 1 {
		t.Errorf("silent mis-edits: A %d, B %d, want 0 and 1", got[0].SilentA, got[0].SilentB)
	}
	if got[0].ReadsB != 5 {
		t.Errorf("B's shell reads: %d, want 5", got[0].ReadsB)
	}
	if got[0].SlippedB != 3 {
		t.Errorf("B's slipped writes: %d, want 3", got[0].SlippedB)
	}
}

// TestTwoStagesAreNeverPooled: a verdict is per stage as well as per model and
// task source (ADR-158 Decision 6). ADR-156 T6 analyses every stage's receipts
// under docs/bench in one run, so Stage 1's and Stage 1b's would otherwise be
// one verdict.
func TestTwoStagesAreNeverPooled(t *testing.T) {
	var rs []Receipt
	for _, stage := range []string{"s1b", "s1"} {
		for _, r := range receiptsFor("sonnet", "own", repeat(1.0, 10), repeat(1.0, 10), repeat(true, 10), repeat(true, 10)) {
			r.Stage = stage
			rs = append(rs, r)
		}
	}
	got, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Stage != "s1" || got[1].Stage != "s1b" || got[0].Attempts != 20 || got[1].Attempts != 20 {
		t.Errorf("two stages: %+v, want one verdict each, s1 first, 20 attempts each", got)
	}
}

// TestTheVariedMultiSiteRowIsDescriptive: where a stage's attempts include
// varied-multi-site tasks, one more row is computed over those tasks alone and
// marked descriptive; the verdict row still counts every task, and a stage with
// none of them gets no such row (ADR-158 Decision 6).
func TestTheVariedMultiSiteRowIsDescriptive(t *testing.T) {
	rs := receiptsFor("sonnet", "own", repeat(1.0, 10), repeat(1.0, 10), repeat(true, 10), repeat(true, 10))
	for i := range rs {
		rs[i].Stage = "s"
		if strings.HasSuffix(rs[i].Task, "xxxxxxx") {
			rs[i].Shape = ShapeVariedMultiSite
		}
	}
	got, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Descriptive != "" || got[0].Tasks != 10 || got[1].Descriptive != onlyVariedMultiSite || got[1].Tasks != 3 {
		t.Errorf("rows: %+v, want the verdict over 10 tasks and a descriptive row over the 3 varied ones", got)
	}
	plain := receiptsFor("sonnet", "own", repeat(1.0, 10), repeat(1.0, 10), repeat(true, 10), repeat(true, 10))
	if got, err := Analyze(plain); err != nil || len(got) != 1 {
		t.Errorf("a stage with no varied-multi-site task: %+v %v, want the verdict alone", got, err)
	}
}

// TestStageOnesPublishedVerdictIsUnchanged: grouping by stage moves nothing in
// the verdict Stage 1 published (docs/bench/stage-1-sonnet, ADR-156 T3).
func TestStageOnesPublishedVerdictIsUnchanged(t *testing.T) {
	dir := filepath.Join("..", "..", "docs", "bench", "stage-1-sonnet")
	var rs []Receipt
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".receipt.json") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var r Receipt
		if err := json.Unmarshal(b, &r); err != nil {
			return err
		}
		rs = append(rs, r)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("rows: %+v, want the verdict and its check_red_at_base row", got)
	}
	v := got[0]
	line := fmt.Sprintf("%s %s %d %d %+.3f [%+.3f, %+.3f] %+.3f [%+.3f, %+.3f] %d/%d %d %d",
		v.Stage, v.Verdict, v.Tasks, v.Attempts, v.RelCost, v.RelLo, v.RelHi, v.SolvedDiff, v.SolvedLo, v.SolvedHi, v.SilentA, v.SilentB, v.ReadsB, v.SlippedB)
	if want := "stage1 INCONCLUSIVE 30 120 +0.026 [-0.039, +0.090] +0.050 [-0.083, +0.167] 2/4 42 0"; line != want {
		t.Errorf("Stage 1's verdict:\n got %s\nwant %s", line, want)
	}
	if got[1].Descriptive != withoutRedAtBase || got[1].Tasks != 23 {
		t.Errorf("Stage 1's descriptive row: %+v", got[1])
	}
}

// TestARowWithoutTheTasksSupersededSinceIsDescriptive: given the ids of the
// tasks superseded since, a group that ran one gets a row without it, labelled
// as registered after the stage ran; the verdict still counts every task, and
// a group that ran none of them gets no such row (ADR-159 Decision 2).
func TestARowWithoutTheTasksSupersededSinceIsDescriptive(t *testing.T) {
	rs := receiptsFor("sonnet", "own", repeat(1.0, 10), repeat(1.0, 10), repeat(true, 10), repeat(true, 10))
	got, err := Analyze(rs, taskName(3), "never-ran")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Descriptive != "" || got[0].Tasks != 10 || got[1].Descriptive != withoutSupersededSince || got[1].Tasks != 9 {
		t.Errorf("rows: %+v, want the verdict over 10 tasks and a descriptive row over 9", got)
	}
	for _, ids := range [][]string{nil, {"never-ran"}} {
		if got, err := Analyze(rs, ids...); err != nil || len(got) != 1 {
			t.Errorf("superseded %v: %+v %v, want the verdict alone", ids, got, err)
		}
	}
}

// TestThePublishedStagesWithoutTheTasksSupersededSince: over the published
// receipts and the corpus's superseded tasks, each verdict is as published and
// the row leaves out the tasks superseded after its stage ran (ADR-159).
func TestThePublishedStagesWithoutTheTasksSupersededSince(t *testing.T) {
	tasks, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	var superseded []string
	for _, task := range tasks {
		if task.SupersededBy != "" {
			superseded = append(superseded, task.ID)
		}
	}
	for _, c := range []struct {
		dir, verdict string
		tasks        int
	}{
		{"stage-1-sonnet", "stage1 INCONCLUSIVE 30 120 +0.026 [-0.039, +0.090] +0.050 [-0.083, +0.167]", 27},
		{"stage-1b-sonnet", "stage1b INCONCLUSIVE 40 160 -0.004 [-0.050, +0.042] -0.013 [-0.037, +0.000]", 39},
	} {
		got, err := Analyze(publishedReceipts(t, c.dir), superseded...)
		if err != nil {
			t.Fatal(err)
		}
		v := got[0]
		if line := fmt.Sprintf("%s %s %d %d %+.3f [%+.3f, %+.3f] %+.3f [%+.3f, %+.3f]",
			v.Stage, v.Verdict, v.Tasks, v.Attempts, v.RelCost, v.RelLo, v.RelHi, v.SolvedDiff, v.SolvedLo, v.SolvedHi); line != c.verdict {
			t.Errorf("%s: the verdict\n got %s\nwant %s", c.dir, line, c.verdict)
		}
		found := false
		for _, r := range got {
			if r.Descriptive == withoutSupersededSince {
				found = true
				if r.Tasks != c.tasks {
					t.Errorf("%s: the row counts %d tasks, want %d", c.dir, r.Tasks, c.tasks)
				}
			}
		}
		if !found {
			t.Errorf("%s: no row without the tasks superseded since: %+v", c.dir, got)
		}
	}
}

// publishedReceipts reads every receipt under docs/bench/<dir>.
func publishedReceipts(t *testing.T, dir string) []Receipt {
	t.Helper()
	var rs []Receipt
	err := filepath.Walk(filepath.Join("..", "..", "docs", "bench", dir), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".receipt.json") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var r Receipt
		if err := json.Unmarshal(b, &r); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		rs = append(rs, r)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return rs
}

// TestAnalyzePrintsAStage4CategoryRow: a group holding ADR-160's categories
// gets one descriptive row per category, and a slice's registered subset its
// own row, each beside the verdict and never in its place; a group of the
// earlier stages, without those categories, gets no category row.
func TestAnalyzePrintsAStage4CategoryRow(t *testing.T) {
	var rs []Receipt
	for i, shape := range []string{ShapeWideRename, ShapeWideRename, ShapeCheckCoupled, ShapeCheckCoupled, ShapeSingleLine, ShapeSingleLine} {
		for _, r := range receiptsFor("sonnet", "own", []float64{1}, []float64{1}, []bool{true}, []bool{i%2 == 0}) {
			r.Task, r.Stage, r.Shape = fmt.Sprintf("%s-%d", shape, i), "stage4", shape
			rs = append(rs, r)
		}
	}
	for i := range 4 {
		for _, r := range receiptsFor("sonnet", "own", []float64{1}, []float64{1}, []bool{true}, []bool{true}) {
			r.Task, r.Stage, r.Shape = fmt.Sprintf("old-%d", i), "stage1b", []string{ShapeSingleLine, ShapeMultiSite}[i%2]
			rs = append(rs, r)
		}
	}
	for i := range 4 {
		for _, r := range receiptsFor("sonnet", "swe-bench", []float64{1}, []float64{1}, []bool{true}, []bool{true}) {
			r.Task, r.Stage, r.Shape = fmt.Sprintf("org__repo-%d", i), "stage-4-swe", ShapeSWEBench
			rs = append(rs, r)
		}
	}
	unnamed := Subset{Label: OnlyUnnamedFiles, Tasks: []string{"org__repo-1", "org__repo-3"}}
	got, err := AnalyzeSubsets(rs, nil, []Subset{unnamed})
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]Result{}
	for _, r := range got {
		rows[r.Stage+"|"+r.Descriptive] = r
	}
	for _, shape := range []string{ShapeWideRename, ShapeCheckCoupled, ShapeSingleLine} {
		r, ok := rows["stage4|"+onlyCategory(shape)]
		if !ok || r.Tasks != 2 || r.Verdict == "" {
			t.Errorf("stage4: no descriptive row over the 2 %s tasks: %+v", shape, r)
		}
	}
	if v, ok := rows["stage4|"]; !ok || v.Tasks != 6 {
		t.Errorf("stage4: the verdict row is not over all 6 tasks: %+v", v)
	}
	if _, ok := rows["stage1b|"+onlyCategory(ShapeSingleLine)]; ok {
		t.Error("stage1b got a category row: only a group holding ADR-160's categories does")
	}
	if r, ok := rows["stage-4-swe|"+OnlyUnnamedFiles]; !ok || r.Tasks != 2 {
		t.Errorf("stage-4-swe: no row over the 2 unnamed-files instances: %+v", r)
	}
	if got[0].Descriptive != "" {
		t.Errorf("the first row is descriptive (%q); the verdict comes first", got[0].Descriptive)
	}
}

// TestAnalyzePrintsPAgainstA: a stage that ran P gets P against A by the same
// rule as B against A, never P against B, and P's silent mis-edits count in
// its own row (ADR-162 Decision 6).
func TestAnalyzePrintsPAgainstA(t *testing.T) {
	rs := receiptsFor("m", "own", repeat(1.0, 10), repeat(0.5, 10), repeat(true, 10), repeat(true, 10))
	for i := range 10 {
		c := 2.0
		rs = append(rs, Receipt{Task: taskName(i), Source: "own", Model: "m", Config: "P", Repeat: 1, CostUSD: &c, Oracle: Verdict{Solved: true}, SilentMisEdit: i == 0})
	}
	results, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	var b, p *Result
	for i := range results {
		if results[i].Descriptive != "" {
			continue
		}
		switch results[i].Comparison {
		case "B":
			b = &results[i]
		case "P":
			p = &results[i]
		default:
			t.Errorf("a row compares %q", results[i].Comparison)
		}
	}
	if b == nil || p == nil {
		t.Fatalf("rows %+v, want B against A and P against A", results)
	}
	if b.RelCost > -0.49 || p.RelCost < 0.99 || p.Verdict != "LOSES" {
		t.Errorf("B against A %+v, P against A %+v: P is not compared with A", b, p)
	}
	if b.SilentB != 0 || p.SilentB != 1 {
		t.Errorf("P's silent mis-edit counted in B's row (%d) or missing from P's (%d)", b.SilentB, p.SilentB)
	}
	if b.Level != 0.95 || p.Level != 0.95 {
		t.Errorf("a stage without looks reads %v and %v, want 95%%", b.Level, p.Level)
	}
}
