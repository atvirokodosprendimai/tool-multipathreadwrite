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
// the secondary variable; Hit and Miss are the primary one. They are counted
// SEPARATELY, because the pre-registration names them as two secondary DVs.
const (
	Hit          Outcome = "hit"
	Miss         Outcome = "miss"
	RefusedParse Outcome = "refused_parse"
	RefusedApply Outcome = "refused_apply"
)

// Score is one trial's verdict. Cell is the REQUESTED size and is what the
// tally groups by; ServedBytes is what the trial actually served, which varies
// with the seed and so cannot be a grouping key.
type Score struct {
	TrialID     string   `json:"trial_id"`
	Cell        int      `json:"cell"`
	ServedBytes int      `json:"served_bytes"`
	Position    Position `json:"position"`
	Outcome     Outcome  `json:"outcome"`
	Target      int      `json:"target"`
	Changed     []int    `json:"changed"`
	// Touched names any OTHER file the plan wrote. A plan that edits the target
	// and also creates a file elsewhere did not change "exactly the planted
	// line", so it is a miss and this says why.
	Touched []string `json:"touched,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

// ScoreTrial scores r against the trial in dir. The error is a refusal of the
// RESULT — it names another trial or another served size — or a failure of the
// harness itself. It is never a verdict on the plan: a plan that does not
// parse, does not apply, or drives the engine into an I/O error is an Outcome,
// because a client's mistake is data and must not be reported as harness
// breakage.
//
// The primary variable is measured by applying, not by comparing addresses.
// The plan runs through plan.Parse and apply.Apply exactly as mrw runs it, on
// a copy of the fixture, and the lines that differ afterwards are the answer.
// A hit is a plan that changed exactly the planted line AND wrote no other
// file; anything else that applied is a miss.
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
	s := Score{
		TrialID: m.TrialID, Cell: m.Cell, ServedBytes: m.ServedBytes,
		Position: a.Position, Target: a.Line,
	}

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
	// A hunk whose path names a DIRECTORY drives Apply into an I/O error rather
	// than a per-hunk refusal, and an error would abort the run. That is the
	// client being wrong, which is data, so it is caught here by name.
	// Out-of-root and absolute paths need no such handling: the engine already
	// refuses those with a verdict.
	if bad := addressesADirectory(scratch, hunks); bad != "" {
		s.Outcome = RefusedApply
		s.Reason = fmt.Sprintf("the plan addresses %s, which is a directory", bad)
		return s, nil
	}
	ledger := map[string]apply.Seen{filepath.Clean(m.File): {SHA: seen.SHA(before)}}
	res, err := apply.Apply(scratch, inputs(hunks), apply.Options{Seen: ledger})
	if err != nil {
		// Apply fills the receipt before returning an error, so a receipt with
		// per-hunk verdicts means the PLAN drove this — a path naming a
		// directory, for instance. That is the client being wrong, which is
		// data. Only a failure with no receipt at all is the harness's.
		if len(res.Hunks) > 0 {
			s.Outcome, s.Reason = RefusedApply, err.Error()
			return s, nil
		}
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
	s.Touched = wroteElsewhere(res, m.File)
	if len(s.Changed) == 1 && s.Changed[0] == a.Line && len(s.Touched) == 0 {
		s.Outcome = Hit
	} else {
		s.Outcome = Miss
	}
	return s, nil
}

// wroteElsewhere lists files the plan wrote that are not the target. Diffing
// only the target file would score a plan that fixed the right line and also
// created something else as a clean hit.
func wroteElsewhere(res apply.Result, target string) []string {
	var out []string
	want := filepath.Clean(target)
	for _, f := range res.Files {
		if f.Written && filepath.Clean(f.Path) != want {
			out = append(out, f.Path)
		}
	}
	sort.Strings(out)
	return out
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
//
// It groups by the REQUESTED size, not the measured one. Padding is fitted to
// reach a size and overshoots it by a seed-dependent amount, so grouping by
// measured bytes would give almost every trial a cell of its own and turn every
// interval into the interval of a single observation. MinServed and MaxServed
// report the spread that grouping absorbs.
type Cell struct {
	Cell      int      `json:"cell"`
	Position  Position `json:"position"`
	N         int      `json:"n"`
	Hits      int      `json:"hits"`
	Misses    int      `json:"misses"`
	Refused   int      `json:"refused"`
	ParseRef  int      `json:"refused_parse"`
	ApplyRef  int      `json:"refused_apply"`
	MinServed int      `json:"min_served_bytes"`
	MaxServed int      `json:"max_served_bytes"`
	Rate      float64  `json:"rate"`
	Low       float64  `json:"low"`
	High      float64  `json:"high"`
}

// Tally groups scores by (requested size, position) and reports each cell as a
// proportion with a 95% Wilson interval. A refused plan is counted beside the
// cell and excluded from N: folding a format failure into a localisation rate
// would bend the primary curve with the secondary variable. The two refusal
// kinds are kept apart, because the pre-registration names them separately.
//
// An unrecognised outcome is REFUSED rather than bucketed. Counting it as a
// refusal would let malformed score data enter the measurement silently, which
// is the whole failure this harness exists to avoid one level up.
func Tally(scores []Score) ([]Cell, error) {
	type key struct {
		c int
		p Position
	}
	cells := map[key]*Cell{}
	for i, s := range scores {
		switch s.Outcome {
		case Hit, Miss, RefusedParse, RefusedApply:
		default:
			return nil, fmt.Errorf("score %d (trial %q) has outcome %q, which is not one of hit, miss, refused_parse or refused_apply", i, s.TrialID, s.Outcome)
		}
		k := key{s.Cell, s.Position}
		c := cells[k]
		if c == nil {
			c = &Cell{Cell: s.Cell, Position: s.Position, MinServed: s.ServedBytes, MaxServed: s.ServedBytes}
			cells[k] = c
		}
		if s.ServedBytes < c.MinServed {
			c.MinServed = s.ServedBytes
		}
		if s.ServedBytes > c.MaxServed {
			c.MaxServed = s.ServedBytes
		}
		switch s.Outcome {
		case Hit:
			c.Hits++
			c.N++
		case Miss:
			c.Misses++
			c.N++
		case RefusedParse:
			c.ParseRef++
			c.Refused++
		case RefusedApply:
			c.ApplyRef++
			c.Refused++
		}
	}
	out := make([]Cell, 0, len(cells))
	for _, c := range cells {
		c.Rate, c.Low, c.High = wilson(c.Hits, c.N)
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cell != out[j].Cell {
			return out[i].Cell < out[j].Cell
		}
		return positionOrder(out[i].Position) < positionOrder(out[j].Position)
	})
	return out, nil
}

// positionOrder sorts the strata the way they sit in the file rather than
// alphabetically, which would report early, late, middle.
func positionOrder(p Position) int {
	switch p {
	case Early:
		return 0
	case Middle:
		return 1
	case Late:
		return 2
	}
	return 3
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

// addressesADirectory returns the first plan path that names a directory in the
// scratch tree, or "". Only a root-relative path is checked: an absolute or
// escaping one is the engine's to refuse, and it does so with a verdict.
func addressesADirectory(root string, hunks []plan.Hunk) string {
	for _, h := range hunks {
		p := filepath.Clean(h.Path)
		if filepath.IsAbs(p) || p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
			continue
		}
		if st, err := os.Stat(filepath.Join(root, p)); err == nil && st.IsDir() {
			return h.Path
		}
	}
	return ""
}
