package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Zy's ruling of 2026-10-20 on Stage 5 block 2: an attempt killed at the budget
// before the CLI wrote its result line has no cost. It counts as unsolved, and
// its instance leaves the cost comparison only; an attempt with no cost that
// did not time out is still refused.
func TestAKilledAttemptWithNoCostLeavesOnlyTheCostComparison(t *testing.T) {
	stage := t.TempDir()
	both := map[string][]string{"m": {"A", "B"}}
	writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, both, func(id, m, c string) outcome { return outcome{cost: 1, solved: true} })
	base, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	b := rowOf(t, base, "m", "B").Result

	edit := func(attempt string, f func(r *Receipt)) {
		t.Helper()
		path := filepath.Join(stage, "block-1", attempt+".receipt.json")
		var r Receipt
		if err := readJSON(path, &r); err != nil {
			t.Fatal(err)
		}
		f(&r)
		out, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	killed := func(r *Receipt) { r.CostUSD, r.TimedOut, r.OverBudget, r.IsError = nil, true, true, true }
	edit("i1-07.A.m.1", killed)
	edit("i1-07.B.m.1", killed)
	r, err := Look(stage)
	if err != nil {
		t.Fatalf("a killed attempt with no cost stopped the look: %v", err)
	}
	got := rowOf(t, r, "m", "B").Result
	if got.Tasks != b.Tasks || !slices.Equal(got.CostUnpaired, []string{"i1-07"}) {
		t.Errorf("paired %d, cost-unpaired %v; want %d paired and [i1-07] out of the cost comparison", got.Tasks, got.CostUnpaired, b.Tasks)
	}
	if got.SolvedDiff != b.SolvedDiff {
		t.Errorf("both unsolved leaves the solved difference %v, got %v", b.SolvedDiff, got.SolvedDiff)
	}

	// No cost without a timeout is still refused.
	edit("i1-08.A.m.1", func(r *Receipt) { r.CostUSD = nil })
	if _, err := Look(stage); err == nil {
		t.Error("an attempt with no cost that did not time out was accepted")
	}

	// When every pair has lost its cost, success still decides: B killed at
	// the budget on all 40 against A solving them all LOSES, never an empty
	// INCONCLUSIVE (the Codex review of 80e50bd2).
	stage = t.TempDir()
	writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, both, func(id, m, c string) outcome { return outcome{cost: 1, solved: c == "A"} })
	for _, id := range ids(1, 40) {
		path := filepath.Join(stage, "block-1", id+".B.m.1.receipt.json")
		var r Receipt
		if err := readJSON(path, &r); err != nil {
			t.Fatal(err)
		}
		killed(&r)
		out, _ := json.Marshal(r)
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if r, err = Look(stage); err != nil {
		t.Fatal(err)
	}
	if row := rowOf(t, r, "m", "B"); row.Verdict != "LOSES" || row.Result.SolvedDiff != -1 || len(row.Result.CostUnpaired) != 40 {
		t.Errorf("every pair without a cost: verdict %s, solved %v, cost-unpaired %d; want LOSES, -1, 40", row.Verdict, row.Result.SolvedDiff, len(row.Result.CostUnpaired))
	}
}
