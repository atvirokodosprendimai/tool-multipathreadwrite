package bench

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// outcome is one instance's attempt in one configuration, in a look fixture.
type outcome struct {
	cost     float64
	solved   bool
	excluded bool
	missing  bool
	reads    int  // H's host Read calls
	timedOut bool // killed at the budget
}

// writeBlock writes block k of a stage5 directory: run.json with the models
// and each model's configurations, and one receipt per instance, model and
// configuration from out(instance, model, config).
func writeBlock(t *testing.T, stage string, k int, ids, models []string, configsFor map[string][]string, out func(id, model, config string) outcome) {
	t.Helper()
	writeBlockUnder(t, ProfileStage5, stage, k, ids, models, configsFor, out)
}

// writeBlockUnder is writeBlock under any profile; stage7's run.json also
// names the sentence's SHA-256, as the runner writes it.
func writeBlockUnder(t *testing.T, profile, stage string, k int, ids, models []string, configsFor map[string][]string, out func(id, model, config string) outcome) {
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
	p := plan{Stage: profile, Tasks: ids, Models: models, Configs: []string{"A", "B", "P"}, Repeats: 1, Seed: 156,
		CLIVersion: "9.9.9 (Claude Code)", Containment: "bypass", Profile: profile, Block: k, Looks: Looks, ConfigsFor: configsFor}
	if profile == ProfileStage7 {
		p.HSentenceSHA256 = hSentenceSHA256()
	}
	put("run.json", p)
	for _, id := range ids {
		for _, m := range models {
			for _, c := range configsFor[m] {
				o := out(id, m, c)
				if o.missing {
					continue
				}
				cost := o.cost
				r := Receipt{Attempt: id + "." + c + "." + m + ".1", Stage: profile, Task: id, Source: "swe-bench", Model: m, Config: c, Repeat: 1,
					CostUSD: &cost, Oracle: Verdict{Solved: o.solved}, HostReads: o.reads, TimedOut: o.timedOut}
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

// index is the instance number inside a fixture id.
func index(id string) int {
	n, _ := strconv.Atoi(id[strings.Index(id, "-")+1:])
	return n
}

// rowBase finds a model's comparison of one configuration with one baseline.
func rowBase(t *testing.T, r LookReport, model, comparison, base string) LookRow {
	t.Helper()
	for _, row := range r.Rows {
		if row.Model == model && row.Comparison == comparison && row.Baseline == base {
			return row
		}
	}
	t.Fatalf("no row for %s, %s against %s in %+v", model, comparison, base, r.Rows)
	return LookRow{}
}

// rewriteReceipt edits one receipt of block k in place.
func rewriteReceipt(t *testing.T, stage string, k int, attempt string, edit func(*Receipt)) {
	t.Helper()
	path := filepath.Join(stage, fmt.Sprintf("block-%d", k), attempt+".receipt.json")
	var r Receipt
	if err := readJSON(path, &r); err != nil {
		t.Fatal(err)
	}
	edit(&r)
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

var stage7All = map[string][]string{"m": {"A", "B", "H"}}

// TestLookReadsHAgainstBAndA: under stage7 a model has three comparisons, each
// read against its own baseline; with B against A and H against A closed and H
// against B open, A is not scheduled and B and H are; a closed row does not
// move when later attempts land (ADR-192 Decision 3).
func TestLookReadsHAgainstBAndA(t *testing.T) {
	stage := t.TempDir()
	cost := map[string]float64{"A": 1, "B": 1.4, "H": 1.2}
	writeBlockUnder(t, ProfileStage7, stage, 1, ids(1, 40), []string{"m"}, stage7All, func(id, m, c string) outcome {
		return outcome{cost: cost[c], solved: true, reads: 1}
	})
	r1, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		cmp, base string
		rel       float64
		closed    int
		verdict   string
	}{
		{"B", "A", 0.4, 1, "LOSES"},
		{"H", "A", 0.2, 1, "LOSES"},
		{"H", "B", (1.2 - 1.4) / 1.4, 0, "INCONCLUSIVE"},
	} {
		row := rowBase(t, r1, "m", want.cmp, want.base)
		if math.Abs(row.Result.RelCost-want.rel) > 1e-9 || row.ClosedAt != want.closed || row.Verdict != want.verdict || !row.Eligible {
			t.Errorf("%s against %s: cost %v closed at %d %s eligible %v, want cost %v closed at %d %s", want.cmp, want.base, row.Result.RelCost, row.ClosedAt, row.Verdict, row.Eligible, want.rel, want.closed, want.verdict)
		}
	}
	if got := strings.Join(r1.Next["m"], ","); got != "B,H" || r1.Complete {
		t.Errorf("after look 1 the model runs %q, complete %v; want B,H and not complete", got, r1.Complete)
	}
	for _, line := range []string{"B against A", "H against A", "H against B"} {
		if !strings.Contains(r1.Text(), line) {
			t.Errorf("the look does not print %q:\n%s", line, r1.Text())
		}
	}
	// A block that runs A again is not the rule's.
	writeBlockUnder(t, ProfileStage7, stage, 2, ids(2, 40), []string{"m"}, stage7All, func(id, m, c string) outcome {
		return outcome{cost: 1, solved: true, reads: 1}
	})
	if _, err := Look(stage); err == nil {
		t.Fatal("a block ran A after its comparisons closed and the look accepted it")
	}
	if err := os.RemoveAll(filepath.Join(stage, "block-2")); err != nil {
		t.Fatal(err)
	}
	// B and H land dearer than ever: the closed rows keep their reading, H against B reads both blocks.
	writeBlockUnder(t, ProfileStage7, stage, 2, ids(2, 40), []string{"m"}, map[string][]string{"m": {"B", "H"}}, func(id, m, c string) outcome {
		return outcome{cost: map[string]float64{"B": 1, "H": 3}[c], solved: true, reads: 1}
	})
	r2, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range [][2]string{{"B", "A"}, {"H", "A"}} {
		before, after := rowBase(t, r1, "m", key[0], key[1]), rowBase(t, r2, "m", key[0], key[1])
		if after.ClosedAt != 1 || after.Verdict != before.Verdict || after.Result.RelCost != before.Result.RelCost || after.Result.Tasks != before.Result.Tasks {
			t.Errorf("%s against %s moved after it closed: %+v became %+v", key[0], key[1], before, after)
		}
	}
	if hb := rowBase(t, r2, "m", "H", "B"); hb.Result.Tasks != 80 || hb.ClosedAt != 2 || hb.Verdict != "LOSES" {
		t.Errorf("H against B after two looks: %d tasks, closed at %d %s; want 80, 2, LOSES", hb.Result.Tasks, hb.ClosedAt, hb.Verdict)
	}
}

// TestHAgainstBIsIneligibleWhenBIsNot: three excluded B attempts among 40 make
// H against B ineligible while A and H are complete, so it closes nothing; with
// B whole it closes (ADR-192 Decision 3).
func TestHAgainstBIsIneligibleWhenBIsNot(t *testing.T) {
	block := func(brokenB bool) func(id, m, c string) outcome {
		return func(id, m, c string) outcome {
			switch c {
			case "B":
				return outcome{cost: 1, solved: true, excluded: brokenB && index(id) < 3}
			case "H":
				return outcome{cost: 1, reads: 1} // H solves nothing: LOSES against A and B
			}
			return outcome{cost: 1, solved: true}
		}
	}
	stage := t.TempDir()
	writeBlockUnder(t, ProfileStage7, stage, 1, ids(1, 40), []string{"m"}, stage7All, block(true))
	r, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	hb := rowBase(t, r, "m", "H", "B")
	if hb.Eligible || hb.ClosedAt != 0 || hb.Result.Verdict != "LOSES" {
		t.Errorf("H against B with B at 7.5%% excluded: eligible %v, closed at %d, interval says %s; want ineligible, open, LOSES", hb.Eligible, hb.ClosedAt, hb.Result.Verdict)
	}
	if ha := rowBase(t, r, "m", "H", "A"); !ha.Eligible {
		t.Errorf("H against A is ineligible although A and H are complete: %+v", ha)
	}
	// The sibling: B whole, and the same data closes H against B.
	whole := t.TempDir()
	writeBlockUnder(t, ProfileStage7, whole, 1, ids(1, 40), []string{"m"}, stage7All, block(false))
	if r, err = Look(whole); err != nil {
		t.Fatal(err)
	}
	if hb := rowBase(t, r, "m", "H", "B"); !hb.Eligible || hb.ClosedAt != 1 || hb.Verdict != "LOSES" {
		t.Errorf("H against B with B whole: %+v, want LOSES at look 1", hb)
	}
}

// TestHClosesUntestedBelowTheUptakeGate: the gate counts block 1's final H
// receipts, an excluded attempt left out and replaced as ADR-156 replaces it, a
// timed-out one counted; fewer than half with a host Read close H against A
// and against B UNTESTED and stop H, and half or more leave them open. Until
// H against B is eligible neither closes, whatever its interval says (ADR-192
// Decision 4).
func TestHClosesUntestedBelowTheUptakeGate(t *testing.T) {
	// H costs and solves as A, so nothing but the gate can close its rows.
	gate := func(h func(i int) outcome) (string, LookReport) {
		stage := t.TempDir()
		writeBlockUnder(t, ProfileStage7, stage, 1, ids(1, 40), []string{"m"}, stage7All, func(id, m, c string) outcome {
			if c == "H" {
				return h(index(id))
			}
			return outcome{cost: 1, solved: true}
		})
		return stage, mustLook(t, stage)
	}
	reads := func(passing func(i int) bool) func(i int) outcome {
		return func(i int) outcome {
			o := outcome{cost: 1, solved: true}
			if passing(i) {
				o.reads = 1
			}
			return o
		}
	}
	untested := func(name string, r LookReport) {
		t.Helper()
		for _, base := range []string{"A", "B"} {
			if row := rowBase(t, r, "m", "H", base); row.Verdict != "UNTESTED" || row.ClosedAt != 1 {
				t.Errorf("%s: H against %s is %s closed at %d, want UNTESTED at look 1", name, base, row.Verdict, row.ClosedAt)
			}
		}
		if got := strings.Join(r.Next["m"], ","); got != "A,B" {
			t.Errorf("%s: after the gate closed H the model runs %q, want A,B", name, got)
		}
	}
	open := func(name string, r LookReport) {
		t.Helper()
		for _, base := range []string{"A", "B"} {
			if row := rowBase(t, r, "m", "H", base); row.ClosedAt != 0 || row.Verdict != "INCONCLUSIVE" {
				t.Errorf("%s: H against %s is %s closed at %d, want open", name, base, row.Verdict, row.ClosedAt)
			}
		}
		if got := strings.Join(r.Next["m"], ","); got != "A,B,H" {
			t.Errorf("%s: the model runs %q, want A,B,H", name, got)
		}
	}

	stage, r := gate(reads(func(i int) bool { return i < 19 }))
	untested("19 of 40", r)
	// H is not scheduled again for the model.
	writeBlockUnder(t, ProfileStage7, stage, 2, ids(2, 40), []string{"m"}, stage7All, func(id, m, c string) outcome { return outcome{cost: 1, solved: true} })
	if _, err := Look(stage); err == nil {
		t.Error("a block ran H after the gate stopped it and the look accepted it")
	}
	_, r = gate(reads(func(i int) bool { return i < 20 }))
	open("20 of 40", r)

	// An excluded attempt and its replacement count once, as the replacement.
	stage, _ = gate(func(i int) outcome {
		o := reads(func(i int) bool { return i < 20 })(i)
		o.excluded = i == 0
		return o
	})
	rewriteReceipt(t, stage, 1, "i1-00.H.m.1.r1", func(r *Receipt) { r.Exclusion, r.HostReads = "", 0 })
	untested("a replacement without a Read", mustLook(t, stage))

	// An attempt excluded to the end is left out: 19 of the 38 that count is half.
	_, r = gate(func(i int) outcome {
		o := reads(func(i int) bool { return i >= 2 && i < 21 })(i)
		o.excluded = i < 2
		return o
	})
	open("two excluded to the end", r)

	// A timed-out attempt counts, with no Read: 19 of 40.
	_, r = gate(func(i int) outcome {
		o := reads(func(i int) bool { return i < 19 })(i)
		o.timedOut = i == 19 || i == 20
		return o
	})
	untested("two timed out", r)

	// With H against B ineligible at look 1, H against A closes nothing though
	// its interval says LOSES; the gate is read at look 2, on block 1's receipts.
	stage = t.TempDir()
	writeBlockUnder(t, ProfileStage7, stage, 1, ids(1, 40), []string{"m"}, stage7All, func(id, m, c string) outcome {
		i := index(id)
		switch c {
		case "B":
			return outcome{cost: 1, solved: true, excluded: i < 3}
		case "H":
			o := outcome{cost: 1}
			if i < 10 {
				o.reads = 1
			}
			return o
		}
		return outcome{cost: 1, solved: true}
	})
	r = mustLook(t, stage)
	ha, hb := rowBase(t, r, "m", "H", "A"), rowBase(t, r, "m", "H", "B")
	if ha.ClosedAt != 0 || ha.Result.Verdict != "LOSES" || hb.ClosedAt != 0 || hb.Eligible {
		t.Errorf("look 1 with B at 7.5%% excluded: H against A closed at %d (interval %s), H against B closed at %d eligible %v; want both open, A's interval LOSES, B's ineligible", ha.ClosedAt, ha.Result.Verdict, hb.ClosedAt, hb.Eligible)
	}
	writeBlockUnder(t, ProfileStage7, stage, 2, ids(2, 40), []string{"m"}, stage7All, func(id, m, c string) outcome {
		return outcome{cost: 1, solved: c != "H", reads: 1}
	})
	r = mustLook(t, stage)
	for _, base := range []string{"A", "B"} {
		if row := rowBase(t, r, "m", "H", base); row.Verdict != "UNTESTED" || row.ClosedAt != 2 {
			t.Errorf("look 2: H against %s is %s closed at %d, want UNTESTED at look 2", base, row.Verdict, row.ClosedAt)
		}
	}
}

func mustLook(t *testing.T, stage string) LookReport {
	t.Helper()
	r, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestALookRefusesABlockThatRanAnotherHSentence: a block whose run.json names
// another H sentence than block 1's is refused by name, as the runner refuses
// it when the block is run (ADR-192 Decision 1).
func TestALookRefusesABlockThatRanAnotherHSentence(t *testing.T) {
	stage := t.TempDir()
	same := func(id, m, c string) outcome { return outcome{cost: 1, solved: true, reads: 1} }
	writeBlockUnder(t, ProfileStage7, stage, 1, ids(1, 40), []string{"m"}, stage7All, same)
	writeBlockUnder(t, ProfileStage7, stage, 2, ids(2, 40), []string{"m"}, stage7All, same)
	if _, err := Look(stage); err != nil {
		t.Fatalf("two blocks with one sentence: %v", err)
	}
	path := filepath.Join(stage, "block-2", "run.json")
	var p plan
	if err := readJSON(path, &p); err != nil {
		t.Fatal(err)
	}
	p.HSentenceSHA256 = strings.Repeat("b", 64)
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Look(stage); err == nil || !strings.Contains(err.Error(), "H sentence") {
		t.Errorf("a block 2 with another H sentence: %v, want a refusal naming it", err)
	}
}

// TestALookRefusesABlockThatRanAnotherEditParagraph: the same refusal for the
// edit paragraph of ADR-193 Decision 1.
func TestALookRefusesABlockThatRanAnotherEditParagraph(t *testing.T) {
	stage := t.TempDir()
	same := func(id, m, c string) outcome { return outcome{cost: 1, solved: true, reads: 1} }
	writeBlockUnder(t, ProfileStage7, stage, 1, ids(1, 40), []string{"m"}, stage7All, same)
	writeBlockUnder(t, ProfileStage7, stage, 2, ids(2, 40), []string{"m"}, stage7All, same)
	path := filepath.Join(stage, "block-2", "run.json")
	var p plan
	if err := readJSON(path, &p); err != nil {
		t.Fatal(err)
	}
	p.EditParagraphSHA = strings.Repeat("c", 64)
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Look(stage); err == nil || !strings.Contains(err.Error(), "edit paragraph") {
		t.Errorf("a block 2 with another edit paragraph: %v, want a refusal naming it", err)
	}
}

// TestADescriptiveRowOfAnUntestedComparisonSaysUntested: the descriptive rows
// of a comparison the uptake gate closed UNTESTED read (UNTESTED), not the
// interval's verdict, which the gate overrode; B against A keeps its own.
func TestADescriptiveRowOfAnUntestedComparisonSaysUntested(t *testing.T) {
	stage := t.TempDir()
	writeBlockUnder(t, ProfileStage7, stage, 1, ids(1, 40), []string{"m"}, stage7All, func(id, m, c string) outcome {
		o := outcome{cost: 1, solved: true}
		if c == "H" && index(id) < 19 {
			o.reads = 1
		}
		return o
	})
	rep := mustLook(t, stage)
	desc, err := LookDescriptive(stage, rep, []Subset{{Label: "all of it", Tasks: ids(1, 40)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(desc) != 3 {
		t.Fatalf("%d descriptive rows, want 3", len(desc))
	}
	for _, r := range desc {
		if (r.Comparison == "H") != (r.Verdict == "UNTESTED") {
			t.Errorf("%s against %s descriptive row says %s", r.Comparison, r.Control, r.Verdict)
		}
	}
}
