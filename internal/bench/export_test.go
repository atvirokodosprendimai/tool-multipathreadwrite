package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExportPairedWritesADR106Rows: the rows quality-harness ADR-106's scorer
// reads, for one model and one pair of configurations: a config line with
// every field its Decision 2 names, then one row per instance, arm and repeat
// with a retry chain collapsed to its last try (ADR-162 Decision 3).
func TestExportPairedWritesADR106Rows(t *testing.T) {
	dir := t.TempDir()
	put := func(name string, v any) {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	yes, no := true, false
	plugin := &PluginRef{Tag: "v3.8.39", Commit: strings.Repeat("a", 40)}
	put("run.json", plan{Stage: "stage5", Tasks: []string{"i1", "i2"}, Models: []string{"m1", "m2"}, Configs: []string{"A", "B", "P"}, Repeats: 1, Seed: 156,
		CLIVersion: "9.9.9 (Claude Code)", Containment: "bypass", Profile: ProfileStage5, Plugin: plugin, StatusLineSHA256: strings.Repeat("b", 64), PromptSHA256: strings.Repeat("d", 64), BudgetSeconds: 1200})
	receipt := func(id, task, model, config, status string, resolved, broke *bool, over bool, excl string) {
		put(id+".receipt.json", Receipt{Attempt: id, Stage: "stage5", Task: task, Model: model, Config: config, Repeat: 1, Status: status,
			Oracle: Verdict{HarnessResolved: resolved, BrokeTests: broke}, OverBudget: over, Exclusion: excl})
	}
	receipt("i1.A.m1.1", "i1", "m1", "A", "", nil, nil, false, ExclusionStart)
	receipt("i1.A.m1.1.r1", "i1", "m1", "A", "done", &yes, &no, false, "")
	receipt("i1.P.m1.1", "i1", "m1", "P", "done", &yes, &no, true, "")
	receipt("i1.B.m1.1", "i1", "m1", "B", "done", &yes, &no, false, "")
	receipt("i2.A.m1.1", "i2", "m1", "A", "", nil, nil, false, ExclusionSetup)
	receipt("i2.A.m1.1.r1", "i2", "m1", "A", "", nil, nil, false, ExclusionSetup)
	receipt("i2.P.m1.1", "i2", "m1", "P", "not done", &no, &yes, false, "")
	receipt("i1.A.m2.1", "i1", "m2", "A", "done", &yes, &no, false, "")
	sel := filepath.Join(dir, "selection.json")
	if err := os.WriteFile(sel, []byte(`{"rule":"the 2 ids with the lowest sha256(\"adr-162:\" + instance_id)","dataset":"SWE-bench/SWE-bench_Verified","revision":"78f4","split":"test","selected":["i1","i2"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	canary := filepath.Join(dir, "canary.json")
	if err := os.WriteFile(canary, []byte(`{"pass":true,"cli_version":"9.9.9 (Claude Code)","profile":"stage5","configs":["A","B","P"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := ExportOptions{Dirs: []string{dir}, X: "A", Y: "P", Model: "m1", ScorerSHA: strings.Repeat("c", 40), Selection: sel, Canary: canary}
	var out bytes.Buffer
	if err := ExportPaired(opts, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var head struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &head); err != nil || head.Config == nil {
		t.Fatalf("the first line is not the config: %s", lines[0])
	}
	for _, k := range []string{"dataset", "split", "revision", "selection_rule", "selection_salt", "instance_count", "instance_list_sha256", "k", "n", "model",
		"cli_version", "plugin", "scaffold", "prompt_sha256", "status_line_sha256", "max_turns", "budget_seconds", "scorer_blob_sha", "exporter", "canary"} {
		if _, ok := head.Config[k]; !ok {
			t.Errorf("the config line lacks %s: %s", k, lines[0])
		}
	}
	type row struct {
		Instance   string `json:"instance"`
		Arm        string `json:"arm"`
		Repeat     int    `json:"repeat"`
		Status     string `json:"status"`
		Resolved   bool   `json:"resolved"`
		BrokeTests bool   `json:"brokeTests"`
		Limit      bool   `json:"limit"`
		Unrun      bool   `json:"unrun"`
	}
	got := map[string]row{}
	for _, l := range lines[1:] {
		var r row
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatalf("a row is not JSON: %s", l)
		}
		got[r.Instance+"."+r.Arm] = r
	}
	want := map[string]row{
		"i1.A": {Instance: "i1", Arm: "A", Repeat: 1, Status: "done", Resolved: true},
		"i1.P": {Instance: "i1", Arm: "P", Repeat: 1, Status: "done", Resolved: true, Limit: true},
		"i2.A": {Instance: "i2", Arm: "A", Repeat: 1, Unrun: true},
		"i2.P": {Instance: "i2", Arm: "P", Repeat: 1, Status: "not done", BrokeTests: true},
	}
	if len(lines) != 1+len(want) {
		t.Errorf("%d rows, want %d (B and the other model are not exported): %s", len(lines)-1, len(want), out.String())
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("row %s = %+v, want %+v", k, got[k], w)
		}
	}
	for name, broken := range map[string]ExportOptions{
		"no scorer sha":   {Dirs: opts.Dirs, X: "A", Y: "P", Model: "m1", Selection: sel, Canary: canary},
		"no canary":       {Dirs: opts.Dirs, X: "A", Y: "P", Model: "m1", ScorerSHA: opts.ScorerSHA, Selection: sel},
		"no selection":    {Dirs: opts.Dirs, X: "A", Y: "P", Model: "m1", ScorerSHA: opts.ScorerSHA, Canary: canary},
		"a model not run": {Dirs: opts.Dirs, X: "A", Y: "P", Model: "m9", ScorerSHA: opts.ScorerSHA, Selection: sel, Canary: canary},
	} {
		if err := ExportPaired(broken, &bytes.Buffer{}); err == nil {
			t.Errorf("%s: the export was written", name)
		}
	}
	if head.Config["prompt_sha256"] != strings.Repeat("d", 64) || head.Config["prompt_sha256"] == head.Config["status_line_sha256"] {
		t.Errorf("prompt_sha256 is %v, want the run's prompt template digest, not the status sentence's", head.Config["prompt_sha256"])
	}
	// Blocks that ran another budget are not described by one config line, and
	// the same attempt in two directories is refused (the review of T2).
	other := t.TempDir()
	b, err := json.Marshal(plan{Stage: "stage5", Tasks: []string{"i3"}, Models: []string{"m1"}, Configs: []string{"A", "P"}, Repeats: 1, Seed: 156,
		CLIVersion: "9.9.9 (Claude Code)", Containment: "bypass", Profile: ProfileStage5, Plugin: plugin, StatusLineSHA256: strings.Repeat("b", 64), PromptSHA256: strings.Repeat("d", 64), BudgetSeconds: 600})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "run.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, dirs := range map[string][]string{"another budget": {dir, other}, "the same attempts twice": {dir, dir}} {
		o := opts
		o.Dirs = dirs
		if err := ExportPaired(o, &bytes.Buffer{}); err == nil {
			t.Errorf("%s: the export was written", name)
		}
	}
}
