package bench

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// outcome is one instance's attempt in one configuration, in a look fixture.
type outcome struct {
	cost     float64
	solved   bool
	excluded bool
	missing  bool
}

// writeBlock writes block k of a stage5 directory: run.json with the models
// and each model's configurations, and one receipt per instance, model and
// configuration from out(instance, model, config).
func writeBlock(t *testing.T, stage string, k int, ids, models []string, configsFor map[string][]string, out func(id, model, config string) outcome) {
	t.Helper()
	dir := filepath.Join(stage, fmt.Sprintf("block-%d", k))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	put := func(name string, v any) {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put("run.json", plan{Stage: "stage5", Tasks: ids, Models: models, Configs: []string{"A", "B", "P"}, Repeats: 1, Seed: 156,
		CLIVersion: "9.9.9 (Claude Code)", Containment: "bypass", Profile: ProfileStage5, Block: k, Looks: Looks, ConfigsFor: configsFor})
	for _, id := range ids {
		for _, m := range models {
			for _, c := range configsFor[m] {
				o := out(id, m, c)
				if o.missing {
					continue
				}
				cost := o.cost
				r := Receipt{Attempt: id + "." + c + "." + m + ".1", Stage: "stage5", Task: id, Source: "swe-bench", Model: m, Config: c, Repeat: 1,
					CostUSD: &cost, Oracle: Verdict{Solved: o.solved}}
				if o.excluded {
					r.Exclusion = ExclusionSetup
					put(r.Attempt+".receipt.json", r)
					r.Attempt += ".r1"
				}
				put(r.Attempt+".receipt.json", r)
			}
		}
	}
}

func ids(block, n int) []string {
	var s []string
	for i := range n {
		s = append(s, fmt.Sprintf("i%d-%02d", block, i))
	}
	return s
}

func rowOf(t *testing.T, r LookReport, model, comparison string) LookRow {
	t.Helper()
	for _, row := range r.Rows {
		if row.Model == model && row.Comparison == comparison {
			return row
		}
	}
	t.Fatalf("no row for %s, %s against A in %+v", model, comparison, r.Rows)
	return LookRow{}
}

// TestALookUsesTheStageLevelAndOnlyWhenEligible: a look reads 99% (ADR-162
// Decision 5), solved by Tango's score interval, so 40 instances with no
// discordant outcome cannot close PAYS on an empty difference; and a look
// with a missing receipt, more than 5% exclusions or less than 95% pairing
// closes nothing.
func TestALookUsesTheStageLevelAndOnlyWhenEligible(t *testing.T) {
	// Costs whose 95% interval sits above 0 and whose 99% one does not.
	noise := func(i int) float64 { return float64((i*37)%11-5) / 10 }
	var shift float64
	found := false
	for s := 0.0; s < 0.5; s += 0.002 {
		var rel []float64
		for i := range 40 {
			rel = append(rel, s+noise(i))
		}
		lo95, _, _, _ := bootstrapAt(rel, make([]float64, 40), 0.95)
		lo99, _, _, _ := bootstrapAt(rel, make([]float64, 40), 0.99)
		if lo95 > eps && lo99 <= eps {
			shift, found = s, true
			break
		}
	}
	if !found {
		t.Fatal("no cost shift separates the 95% and the 99% interval")
	}
	stage := t.TempDir()
	cfg := map[string][]string{"m": {"A", "B"}}
	writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, cfg, func(id, m, c string) outcome {
		var i int
		fmt.Sscanf(id, "i1-%d", &i)
		if c == "B" {
			return outcome{cost: 1 + shift + noise(i), solved: true}
		}
		return outcome{cost: 1, solved: true}
	})
	r, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	if row := rowOf(t, r, "m", "B"); row.ClosedAt != 0 || row.Result.Level != 0.99 || !row.Eligible {
		t.Errorf("a cost LOSES at 95%% and not at 99%% closed at look %d (level %v, eligible %v)", row.ClosedAt, row.Result.Level, row.Eligible)
	}
	// An empty solved difference with a large saving: Tango's interval is
	// not [0, 0], so it is not PAYS; the percentile bootstrap would read it so.
	stage = t.TempDir()
	writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, cfg, func(id, m, c string) outcome {
		if c == "B" {
			return outcome{cost: 0.8, solved: true}
		}
		return outcome{cost: 1, solved: true}
	})
	if r, err = Look(stage); err != nil {
		t.Fatal(err)
	}
	if row := rowOf(t, r, "m", "B"); row.ClosedAt != 0 || row.Result.SolvedLo > -0.05 {
		t.Errorf("no discordant instance closed PAYS: %+v", row)
	}
	if lo, hi := tangoInterval(0, 0, 40, 0.99); lo > -0.1 || hi < 0.1 {
		t.Errorf("Tango's 99%% interval with nothing discordant in 40 is [%v, %v]", lo, hi)
	}
	// Ineligible looks close nothing, even when B loses outright.
	for name, broken := range map[string]func(id, m, c string) outcome{
		"a missing receipt": func(id, m, c string) outcome {
			return outcome{cost: 1, solved: c == "A", missing: c == "B" && id == "i1-07"}
		},
		"6% exclusions": func(id, m, c string) outcome {
			var i int
			fmt.Sscanf(id, "i1-%d", &i)
			return outcome{cost: 1, solved: c == "A", excluded: c == "B" && i < 3}
		},
	} {
		stage := t.TempDir()
		writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, cfg, broken)
		r, err := Look(stage)
		if err != nil {
			t.Fatal(err)
		}
		if row := rowOf(t, r, "m", "B"); row.Eligible || row.ClosedAt != 0 {
			t.Errorf("%s: the look was eligible or closed: %+v", name, row)
		}
	}
	// A missing receipt of the other comparison makes the look ineligible
	// too, and a last retry with no first try counts as missing (the review
	// of T3).
	all := map[string][]string{"m": {"A", "B", "P"}}
	for name, setup := range map[string]func(stage string){
		"a missing P receipt": func(stage string) {
			writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, all, func(id, m, c string) outcome {
				return outcome{cost: 1, solved: c == "A", missing: c == "P" && id == "i1-07"}
			})
		},
		"a retry with no first try": func(stage string) {
			writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, cfg, func(id, m, c string) outcome {
				return outcome{cost: 1, solved: c == "A", missing: c == "B" && id == "i1-07"}
			})
			cost := 1.0
			b, _ := json.Marshal(Receipt{Attempt: "i1-07.B.m.1.r2", Stage: "stage5", Task: "i1-07", Source: "swe-bench", Model: "m", Config: "B", Repeat: 1, CostUSD: &cost})
			if err := os.WriteFile(filepath.Join(stage, "block-1", "i1-07.B.m.1.r2.receipt.json"), b, 0o644); err != nil {
				t.Fatal(err)
			}
		},
	} {
		stage := t.TempDir()
		setup(stage)
		r, err := Look(stage)
		if err != nil {
			t.Fatal(err)
		}
		if row := rowOf(t, r, "m", "B"); row.Eligible || row.ClosedAt != 0 {
			t.Errorf("%s: B closed on an ineligible look: %+v", name, row)
		}
	}
	// 5% exclusions is eligible: the bound is inclusive.
	stage = t.TempDir()
	writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, cfg, func(id, m, c string) outcome {
		var i int
		fmt.Sscanf(id, "i1-%d", &i)
		return outcome{cost: 1, solved: c == "A", excluded: c == "B" && i < 2}
	})
	if r, err = Look(stage); err != nil {
		t.Fatal(err)
	}
	if row := rowOf(t, r, "m", "B"); !row.Eligible || row.ClosedAt != 1 || row.Verdict != "LOSES" {
		t.Errorf("B never solving, 5%% excluded: %+v, want LOSES at look 1", row)
	}
}

// TestAModelStopsWhenBothComparisonsAreConclusive: a comparison closes at its
// first conclusive eligible look; a model with one open comparison runs A and
// that configuration; the stage is complete after the fifth look or when no
// model runs on; a block holding an attempt the rule did not schedule is
// refused (ADR-162 Decision 5).
func TestAModelStopsWhenBothComparisonsAreConclusive(t *testing.T) {
	stage := t.TempDir()
	both := map[string][]string{"m": {"A", "B", "P"}}
	// B never solves: LOSES at look 1. P costs and solves as A: open.
	block := func(id, m, c string) outcome { return outcome{cost: 1, solved: c != "B"} }
	writeBlock(t, stage, 1, ids(1, 10), []string{"m"}, both, block)
	r, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	if row := rowOf(t, r, "m", "B"); row.ClosedAt != 1 || row.Verdict != "LOSES" {
		t.Fatalf("B against A: %+v, want LOSES at look 1", row)
	}
	if row := rowOf(t, r, "m", "P"); row.ClosedAt != 0 {
		t.Fatalf("P against A closed: %+v", row)
	}
	if got := strings.Join(r.Next["m"], ","); got != "A,P" || r.Complete {
		t.Fatalf("after look 1 the model runs %q, complete %v; want A,P and not complete", got, r.Complete)
	}
	// A block that runs B again for the model is not the rule's.
	writeBlock(t, stage, 2, ids(2, 10), []string{"m"}, both, block)
	if _, err := Look(stage); err == nil {
		t.Fatal("a block ran a closed comparison's configuration and the look accepted it")
	}
	if err := os.RemoveAll(filepath.Join(stage, "block-2")); err != nil {
		t.Fatal(err)
	}
	next := map[string][]string{"m": {"A", "P"}}
	for k := 2; k <= Looks; k++ {
		writeBlock(t, stage, k, ids(k, 10), []string{"m"}, next, block)
	}
	if r, err = Look(stage); err != nil {
		t.Fatal(err)
	}
	if !r.Complete || len(r.Next) != 0 || rowOf(t, r, "m", "B").ClosedAt != 1 || rowOf(t, r, "m", "P").ClosedAt != 0 || rowOf(t, r, "m", "P").Verdict != "INCONCLUSIVE" {
		t.Errorf("after the fifth look: %+v, want complete with nothing next, B closed at 1, P INCONCLUSIVE", r)
	}
	// A model a later block adds, and a block of two repeats, are refused.
	for name, edit := range map[string]func(p map[string]any){
		"a model added later": func(p map[string]any) { p["models"] = []string{"m", "m2"} },
		"two repeats":         func(p map[string]any) { p["repeats"] = 2 },
	} {
		path := filepath.Join(stage, "block-2", "run.json")
		orig, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var p map[string]any
		if err := json.Unmarshal(orig, &p); err != nil {
			t.Fatal(err)
		}
		edit(p)
		b, _ := json.Marshal(p)
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Look(stage); err == nil {
			t.Errorf("%s: the look accepted it", name)
		}
		if err := os.WriteFile(path, orig, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Both closing at look 1 ends the model's run.
	stage = t.TempDir()
	writeBlock(t, stage, 1, ids(1, 10), []string{"m"}, both, func(id, m, c string) outcome { return outcome{cost: 1, solved: c == "A"} })
	if r, err = Look(stage); err != nil {
		t.Fatal(err)
	}
	if len(r.Next["m"]) != 0 || !r.Complete {
		t.Errorf("both comparisons closed at look 1: next %v, complete %v", r.Next["m"], r.Complete)
	}
	if !errors.Is(func() error { _, err := Look(filepath.Join(stage, "nothing")); return err }(), os.ErrNotExist) {
		t.Error("a stage directory with no block-1 was read")
	}
}
