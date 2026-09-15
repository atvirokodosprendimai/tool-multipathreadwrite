package curve

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestScoreTrialHitsAReplaceFromBodyAtPath is ADR-060 T7: a replace whose
// body is the planted line, loaded from body=@, must Hit. Today validate
// refuses the empty Body before Load, so the outcome is RefusedParse.
func TestScoreTrialHitsAReplaceFromBodyAtPath(t *testing.T) {
	_, m, a := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 23})
	dir := filepath.Dir(m.Tree)
	if err := os.WriteFile(filepath.Join(m.Tree, "payload.txt"), []byte("timeout = 45\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := ok(m)
	r.Plan = fmt.Sprintf("@@ %s %d replace body=@payload.txt\n", m.File, a.Line)
	s, err := ScoreTrial(dir, r)
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	if s.Outcome != Hit {
		t.Fatalf("replace body=@payload.txt scored %s (%s), want %s — parse must accept empty Body while BodyFile is set", s.Outcome, s.Reason, Hit)
	}
}
