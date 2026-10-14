package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// stubClaude writes a shell script that stands in for the CLI: it answers
// --version, logs each run, applies the task's reference fix in its working
// directory, and prints stream-json lines in the shape bench parses. It never
// calls a model; that is the point.
func stubClaude(t *testing.T, dir, patch, say string) (claude, log string) {
	t.Helper()
	claude = filepath.Join(dir, "claude")
	log = filepath.Join(dir, "runs.log")
	script := `#!/bin/sh
tools=""; prev=""
for a in "$@"; do
	[ "$a" = "--version" ] && { echo "9.9.9 (Claude Code)"; exit 0; }
	[ "$prev" = "--tools" ] && tools="$a"
	prev="$a"
done
echo "$PWD" >> "` + log + `"
echo "mrw=$(command -v mrw || true)" >> "` + filepath.Join(dir, "path.log") + `"
git apply "` + patch + `" >/dev/null 2>&1
printf '{"type":"system","subtype":"init","model":"stub-model","tools":["%s"],"mcp_servers":[]}\n' "$(printf '%s' "$tools" | sed 's/,/","/g')"
printf '%s\n' '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}},{"type":"text","text":"` + say + `"}]}}'
printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"num_turns":2,"duration_ms":10,"total_cost_usd":0.25,"usage":{"input_tokens":100,"output_tokens":20,"cache_read_input_tokens":300,"cache_creation_input_tokens":40}}'
`
	if err := os.WriteFile(claude, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return claude, log
}

// stubMrw stands in for the pinned mrw build: it prints a version and the
// instructions configuration B appends.
func stubMrw(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "mrw")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n[ \"$1\" = version ] && { echo 'mrw stub'; exit 0; }\necho 'use it'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// fixtureTask writes a one-file fixture task and loads it.
func fixtureTask(t *testing.T, root string) Task {
	t.Helper()
	dir := filepath.Join(root, "fix-one")
	if err := os.MkdirAll(filepath.Join(dir, "base"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"task.json":       `{"id":"fix-one","shape":"fixture","repo":"fixture","instruction":"Change beta to gamma in notes.txt.","require":[{"file":"notes.txt","match":"(?m)^gamma$"}],"forbid":[{"file":"notes.txt","match":"(?m)^beta$"}]}`,
		"base/notes.txt":  "alpha\nbeta\ndelta\nepsilon\n",
		"reference.patch": "diff --git a/notes.txt b/notes.txt\n--- a/notes.txt\n+++ b/notes.txt\n@@ -1,4 +1,4 @@\n alpha\n-beta\n+gamma\n delta\n epsilon\n",
	}
	for p, s := range files {
		if err := os.WriteFile(filepath.Join(dir, p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	task, err := LoadTask(dir)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func runOptions(t *testing.T) (RunOptions, string) {
	t.Helper()
	needGit(t)
	work := t.TempDir()
	task := fixtureTask(t, work)
	claude, log := stubClaude(t, work, filepath.Join(task.Dir, "reference.patch"), "done")
	canary := filepath.Join(work, "canary.json")
	if err := os.WriteFile(canary, []byte(`{"pass":true,"cli_version":"9.9.9 (Claude Code)"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return RunOptions{
		Tasks:       []Task{task},
		Stage:       "test",
		Out:         filepath.Join(work, "out"),
		Models:      []string{"stub-model"},
		Configs:     []string{"A", "B"},
		Repeats:     1,
		Seed:        156,
		Claude:      claude,
		Mrw:         stubMrw(t, work),
		Bench:       "/bin/true",
		ConfigDir:   filepath.Join(work, "config"),
		Containment: "bypass",
		Canary:      canary,
		Budget:      time.Minute,
		Scratch:     filepath.Join(work, "scratch"),
	}, log
}

func receipts(t *testing.T, out string) map[string][]byte {
	t.Helper()
	got := map[string][]byte{}
	ents, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".receipt.json") {
			b, err := os.ReadFile(filepath.Join(out, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			got[e.Name()] = b
		}
	}
	return got
}

func TestTheRunnerRecordsAReceiptPerAttemptFromAStubClaude(t *testing.T) {
	opts, log := runOptions(t)
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	got := receipts(t, opts.Out)
	if len(got) != 2 {
		t.Fatalf("receipts: %d, want one per attempt (2)", len(got))
	}
	for name, b := range got {
		var r Receipt
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !r.Oracle.Solved || r.CostUSD == nil || *r.CostUSD != 0.25 || r.Usage.InputTokens != 100 || r.Usage.CacheReadTokens != 300 ||
			r.ToolCalls != 1 || r.CLIVersion != "9.9.9 (Claude Code)" || r.Model != "stub-model" || r.Diff.Files != 1 || r.Task != "fix-one" {
			t.Errorf("%s: %+v", name, r)
		}
		if r.Config == "B" && r.MrwVersion != "mrw stub" {
			t.Errorf("%s: configuration B does not record the mrw build: %+v", name, r)
		}
		if r.Config == "A" && r.MrwVersion != "" {
			t.Errorf("%s: configuration A names an mrw build: %+v", name, r)
		}
	}
	runs, _ := os.ReadFile(log)
	if strings.Count(string(runs), "\n") != 2 {
		t.Errorf("the CLI ran %q times, want 2", runs)
	}
}

func TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts(t *testing.T) {
	opts, _ := runOptions(t)
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	before := receipts(t, opts.Out)
	other := opts
	other.Repeats = 2
	err := Run(other)
	if err == nil || !strings.Contains(err.Error(), "an earlier run") {
		t.Fatalf("a different run into the same directory was not refused: %v", err)
	}
	after := receipts(t, opts.Out)
	if len(after) != len(before) {
		t.Fatalf("receipts changed: %d -> %d", len(before), len(after))
	}
	for name, b := range before {
		if !bytes.Equal(after[name], b) {
			t.Errorf("%s was rewritten", name)
		}
	}
}

func TestTheRunnerResumesWithoutRepeatingAStartedAttempt(t *testing.T) {
	opts, log := runOptions(t)
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	got := receipts(t, opts.Out)
	var names []string
	for n := range got {
		names = append(names, n)
	}
	// One attempt looks crashed (started, no receipt); the other never began.
	crashed := filepath.Join(opts.Out, names[0])
	if err := os.Remove(crashed); err != nil {
		t.Fatal(err)
	}
	fresh := strings.TrimSuffix(names[1], ".receipt.json")
	for _, suffix := range []string{".receipt.json", ".started", ".transcript.jsonl", ".stderr"} {
		_ = os.Remove(filepath.Join(opts.Out, fresh+suffix))
	}
	if err := os.Truncate(log, 0); err != nil {
		t.Fatal(err)
	}
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	runs, _ := os.ReadFile(log)
	if strings.Count(string(runs), "\n") != 1 {
		t.Fatalf("the resumed run started the CLI %d times, want 1 (the fresh attempt only)", strings.Count(string(runs), "\n"))
	}
	if _, err := os.Stat(crashed); err == nil {
		t.Error("the started attempt was run again")
	}
	if _, err := os.Stat(filepath.Join(opts.Out, fresh+".receipt.json")); err != nil {
		t.Errorf("the fresh attempt has no receipt: %v", err)
	}
}

// TestVerifyNamesAnAttemptWithNoReceipt: a stage directory verifies only when
// every planned attempt has its receipt.
func TestVerifyNamesAnAttemptWithNoReceipt(t *testing.T) {
	opts, _ := runOptions(t)
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if err := Verify(opts.Out, "test", 1, 1); err != nil {
		t.Fatalf("a complete stage did not verify: %v", err)
	}
	if err := Verify(opts.Out, "test", 1, 2); err == nil {
		t.Error("a stage verified against counts it was not planned with")
	}
	if err := os.Remove(filepath.Join(opts.Out, "fix-one.A.stub-model.1.receipt.json")); err != nil {
		t.Fatal(err)
	}
	if err := Verify(opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), "fix-one.A.stub-model.1") {
		t.Errorf("a missing receipt was not named: %v", err)
	}
}

// TestTheCanaryRunsBothConfigurationsAndRecordsItsVerdict drives RunCanary
// with a stub CLI: a clean stub passes and writes canary.json; one that names
// mrw in configuration A fails, and a stage then refuses to start on it.
func TestTheCanaryRunsBothConfigurationsAndRecordsItsVerdict(t *testing.T) {
	for _, c := range []struct {
		say  string
		pass bool
	}{{"done", true}, {"I would use mrw for this", false}} {
		opts, _ := runOptions(t)
		work := filepath.Dir(opts.Out)
		claude, _ := stubClaude(t, work, filepath.Join(opts.Tasks[0].Dir, "reference.patch"), c.say)
		opts.Claude, opts.Out = claude, filepath.Join(work, "canary-"+strings.Fields(c.say)[0])
		got, err := RunCanary(opts, opts.Tasks[0], "")
		if err != nil {
			t.Fatal(err)
		}
		if got.Pass != c.pass || got.CLIVersion != "9.9.9 (Claude Code)" {
			t.Errorf("say %q: %+v", c.say, got)
		}
		opts.Canary = filepath.Join(opts.Out, "canary.json")
		opts.Out = filepath.Join(work, "stage-"+strings.Fields(c.say)[0])
		if err := Run(opts); (err == nil) != c.pass {
			t.Errorf("say %q: a stage on this canary: %v", c.say, err)
		}
	}
}

func TestTheRunnerRefusesWithoutTheGatesZyHolds(t *testing.T) {
	opts, log := runOptions(t)
	for name, mutate := range map[string]func(*RunOptions){
		"no containment":       func(o *RunOptions) { o.Containment = "" },
		"a dedicated user":     func(o *RunOptions) { o.Containment = "user:bench" },
		"no canary":            func(o *RunOptions) { o.Canary = "" },
		"no config directory":  func(o *RunOptions) { o.ConfigDir = "" },
		"the user's own setup": func(o *RunOptions) { home, _ := os.UserHomeDir(); o.ConfigDir = filepath.Join(home, ".claude") },
	} {
		o := opts
		o.Out = filepath.Join(t.TempDir(), "out")
		mutate(&o)
		if err := Run(o); err == nil {
			t.Errorf("%s: the run started", name)
		}
	}
	if b, _ := os.ReadFile(log); len(b) != 0 {
		t.Errorf("the CLI ran although a gate refused: %q", b)
	}
	failing := filepath.Join(t.TempDir(), "canary.json")
	if err := os.WriteFile(failing, []byte(`{"pass":false,"cli_version":"9.9.9 (Claude Code)"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	o := opts
	o.Canary = failing
	if err := Run(o); err == nil {
		t.Error("a failing canary let the run start")
	}
}

// TestAnExcludedAttemptIsReplacedAndKept: a CLI that fails before its first
// tool call with a rate limit is a start error, replaced twice under ids of
// its own, every receipt kept; a stage then verifies on the planned ids.
func TestAnExcludedAttemptIsReplacedAndKept(t *testing.T) {
	opts, _ := runOptions(t)
	work := filepath.Dir(opts.Out)
	failing := filepath.Join(work, "claude-limited")
	script := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = \"--version\" ] && { echo \"9.9.9 (Claude Code)\"; exit 0; }; done\necho 'API Error: 429 rate limit reached' >&2\nexit 1\n"
	if err := os.WriteFile(failing, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	opts.Claude, opts.Configs = failing, []string{"A"}
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	got := receipts(t, opts.Out)
	for _, id := range []string{"fix-one.A.stub-model.1", "fix-one.A.stub-model.1.r1", "fix-one.A.stub-model.1.r2"} {
		var r Receipt
		if err := json.Unmarshal(got[id+".receipt.json"], &r); err != nil || r.Exclusion != ExclusionStart {
			t.Errorf("%s: %+v %v, want a start-error receipt", id, r, err)
		}
	}
	if len(got) != 3 {
		t.Errorf("receipts: %d, want the attempt and two replacements, no more", len(got))
	}
}

// TestConfigurationAHasNoMrwOnItsPath: an mrw installed beside the host's git
// does not reach configuration A, and configuration B gets the pinned build.
func TestConfigurationAHasNoMrwOnItsPath(t *testing.T) {
	opts, _ := runOptions(t)
	work := filepath.Dir(opts.Out)
	host := filepath.Join(work, "hostbin")
	if err := os.Mkdir(host, 0o755); err != nil {
		t.Fatal(err)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not on PATH")
	}
	if err := os.Symlink(git, filepath.Join(host, "git")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(host, "mrw"), []byte("#!/bin/sh\necho host mrw\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", host+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(work, "path.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(b))
	if len(lines) != 2 {
		t.Fatalf("path log: %q", b)
	}
	sawEmpty, sawPinned := false, false
	for _, l := range lines {
		switch {
		case l == "mrw=":
			sawEmpty = true
		case strings.HasSuffix(l, "/bin/mrw") && !strings.Contains(l, host):
			sawPinned = true
		default:
			t.Errorf("an attempt found %s", l)
		}
	}
	if !sawEmpty || !sawPinned {
		t.Errorf("want one attempt with no mrw (A) and one with the pinned build (B): %q", b)
	}
}

// TestTheConfigDirectoryIsJudgedByIdentity: a link to the user's own
// ~/.claude is refused like the directory itself.
func TestTheConfigDirectoryIsJudgedByIdentity(t *testing.T) {
	opts, log := runOptions(t)
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	alias := filepath.Join(t.TempDir(), "looks-dedicated")
	if err := os.Symlink(filepath.Join(home, ".claude"), alias); err != nil {
		t.Fatal(err)
	}
	opts.ConfigDir = alias
	if err := Run(opts); err == nil || !strings.Contains(err.Error(), "own ~/.claude") {
		t.Fatalf("a link to ~/.claude was accepted: %v", err)
	}
	if b, _ := os.ReadFile(log); len(b) != 0 {
		t.Errorf("the CLI ran: %q", b)
	}
}

// TestVerifyRefusesAReceiptThatIsNotItsAttempt and the hook's command: a
// corrupted receipt does not verify, and a hook path with a space in it is
// quoted in the settings an attempt runs with.
func TestVerifyRefusesAReceiptThatIsNotItsAttempt(t *testing.T) {
	opts, _ := runOptions(t)
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opts.Out, "fix-one.A.stub-model.1.receipt.json"), []byte("{\"attempt\":\"somethi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), "fix-one.A.stub-model.1") {
		t.Errorf("a corrupted receipt verified: %v", err)
	}
	scratch := t.TempDir()
	opts.Bench = "/opt/with space/bench"
	t.Setenv("USER", "bench-user")
	_, env, err := invocation(opts, attempt{ID: "x", Task: opts.Tasks[0], Model: "m", Config: Configs["B"], Repeat: 1}, scratch)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(env, "USER=bench-user") {
		t.Errorf("USER is not passed through, and the CLI cannot find the subscription without it: %v", env)
	}
	b, err := os.ReadFile(filepath.Join(scratch, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `'/opt/with space/bench' hook --log '`) {
		t.Errorf("the hook command is not quoted: %s", b)
	}
	if !strings.Contains(string(b), " --tree "+shellQuote(attemptTree(scratch))) {
		t.Errorf("the hook is not told the attempt's tree, so a write outside it is refused: %s", b)
	}
}

// TestBothConfigurationsKeepTheSystemPromptStable: every attempt runs in a
// directory of its own, and the CLI writes that directory's sections (cwd,
// memory paths, git status) into the system prompt ahead of the appended mrw
// instructions, so configuration B paid to write those instructions to the
// cache in every attempt while configuration A's tool definitions were read
// from it. Measured 2026-10-14 on CLI 2.1.300: with the flag, a second
// directory read 5,892 tokens from the cache where it read 2,431 without it.
func TestBothConfigurationsKeepTheSystemPromptStable(t *testing.T) {
	opts, _ := runOptions(t)
	for _, name := range []string{"A", "B"} {
		args, _, err := invocation(opts, attempt{ID: "x", Task: opts.Tasks[0], Model: "m", Config: Configs[name], Repeat: 1}, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(args, "--exclude-dynamic-system-prompt-sections") {
			t.Errorf("configuration %s keeps the per-directory sections in the system prompt: %v", name, args)
		}
	}
}

// TestAnAttemptLeavesNothingRunning: a background process the agent started is
// stopped with the attempt, before its tree is graded and removed (the review
// of #411, P1).
func TestAnAttemptLeavesNothingRunning(t *testing.T) {
	opts, _ := runOptions(t)
	work := filepath.Dir(opts.Out)
	pidfile := filepath.Join(work, "bg.pid")
	stub := filepath.Join(work, "claude-bg")
	script := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = \"--version\" ] && { echo \"9.9.9 (Claude Code)\"; exit 0; }; done\nsleep 300 &\necho $! > '" + pidfile + "'\n" +
		"printf '%s\\n' '{\"type\":\"system\",\"subtype\":\"init\",\"model\":\"m\",\"tools\":[],\"mcp_servers\":[]}' '{\"type\":\"result\",\"subtype\":\"success\",\"is_error\":false,\"num_turns\":1,\"total_cost_usd\":0.1,\"usage\":{}}'\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	opts.Claude, opts.Configs = stub, []string{"A"}
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(pidfile)
	if err != nil {
		t.Fatal(err)
	}
	pid := strings.TrimSpace(string(b))
	if err := exec.Command("kill", "-0", pid).Run(); err == nil {
		_ = exec.Command("kill", "-9", pid).Run()
		t.Fatalf("the attempt's background process %s outlived it", pid)
	}
}

// TestAFailingExitIsAnErrorAndAnInterruptStopsTheStage: a CLI that prints a
// success result and exits 1 is an error; a signal to bench during an attempt
// stops the stage, writes no receipt for that attempt and starts no other; and
// an excluded attempt's missing replacement fails Verify (the re-review of #411).
func TestAFailingExitIsAnErrorAndAnInterruptStopsTheStage(t *testing.T) {
	opts, _ := runOptions(t)
	work := filepath.Dir(opts.Out)
	result := `printf '%s\n' '{"type":"system","subtype":"init","model":"m","tools":[],"mcp_servers":[]}' '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash"}]}}' '{"type":"result","subtype":"success","is_error":false,"num_turns":1,"total_cost_usd":0.1,"usage":{}}'`
	stub := func(name, body string) string {
		p := filepath.Join(work, name)
		script := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = \"--version\" ] && { echo \"9.9.9 (Claude Code)\"; exit 0; }; done\n" + body + "\n"
		if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	o := opts
	o.Claude, o.Configs, o.Out = stub("claude-exit1", result+"\nexit 1"), []string{"A"}, filepath.Join(work, "exit1")
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	var r Receipt
	b, _ := os.ReadFile(filepath.Join(o.Out, "fix-one.A.stub-model.1.receipt.json"))
	if err := json.Unmarshal(b, &r); err != nil || !r.IsError || r.ExitCode != 1 {
		t.Errorf("a success result with exit 1: %+v %v, want is_error", r, err)
	}

	o = opts
	o.Out = filepath.Join(work, "interrupted")
	o.Claude = stub("claude-int", "kill -INT "+strconv.Itoa(os.Getpid())+"\nsleep 30\n"+result)
	err := Run(o)
	if err == nil || !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("an interrupt did not stop the stage: %v", err)
	}
	if got := receipts(t, o.Out); len(got) != 0 {
		t.Errorf("an interrupted stage wrote receipts: %v", got)
	}
	starts, _ := filepath.Glob(filepath.Join(o.Out, "*.started"))
	if len(starts) != 1 {
		t.Errorf("started markers: %v, want only the interrupted attempt's", starts)
	}

	o = opts
	o.Out = filepath.Join(work, "chain")
	o.Configs = []string{"A"}
	o.Claude = stub("claude-limit", "echo 'rate limit' >&2\nexit 1")
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	if err := Verify(o.Out, "test", 1, 1); err != nil {
		t.Fatalf("a complete chain did not verify: %v", err)
	}
	if err := os.Remove(filepath.Join(o.Out, "fix-one.A.stub-model.1.r2.receipt.json")); err != nil {
		t.Fatal(err)
	}
	if err := Verify(o.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), ".r2") {
		t.Errorf("a missing replacement verified: %v", err)
	}
	// A replacement that names itself but not its attempt's dimensions does
	// not verify either (the second re-review of #411).
	if err := os.WriteFile(filepath.Join(o.Out, "fix-one.A.stub-model.1.r2.receipt.json"), []byte(`{"attempt":"fix-one.A.stub-model.1.r2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(o.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), ".r2") {
		t.Errorf("a replacement with no dimensions verified: %v", err)
	}

	// A slip check that cannot run is a setup exclusion, never a count of
	// zero (the final review of #411).
	o = opts
	o.Out = filepath.Join(work, "no-backstop")
	o.Configs = []string{"B"}
	o.Claude = stub("claude-bad-ledger", `mkdir -p "$XDG_STATE_HOME/mrw/x" && echo x > "$XDG_STATE_HOME/mrw/x/seen"`+"\n"+result)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(o.Out, "fix-one.B.stub-model.1.receipt.json"))
	if err := json.Unmarshal(b, &r); err != nil || r.Exclusion != ExclusionSetup || !strings.Contains(r.Note, "slipped writes") {
		t.Errorf("a slip check that could not run: %+v %v, want a setup exclusion naming it", r, err)
	}

	// An interrupt during the canary stops it: no canary.json, no second
	// configuration.
	o = opts
	o.Out = filepath.Join(work, "canary-interrupted")
	o.Claude = stub("claude-int-canary", "kill -INT "+strconv.Itoa(os.Getpid())+"\nsleep 30\n"+result)
	if _, err := RunCanary(o, o.Tasks[0], ""); err == nil || !strings.Contains(err.Error(), "interrupted") {
		t.Errorf("an interrupted canary went on: %v", err)
	}
	if _, err := os.Stat(filepath.Join(o.Out, "canary.json")); err == nil {
		t.Error("an interrupted canary wrote canary.json")
	}
	if b, _ := filepath.Glob(filepath.Join(o.Out, "canary.B.*")); len(b) != 0 {
		t.Errorf("an interrupted canary started configuration B: %v", b)
	}
}
