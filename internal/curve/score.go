package curve

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Result is what a client hands back for one trial. TrialID and ServedBytes are
// echoed so the scorer can refuse a result pasted in from another cell — a
// manifest emitted at one size and answered at another would otherwise score
// cleanly and mean nothing.
type Result struct {
	TrialID     string `json:"trial_id"`
	ServedBytes int    `json:"served_bytes"`
	Plan        string `json:"plan"`
}

// Outcome is one of three answers. Only Hit and Miss enter the correct-address
// denominator; a refused plan is reported beside them and never folded in.
type Outcome string

// The outcomes. RefusedParse and RefusedApply are the format failing, which is
// the secondary variable; Hit and Miss are the primary one.
const (
	Hit          Outcome = "hit"
	Miss         Outcome = "miss"
	RefusedParse Outcome = "refused_parse"
	RefusedApply Outcome = "refused_apply"
)

// Score is one trial's verdict.
type Score struct {
	TrialID     string   `json:"trial_id"`
	ServedBytes int      `json:"served_bytes"`
	Position    Position `json:"position"`
	Outcome     Outcome  `json:"outcome"`
	Target      int      `json:"target"`
	Changed     []int    `json:"changed"`
	Reason      string   `json:"reason,omitempty"`
}

// ScoreTrial scores r against the trial in dir. The error is a refusal of the
// RESULT — it names another trial or another served size — and never a verdict
// on the plan: a plan that does not parse or does not apply is an Outcome.
//
// The primary variable is measured by applying, not by comparing addresses.
// The plan runs through plan.Parse and apply.Apply exactly as mrw runs it, on
// a copy of the fixture, and the lines that differ afterwards are the answer.
// A hit is a plan that changed exactly the planted line; anything else that
// applied is a miss, including a plan that changed that line and another.
func ScoreTrial(dir string, r Result) (Score, error) {
	m, a, err := Load(dir)
	if err != nil {
		return Score{}, err
	}
	if r.TrialID != m.TrialID {
		return Score{}, fmt.Errorf("result echoes trial %q but %s holds trial %q", r.TrialID, dir, m.TrialID)
	}
	if r.ServedBytes != m.ServedBytes {
		return Score{}, fmt.Errorf("result echoes %d served bytes but trial %s served %d", r.ServedBytes, m.TrialID, m.ServedBytes)
	}
	s := Score{TrialID: m.TrialID, ServedBytes: m.ServedBytes, Position: m.Position, Target: a.Line}

	hunks, err := plan.Parse(strings.NewReader(r.Plan))
	if err != nil {
		s.Outcome, s.Reason = RefusedParse, err.Error()
		return s, nil
	}

	scratch, err := os.MkdirTemp("", "curve-score-")
	if err != nil {
		return Score{}, err
	}
	defer os.RemoveAll(scratch)
	if err := copyTree(m.Tree, scratch); err != nil {
		return Score{}, err
	}
	before, err := os.ReadFile(filepath.Join(scratch, m.File))
	if err != nil {
		return Score{}, err
	}

	// The ledger is seeded WHOLE. The read-before-write guard is not what is
	// under test, and left in force it would turn a plan addressing an
	// unserved line — the worst kind of miss — into a refusal and quietly
	// remove it from the primary denominator (ADR-020, Decision 4).
	ledger := map[string]apply.Seen{filepath.Clean(m.File): {SHA: seen.SHA(before)}}
	res, err := apply.Apply(scratch, inputs(hunks), apply.Options{Seen: ledger})
	if err != nil {
		return Score{}, fmt.Errorf("apply: %w", err)
	}
	if !res.Applied {
		s.Outcome, s.Reason = RefusedApply, firstRefusal(res)
		return s, nil
	}
	after, err := os.ReadFile(filepath.Join(scratch, m.File))
	if err != nil {
		return Score{}, err
	}
	s.Changed = changed(string(before), string(after))
	if len(s.Changed) == 1 && s.Changed[0] == a.Line {
		s.Outcome = Hit
	} else {
		s.Outcome = Miss
	}
	return s, nil
}

// inputs is the mapping cmd/mrw performs before apply, minus the working-set
// pointer resolution a benchmark plan has no use for.
func inputs(hunks []plan.Hunk) []apply.Input {
	in := make([]apply.Input, 0, len(hunks))
	for _, h := range hunks {
		in = append(in, apply.Input{
			Path: h.Path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
			StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat,
			Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor,
			SrcLine: h.SrcLine, Index: h.Index,
		})
	}
	return in
}

func firstRefusal(res apply.Result) string {
	for _, h := range res.Hunks {
		if h.Status != apply.StatusOK && h.Reason != "" {
			return h.Reason
		}
	}
	return fmt.Sprintf("%d hunk(s) failed", res.Failed)
}

// changed lists the 1-based lines that differ between two texts. Where the
// line counts differ, every line past the shorter text counts as changed, so
// an insertion or deletion never masquerades as a one-line edit.
func changed(before, after string) []int {
	b := strings.Split(before, "\n")
	a := strings.Split(after, "\n")
	n := len(b)
	if len(a) > n {
		n = len(a)
	}
	var out []int
	for i := 0; i < n; i++ {
		if i >= len(b) || i >= len(a) || b[i] != a[i] {
			out = append(out, i+1)
		}
	}
	return out
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o700)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o600)
	})
}

// Cell is one point on the curve: a proportion with its interval, and the
// refusals that were kept OUT of its denominator.
type Cell struct {
	ServedBytes int      `json:"served_bytes"`
	Position    Position `json:"position"`
	N           int      `json:"n"`
	Hits        int      `json:"hits"`
	Misses      int      `json:"misses"`
	Refused     int      `json:"refused"`
	Rate        float64  `json:"rate"`
	Low         float64  `json:"low"`
	High        float64  `json:"high"`
}

// Tally groups scores by (served bytes, position) and reports each cell as a
// proportion with a 95% Wilson interval. A refused plan is counted beside the
// cell and excluded from N: folding a format failure into a localisation rate
// would bend the primary curve with the secondary variable.
func Tally(scores []Score) []Cell {
	type key struct {
		b int
		p Position
	}
	cells := map[key]*Cell{}
	for _, s := range scores {
		k := key{s.ServedBytes, s.Position}
		c := cells[k]
		if c == nil {
			c = &Cell{ServedBytes: s.ServedBytes, Position: s.Position}
			cells[k] = c
		}
		switch s.Outcome {
		case Hit:
			c.Hits++
			c.N++
		case Miss:
			c.Misses++
			c.N++
		default:
			c.Refused++
		}
	}
	out := make([]Cell, 0, len(cells))
	for _, c := range cells {
		c.Rate, c.Low, c.High = wilson(c.Hits, c.N)
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ServedBytes != out[j].ServedBytes {
			return out[i].ServedBytes < out[j].ServedBytes
		}
		return out[i].Position < out[j].Position
	})
	return out
}

// wilson is the score interval at z=1.96. It is taken over the normal
// approximation because cells are small and rates near 0 or 1 are exactly
// where the normal interval leaves the unit square.
func wilson(hits, n int) (rate, low, high float64) {
	if n == 0 {
		return 0, 0, 0
	}
	const z = 1.96
	nn := float64(n)
	p := float64(hits) / nn
	denom := 1 + z*z/nn
	centre := (p + z*z/(2*nn)) / denom
	half := z * math.Sqrt(p*(1-p)/nn+z*z/(4*nn*nn)) / denom
	return p, centre - half, centre + half
}
