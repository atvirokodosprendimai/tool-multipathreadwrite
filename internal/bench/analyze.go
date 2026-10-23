package bench

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sort"
	"strings"
)

// Result is one verdict of the registered bar (BACKLOG, "Pre-registered for
// ADR-156"): one stage, one model, one task source, never pooled (ADR-158
// Decision 6).
type Result struct {
	Stage      string   `json:"stage"`
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
	ReadsB     int      `json:"shell_reads_b"`    // B's reads through cat, grep and kin, counted not refused
	SlippedB   int      `json:"slipped_writes_b"` // files B changed by a means its hook missed
	// Descriptive is empty on the verdict. On a second row it names what was
	// left out; that row is not a verdict of the registered bar (ADR-156 T6 S5).
	Descriptive string `json:"descriptive,omitempty"`
	// Comparison is the configuration compared with A: "B" (mrw) or "P" (the
	// quality-harness plugin, ADR-162). Level is the interval's level: 0.95
	// for ADR-156's bar, 1 − 0.05/5 at an ADR-162 look; Look is the look a
	// Stage 5 verdict was read at.
	Comparison string  `json:"comparison"`
	Level      float64 `json:"level"`
	Look       int     `json:"look,omitempty"`

	// CostUnpaired names the paired instances left out of the cost comparison
	// only: an attempt killed at the budget before the CLI wrote its result
	// line has no cost (Zy's ruling of 2026-10-20). They stay in the success
	// comparison, unsolved.
	CostUnpaired []string `json:"cost_unpaired,omitempty"`
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
// attempt over its budget, or one that ended in an error the registration does
// not exclude, is counted unsolved, with its cost.
//
// Where PAYS and LOSES both hold — the solved interval wholly below 0 but no
// lower than -5 points — LOSES wins: Zy's clarification of the registration,
// 2026-10-14, before any stage ran. B's counted shell reads are summed beside
// each verdict (Zy, the same day).
//
// superseded names the tasks superseded since a stage ran: a group that ran
// one gets a descriptive row without them (ADR-159 Decision 2).
func Analyze(rs []Receipt, superseded ...string) ([]Result, error) {
	return AnalyzeSubsets(rs, superseded, nil)
}

// Subset is a registered set of a slice's tasks that gets a descriptive row of
// its own, such as ADR-160's instances whose problem statement names none of
// the patched files.
type Subset struct {
	Label string
	Tasks []string
}

// OnlyUnnamedFiles labels ADR-160 Decision 5's subset row.
const OnlyUnnamedFiles = "only the instances whose problem statement names no patched file; descriptive, not the verdict"

// stage4Categories are ADR-160's categories, the control included: a group
// holding either of the first two gets one descriptive row per category it
// holds (ADR-160 Decision 5). The earlier stages' groups, which hold neither,
// get none, so their published rows do not change.
var stage4Categories = []string{ShapeWideRename, ShapeCheckCoupled, ShapeSingleLine}

func onlyCategory(shape string) string {
	return "only the " + shape + " tasks; descriptive, not the verdict"
}

// AnalyzeSubsets is Analyze with the registered subsets of a slice: a group
// holding some of a subset's tasks gets a descriptive row over them.
func AnalyzeSubsets(rs []Receipt, superseded []string, subsets []Subset) ([]Result, error) {
	type key struct{ stage, model, source string }
	groups := map[key][]Receipt{}
	for _, r := range rs {
		// A pilot is never counted (the registration), wherever its receipts
		// sit: the stage's own name for it decides, not the directory's.
		if strings.HasPrefix(r.Stage, "pilot") {
			continue
		}
		switch r.Exclusion {
		case "", ExclusionStart, ExclusionSetup, ExclusionInfra:
		default:
			return nil, fmt.Errorf("attempt %s: exclusion %q is not one the registration admits", r.Attempt, r.Exclusion)
		}
		k := key{r.Stage, r.Model, r.Source}
		groups[k] = append(groups[k], r)
	}
	var out []Result
	for k, g := range groups {
		res, err := judge(k.stage, k.model, k.source, g)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
		// A stage that ran P gets P against A by the same rule, never P
		// against B (ADR-162 Decision 6).
		if slices.ContainsFunc(g, func(r Receipt) bool { return r.Config == "P" }) {
			p, err := judgeWith(k.stage, k.model, k.source, g, "P", 0.95, false)
			if err != nil {
				return nil, err
			}
			out = append(out, p)
		}
		// One descriptive row beside the verdict, without the tasks whose own
		// check was measured red at base (Zy, 2026-10-14): it shows how much of
		// the verdict those tasks carry, and is never the verdict itself.
		var kept, varied, current []Receipt
		for _, r := range g {
			if !r.CheckRedAtBase {
				kept = append(kept, r)
			}
			if r.Shape == ShapeVariedMultiSite {
				varied = append(varied, r)
			}
			if !slices.Contains(superseded, r.Task) {
				current = append(current, r)
			}
		}
		if len(kept) != len(g) {
			d, err := judge(k.stage, k.model, k.source, kept)
			if err != nil {
				return nil, err
			}
			d.Descriptive = withoutRedAtBase
			out = append(out, d)
		}
		// A second descriptive row over the varied-multi-site tasks alone,
		// registered before any of them ran (ADR-158 Decision 6).
		if len(varied) > 0 {
			d, err := judge(k.stage, k.model, k.source, varied)
			if err != nil {
				return nil, err
			}
			d.Descriptive = onlyVariedMultiSite
			out = append(out, d)
		}
		// A row without the tasks superseded since the stage ran. It was
		// registered after the stages it applies to had run, for a defect found
		// in a task's reference fix (Zy, ADR-159 Decision 2): it shows how much of
		// a result that defect carries, and is never the verdict.
		if len(current) != len(g) {
			d, err := judge(k.stage, k.model, k.source, current)
			if err != nil {
				return nil, err
			}
			d.Descriptive = withoutSupersededSince
			out = append(out, d)
		}
		// ADR-160's rows: one per category in a group that holds its
		// categories, and one per registered subset the group holds.
		byShape := map[string][]Receipt{}
		for _, r := range g {
			byShape[r.Shape] = append(byShape[r.Shape], r)
		}
		if len(byShape[ShapeWideRename])+len(byShape[ShapeCheckCoupled]) > 0 {
			for _, shape := range stage4Categories {
				if len(byShape[shape]) == 0 {
					continue
				}
				d, err := judge(k.stage, k.model, k.source, byShape[shape])
				if err != nil {
					return nil, err
				}
				d.Descriptive = onlyCategory(shape)
				out = append(out, d)
			}
		}
		for _, s := range subsets {
			var in []Receipt
			for _, r := range g {
				if slices.Contains(s.Tasks, r.Task) {
					in = append(in, r)
				}
			}
			if len(in) == 0 {
				continue
			}
			d, err := judge(k.stage, k.model, k.source, in)
			if err != nil {
				return nil, err
			}
			d.Descriptive = s.Label
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Stage != out[j].Stage {
			return out[i].Stage < out[j].Stage
		}
		if out[i].Model != out[j].Model {
			return out[i].Model < out[j].Model
		}
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Descriptive != out[j].Descriptive {
			return out[i].Descriptive < out[j].Descriptive
		}
		return out[i].Comparison < out[j].Comparison
	})
	return out, nil
}

// withoutRedAtBase labels the row computed without the tasks flagged
// check_red_at_base.
const withoutRedAtBase = "without the tasks whose own check is red at base; descriptive, not the verdict"

// onlyVariedMultiSite labels the row computed over the varied-multi-site tasks
// alone.
const onlyVariedMultiSite = "only the varied-multi-site tasks; descriptive, not the verdict"

// withoutSupersededSince labels the row computed without the tasks superseded
// since the stage ran.
const withoutSupersededSince = "without the tasks superseded after the stage ran; registered after it ran, descriptive, not the verdict"

// judge computes one row of the registered bar from one model's and one task
// source's receipts: B against A at 95%, ADR-156's bar.
func judge(stage, model, source string, g []Receipt) (Result, error) {
	return judgeWith(stage, model, source, g, "B", 0.95, false)
}

// judgeWith is judge for any configuration against A, at any interval level.
// With tango set, the solved interval is Tango's score interval for a paired
// difference of proportions, which needs one attempt per instance and
// configuration; the cost interval stays the percentile bootstrap (ADR-162
// Decision 5).
func judgeWith(stage, model, source string, g []Receipt, second string, level float64, tango bool) (Result, error) {
	return judgeAgainst("A", stage, model, source, g, second, level, tango)
}

// judgeAgainst is judgeWith with any configuration as the control: bench side
// reads a side arm against P (ADR-163 Decision 9).
func judgeAgainst(control, stage, model, source string, g []Receipt, second string, level float64, tango bool) (Result, error) {
	res := Result{Stage: stage, Model: model, Source: source, Comparison: second, Level: level}
	type side struct {
		cost         float64
		solved, runs int
		costless     bool // an attempt killed at the budget with no cost
	}
	perTask := map[string]map[string]*side{}
	for _, r := range g {
		if r.Config != control && r.Config != second {
			continue
		}
		res.Attempts++
		if r.Exclusion != "" {
			res.Excluded++
			continue
		}
		if r.CostUSD == nil && !r.TimedOut {
			return res, fmt.Errorf("attempt %s: no cost", r.Attempt)
		}
		if r.Config == "B" {
			res.ReadsB += r.HookReads
			res.SlippedB += len(r.SlippedWrites)
		}
		if r.SilentMisEdit {
			if r.Config == control {
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
		if r.CostUSD == nil {
			// Killed at the budget before the CLI wrote its result line (Zy's
			// ruling of 2026-10-20): unsolved, and out of the cost comparison.
			s.costless = true
		} else {
			s.cost += *r.CostUSD
		}
		s.runs++
		if r.Oracle.Solved && !r.OverBudget && !r.IsError {
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
		a, b := perTask[t][control], perTask[t][second]
		if a == nil || b == nil {
			res.Unpaired = append(res.Unpaired, t)
			continue
		}
		if tango && (a.runs != 1 || b.runs != 1) {
			return res, fmt.Errorf("task %s: Tango's interval needs one attempt per instance and configuration, found %d and %d", t, a.runs, b.runs)
		}
		diff = append(diff, float64(b.solved)/float64(b.runs)-float64(a.solved)/float64(a.runs))
		if a.costless || b.costless {
			res.CostUnpaired = append(res.CostUnpaired, t)
			continue
		}
		meanA, meanB := a.cost/float64(a.runs), b.cost/float64(b.runs)
		if meanA <= 0 {
			return res, fmt.Errorf("task %s: configuration %s's mean cost is %v", t, control, meanA)
		}
		rel = append(rel, (meanB-meanA)/meanA)
	}
	res.Tasks = len(diff)
	if len(diff) == 0 {
		res.Verdict = "INCONCLUSIVE"
		return res, nil
	}
	res.SolvedDiff = mean(diff)
	switch {
	case len(rel) == len(diff):
		res.RelCost = mean(rel)
		res.RelLo, res.RelHi, res.SolvedLo, res.SolvedHi = bootstrapAt(rel, diff, level)
	case len(rel) > 0:
		// The two comparisons cover different instances, so each is resampled
		// on its own. With none left out, the joint draw above is unchanged.
		res.RelCost = mean(rel)
		res.RelLo, res.RelHi, _, _ = bootstrapAt(rel, rel, level)
		_, _, res.SolvedLo, res.SolvedHi = bootstrapAt(diff, diff, level)
	default:
		// No pair kept its cost: success alone can decide, and only LOSES.
		_, _, res.SolvedLo, res.SolvedHi = bootstrapAt(diff, diff, level)
	}
	if tango {
		var x12, x21 int
		for _, d := range diff {
			switch {
			case d == 1:
				x12++
			case d == -1:
				x21++
			case d != 0:
				return res, fmt.Errorf("stage %s, model %s: Tango's interval needs one attempt per instance and configuration", stage, model)
			}
		}
		res.SolvedLo, res.SolvedHi = tangoInterval(x12, x21, len(diff), level)
	}
	switch {
	case len(rel) > 0 && res.RelLo > eps || res.SolvedHi < -eps:
		res.Verdict = "LOSES"
	case len(rel) > 0 && res.RelCost <= -0.10+eps && res.RelHi < -eps && res.SolvedLo >= -0.05-eps:
		res.Verdict = "PAYS"
	default:
		res.Verdict = "INCONCLUSIVE"
	}
	return res, nil
}

func mean(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// bootstrapAt resamples tasks with replacement, both measures from the same
// draw, and returns the percentile intervals at the level given: at 0.95 the
// 250th and 9,750th of 10,000 sorted means, as ADR-156 registered.
func bootstrapAt(rel, diff []float64, level float64) (relLo, relHi, diffLo, diffHi float64) {
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
	tail := (1 - level) / 2
	lo, hi := int(math.Round(tail*bootstrapResamples)), int(math.Round((1-tail)*bootstrapResamples))-1
	return rs[lo], rs[hi], ds[lo], ds[hi]
}

// tangoInterval is Tango's (1998) score interval for the difference of two
// paired proportions, here p₂ − p₁ for the second configuration against A:
// x12 instances the second solved and A did not, x21 the reverse, n in all.
// It solves |T(δ)| = z, where T(δ) = (x12 − x21 − nδ) / √(n(2p̃21 + δ(1 − δ)))
// with p̃21 the restricted maximum-likelihood estimate; at δ = 0 T is
// McNemar's statistic. Unlike a percentile bootstrap it does not collapse to
// [0, 0] when no instance is discordant: there it is ±z²/(n + z²).
func tangoInterval(x12, x21, n int, level float64) (lo, hi float64) {
	if n == 0 {
		return -1, 1
	}
	z := math.Sqrt2 * math.Erfinv(level)
	fn, b, c := float64(n), float64(x12), float64(x21)
	t := func(d float64) float64 {
		a := 2 * fn
		bb := -b - c + (2*fn-b+c)*d
		cc := -c * d * (1 - d)
		p21 := (math.Sqrt(math.Max(bb*bb-4*a*cc, 0)) - bb) / (2 * a)
		num := b - c - fn*d
		v := fn * (2*p21 + d*(1-d))
		if v <= 0 {
			switch {
			case num > 0:
				return math.Inf(1)
			case num < 0:
				return math.Inf(-1)
			}
			return 0
		}
		return num / math.Sqrt(v)
	}
	est := (b - c) / fn
	// T falls as δ rises through 0 at the estimate, so |T| grows on either
	// side: each bound is the last δ, walking out from the estimate, where
	// |T| ≤ z.
	solve := func(in, out float64) float64 {
		for range 200 {
			mid := (in + out) / 2
			if math.Abs(t(mid)) <= z {
				in = mid
			} else {
				out = mid
			}
		}
		return in
	}
	const edge = 1 - 1e-12
	return solve(est, -edge), solve(est, edge)
}
