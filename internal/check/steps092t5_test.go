package check

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// ADR-092 T5 (the stress round of 2026-09-28). A steps block nobody asked for
// does not stop a load: a typo in "steps" refused every write in the tree, a
// key the project may never use. Asked for, the same block is refused.
func TestAMalformedStepsBlockStillLoads(t *testing.T) {
	for _, doc := range []string{
		`{"check":"true","steps":{"my step":"go vet ./..."}}`,
		`{"check":"true","steps":{"a":1}}`,
		`{"check":"true","steps":["a"]}`,
		`{"check":"true","steps":{"e":""}}`,
	} {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(root)
		if err != nil || cfg.Check != "true" {
			t.Errorf("%s: a malformed steps block stopped the load: %v", doc, err)
			continue
		}
		if _, err := cfg.StepCommands(); err == nil {
			t.Errorf("%s: asked for, a malformed steps block was accepted", doc)
		}
	}
}

// ADR-092 T5. A step runs with MRW_STEP_DEPTH one higher than mrw's own, so a
// step that runs mrw with steps again is one level deeper.
func TestAStepRunsOneLevelDeeper(t *testing.T) {
	needShell(t)
	tempDirForLogs(t)
	root := t.TempDir()
	t.Setenv("MRW_STEP_DEPTH", "3")
	if got := StepDepth(); got != 3 {
		t.Errorf("StepDepth() = %d under MRW_STEP_DEPTH=3", got)
	}
	res := RunSteps(context.Background(), root, Config{}, []Step{{Name: "d", Command: `printf "%s" "$MRW_STEP_DEPTH" > depth`}})
	b, _ := os.ReadFile(filepath.Join(root, "depth"))
	if len(res.Steps) != 1 || res.Steps[0].Status != StepPass || string(b) != "4" {
		t.Errorf("the step saw MRW_STEP_DEPTH=%q, want 4: %+v", b, res.Steps)
	}
}
