package bench

import (
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
)

// The verdicts of the side reader. A row below SideFloor paired tasks is
// descriptive only: its numbers are published and no claim is read from them.
const (
	SideHolds       = "HOLDS"
	SideFails       = "DOES NOT HOLD"
	SideDescriptive = "DESCRIPTIVE ONLY"

	// SideFloor is the paired tasks per model below which no verdict is read:
	// at 60, with every outcome concordant, the 99% Tango lower bound is
	// -0.0996, just inside the margin; at 40 it is -0.142 (BACKLOG,
	// "Pre-registered for ADR-163").
	SideFloor = 60

	// sideSolvedMargin is the 10-point non-inferiority margin of both arms.
	sideSolvedMargin = -0.10
)

// sideBar is an arm's registered claim: the upper bound of (arm-P)/P must be
// below costBelow, and the lower bound of solved arm-P above the margin.
var sideBars = map[string]struct {
	claim     string
	costBelow float64
}{
	"C": {"the quiet Stop (turnEnd notify) costs less than P, with solving inside a 10-point non-inferiority margin", 0},
	"G": {"Go hooks stay inside a +5% cost margin and a 10-point solving margin of Node", 0.05},
}

// SideRow is one model's reading of one side arm against P.
type SideRow struct {
	Stage, Model, Arm, Control string
	Result                     Result
	Verdict                    string
	Claim                      string
	// Reported beside the verdict, not part of it: the per-task means over
	// the paired tasks, wall time in seconds.
	TurnsControl, TurnsArm float64
	WallControl, WallArm   float64
}

// sideVerdict applies the registered bar. The inequalities are strict, as the
// record words them ("below 0", "below +0.05", "above -0.10"), with the
// tolerance the other bars take so float arithmetic cannot move a result that
// sits on a boundary. Fewer than SideFloor paired tasks read no verdict; and a
// verdict about cost needs at least one pair that has a cost.
func sideVerdict(arm string, relHi, solvedLo float64, pairs, costPairs int) string {
	if pairs < SideFloor {
		return SideDescriptive
	}
	if costPairs > 0 && relHi < sideBars[arm].costBelow-eps && solvedLo > sideSolvedMargin+eps {
		return SideHolds
	}
	return SideFails
}

// Side reads a side arm against P, paired by task and cumulated over the
// blocks given (ADR-163 Decision 9). dirs are block directories: P's receipts
// are the ones in them, the arm's the ones in their side/ directories. Cost
// uses the percentile bootstrap and solved Tango's interval, both at the
// look's 99%. A pilot's receipts are never counted; an attempt the stage's
// exclusions.json rules is excluded as the look excludes it.
func Side(dirs []string, control, arm string) ([]SideRow, error) {
	bar, ok := sideBars[arm]
	switch {
	case control != "P":
		return nil, fmt.Errorf("--control %q: a side arm is read against P alone (ADR-163 Decision 9)", control)
	case !ok:
		return nil, fmt.Errorf("--arm %q: G or C", arm)
	case len(dirs) == 0:
		return nil, fmt.Errorf("side needs at least one block directory")
	}
	var rs []Receipt
	var first *plan
	firstDir := ""
	for _, dir := range dirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		ruled, err := rulings(filepath.Dir(abs))
		if err != nil {
			return nil, err
		}
		// The blocks are one stage: the same plugin and the same registration of the arm (asset, settings, tag).
		var p plan
		if err := readJSON(filepath.Join(dir, "run.json"), &p); err != nil {
			return nil, fmt.Errorf("side reads each block's run.json: %w", err)
		}
		if !strings.HasPrefix(p.Stage, "pilot") {
			switch {
			case !slices.Contains(p.SideAdmitted, arm):
				return nil, fmt.Errorf("%s did not run side arm %s, and holds another stage's receipts", dir, arm)
			case first == nil:
				first, firstDir = &p, dir
			case !samePlugin(p.Plugin, first.Plugin) || !reflect.DeepEqual(p.Side[arm], first.Side[arm]):
				return nil, fmt.Errorf("%s ran another plugin or another %s registration (tag, asset, setting) than %s: blocks of one stage share them (ADR-163 Decision 9)", dir, arm, firstDir)
			}
		}
		for _, src := range []struct{ glob, config string }{{filepath.Join(dir, "*.receipt.json"), control}, {filepath.Join(dir, sideDir, "*.receipt.json"), arm}} {
			paths, err := filepath.Glob(src.glob)
			if err != nil {
				return nil, err
			}
			for _, path := range paths {
				var r Receipt
				if err := readJSON(path, &r); err != nil {
					return nil, err
				}
				if r.Config != src.config || strings.HasPrefix(r.Stage, "pilot") {
					continue
				}
				// A ruling names an attempt the blocks hold; a side arm's are in side/, which no ruling names.
				if src.config == control {
					if err := rule(ruled, &r, map[string]bool{}); err != nil {
						return nil, err
					}
				}
				switch r.Exclusion {
				case "", ExclusionStart, ExclusionSetup, ExclusionInfra:
				default:
					return nil, fmt.Errorf("attempt %s: exclusion %q is not one the registration admits", r.Attempt, r.Exclusion)
				}
				rs = append(rs, r)
			}
		}
	}
	type key struct{ stage, model, source string }
	groups := map[key][]Receipt{}
	armSeen := false
	for _, r := range rs {
		k := key{r.Stage, r.Model, r.Source}
		groups[k] = append(groups[k], r)
		armSeen = armSeen || r.Config == arm
	}
	if !armSeen {
		return nil, fmt.Errorf("no receipt of side arm %s in %s", arm, strings.Join(dirs, ", "))
	}
	var rows []SideRow
	for k, g := range groups {
		res, err := judgeAgainst(control, k.stage, k.model, k.source, g, arm, lookLevel, true)
		if err != nil {
			return nil, err
		}
		row := SideRow{Stage: k.stage, Model: k.model, Arm: arm, Control: control, Result: res, Claim: bar.claim}
		row.Verdict = sideVerdict(arm, res.RelHi, res.SolvedLo, res.Tasks, res.Tasks-len(res.CostUnpaired))
		row.TurnsControl, row.TurnsArm, row.WallControl, row.WallArm = pairedMeans(g, control, arm, res.Unpaired)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Stage != rows[j].Stage {
			return rows[i].Stage < rows[j].Stage
		}
		return rows[i].Model < rows[j].Model
	})
	return rows, nil
}

// pairedMeans are the mean turns and wall seconds of the control's and the
// arm's attempts over the tasks both have, an excluded attempt and an unpaired
// task left out.
func pairedMeans(g []Receipt, control, arm string, unpaired []string) (tc, ta, wc, wa float64) {
	type pair struct{ c, a *Receipt }
	byTask := map[string]*pair{}
	for i := range g {
		r := &g[i]
		if r.Exclusion != "" || slices.Contains(unpaired, r.Task) || (r.Config != control && r.Config != arm) {
			continue
		}
		p := byTask[r.Task]
		if p == nil {
			p = &pair{}
			byTask[r.Task] = p
		}
		if r.Config == control {
			p.c = r
		} else {
			p.a = r
		}
	}
	n := 0.0
	for _, p := range byTask {
		if p.c == nil || p.a == nil {
			continue
		}
		n++
		tc += float64(p.c.NumTurns)
		ta += float64(p.a.NumTurns)
		wc += float64(p.c.WallMS) / 1000
		wa += float64(p.a.WallMS) / 1000
	}
	if n == 0 {
		return 0, 0, 0, 0
	}
	return tc / n, ta / n, wc / n, wa / n
}
