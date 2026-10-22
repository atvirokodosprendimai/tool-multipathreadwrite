package curve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-167 T2 through the scorer: its ledger holds m.File alone, so a from=
// source in that file copies and any other source is refused.
func TestAFromCopyThroughTheCurveScorer(t *testing.T) {
	_, m, a := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 31})
	dir := filepath.Dir(m.Tree)
	n := len(lines(t, m))
	if err := os.WriteFile(filepath.Join(m.Tree, "other.txt"), []byte("one\ntwo\nthree\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	score := func(plan string) Score {
		t.Helper()
		r := ok(m)
		r.Plan = plan
		s, err := ScoreTrial(dir, r)
		if err != nil {
			t.Fatalf("score %q: %v", plan, err)
		}
		return s
	}

	t.Run("a source in the scored file copies", func(t *testing.T) {
		s := score(fmt.Sprintf("@@ %s %d insert-after from=%s:%d\n", m.File, a.Line, m.File, a.Line))
		if s.Outcome != Miss || len(s.Changed) == 0 {
			t.Errorf("a copy within %s scored %s (%s), want it applied (a Miss: it adds a line)", m.File, s.Outcome, s.Reason)
		}
	})
	t.Run("a create, $ and A,+N each land", func(t *testing.T) {
		for _, spec := range []string{
			fmt.Sprintf("%s:1-2", m.File),
			fmt.Sprintf("%s:$", m.File),
			fmt.Sprintf("%s:%d,+1", m.File, n-1),
		} {
			s := score("@@ copy.txt - create from=" + spec + "\n")
			if s.Outcome == RefusedParse || s.Outcome == RefusedApply || len(s.Touched) != 1 || s.Touched[0] != "copy.txt" {
				t.Errorf("create from=%s scored %s (%s) touched=%v, want copy.txt created", spec, s.Outcome, s.Reason, s.Touched)
			}
		}
	})
	t.Run("a relative end is honoured, not dropped", func(t *testing.T) {
		// n,+1 runs past the last line: refused when the builder carries the
		// relative end, and a one-line copy of line n when it drops it.
		s := score(fmt.Sprintf("@@ copy.txt - create from=%s:%d,+1\n", m.File, n))
		if s.Outcome != RefusedApply {
			t.Errorf("from=%s:%d,+1 scored %s (%s), want RefusedApply: the span runs past the last line", m.File, n, s.Outcome, s.Reason)
		}
	})
	t.Run("a source in another file is refused", func(t *testing.T) {
		s := score(fmt.Sprintf("@@ %s %d insert-after from=other.txt:1-2\n", m.File, a.Line))
		if s.Outcome != RefusedApply || !strings.Contains(s.Reason, "other.txt") {
			t.Errorf("a copy from another file scored %s (%s), want RefusedApply naming it", s.Outcome, s.Reason)
		}
	})
}
