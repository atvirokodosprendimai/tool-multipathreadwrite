package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The tests of ADR-163 T2: configuration W, the declared check and the caps.

const (
	realChecks    = "../../bench/swe-bench/checks.json"
	realSelection = "../../bench/swe-bench/selection-adr-162.json"
	workLineText  = "Run the repository's check before you call the task done."
)

func hexSHA(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// poolRepos are the repositories of ADR-162's 191 instances, read from their
// ids: the owner before "__", the name before the last "-".
func poolRepos(t *testing.T) []string {
	t.Helper()
	var sel struct{ Selected []string }
	if err := readJSON(realSelection, &sel); err != nil {
		t.Fatal(err)
	}
	var repos []string
	for _, id := range sel.Selected {
		owner, rest, ok := strings.Cut(id, "__")
		i := strings.LastIndex(rest, "-")
		if !ok || i < 0 {
			t.Fatalf("instance id %q is not owner__name-number", id)
		}
		if r := owner + "/" + rest[:i]; !slices.Contains(repos, r) {
			repos = append(repos, r)
		}
	}
	return repos
}

// tableWith is checks.json with one repository's entry edited; edit returns
// false to drop the entry.
func tableWith(t *testing.T, repo string, edit func(e map[string]any) bool) string {
	t.Helper()
	b, err := os.ReadFile(realChecks)
	if err != nil {
		t.Fatal(err)
	}
	var entries []map[string]any
	if err := json.Unmarshal(b, &entries); err != nil {
		t.Fatal(err)
	}
	var kept []map[string]any
	for _, e := range entries {
		if e["repo"] != repo || edit(e) {
			kept = append(kept, e)
		}
	}
	out, err := json.Marshal(kept)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "checks.json")
	if err := os.WriteFile(p, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func set(key string, v any) func(map[string]any) bool {
	return func(e map[string]any) bool { e[key] = v; return true }
}

// checkRig is the stage6 rig with configurations A, P and W, a plugin, a work
// line and a checks table that names the fixture's repository.
func checkRig(t *testing.T) *stage6Rig {
	t.Helper()
	g := newStage6Rig(t)
	dir := t.TempDir()
	put(t, filepath.Join(dir, "work-line.txt"), workLineText+"\n", 0o644)
	put(t, filepath.Join(dir, "checks.json"), `[{"repo":"fixture","command":"pytest","source":"setup.cfg:1@`+strings.Repeat("a", 40)+`"}]`+"\n", 0o644)
	g.opts.Configs, g.opts.PluginDir = []string{"A", "P", "W"}, pluginRepo(t, true)
	g.opts.WorkLine, g.opts.Checks = filepath.Join(dir, "work-line.txt"), filepath.Join(dir, "checks.json")
	if err := resolveProfile(&g.opts); err != nil {
		t.Fatal(err)
	}
	return g
}

// TestConfigurationWIsPPlusTheWorkLine: W's arguments are P's; its instruction
// is P's plus the line, whose sha is in run.json; A and P carry no line; W
// without --work-line or without stage6 is refused; the canary refuses the
// line in P (ADR-163 Decision 3).
func TestConfigurationWIsPPlusTheWorkLine(t *testing.T) {
	g := checkRig(t)
	o := g.opts
	for _, status := range []bool{false, true} {
		o.StatusLine = status
		argsOf := func(name string) (rest []string, instruction string) {
			scratch := t.TempDir()
			args, _, err := invocation(o, attempt{ID: "x", Task: o.Tasks[0], Model: "m", Config: Configs[name], Repeat: 1}, scratch)
			if err != nil {
				t.Fatal(err)
			}
			args = scrub(args, scratch)
			i := slices.Index(args, "-p")
			return slices.Delete(slices.Clone(args), i, i+2), args[i+1]
		}
		_, aText := argsOf("A")
		pRest, pText := argsOf("P")
		wRest, wText := argsOf("W")
		if !slices.Equal(wRest, pRest) {
			t.Errorf("status %v: W's arguments are not P's:\nP %v\nW %v", status, pRest, wRest)
		}
		if strings.Contains(aText, workLineText) || strings.Contains(pText, workLineText) {
			t.Errorf("status %v: A or P carries the line", status)
		}
		if strings.Count(wText, workLineText) != 1 || strings.Replace(wText, "\n\n"+workLineText, "", 1) != pText {
			t.Errorf("status %v: W's instruction is not P's plus the line:\nP %q\nW %q", status, pText, wText)
		}
	}

	for name, mutate := range map[string]func(*RunOptions){
		"without --work-line": func(x *RunOptions) { x.WorkLine = "" },
		"under stage5":        func(x *RunOptions) { x.Profile = ProfileStage5 },
		"under no profile":    func(x *RunOptions) { x.Profile = "" },
	} {
		x := g.opts
		mutate(&x)
		if _, err := gates(x); err == nil {
			t.Errorf("W ran %s", name)
		}
	}
	x := g.opts
	x.Configs = []string{"A", "P"}
	if _, err := gates(x); err != nil {
		t.Errorf("a stage without W, with the line given, was refused: %v", err)
	}
	if _, err := gates(g.opts); err != nil {
		t.Errorf("A, P and W under stage6 with the line were refused: %v", err)
	}

	// The canary reads the instruction each configuration is given.
	instr := func(c string) string { return instructionFor(g.opts, Configs[c], g.opts.Tasks[0]) }
	for _, c := range []struct {
		config, text string
		clean        bool
	}{{"W", instr("W"), true}, {"P", instr("P"), true}, {"A", instr("A"), true}, {"P", instr("W"), false}, {"W", instr("P"), false}, {"A", instr("W"), false}} {
		if p := CheckWorkLine(Configs[c.config], c.text, g.opts.workLine); (len(p) == 0) != c.clean {
			t.Errorf("the canary of %s on an instruction with the line=%v: %v", c.config, strings.Contains(c.text, workLineText), p)
		}
	}

	// run.json and the receipts.
	g.claude("")
	canary := `{"pass":true,"cli_version":"9.9.9","profile":"stage6","configs":["A","P","W"],"plugin":` + mustJSON(t, g.opts.plugin) + `}`
	put(t, g.opts.Canary, canary, 0o644)
	if err := Run(g.opts); err != nil {
		t.Fatal(err)
	}
	var p plan
	b, err := os.ReadFile(filepath.Join(g.opts.Out, "run.json"))
	if err != nil || json.Unmarshal(b, &p) != nil {
		t.Fatalf("run.json: %v %s", err, b)
	}
	line, _ := os.ReadFile(g.opts.WorkLine)
	if p.WorkLineSHA256 != hexSHA(line) || !slices.Equal(p.Configs, []string{"A", "P", "W"}) {
		t.Errorf("run.json names work_line_sha256 %q for configs %v, want %s", p.WorkLineSHA256, p.Configs, hexSHA(line))
	}
	var r Receipt
	rb, err := os.ReadFile(filepath.Join(g.opts.Out, g.opts.Tasks[0].ID+".W.stub-model.1.receipt.json"))
	if err != nil || json.Unmarshal(rb, &r) != nil || r.Config != "W" || r.Plugin == nil || *r.Plugin != *g.opts.plugin {
		t.Errorf("W's receipt: %v %+v", err, r)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestTheDeclaredCheckIsWrittenBeforeTheBaseCommitInEveryArm: under stage6, A,
// P and W trees hold the same .quality-harness.json in the base commit and not
// in the diff; without the profile no file is written (ADR-163 Decision 2).
func TestTheDeclaredCheckIsWrittenBeforeTheBaseCommitInEveryArm(t *testing.T) {
	g := checkRig(t)
	log := filepath.Join(g.work, "base.log")
	// What the CLI finds when it starts: the file in HEAD, and a clean status.
	g.claude("{ git show HEAD:.quality-harness.json 2>&1; git status --porcelain; test -e .quality-harness.json || echo ABSENT; echo --; } >> " + shellQuote(log) + "; " + g.applyFix())
	blocks := func() []string {
		b, _ := os.ReadFile(log)
		if len(b) == 0 {
			return nil
		}
		return strings.Split(strings.TrimSuffix(string(b), "--\n"), "--\n")
	}
	for _, name := range []string{"A", "P", "W"} {
		a := g.attempt("check." + name)
		a.Config = Configs[name]
		if r := runAttempt(g.opts, a, 0, "9.9.9", ""); r.Exclusion != "" {
			t.Fatalf("%s: %s %s", name, r.Exclusion, r.Note)
		}
	}
	got := blocks()
	if len(got) != 3 {
		t.Fatalf("%d logged starts, want 3: %q", len(got), got)
	}
	for i, b := range got {
		if b != `{"check":"pytest"}`+"\n" {
			t.Errorf("arm %d found %q in its base commit and tree, want exactly the declared check and a clean status", i, b)
		}
	}

	// Without the profile, and under stage5, nothing is written.
	for _, profile := range []string{"", ProfileStage5} {
		if err := os.Remove(log); err != nil {
			t.Fatal(err)
		}
		o := g.opts
		o.Profile = profile
		if r := runAttempt(o, g.attempt("plain."+profile), 0, "9.9.9", ""); r.Exclusion != "" {
			t.Fatalf("profile %q: %s %s", profile, r.Exclusion, r.Note)
		}
		if b := blocks(); len(b) != 1 || !strings.Contains(b[0], "ABSENT") || strings.Contains(b[0], `"check"`) {
			t.Errorf("profile %q wrote the check file: %q", profile, b)
		}
	}

	// The table is looked up by the task's repository, and a SWE-bench task
	// the table does not name is refused where the table is read.
	real, err := LoadChecks(realChecks, []string{"django/django"})
	if err != nil {
		t.Fatal(err)
	}
	o := g.opts
	o.checks = real
	if got := o.declaredFor(Task{Repo: "django/django"}); got != "cd tests && python runtests.py" {
		t.Errorf("django's declared check is %q", got)
	}
	if got := o.declaredFor(Task{Repo: "not/pooled"}); got != "" {
		t.Errorf("a repository the table does not name got %q", got)
	}
	o.Profile = ProfileStage5
	if got := o.declaredFor(Task{Repo: "django/django"}); got != "" {
		t.Errorf("stage5 got the declared check %q", got)
	}
}

// TestTheChecksTableNamesARepositorySourceForEveryPoolRepository: every
// repository of selection-adr-162.json has a command and a source; a missing
// repository, a missing source, or a command with a path or -k argument is
// refused (ADR-163 Decision 2).
func TestTheChecksTableNamesARepositorySourceForEveryPoolRepository(t *testing.T) {
	repos := poolRepos(t)
	if len(repos) != 11 {
		t.Fatalf("the pool names %d repositories, want 11: %v", len(repos), repos)
	}
	c, err := LoadChecks(realChecks, repos)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(realChecks)
	if err != nil || c.SHA256 != hexSHA(raw) {
		t.Errorf("the table's sha256 is %q, want the file's %s (%v)", c.SHA256, hexSHA(raw), err)
	}
	for _, r := range repos {
		if e := c.byRepo[r]; e.Command == "" || e.Source == "" {
			t.Errorf("%s has command %q, source %q", r, e.Command, e.Source)
		}
	}
	for name, tc := range map[string]struct {
		repo string
		edit func(map[string]any) bool
	}{
		"a pool repository the table does not name": {"django/django", func(map[string]any) bool { return false }},
		"an entry without a source":                 {"sympy/sympy", set("source", "")},
		"a source with no line or commit":           {"sympy/sympy", set("source", "bin/test")},
		"a command naming a file":                   {"astropy/astropy", set("command", "pytest tests/test_x.py")},
		"a command with -k":                         {"astropy/astropy", set("command", "pytest -k foo")},
		"a command naming a test":                   {"astropy/astropy", set("command", "pytest x::y")},
		"a quoted -k":                               {"astropy/astropy", set("command", `pytest "-k" foo`)},
		"options set before the runner":             {"astropy/astropy", set("command", "PYTEST_ADDOPTS=-kfoo pytest")},
		"a command naming a test file":              {"astropy/astropy", set("command", "pytest test_x.py")},
	} {
		if _, err := LoadChecks(tableWith(t, tc.repo, tc.edit), repos); err == nil || !strings.Contains(err.Error(), tc.repo) {
			t.Errorf("%s: %v, want a refusal naming %s", name, err, tc.repo)
		}
	}
	// The sibling that must pass: a runner with flags and no path.
	if _, err := LoadChecks(tableWith(t, "astropy/astropy", set("command", "python -X dev -m pytest --durations 25")), repos); err != nil {
		t.Errorf("a whole runner with flags was refused: %v", err)
	}
}

// TestMaxTurnsIsTheStagesAndTheRowsCarryIt: --max-turns 90 makes 61 to 90
// turns within budget and is in run.json, the rows and scaffold; without the
// flag it is 60; two blocks with different checks_sha256, work_line_sha256 or
// max_turns are refused by the export (ADR-163 Decision 3).
func TestMaxTurnsIsTheStagesAndTheRowsCarryIt(t *testing.T) {
	opts, _ := runOptions(t)
	opts.Configs = []string{"A"}
	script, err := os.ReadFile(opts.Claude)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(opts.Claude, bytes.Replace(script, []byte(`"num_turns":2`), []byte(`"num_turns":75`), 1), 0o755); err != nil {
		t.Fatal(err)
	}
	over := func(maxTurns int, out string) (bool, plan) {
		o := opts
		o.MaxTurns, o.Out = maxTurns, filepath.Join(filepath.Dir(opts.Out), out)
		if err := Run(o); err != nil {
			t.Fatal(err)
		}
		var r Receipt
		var p plan
		for _, f := range []struct {
			name string
			v    any
		}{{o.Tasks[0].ID + ".A.stub-model.1.receipt.json", &r}, {"run.json", &p}} {
			b, err := os.ReadFile(filepath.Join(o.Out, f.name))
			if err != nil || json.Unmarshal(b, f.v) != nil {
				t.Fatalf("%s: %v", f.name, err)
			}
		}
		return r.OverBudget, p
	}
	if o, p := over(90, "cap-90"); o || p.MaxTurns != 90 {
		t.Errorf("--max-turns 90: 75 turns over budget %v, run.json max_turns %d", o, p.MaxTurns)
	}
	if o, p := over(75, "cap-75"); o || p.MaxTurns != 75 {
		t.Errorf("--max-turns 75: 75 turns over budget %v, run.json max_turns %d", o, p.MaxTurns)
	}
	if o, p := over(70, "cap-70"); !o || p.MaxTurns != 70 {
		t.Errorf("--max-turns 70: 75 turns over budget %v, run.json max_turns %d", o, p.MaxTurns)
	}
	// Without the flag the cap is 60, and a stage without stage6 writes no
	// new key, so Stage 5's run.json is what it was.
	if o, p := over(0, "cap-default"); !o || p.MaxTurns != 0 {
		t.Errorf("no --max-turns: 75 turns over budget %v, run.json max_turns %d, want over budget and no key", o, p.MaxTurns)
	}
	g := checkRig(t)
	g.claude("")
	put(t, g.opts.Canary, `{"pass":true,"cli_version":"9.9.9","profile":"stage6","configs":["A","P","W"],"plugin":`+mustJSON(t, g.opts.plugin)+`}`, 0o644)
	if err := Run(g.opts); err != nil {
		t.Fatal(err)
	}
	var p6 plan
	if b, err := os.ReadFile(filepath.Join(g.opts.Out, "run.json")); err != nil || json.Unmarshal(b, &p6) != nil || p6.MaxTurns != 60 {
		t.Errorf("a stage6 run.json without the flag: max_turns %d (%v), want 60", p6.MaxTurns, err)
	}
	if p6.ChecksSHA256 != hexSHA(mustRead(t, g.opts.Checks)) {
		t.Errorf("run.json checks_sha256 is %q, want the table's", p6.ChecksSHA256)
	}

	// The export.
	dir := t.TempDir()
	plugin := &PluginRef{Tag: "v3.8.39", Commit: strings.Repeat("a", 40)}
	block := func(name string, edit func(*plan)) string {
		p := plan{Stage: "stage6", Tasks: []string{"i1"}, Models: []string{"m1"}, Configs: []string{"A", "W"}, Repeats: 1, Seed: 163,
			CLIVersion: "9.9.9 (Claude Code)", Containment: "bypass", Profile: ProfileStage6, Plugin: plugin,
			StatusLineSHA256: strings.Repeat("b", 64), PromptSHA256: strings.Repeat("d", 64), BudgetSeconds: 1200,
			MaxTurns: 90, ChecksSHA256: strings.Repeat("c", 64), WorkLineSHA256: strings.Repeat("e", 64)}
		edit(&p)
		d := filepath.Join(dir, name)
		put(t, filepath.Join(d, "run.json"), mustJSON(t, p), 0o644)
		return d
	}
	sel := filepath.Join(dir, "selection.json")
	put(t, sel, `{"rule":"the 1 ids with the lowest sha256(\"adr-163:\" + instance_id)","dataset":"SWE-bench/SWE-bench_Verified","revision":"78f4","split":"test","selected":["i1"]}`, 0o644)
	can := filepath.Join(dir, "canary.json")
	put(t, can, `{"pass":true}`, 0o644)
	export := func(dirs ...string) (map[string]any, error) {
		var out bytes.Buffer
		err := ExportPaired(ExportOptions{Dirs: dirs, X: "A", Y: "W", Model: "m1", ScorerSHA: strings.Repeat("c", 40), Selection: sel, Canary: can}, &out)
		var head struct {
			Config map[string]any `json:"config"`
		}
		if err == nil {
			err = json.Unmarshal([]byte(strings.SplitN(out.String(), "\n", 2)[0]), &head)
		}
		return head.Config, err
	}
	same := func(*plan) {}
	cfg, err := export(block("one", same), block("two", same))
	if err != nil {
		t.Fatal(err)
	}
	scaffold, _ := cfg["scaffold"].(map[string]any)
	if cfg["max_turns"] != float64(90) || scaffold["max_turns"] != float64(90) || scaffold["checks_sha256"] != strings.Repeat("c", 64) || scaffold["work_line_sha256"] != strings.Repeat("e", 64) {
		t.Errorf("the config line does not carry the caps and the two digests in max_turns and scaffold: %v", cfg)
	}
	for name, edit := range map[string]func(*plan){
		"another max_turns":        func(p *plan) { p.MaxTurns = 60 },
		"another checks_sha256":    func(p *plan) { p.ChecksSHA256 = strings.Repeat("f", 64) },
		"another work_line_sha256": func(p *plan) { p.WorkLineSHA256 = strings.Repeat("f", 64) },
	} {
		if _, err := export(block("one-"+strings.ReplaceAll(name, " ", "-"), same), block("two-"+strings.ReplaceAll(name, " ", "-"), func(p *plan) { edit(p) })); err == nil {
			t.Errorf("blocks with %s were exported as one", name)
		}
	}
	// A run.json from before this task has no caps keys: 60 and an unchanged scaffold.
	old, err := export(block("old", func(p *plan) { p.MaxTurns, p.ChecksSHA256, p.WorkLineSHA256 = 0, "", "" }))
	if err != nil {
		t.Fatal(err)
	}
	if sc, _ := old["scaffold"].(map[string]any); old["max_turns"] != float64(60) || len(sc) != 5 {
		t.Errorf("a Stage 5 run.json exports max_turns %v and scaffold %v, want 60 and the five keys it always had", old["max_turns"], sc)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
