package curve

import (
	"os"
	"path/filepath"
	"regexp"
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
