package curve

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestATargetChosenByRelationNotName is ADR-020-T2's Enforced-by. The named
// selector measures whether a client can MATCH a string, and the first reading
// returned 42 of 45 against it — a ceiling, not a curve. This selector measures
// whether a client can READ, so the instruction must give it nothing to match.
func TestATargetChosenByRelationNotName(t *testing.T) {
	p := Params{ServedBytes: 4000, Position: Middle, Distractors: 3, Seed: 7, Selector: ByOddRetries}
	dir := t.TempDir()
	m, err := Generate(dir, p)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	_, a, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(m.Tree, m.File))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")

	// 1. The instruction names no service. A name in the instruction is a
	//    string to match, and matching is what the named selector measures.
	for _, n := range regexp.MustCompile(`svc-[a-z]+`).FindAllString(string(b), -1) {
		if strings.Contains(m.Instruction, n) {
			t.Fatalf("the instruction names service %q, so the target is findable without reading:\n%s", n, m.Instruction)
		}
	}

	// 2. Exactly one block's retries differs from every other, over at least
	//    three blocks — with two, each differs from the other and "the odd one"
	//    picks out nothing.
	counts := map[string]int{}
	for _, l := range lines {
		if strings.HasPrefix(l, "retries = ") {
			counts[l]++
		}
	}
	blocks := 0
	odd, common := "", ""
	for value, n := range counts {
		blocks += n
		if n == 1 {
			odd = value
		} else {
			common = value
		}
	}
	if len(counts) != 2 || odd == "" || common == "" {
		t.Fatalf("want one odd retries value and one common one, got %v", counts)
	}
	if blocks < 3 {
		t.Fatalf("want at least three blocks so the odd one is unambiguous, got %d", blocks)
	}

	// 3. The answer names the odd block's timeout line — the line below its
	//    retries. If the answer pointed at any other block the trial would be
	//    scored against a line the instruction does not describe.
	want := 0
	for i, l := range lines {
		if l == odd {
			want = i + 2 // lines is 0-based; retries is i+1, its timeout i+2
		}
	}
	if a.Line != want {
		t.Fatalf("the answer names line %d; the odd block's timeout line is %d", a.Line, want)
	}
	if got := lines[a.Line-1]; got != targetLine {
		t.Fatalf("line %d is %q, want %q", a.Line, got, targetLine)
	}

	// 4. Two selectors are two trials. Same size, position, distractors and
	//    seed, so without this a result answering the named cell would score
	//    against the relational one and the scorer could not tell.
	named := p
	named.Selector = ByName
	if trialID(named) == trialID(p) {
		t.Fatalf("a named cell and a relational cell with the same parameters share trial id %s", trialID(p))
	}
}

// TestTheNamedSelectorIsUnchangedByTheRelationalOne pins the zero value: every
// caller written before T2 keeps the fixture it had, which is what lets the two
// readings be compared at all.
func TestTheNamedSelectorIsUnchangedByTheRelationalOne(t *testing.T) {
	p := Params{ServedBytes: 4000, Position: Middle, Distractors: 3, Seed: 7}
	if p.Selector != ByName {
		t.Fatalf("the zero Selector is %q, want the named one so existing callers are unchanged", p.Selector)
	}
	dir := t.TempDir()
	m, err := Generate(dir, p)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(m.Tree, m.File))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if n := strings.Count(string(b), "retries = 3"); n != p.Distractors+1 {
		t.Fatalf("the named fixture has %d blocks at the common retries value, want %d — T2 changed a fixture it does not own", n, p.Distractors+1)
	}
	named := regexp.MustCompile(`svc-[a-z]+`).FindAllString(string(b), -1)
	found := false
	for _, n := range named {
		if strings.Contains(m.Instruction, n) {
			found = true
		}
	}
	if !found {
		t.Fatalf("the named instruction no longer names a service:\n%s", m.Instruction)
	}
}

// TestALegacyTrialIDStillRegenerates pins a trial id recorded before the
// selector existed. It is a regression test in the strict sense: the first
// version of trialID appended the selector unconditionally, so ByName cells got
// a NEW id, and regenerating a historical cell would have refused the raw result
// collected against it — collected data invalidated with nothing failing. Found
// in review of PR #93.
//
// The pinned value is the trial_id in docs/curve/reading-02-scores/2000-early-1.score.json,
// which is committed evidence from a reading taken before T2 existed.
func TestALegacyTrialIDStillRegenerates(t *testing.T) {
	const recorded = "96bbcee067ba" // 2000-early-1, reading 2
	got := trialID(Params{ServedBytes: 2000, Position: Early, Distractors: 3, Seed: 1})
	if got != recorded {
		t.Fatalf("a cell generated before the selector existed regenerates as %s, but the reading recorded %s — every raw result from that reading would now be refused", got, recorded)
	}
	// And the relational cell of the same parameters is still a different trial.
	rel := trialID(Params{ServedBytes: 2000, Position: Early, Distractors: 3, Seed: 1, Selector: ByOddRetries})
	if rel == recorded {
		t.Fatalf("the relational cell shares the named cell's trial id %s", rel)
	}
}

// TestTheOddRetryBudgetIsNotAFixedSignature is ADR-020-T3's Enforced-by. T2
// removed the unique NAME and left a constant VALUE: every relational cell at
// every seed rendered the target at "retries = 5", so a client that had seen one
// cell could search for it in every other at a cost independent of served size —
// the single-match shortcut T2 exists to remove, surviving inside T2's own
// fixture. Found by review of PR #93.
func TestTheOddRetryBudgetIsNotAFixedSignature(t *testing.T) {
	odds, commons := map[string]bool{}, map[string]bool{}
	// Seeds 1..30 rather than a handful. Review of PR #94 found that a
	// generator accepting the FIRST odd draw — no inequality guarantee at all —
	// produces a cell with no odd block at seeds 13, 18, 23 and 24, and passes
	// at every seed a shorter sweep would have exercised. The range is the
	// assertion here as much as the clauses are.
	for seed := int64(1); seed <= 30; seed++ {
		p := Params{ServedBytes: 2500, Position: Middle, Distractors: 3, Seed: seed, Selector: ByOddRetries}
		dir := t.TempDir()
		m, err := Generate(dir, p)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		odd, common := retriesShape(t, m, p.Distractors+1)
		odds[odd], commons[common] = true, true
	}
	if len(odds) < 2 {
		t.Fatalf("the odd budget is %v at every seed — a client that has seen one cell can search for it in all of them", keys(odds))
	}
	if len(commons) < 2 {
		t.Fatalf("the common budget is %v at every seed, which identifies the odd block by elimination", keys(commons))
	}

	// The named fixture is untouched: it still renders the constant in every
	// block, which is what keeps its recorded trial ids regenerable.
	dir := t.TempDir()
	m, err := Generate(dir, Params{ServedBytes: 2500, Position: Middle, Distractors: 3, Seed: 1})
	if err != nil {
		t.Fatalf("named: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(m.Tree, m.File))
	if err != nil {
		t.Fatalf("read named fixture: %v", err)
	}
	if n := strings.Count(string(b), commonRetries); n != 4 {
		t.Fatalf("the named fixture renders %s in %d of 4 blocks — T3 reached a fixture it does not own", commonRetries, n)
	}
}

// retriesOf returns the singleton retries line, the common one, and the block
// count of a generated fixture.
// retriesShape returns the singleton retries line and the common one, and FAILS
// unless the fixture has exactly two distinct values at multiplicities 1 and
// blocks-1. The earlier version returned an empty odd value when every block
// matched and left the caller to notice; it did not, so a fixture with no odd
// block at all passed. Found by review of PR #94.
func retriesShape(t *testing.T, m Manifest, blocks int) (odd, common string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(m.Tree, m.File))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	counts := map[string]int{}
	total := 0
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "retries = ") {
			counts[l]++
			total++
		}
	}
	if total != blocks {
		t.Fatalf("want %d retries lines, got %d: %v", blocks, total, counts)
	}
	if len(counts) != 2 {
		t.Fatalf("want exactly two distinct retries values at multiplicities 1 and %d, got %v", blocks-1, counts)
	}
	for value, n := range counts {
		switch n {
		case 1:
			odd = value
		case blocks - 1:
			common = value
		default:
			t.Fatalf("want multiplicities 1 and %d, got %v", blocks-1, counts)
		}
	}
	if odd == "" || common == "" {
		t.Fatalf("no singleton retries value in %v", counts)
	}
	return odd, common
}

// keys renders a set for a failure message in a stable order.
func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestTheNamedFixtureMatchesItsGoldenBytes pins the BYTES of the named fixture,
// not its trial id.
//
// This exists because the id cannot do the job the record claimed it did.
// trialID hashes PARAMETERS — served bytes, position, distractors, seed — and
// never the rendered file, so a change to the renderer does not change the id:
// drifted bytes would silently reuse the recorded one, which is the opposite of
// a guard. Reading 2's forty-five results were collected against these bytes, so
// the bytes are what has to be pinned. Found by review of PR #94, which named
// the rationale as backwards.
//
// The digests were taken from the binary built at origin/main (e96504a) and
// confirmed identical on this head across 6 seeds x 3 positions.
func TestTheNamedFixtureMatchesItsGoldenBytes(t *testing.T) {
	golden := []struct {
		p      Params
		digest string
	}{
		{Params{ServedBytes: 2000, Position: Early, Distractors: 3, Seed: 1}, "a9504af9b15f03df2ad26bddbb0016d5"},
		{Params{ServedBytes: 20000, Position: Middle, Distractors: 3, Seed: 3}, "ce3922d276340d0136af54378719ce55"},
		{Params{ServedBytes: 200000, Position: Late, Distractors: 3, Seed: 5}, "53516277fd562c27f583a7f5a13ba230"},
	}
	for _, g := range golden {
		dir := t.TempDir()
		m, err := Generate(dir, g.p)
		if err != nil {
			t.Fatalf("%v: %v", g.p, err)
		}
		h := sha256.New()
		// The manifest is excluded on purpose: it carries the absolute -out
		// path, which differs per run and is not fixture content.
		for _, f := range []string{filepath.Join(m.Tree, m.File), filepath.Join(dir, servedName), filepath.Join(dir, answerName)} {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read %s: %v", f, err)
			}
			h.Write(b)
		}
		if got := hex.EncodeToString(h.Sum(nil))[:32]; got != g.digest {
			t.Fatalf("the named fixture for %v now hashes %s, was %s — reading 2's results were collected against the old bytes and its trial ids would NOT change to say so", g.p, got, g.digest)
		}
	}
}

// TestTheDrawIsStableAcrossThePaddingFit renders one cell at several padding
// widths and requires the budgets not to move.
//
// Generate calls render repeatedly while fitting the padding to the size cell.
// If retryPair drew from a generator that advanced with each call, the budgets
// would change between steps — and the fit loop would still converge, because
// every budget 2..8 is one digit and the rendered length does not change. So
// nothing else in the suite would notice: the cell would simply be a different
// cell from the one measured. Review of PR #94 pointed out that the task's own
// risk text claimed convergence would catch it, which is false.
func TestTheDrawIsStableAcrossThePaddingFit(t *testing.T) {
	p := Params{ServedBytes: 2500, Position: Middle, Distractors: 3, Seed: 4, Selector: ByOddRetries}
	names := serviceNames(p)
	target := targetIndex(p)
	var first []string
	for _, pad := range []int{0, 1, 7, 40, 250} {
		body, _ := render(p, names, target, pad)
		var got []string
		for _, l := range strings.Split(body, "\n") {
			if strings.HasPrefix(l, "retries = ") {
				got = append(got, l)
			}
		}
		if first == nil {
			first = got
			continue
		}
		if strings.Join(got, ",") != strings.Join(first, ",") {
			t.Fatalf("pad %d renders %v; pad 0 rendered %v — the budgets move between fitting steps, so the cell measured is not the cell generated", pad, got, first)
		}
	}
}

// TestAServedWindowNeedNotBeginAtLineOne is ADR-020-T4's Enforced-by. Every
// miss in 135 read-arm trials sits at target+2, and every cell so far serves
// from line 1, where a row count in the rendering — whose first two rows carry
// no line number — and the line number plus two are the same integer. A window
// served from line N makes them differ by N-1, which is what lets a reading
// tell the two apart.
func TestAServedWindowNeedNotBeginAtLineOne(t *testing.T) {
	base := Params{ServedBytes: 20000, Position: Late, Distractors: 3, Seed: 2, Selector: ByOddRetries}

	whole, err := Generate(t.TempDir(), base)
	if err != nil {
		t.Fatalf("whole-file cell: %v", err)
	}

	from := base
	from.ServeFrom = 120
	dir := t.TempDir()
	m, err := Generate(dir, from)
	if err != nil {
		t.Fatalf("windowed cell: %v", err)
	}
	_, a, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	served, err := os.ReadFile(filepath.Join(dir, servedName))
	if err != nil {
		t.Fatalf("read served: %v", err)
	}
	lines := strings.Split(string(served), "\n")

	// 1. The rendering's range header starts where asked. This is the line a
	//    row-counting client would mis-add; it must not start at 1.
	if len(lines) < 2 || !strings.HasPrefix(lines[1], "@@ 120-") {
		t.Fatalf("the served window's range header is %q, want it to begin at 120", lines[1])
	}
	// 2. The first numbered row is the requested line, so row 3 of the
	//    rendering is line 120 and a row count is off by 119, not by 2.
	if !strings.HasPrefix(lines[2], "  120|") && !strings.HasPrefix(lines[2], " 120|") && !strings.HasPrefix(lines[2], "120|") {
		t.Fatalf("the first served row is %q, want line 120", lines[2])
	}
	// 3. The target is inside the window — a window that excludes it would be
	//    unanswerable rather than hard.
	if a.Line < 120 {
		t.Fatalf("the answer is line %d, outside a window that starts at 120", a.Line)
	}
	// 4. The window is what the client is served, so it is what the fit loop
	//    sized: the cell reaches its byte target with FEWER file lines than the
	//    whole-file cell, not by serving the whole file and hiding the top.
	if m.ServedBytes < base.ServedBytes {
		t.Fatalf("the windowed cell served %d bytes, below its %d-byte target", m.ServedBytes, base.ServedBytes)
	}
	_ = whole

	// 5. A window past the target is refused at generate time.
	past := base
	past.ServeFrom = 100000
	_, err = Generate(t.TempDir(), past)
	if err == nil {
		t.Fatalf("a window starting past every line was generated; the client would be served no answer")
	}
	// And refused for the RIGHT reason. The first version of this test took
	// any error, and the error it took was the fit loop giving up on a byte
	// count — a symptom, not the cause — which the contract row then exposed.
	if !strings.Contains(err.Error(), "excludes the target") {
		t.Fatalf("refused, but not for excluding the target: %v", err)
	}

	// 6. And the case that the previous one does NOT reach: a window that
	//    EXISTS but starts after the target. With an early target at line ~76
	//    of a ~374-line file, a window from 200 is served in full and holds no
	//    answer. The fit-loop refusal above never fires here, so this is the
	//    only case that exercises the post-fit check — a mutant that deleted
	//    that check survived until this case was added.
	exists := Params{ServedBytes: 20000, Position: Early, Distractors: 3, Seed: 2, Selector: ByOddRetries, ServeFrom: 200}
	_, err = Generate(t.TempDir(), exists)
	if err == nil {
		t.Fatalf("a window from 200 over an early target was generated; the client would be served no answer")
	}
	if !strings.Contains(err.Error(), "excludes the target") {
		t.Fatalf("refused, but not for excluding the target: %v", err)
	}
}
