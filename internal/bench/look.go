package bench

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// BlockSize and Looks are ADR-162 Decision 5's registration: a look after
// each block of 40 instances, five looks, the fifth block the rest of the pool.
const (
	BlockSize = 40
	Looks     = 5
)

// lookLevel is the interval each look reads: 1 − 0.05/5, Bonferroni over the
// five looks, a nominal per-comparison allocation (ADR-162 Decision 5).
const lookLevel = 1 - 0.05/Looks

// BlockTasks is block k of the pool: instances 40(k−1)+1 to 40k, the fifth
// block the rest.
func BlockTasks(ts []Task, k int) ([]Task, error) {
	if k < 1 || k > Looks {
		return nil, fmt.Errorf("block %d: the registration has blocks 1 to %d", k, Looks)
	}
	lo := (k - 1) * BlockSize
	hi := k * BlockSize
	if k == Looks || hi > len(ts) {
		hi = len(ts)
	}
	if lo >= hi {
		return nil, fmt.Errorf("block %d is empty: the pool has %d instances", k, len(ts))
	}
	return ts[lo:hi], nil
}

// LookRow is one model's comparison with A at the latest look: closed at the
// look it first read eligible and PAYS or LOSES, else open, and INCONCLUSIVE
// once the stage is complete.
type LookRow struct {
	Model      string
	Comparison string
	Verdict    string
	ClosedAt   int    // the look it closed at, 0 while open
	Eligible   bool   // whether the latest look it was read at was eligible
	Result     Result // the reading it closed at, or the latest one
}

// LookReport is bench look's answer: every comparison, what each model runs
// next (nothing once both its comparisons are closed), and whether the stage
// is complete.
type LookReport struct {
	Rows     []LookRow
	Next     map[string][]string
	Blocks   int
	Complete bool
}

type lookBlock struct {
	plan plan
	rs   []Receipt
}

// Look reads a Stage 5 directory's blocks, block-1 onward, and applies ADR-162
// Decision 5's stop rule. A block that ran a configuration the rule had not
// scheduled for a model is refused.
func Look(stageDir string) (LookReport, error) {
	rep := LookReport{Next: map[string][]string{}}
	ruled, err := rulings(stageDir)
	if err != nil {
		return rep, err
	}
	applied := map[string]bool{}
	var blocks []lookBlock
	for k := 1; k <= Looks; k++ {
		dir := filepath.Join(stageDir, fmt.Sprintf("block-%d", k))
		var p plan
		if err := readJSON(filepath.Join(dir, "run.json"), &p); err != nil {
			if errors.Is(err, os.ErrNotExist) && k > 1 {
				break
			}
			return rep, err
		}
		if p.Block != k || p.Looks != Looks || p.Repeats != 1 {
			return rep, fmt.Errorf("%s: run.json names block %d of %d looks with %d repeats, want block %d of %d with one", dir, p.Block, p.Looks, p.Repeats, k, Looks)
		}
		if len(blocks) > 0 {
			f := blocks[0].plan
			if p.Stage != f.Stage || p.CLIVersion != f.CLIVersion || p.MrwVersion != f.MrwVersion || p.Profile != f.Profile || !samePlugin(p.Plugin, f.Plugin) {
				return rep, fmt.Errorf("%s ran another stage, CLI, mrw, profile or plugin than block 1", dir)
			}
		}
		paths, err := filepath.Glob(filepath.Join(dir, "*.receipt.json"))
		if err != nil {
			return rep, err
		}
		b := lookBlock{plan: p}
		for _, path := range paths {
			var r Receipt
			if err := readJSON(path, &r); err != nil {
				return rep, err
			}
			if err := rule(ruled, &r, applied); err != nil {
				return rep, err
			}
			if !slices.Contains(plannedConfigs(p, r.Model), r.Config) {
				return rep, fmt.Errorf("%s holds %s, a configuration block %d did not plan for %s", dir, r.Attempt, k, r.Model)
			}
			b.rs = append(b.rs, r)
		}
		blocks = append(blocks, b)
	}
	if err := unapplied(ruled, applied); err != nil {
		return rep, err
	}
	rep.Blocks = len(blocks)
	first := blocks[0].plan
	// A model a later block adds would get no row and no scheduling check
	// (the review of T3): the stage's models are block 1's.
	for _, b := range blocks[1:] {
		for _, m := range b.plan.Models {
			if !slices.Contains(first.Models, m) {
				return rep, fmt.Errorf("block %d runs model %s, which block 1 did not", b.plan.Block, m)
			}
		}
	}
	for _, m := range first.Models {
		var comps []string
		for _, c := range plannedConfigs(first, m) {
			if c != "A" {
				comps = append(comps, c)
			}
		}
		rows := map[string]*LookRow{}
		for _, c := range comps {
			rows[c] = &LookRow{Model: m, Comparison: c}
		}
		for j := range blocks {
			open := []string{"A"}
			for _, c := range comps {
				if rows[c].ClosedAt == 0 {
					open = append(open, c)
				}
			}
			if len(open) == 1 {
				open = nil
			}
			if j > 0 && !sameSet(plannedConfigs(blocks[j].plan, m), open) {
				return rep, fmt.Errorf("block %d runs %v for %s; after look %d the rule schedules %v", j+1, plannedConfigs(blocks[j].plan, m), m, j, open)
			}
			for _, c := range comps {
				row := rows[c]
				if row.ClosedAt != 0 {
					continue
				}
				var rs []Receipt
				for _, b := range blocks[:j+1] {
					for _, r := range b.rs {
						if r.Model == m && (r.Config == "A" || r.Config == c) {
							rs = append(rs, r)
						}
					}
				}
				res, err := judgeWith(first.Stage, m, "swe-bench", rs, c, lookLevel, true)
				if err != nil {
					return rep, err
				}
				res.Look = j + 1
				row.Result, row.Eligible = res, eligible(blocks[:j+1], m, c)
				if row.Eligible && (res.Verdict == "PAYS" || res.Verdict == "LOSES") {
					row.ClosedAt, row.Verdict = j+1, res.Verdict
				}
			}
		}
		var next []string
		for _, c := range comps {
			if rows[c].ClosedAt == 0 {
				rows[c].Verdict = "INCONCLUSIVE"
				next = append(next, c)
			}
			rep.Rows = append(rep.Rows, *rows[c])
		}
		if len(next) > 0 {
			rep.Next[m] = append([]string{"A"}, next...)
		}
	}
	rep.Complete = rep.Blocks == Looks || len(rep.Next) == 0
	if rep.Complete {
		rep.Next = map[string][]string{}
	}
	return rep, nil
}

// eligible is ADR-162 Decision 5's rule for a look. Every attempt a block
// scheduled for the model — every configuration it planned, not only this
// comparison's — has its receipt, walked from the first try through each
// replacement the registration owes, as Verify walks it; then at most 5% of
// the instances are excluded in A or c, and at least 95% are paired.
func eligible(blocks []lookBlock, model, c string) bool {
	n, paired := 0, 0
	excluded := map[string]int{}
	for _, b := range blocks {
		byID := map[string]Receipt{}
		for _, r := range b.rs {
			if r.Model == model {
				byID[r.Attempt] = r
			}
		}
		final := func(task, cfg string) (Receipt, bool) {
			id := fmt.Sprintf("%s.%s.%s.1", task, cfg, safe(model))
			r, ok := byID[id]
			for try := 0; ok && replaced(r.Exclusion, try); try++ {
				r, ok = byID[fmt.Sprintf("%s.r%d", id, try+1)]
			}
			return r, ok
		}
		for _, task := range b.plan.Tasks {
			for _, cfg := range plannedConfigs(b.plan, model) {
				if _, ok := final(task, cfg); !ok {
					return false
				}
			}
			n++
			both := true
			for _, cfg := range []string{"A", c} {
				if r, _ := final(task, cfg); r.Exclusion != "" {
					excluded[cfg]++
					both = false
				}
			}
			if both {
				paired++
			}
		}
	}
	if n == 0 {
		return false
	}
	for _, cfg := range []string{"A", c} {
		if float64(excluded[cfg]) > 0.05*float64(n)+eps {
			return false
		}
	}
	return float64(paired) >= 0.95*float64(n)-eps
}

// plannedConfigs is what a block planned for a model: its own list when the
// plan has one, else the block's.
func plannedConfigs(p plan, model string) []string {
	if !slices.Contains(p.Models, model) {
		return nil
	}
	if cs, ok := p.ConfigsFor[model]; ok {
		return cs
	}
	return p.Configs
}

func sameSet(a, b []string) bool {
	return slices.Equal(sorted(a), sorted(b))
}

// Text is the report as bench look prints it.
func (r LookReport) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "looks read: %d of %d\n", r.Blocks, Looks)
	for _, row := range r.Rows {
		state := "open"
		if row.ClosedAt != 0 {
			state = fmt.Sprintf("closed at look %d", row.ClosedAt)
		}
		res := row.Result
		fmt.Fprintf(&b, "%s · %s against A · %s · %s · eligible %v · cost %+.3f [%+.3f, %+.3f] · solved %+.3f [%+.3f, %+.3f] at %.0f%% · tasks %d",
			row.Model, row.Comparison, state, row.Verdict, row.Eligible, res.RelCost, res.RelLo, res.RelHi, res.SolvedDiff, res.SolvedLo, res.SolvedHi, res.Level*100, res.Tasks)
		// What a ruling or a killed attempt left out is said on the row it
		// shaped, and only there, so a row without either reads as before.
		if res.Excluded > 0 {
			fmt.Fprintf(&b, " · excluded %d", res.Excluded)
		}
		if len(res.CostUnpaired) > 0 {
			fmt.Fprintf(&b, " · cost leaves out %s", strings.Join(res.CostUnpaired, ", "))
		}
		b.WriteString("\n")
	}
	for _, m := range sortedKeysOf(r.Next) {
		fmt.Fprintf(&b, "next: %s runs %s\n", m, strings.Join(r.Next[m], ","))
	}
	if r.Complete {
		b.WriteString("the stage is complete\n")
	} else {
		fmt.Fprintf(&b, "block %d is due\n", r.Blocks+1)
	}
	return b.String()
}

func sortedKeysOf(m map[string][]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return sorted(ks)
}

// LookStage reports whether dir is a Stage 5 directory: its block-1 run.json
// names the looks.
func LookStage(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "block-1", "run.json"))
	if err != nil {
		return false
	}
	var p plan
	return json.Unmarshal(b, &p) == nil && p.Looks > 0
}

// OnlyHunks labels a selection's hunk-count subset row.
func OnlyHunks(n int) string {
	return fmt.Sprintf("only the instances with %d or more hunks; descriptive, not the verdict", n)
}

// LookDescriptive is each comparison's descriptive rows over the registered
// subsets of a Stage 5 directory, read as its verdict was: at the look it
// closed at, or the latest, at the look's level (ADR-162 Decision 4).
func LookDescriptive(stageDir string, rep LookReport, subsets []Subset) ([]Result, error) {
	var out []Result
	ruled, err := rulings(stageDir)
	if err != nil {
		return nil, err
	}
	for _, row := range rep.Rows {
		upTo := row.ClosedAt
		if upTo == 0 {
			upTo = rep.Blocks
		}
		var rs []Receipt
		for k := 1; k <= upTo; k++ {
			paths, err := filepath.Glob(filepath.Join(stageDir, fmt.Sprintf("block-%d", k), "*.receipt.json"))
			if err != nil {
				return nil, err
			}
			for _, p := range paths {
				var r Receipt
				if err := readJSON(p, &r); err != nil {
					return nil, err
				}
				if err := rule(ruled, &r, map[string]bool{}); err != nil {
					return nil, err
				}
				if r.Model == row.Model && (r.Config == "A" || r.Config == row.Comparison) {
					rs = append(rs, r)
				}
			}
		}
		for _, s := range subsets {
			var in []Receipt
			for _, r := range rs {
				if slices.Contains(s.Tasks, r.Task) {
					in = append(in, r)
				}
			}
			if len(in) == 0 {
				continue
			}
			res, err := judgeWith(row.Result.Stage, row.Model, "swe-bench", in, row.Comparison, lookLevel, true)
			if err != nil {
				return nil, err
			}
			res.Descriptive, res.Look = s.Label, upTo
			out = append(out, res)
		}
	}
	return out, nil
}

// rulings reads the stage's exclusions.json: an attempt id and why the
// machine, not the agent, ended it (Zy's ruling of 2026-10-20 on Stage 5 block
// 2, an attempt cut by a sleeping Mac). No file, no rulings. The receipts stay
// as the runner wrote them; a ruling is applied where they are read.
func rulings(stageDir string) (map[string]string, error) {
	path := filepath.Join(stageDir, "exclusions.json")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	// Every id must be exactly an attempt the stage holds, so a misspelled,
	// wildcard or path-shaped ruling fails here on every reader rather than
	// excluding nothing. The pattern is fixed; an id is only compared.
	held := map[string]bool{}
	paths, err := filepath.Glob(filepath.Join(stageDir, "*", "*.receipt.json"))
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		held[strings.TrimSuffix(filepath.Base(p), ".receipt.json")] = true
	}
	var missing []string
	for id := range m {
		if !held[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return nil, fmt.Errorf("exclusions.json names %s, which no block of the stage holds", strings.Join(missing, ", "))
	}
	return m, nil
}

// rule excludes r as infrastructure when a ruling names it, and refuses a
// ruling over an attempt the runner already excluded.
func rule(ruled map[string]string, r *Receipt, applied map[string]bool) error {
	why, ok := ruled[r.Attempt]
	if !ok {
		return nil
	}
	if r.Exclusion != "" {
		return fmt.Errorf("exclusions.json rules %s, which the runner already excluded (%s)", r.Attempt, r.Exclusion)
	}
	r.Exclusion, r.Note = ExclusionInfra, why
	applied[r.Attempt] = true
	return nil
}

// unapplied refuses a ruling that named no attempt the stage holds: a typo
// would otherwise exclude nothing and say nothing.
func unapplied(ruled map[string]string, applied map[string]bool) error {
	var missing []string
	for id := range ruled {
		if !applied[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return fmt.Errorf("exclusions.json names %s, which no block of the stage holds", strings.Join(missing, ", "))
	}
	return nil
}
