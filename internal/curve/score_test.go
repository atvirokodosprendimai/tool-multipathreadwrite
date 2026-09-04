package curve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cell generates one trial into a fresh directory and returns what the client
// would receive plus the planted answer. Every test starts here so that the
// scorer is always exercised against a tree the generator produced, never a
// hand-built one — a hand-built tree would let the two halves drift apart while
// every test stayed green.
func cell(t *testing.T, p Params) (string, Manifest, Answer) {
	t.Helper()
	dir := t.TempDir()
	m, err := Generate(dir, p)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	_, a, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return dir, m, a
}

// lines returns the target file's lines, 1-based in the returned map so a test
// can talk about line N the way a plan does.
func lines(t *testing.T, dir string, m Manifest) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(m.Tree, m.File))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func replaceAt(m Manifest, line int) string {
	return fmt.Sprintf("@@ %s %d replace\ntimeout = 45\n", m.File, line)
}

func TestTheScorerCountsAWrongLineAsAMiss(t *testing.T) {
	dir, m, a := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 1})
	all := lines(t, dir, m)
	target := all[a.Line-1]

	// The right line is a hit, and the changed set is exactly that line.
	s, err := ScoreTrial(dir, Result{TrialID: m.TrialID, ServedBytes: m.ServedBytes, Plan: replaceAt(m, a.Line)})
	if err != nil {
		t.Fatalf("score right plan: %v", err)
	}
	if s.Outcome != Hit || len(s.Changed) != 1 || s.Changed[0] != a.Line {
		t.Fatalf("right plan: outcome=%s changed=%v target=%d", s.Outcome, s.Changed, a.Line)
	}

	// A distractor's line has the SAME content as the target — that is the
	// whole threat to validity — so a plan that parses, applies cleanly and
	// reports ok can still have changed the wrong one. Find such a line.
	wrong := 0
	for i, l := range all {
		if i+1 != a.Line && l == target {
			wrong = i + 1
			break
		}
	}
	if wrong == 0 {
		t.Fatalf("no distractor shares the target's content %q — the fixture does not carry the threat it exists to measure", target)
	}
	s, err = ScoreTrial(dir, Result{TrialID: m.TrialID, ServedBytes: m.ServedBytes, Plan: replaceAt(m, wrong)})
	if err != nil {
		t.Fatalf("score wrong plan: %v", err)
	}
	if s.Outcome != Miss {
		t.Fatalf("a plan that changed line %d instead of %d scored %s; the scorer cannot see a wrong line", wrong, a.Line, s.Outcome)
	}
	if len(s.Changed) != 1 || s.Changed[0] != wrong {
		t.Fatalf("changed=%v, want [%d]", s.Changed, wrong)
	}
}

func TestTheScorerRefusesResultsFromADifferentTrial(t *testing.T) {
	dir, m, a := cell(t, Params{ServedBytes: 4000, Position: Early, Distractors: 4, Seed: 2})
	plan := replaceAt(m, a.Line)

	if _, err := ScoreTrial(dir, Result{TrialID: "someone-else", ServedBytes: m.ServedBytes, Plan: plan}); err == nil {
		t.Fatal("a result echoing another trial's id was scored instead of refused")
	} else if !strings.Contains(err.Error(), "trial") {
		t.Fatalf("refusal does not name what mismatched: %v", err)
	}
	if _, err := ScoreTrial(dir, Result{TrialID: m.TrialID, ServedBytes: m.ServedBytes + 1, Plan: plan}); err == nil {
		t.Fatal("a result echoing a different served size was scored instead of refused")
	} else if !strings.Contains(err.Error(), "served") {
		t.Fatalf("refusal does not name what mismatched: %v", err)
	}
	if _, err := ScoreTrial(dir, Result{TrialID: m.TrialID, ServedBytes: m.ServedBytes, Plan: plan}); err != nil {
		t.Fatalf("a result that echoes both correctly was refused: %v", err)
	}
}

func TestARefusedPlanIsNotCountedAsALocalisationMiss(t *testing.T) {
	dir, m, a := cell(t, Params{ServedBytes: 4000, Position: Late, Distractors: 4, Seed: 3})
	ok := Result{TrialID: m.TrialID, ServedBytes: m.ServedBytes}

	unparseable := ok
	unparseable.Plan = "this is not a plan\n"
	s, err := ScoreTrial(dir, unparseable)
	if err != nil {
		t.Fatalf("an unparseable PLAN is an outcome, not a refusal of the RESULT: %v", err)
	}
	if s.Outcome != RefusedParse {
		t.Fatalf("unparseable plan scored %s, want %s", s.Outcome, RefusedParse)
	}

	// Parses, but a sha guard that cannot match: the engine refuses it.
	guarded := ok
	guarded.Plan = fmt.Sprintf("@@ %s %d replace sha=%s\ntimeout = 45\n", m.File, a.Line, strings.Repeat("0", 64))
	s, err = ScoreTrial(dir, guarded)
	if err != nil {
		t.Fatalf("score guarded plan: %v", err)
	}
	if s.Outcome != RefusedApply {
		t.Fatalf("a plan the engine refused scored %s, want %s", s.Outcome, RefusedApply)
	}

	// And in the tally the refusal is OUTSIDE the correct-address denominator.
	hit := Score{ServedBytes: m.ServedBytes, Position: m.Position, Outcome: Hit}
	miss := Score{ServedBytes: m.ServedBytes, Position: m.Position, Outcome: Miss}
	cells := Tally([]Score{hit, miss, s})
	if len(cells) != 1 {
		t.Fatalf("three scores in one cell tallied as %d cells", len(cells))
	}
	c := cells[0]
	if c.N != 2 || c.Hits != 1 || c.Refused != 1 {
		t.Fatalf("N=%d hits=%d refused=%d; a refusal leaked into the denominator or was dropped", c.N, c.Hits, c.Refused)
	}
	if c.Rate != 0.5 {
		t.Fatalf("rate=%v, want 0.5 — the refusal must not move the rate", c.Rate)
	}
	if !(c.Low < c.Rate && c.Rate < c.High) {
		t.Fatalf("interval [%v, %v] does not contain the rate %v", c.Low, c.High, c.Rate)
	}
}

func TestEachCellGetsItsOwnLedger(t *testing.T) {
	p := Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 4}
	d1, m1, _ := cell(t, p)
	d2, m2, _ := cell(t, p)

	for _, x := range []struct {
		dir string
		m   Manifest
	}{{d1, m1}, {d2, m2}} {
		if !filepath.IsAbs(x.m.StateHome) {
			t.Fatalf("state home %q is not absolute; mrw refuses a relative XDG_STATE_HOME", x.m.StateHome)
		}
		if st, err := os.Stat(x.m.StateHome); err != nil || !st.IsDir() {
			t.Fatalf("state home %q does not exist as a directory: %v", x.m.StateHome, err)
		}
		if !strings.HasPrefix(x.m.StateHome, x.dir) {
			t.Fatalf("state home %q is not inside the cell %q", x.m.StateHome, x.dir)
		}
	}
	if m1.StateHome == m2.StateHome {
		t.Fatalf("two cells share a state home %q — one trial's ledger would license the next", m1.StateHome)
	}

	// Served bytes is the size of what mrw's own renderer produced, measured,
	// and it is what the manifest carries.
	served, err := os.ReadFile(filepath.Join(d1, "served.txt"))
	if err != nil {
		t.Fatalf("served.txt: %v", err)
	}
	if len(served) != m1.ServedBytes {
		t.Fatalf("manifest says %d served bytes, served.txt holds %d", m1.ServedBytes, len(served))
	}
	if !strings.Contains(string(served), "==> "+m1.File) {
		t.Fatalf("served.txt is not mrw's read rendering (no file header): %.80q", served)
	}
}

func TestDistractorCountDoesNotVaryWithSize(t *testing.T) {
	const k = 4
	candidates := func(t *testing.T, dir string, m Manifest, a Answer) int {
		n := 0
		for _, l := range lines(t, dir, m) {
			if l == lines(t, dir, m)[a.Line-1] {
				n++
			}
		}
		return n
	}
	ds, ms, as := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: k, Seed: 5})
	dl, ml, al := cell(t, Params{ServedBytes: 40000, Position: Middle, Distractors: k, Seed: 5})

	if ms.ServedBytes < 4000 || ml.ServedBytes < 40000 || ml.ServedBytes <= ms.ServedBytes {
		t.Fatalf("served bytes small=%d large=%d; the size cell was not honoured", ms.ServedBytes, ml.ServedBytes)
	}
	if cs, cl := candidates(t, ds, ms, as), candidates(t, dl, ml, al); cs != k+1 || cl != k+1 {
		t.Fatalf("candidate lines small=%d large=%d, want %d in both — distractor count rode along with size", cs, cl, k+1)
	}
	if ms.Distractors != k || ml.Distractors != k {
		t.Fatalf("manifest distractors small=%d large=%d, want %d", ms.Distractors, ml.Distractors, k)
	}

	// Position is a stratum: for one size, early < middle < late by line.
	_, _, e := cell(t, Params{ServedBytes: 8000, Position: Early, Distractors: k, Seed: 6})
	_, _, mid := cell(t, Params{ServedBytes: 8000, Position: Middle, Distractors: k, Seed: 6})
	_, _, l := cell(t, Params{ServedBytes: 8000, Position: Late, Distractors: k, Seed: 6})
	if !(e.Line < mid.Line && mid.Line < l.Line) {
		t.Fatalf("target lines early=%d middle=%d late=%d are not ordered as strata", e.Line, mid.Line, l.Line)
	}
}
