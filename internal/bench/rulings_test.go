package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Zy's ruling of 2026-10-20 on Stage 5 block 2: an attempt the machine ended
// (the Mac asleep, the API unreachable) is excluded as infrastructure by a
// committed exclusions.json beside the blocks, while its receipt stays as the
// runner wrote it. The look counts it toward the 5% cap and drops its pair;
// a ruling that names no attempt, or one the runner already excluded, is
// refused rather than ignored.
func TestARulingExcludesAnAttemptTheMachineEnded(t *testing.T) {
	stage := t.TempDir()
	both := map[string][]string{"m": {"A", "B"}}
	solved := func(id, m, c string) outcome {
		return outcome{cost: 1, solved: true, excluded: id == "i1-39" && c == "B"}
	}
	writeBlock(t, stage, 1, ids(1, 40), []string{"m"}, both, solved)
	rule := func(m map[string]string) {
		t.Helper()
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stage, "exclusions.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The fixture's own setup exclusion is the baseline the ruling adds to.
	base, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	b := rowOf(t, base, "m", "B").Result

	// One ruled attempt of 40: one more excluded, one fewer pair, and the look
	// still eligible.
	rule(map[string]string{"i1-00.A.m.1": "the Mac slept and the API was unreachable"})
	r, err := Look(stage)
	if err != nil {
		t.Fatal(err)
	}
	if row := rowOf(t, r, "m", "B"); !row.Eligible || row.Result.Excluded != b.Excluded+1 || row.Result.Tasks != b.Tasks-1 {
		t.Errorf("one ruled attempt: eligible %v, excluded %d, paired %d; want eligible, %d, %d", row.Eligible, row.Result.Excluded, row.Result.Tasks, b.Excluded+1, b.Tasks-1)
	}
	desc, err := LookDescriptive(stage, r, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range desc {
		if d.Comparison == "B" && d.Excluded != b.Excluded+1 {
			t.Errorf("the descriptive row counts %d excluded, want %d", d.Excluded, b.Excluded+1)
		}
	}

	// Three of 40 is over the 5% cap: the look is not eligible.
	rule(map[string]string{"i1-00.A.m.1": "x", "i1-01.A.m.1": "x", "i1-02.A.m.1": "x"})
	if r, err = Look(stage); err != nil {
		t.Fatal(err)
	}
	if rowOf(t, r, "m", "B").Eligible {
		t.Error("three ruled attempts of 40 left the look eligible")
	}

	// A ruling naming no attempt, and one naming an attempt the runner already
	// excluded, are refused by name.
	for name, m := range map[string]map[string]string{
		"no such attempt":  {"i1-00.A.m.9": "x"},
		"already excluded": {"i1-39.B.m.1": "x"},
	} {
		rule(m)
		_, err := Look(stage)
		if err == nil {
			t.Errorf("%s: the look accepted the ruling", name)
			continue
		}
		for id := range m {
			if !strings.Contains(err.Error(), id) {
				t.Errorf("%s: the refusal does not name %s: %v", name, id, err)
			}
		}
	}
}
