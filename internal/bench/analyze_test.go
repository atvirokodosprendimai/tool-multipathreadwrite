package bench

import (
	"math"
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
