package curve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScoreTrialLoadsABodyAtPath is ADR-060 T6: ScoreTrial must call
// LoadBodyFiles after parse, on the copied fixture, before Apply. CLI and MCP
// already do. Skipping the load lets create body=@missing apply an empty file
// and score Miss — a client's missing path must be RefusedParse, like Parse.
func TestScoreTrialLoadsABodyAtPath(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		_, m, _ := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 21})
		dir := filepath.Dir(m.Tree)
		r := ok(m)
		r.Plan = "@@ dest.txt 0 create body=@missing.txt\n"
		s, err := ScoreTrial(dir, r)
		if err != nil {
			t.Fatalf("a missing body=@ path is an Outcome, not harness breakage: %v", err)
		}
		if s.Outcome != RefusedParse {
			t.Fatalf("create body=@missing.txt scored %s, want %s — skipping LoadBodyFiles applies an empty create", s.Outcome, RefusedParse)
		}
		if !strings.Contains(s.Reason, "missing.txt") {
			t.Fatalf("refusal does not name the path:\n%s", s.Reason)
		}
	})
	t.Run("present", func(t *testing.T) {
		_, m, _ := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 22})
		dir := filepath.Dir(m.Tree)
		if err := os.WriteFile(filepath.Join(m.Tree, "src.txt"), []byte("alpha\nbeta\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		r := ok(m)
		r.Plan = "@@ dest.txt 0 create body=@src.txt\n"
		s, err := ScoreTrial(dir, r)
		if err != nil {
			t.Fatalf("score: %v", err)
		}
		if s.Outcome == RefusedParse {
			t.Fatalf("create body=@src.txt was RefusedParse (%s); a present file must load", s.Reason)
		}
		if s.Outcome != Miss {
			t.Fatalf("a create of another file scored %s, want %s", s.Outcome, Miss)
		}
		if len(s.Touched) != 1 || s.Touched[0] != "dest.txt" {
			t.Fatalf("touched=%v, want [dest.txt]", s.Touched)
		}
		if _, err := os.Stat(filepath.Join(m.Tree, "dest.txt")); !os.IsNotExist(err) {
			t.Fatalf("scoring wrote into the fixture tree itself: %v", err)
		}
	})
}
