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

// lines returns the target file's lines, so a test can talk about line N the
// way a plan does.
func lines(t *testing.T, m Manifest) []string {
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

func ok(m Manifest) Result {
	return Result{TrialID: m.TrialID, ServedBytes: m.ServedBytes}
}

func TestTheScorerCountsAWrongLineAsAMiss(t *testing.T) {
	_, m, a := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 1})
	dir := filepath.Dir(m.Tree)
	all := lines(t, m)
	target := all[a.Line-1]

	// The right line is a hit, and the changed set is exactly that line.
	r := ok(m)
	r.Plan = replaceAt(m, a.Line)
	s, err := ScoreTrial(dir, r)
	if err != nil {
		t.Fatalf("score right plan: %v", err)
	}
	if s.Outcome != Hit || len(s.Changed) != 1 || s.Changed[0] != a.Line {
		t.Fatalf("right plan: outcome=%s changed=%v target=%d", s.Outcome, s.Changed, a.Line)
	}

	// A distractor's line has the SAME content as the target — that is the
	// whole threat to validity — so a plan that parses, applies cleanly and
	// reports ok can still have changed the wrong one.
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
	r.Plan = replaceAt(m, wrong)
	s, err = ScoreTrial(dir, r)
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

// TestAPlanThatAlsoWritesElsewhereIsNotAHit covers the plan that fixes the
// right line and does something else too. Diffing only the target file scores
// it a clean hit, which is not "changed exactly the planted line".
func TestAPlanThatAlsoWritesElsewhereIsNotAHit(t *testing.T) {
	_, m, a := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 11})
	dir := filepath.Dir(m.Tree)

	r := ok(m)
	r.Plan = replaceAt(m, a.Line) + "@@ elsewhere.txt - create\nsomething the task never asked for\n"
	s, err := ScoreTrial(dir, r)
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	if s.Outcome != Miss {
		t.Fatalf("a plan that also created another file scored %s, want %s", s.Outcome, Miss)
	}
	if len(s.Touched) != 1 || s.Touched[0] != "elsewhere.txt" {
		t.Fatalf("touched=%v, want [elsewhere.txt] — the verdict must say WHY it was not a hit", s.Touched)
	}
	if _, err := os.Stat(filepath.Join(m.Tree, "elsewhere.txt")); !os.IsNotExist(err) {
		t.Fatalf("scoring wrote into the fixture tree itself: %v", err)
	}
}

// TestTheManifestCarriesNoGroundTruth is the leak check. The stratum and the
// distractor count are not secrets on their own, but together they name the
// target's block by arithmetic, so a client holding them can localise by
// counting instead of by reading.
func TestTheManifestCarriesNoGroundTruth(t *testing.T) {
	dir, m, a := cell(t, Params{ServedBytes: 5000, Position: Late, Distractors: 4, Seed: 12})
	raw, err := os.ReadFile(filepath.Join(dir, manifestName))
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	for _, leak := range []string{"position", "distractor", string(Early), string(Middle), string(Late)} {
		if strings.Contains(strings.ToLower(string(raw)), leak) {
			t.Errorf("manifest.json mentions %q, which lets a client derive the target's block without reading:\n%s", leak, raw)
		}
	}
	if strings.Contains(string(raw), fmt.Sprint(a.Line)) && a.Line > 999 {
		t.Errorf("manifest.json contains the planted line %d verbatim:\n%s", a.Line, raw)
	}
	// And the answer still has them, because the scorer needs the stratum.
	if a.Position != Late || a.Distractors != 4 {
		t.Fatalf("answer.json lost the stratum or the count: %+v", a)
	}
	if m.Cell != 5000 {
		t.Fatalf("manifest cell = %d, want the REQUESTED 5000", m.Cell)
	}
}

func TestTheScorerRefusesResultsFromADifferentTrial(t *testing.T) {
	_, m, a := cell(t, Params{ServedBytes: 4000, Position: Early, Distractors: 4, Seed: 2})
	dir := filepath.Dir(m.Tree)
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
	_, m, a := cell(t, Params{ServedBytes: 4000, Position: Late, Distractors: 4, Seed: 3})
	dir := filepath.Dir(m.Tree)

	unparseable := ok(m)
	unparseable.Plan = "this is not a plan\n"
	s, err := ScoreTrial(dir, unparseable)
	if err != nil {
		t.Fatalf("an unparseable PLAN is an outcome, not a refusal of the RESULT: %v", err)
	}
	if s.Outcome != RefusedParse {
		t.Fatalf("unparseable plan scored %s, want %s", s.Outcome, RefusedParse)
	}

	// Parses, but a sha guard that cannot match: the engine refuses it.
	guarded := ok(m)
	guarded.Plan = fmt.Sprintf("@@ %s %d replace sha=%s\ntimeout = 45\n", m.File, a.Line, strings.Repeat("0", 64))
	applyRef, err := ScoreTrial(dir, guarded)
	if err != nil {
		t.Fatalf("score guarded plan: %v", err)
	}
	if applyRef.Outcome != RefusedApply {
		t.Fatalf("a plan the engine refused scored %s, want %s", applyRef.Outcome, RefusedApply)
	}

	// A plan whose path names a DIRECTORY drives the engine into an I/O error.
	// That is still the client being wrong, so it is data, not harness breakage.
	dirPath := ok(m)
	dirPath.Plan = "@@ . 1 replace\nx\n"
	s2, err := ScoreTrial(dir, dirPath)
	if err != nil {
		t.Fatalf("a plan naming a directory was reported as a HARNESS failure, so a client's mistake would abort the run: %v", err)
	}
	if s2.Outcome != RefusedApply {
		t.Fatalf("a plan naming a directory scored %s, want %s", s2.Outcome, RefusedApply)
	}

	// In the tally the refusals stay OUTSIDE the correct-address denominator,
	// and the two kinds are counted apart.
	hit := Score{Cell: m.Cell, ServedBytes: m.ServedBytes, Position: a.Position, Outcome: Hit}
	miss := Score{Cell: m.Cell, ServedBytes: m.ServedBytes, Position: a.Position, Outcome: Miss}
	cells, err := Tally([]Score{hit, miss, s, applyRef})
	if err != nil {
		t.Fatalf("Tally: %v", err)
	}
	if len(cells) != 1 {
		t.Fatalf("four scores in one cell tallied as %d cells", len(cells))
	}
	c := cells[0]
	if c.N != 2 || c.Hits != 1 || c.Refused != 2 {
		t.Fatalf("N=%d hits=%d refused=%d; a refusal leaked into the denominator or was dropped", c.N, c.Hits, c.Refused)
	}
	if c.ParseRef != 1 || c.ApplyRef != 1 {
		t.Fatalf("refused_parse=%d refused_apply=%d; the pre-registration names them as two DVs, not one", c.ParseRef, c.ApplyRef)
	}
	if c.Rate != 0.5 {
		t.Fatalf("rate=%v, want 0.5 — the refusals must not move the rate", c.Rate)
	}
	if !(c.Low < c.Rate && c.Rate < c.High) {
		t.Fatalf("interval [%v, %v] does not contain the rate %v", c.Low, c.High, c.Rate)
	}

	// An outcome the scorer never produces is malformed data and is refused,
	// not bucketed — silently counting it would let it into the measurement.
	if _, err := Tally([]Score{{Cell: 1, Position: Early, Outcome: Outcome("garbage")}}); err == nil {
		t.Fatal("an unrecognised outcome was tallied instead of refused")
	}
}

// TestRepeatsOfOneCellTallyTogether is the aggregation check. Padding is fitted
// to REACH a size and overshoots by a seed-dependent amount, so keying the
// tally on measured bytes would give nearly every trial a cell of its own and
// turn every interval into the interval of a single observation.
func TestRepeatsOfOneCellTallyTogether(t *testing.T) {
	const want = 6000
	var scores []Score
	var served []int
	for seed := int64(1); seed <= 5; seed++ {
		_, m, a := cell(t, Params{ServedBytes: want, Position: Middle, Distractors: 4, Seed: seed})
		if m.Cell != want {
			t.Fatalf("seed %d: cell=%d, want %d", seed, m.Cell, want)
		}
		served = append(served, m.ServedBytes)
		scores = append(scores, Score{Cell: m.Cell, ServedBytes: m.ServedBytes, Position: a.Position, Outcome: Hit})
	}
	distinct := map[int]bool{}
	for _, b := range served {
		distinct[b] = true
	}
	if len(distinct) < 2 {
		t.Fatalf("served sizes %v are all equal, so this test cannot show that grouping absorbs the spread", served)
	}
	cells, err := Tally(scores)
	if err != nil {
		t.Fatalf("Tally: %v", err)
	}
	if len(cells) != 1 {
		t.Fatalf("five repeats of one cell tallied as %d cells (served %v) — N repetitions became N cells of one", len(cells), served)
	}
	c := cells[0]
	if c.N != 5 {
		t.Fatalf("N=%d, want 5", c.N)
	}
	if c.MinServed > c.MaxServed || c.MinServed == 0 {
		t.Fatalf("the cell does not report the byte spread it absorbed: [%d, %d]", c.MinServed, c.MaxServed)
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
		entries, err := os.ReadDir(x.m.StateHome)
		if err != nil {
			t.Fatalf("state home %q: %v", x.m.StateHome, err)
		}
		if len(entries) != 0 {
			t.Fatalf("state home %q is not empty (%d entries)", x.m.StateHome, len(entries))
		}
	}
	if m1.StateHome == m2.StateHome {
		t.Fatalf("two cells share a state home %q — one trial's ledger would license the next", m1.StateHome)
	}

	// Regenerating into a USED output directory is how a runner following the
	// documented protocol inherits a ledger, and it looks like success.
	if err := os.WriteFile(filepath.Join(m1.StateHome, "seen"), []byte("a ledger from an earlier trial\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(d1, p); err == nil {
		t.Fatal("regenerating into a directory whose state home already holds a ledger was accepted")
	} else if !strings.Contains(err.Error(), "state") {
		t.Fatalf("the refusal does not name the state directory: %v", err)
	}

	// Served bytes is the size of what mrw's own renderer produced, measured.
	servedBytes, err := os.ReadFile(filepath.Join(d2, servedName))
	if err != nil {
		t.Fatalf("served.txt: %v", err)
	}
	if len(servedBytes) != m2.ServedBytes {
		t.Fatalf("manifest says %d served bytes, served.txt holds %d", m2.ServedBytes, len(servedBytes))
	}
	if !strings.Contains(string(servedBytes), "==> "+m2.File) {
		t.Fatalf("served.txt is not mrw's read rendering (no file header): %.80q", servedBytes)
	}
}

func TestDistractorCountDoesNotVaryWithSize(t *testing.T) {
	const k = 4
	candidates := func(t *testing.T, m Manifest, a Answer) int {
		all := lines(t, m)
		n := 0
		for _, l := range all {
			if l == all[a.Line-1] {
				n++
			}
		}
		return n
	}
	_, ms, as := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: k, Seed: 5})
	_, ml, al := cell(t, Params{ServedBytes: 40000, Position: Middle, Distractors: k, Seed: 5})

	if ms.ServedBytes < 4000 || ml.ServedBytes < 40000 || ml.ServedBytes <= ms.ServedBytes {
		t.Fatalf("served bytes small=%d large=%d; the size cell was not honoured", ms.ServedBytes, ml.ServedBytes)
	}
	if cs, cl := candidates(t, ms, as), candidates(t, ml, al); cs != k+1 || cl != k+1 {
		t.Fatalf("candidate lines small=%d large=%d, want %d in both — distractor count rode along with size", cs, cl, k+1)
	}
	if as.Distractors != k || al.Distractors != k {
		t.Fatalf("answer distractors small=%d large=%d, want %d", as.Distractors, al.Distractors, k)
	}

	// Position is a stratum: for one size, early < middle < late by line.
	_, _, e := cell(t, Params{ServedBytes: 8000, Position: Early, Distractors: k, Seed: 6})
	_, _, mid := cell(t, Params{ServedBytes: 8000, Position: Middle, Distractors: k, Seed: 6})
	_, _, l := cell(t, Params{ServedBytes: 8000, Position: Late, Distractors: k, Seed: 6})
	if !(e.Line < mid.Line && mid.Line < l.Line) {
		t.Fatalf("target lines early=%d middle=%d late=%d are not ordered as strata", e.Line, mid.Line, l.Line)
	}

	// One distractor gives two blocks, so early and middle would be the SAME
	// trial under two stratum names. Refused rather than silently degenerate.
	if _, err := Generate(t.TempDir(), Params{ServedBytes: 3000, Position: Middle, Distractors: 1, Seed: 7}); err == nil {
		t.Fatal("a trial with one distractor was accepted, so two strata name the same block")
	}
	for _, p := range []Position{Early, Middle, Late} {
		if _, err := Generate(t.TempDir(), Params{ServedBytes: 3000, Position: p, Distractors: 2, Seed: 7}); err != nil {
			t.Fatalf("two distractors is the documented minimum but %s was refused: %v", p, err)
		}
	}
}
