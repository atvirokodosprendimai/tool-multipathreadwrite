package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Review of #751. A run that schedules several models needs a canary of each: a
// canary of one model authorises that model alone, so `--model opus --model sonnet`
// on an Opus-only canary is refused before any attempt runs.
func TestARunOfSeveralModelsNeedsACanaryOfEach(t *testing.T) {
	rig := newPinRig(t, []string{"B"}, 1)
	rig.opts.Models = []string{"stub-model", "other-model"}
	err := Run(rig.opts)
	if err == nil || !strings.Contains(err.Error(), "other-model") {
		t.Fatalf("a run of two models on one model's canary: %v, want a refusal naming the model with no canary", err)
	}
	if n := attemptsRun(rig); n != 0 {
		t.Errorf("%d attempts ran before the refusal", n)
	}
	// The pair: a second canary, of the other model, covers it.
	second := filepath.Join(filepath.Dir(rig.opts.Canary), "canary-other.json")
	body := fmt.Sprintf(`{"pass":true,"cli_version":"9.9.9 (Claude Code)","model":"other-model","mrw_sha256":%q,"mrw_instructions_sha256":""}`, rig.sha)
	if werr := os.WriteFile(second, []byte(body), 0o644); werr != nil {
		t.Fatal(werr)
	}
	rig.opts.MoreCanaries = []string{second}
	rig.opts.Out = filepath.Join(filepath.Dir(rig.opts.Out), "two-models")
	if err := Run(rig.opts); err != nil {
		t.Errorf("a run of two models on a canary of each: %v", err)
	}
}
