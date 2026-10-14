package bench

import (
	"fmt"
	"math/rand"
	"sort"
)

// Result is one verdict of the registered bar (BACKLOG, "Pre-registered for
// ADR-156"): one model, one task source, never pooled.
type Result struct {
	Model      string   `json:"model"`
	Source     string   `json:"source"`
	Verdict    string   `json:"verdict"` // PAYS, LOSES or INCONCLUSIVE
	Tasks      int      `json:"tasks"`   // paired tasks
	Attempts   int      `json:"attempts"`
	Excluded   int      `json:"excluded"`
	Unpaired   []string `json:"unpaired,omitempty"`
	RelCost    float64  `json:"rel_cost"` // mean per-task (B - A) / A
	RelLo      float64  `json:"rel_cost_lo"`
	RelHi      float64  `json:"rel_cost_hi"`
	SolvedDiff float64  `json:"solved_diff"` // mean per-task solved rate B - A
	SolvedLo   float64  `json:"solved_diff_lo"`
	SolvedHi   float64  `json:"solved_diff_hi"`
	SilentA    int      `json:"silent_mis_edits_a"`
	SilentB    int      `json:"silent_mis_edits_b"`
}

// The bootstrap the registration fixes, and the tolerance a boundary of the
// bar is compared with: the bar's edges are inclusive, and float arithmetic
// must not move a result that sits exactly on one.
const (
	bootstrapSeed      = 156
	bootstrapResamples = 10000
	eps                = 1e-9
)

// Analyze computes each verdict from the receipts. Only the registered
// exclusions leave an attempt out of the pairing; any other is refused. An
// attempt over its budget is counted, unsolved, with its cost.
//
// The registration's PAYS and LOSES overlap when the solved interval lies
// wholly below 0 but no lower than -5 points; LOSES is checked first, the
// conservative reading, which ADR-156 asks Zy to confirm before Stage 1.
func Analyze(rs []Receipt) ([]Result, error) {
	type key struct{ model, source string }
	groups := map[key][]Receipt{}
	for _, r := range rs {
		switch r.Exclusion {
		case "", ExclusionStart, ExclusionSetup:
		default:
			return nil, fmt.Errorf("attempt %s: exclusion %q is not one the registration admits", r.Attempt, r.Exclusion)
		}
		k := key{r.Model, r.Source}
		groups[k] = append(groups[k], r)
	}
	var out []Result
	for k, g := range groups {
		res := Result{Model: k.model, Source: k.source, Attempts: len(g)}
		type side struct {
			cost         float64
			solved, runs int
		}
		perTask := map[string]map[string]*side{}
		for _, r := range g {
			if r.Exclusion != "" {
				res.Excluded++
				continue
			}
			if r.CostUSD == nil {
				return nil, fmt.Errorf("attempt %s: no cost", r.Attempt)
			}
			if r.SilentMisEdit {
				if r.Config == "A" {
					res.SilentA++
				} else {
					res.SilentB++
				}
			}
			if perTask[r.Task] == nil {
				perTask[r.Task] = map[string]*side{}
			}
			s := perTask[r.Task][r.Config]
			if s == nil {
				s = &side{}
				perTask[r.Task][r.Config] = s
			}
			s.cost += *r.CostUSD
			s.runs++
			if r.Oracle.Solved && !r.OverBudget {
				s.solved++
			}
		}
		var tasks []string
		for t := range perTask {
			tasks = append(tasks, t)
		}
		sort.Strings(tasks)
		var rel, diff []float64
		for _, t := range tasks {
			a, b := perTask[t]["A"], perTask[t]["B"]
			if a == nil || b == nil {
				res.Unpaired = append(res.Unpaired, t)
				continue
			}
			meanA, meanB := a.cost/float64(a.runs), b.cost/float64(b.runs)
			if meanA <= 0 {
				return nil, fmt.Errorf("task %s: configuration A's mean cost is %v", t, meanA)
			}
			rel = append(rel, (meanB-meanA)/meanA)
			diff = append(diff, float64(b.solved)/float64(b.runs)-float64(a.solved)/float64(a.runs))
		}
		res.Tasks = len(rel)
		if res.Tasks == 0 {
			res.Verdict = "INCONCLUSIVE"
			out = append(out, res)
			continue
		}
		res.RelCost, res.SolvedDiff = mean(rel), mean(diff)
		res.RelLo, res.RelHi, res.SolvedLo, res.SolvedHi = bootstrap(rel, diff)
		switch {
		case res.RelLo > eps || res.SolvedHi < -eps:
			res.Verdict = "LOSES"
		case res.RelCost <= -0.10+eps && res.RelHi < -eps && res.SolvedLo >= -0.05-eps:
			res.Verdict = "PAYS"
		default:
			res.Verdict = "INCONCLUSIVE"
		}
		out = append(out, res)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Model != out[j].Model {
			return out[i].Model < out[j].Model
		}
		return out[i].Source < out[j].Source
	})
	return out, nil
}

func mean(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// bootstrap resamples tasks with replacement, both measures from the same
// draw, and returns the 95% percentile intervals.
func bootstrap(rel, diff []float64) (relLo, relHi, diffLo, diffHi float64) {
	rng := rand.New(rand.NewSource(bootstrapSeed))
	n := len(rel)
	rs, ds := make([]float64, bootstrapResamples), make([]float64, bootstrapResamples)
	for b := range bootstrapResamples {
		sr, sd := 0.0, 0.0
		for range n {
			i := rng.Intn(n)
			sr += rel[i]
			sd += diff[i]
		}
		rs[b], ds[b] = sr/float64(n), sd/float64(n)
	}
	sort.Float64s(rs)
	sort.Float64s(ds)
	lo, hi := int(0.025*bootstrapResamples), int(0.975*bootstrapResamples)-1
	return rs[lo], rs[hi], ds[lo], ds[hi]
}
