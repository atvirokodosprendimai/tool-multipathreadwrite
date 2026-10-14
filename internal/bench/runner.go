package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc"
)

// Config is one of the two configurations ADR-156 Decision 1 compares.
type Config struct {
	Name  string
	Tools []string
	Mrw   bool // a pinned mrw on PATH, its instructions appended, the hook installed
}

// Configs are A (the baseline) and B (mrw forced). Task, Agent, WebFetch and
// WebSearch are offered to neither.
var Configs = map[string]Config{
	"A": {Name: "A", Tools: []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep"}},
	"B": {Name: "B", Tools: []string{"Bash", "Glob"}, Mrw: true},
}

// The registered budget (BACKLOG, "Pre-registered for ADR-156"): an attempt
// past either is unsolved, its cost counted.
const (
	DefaultBudget = 20 * time.Minute
	MaxTurns      = 60
)

// Exclusions the registration admits; no other reason leaves an attempt out.
const (
	ExclusionStart = "start-error" // ended before its first tool call: auth, rate limit, CLI start
	ExclusionSetup = "setup"       // bench could not build the tree or grade it
)

// Receipt is one attempt's record, written once and never replaced.
type Receipt struct {
	Attempt       string   `json:"attempt"`
	Stage         string   `json:"stage"`
	Task          string   `json:"task"`
	Shape         string   `json:"shape"`
	Source        string   `json:"source"` // "own" or "swe-bench"
	Model         string   `json:"model"`
	Config        string   `json:"config"`
	Repeat        int      `json:"repeat"`
	Order         int      `json:"order"`
	Started       string   `json:"started"`
	WallMS        int64    `json:"wall_ms"`
	CLIVersion    string   `json:"cli_version"`
	MrwVersion    string   `json:"mrw_version,omitempty"`
	ExitCode      int      `json:"exit_code"`
	TimedOut      bool     `json:"timed_out,omitempty"`
	OverBudget    bool     `json:"over_budget,omitempty"`
	IsError       bool     `json:"is_error"`
	NumTurns      int      `json:"num_turns"`
	ToolCalls     int      `json:"tool_calls"`
	CostUSD       *float64 `json:"total_cost_usd"`
	Usage         Usage    `json:"usage"`
	InitTools     []string `json:"init_tools"`
	InitMCP       []string `json:"init_mcp_servers"`
	Oracle        Verdict  `json:"oracle"`
	SilentMisEdit bool     `json:"silent_mis_edit"`
	Diff          DiffStat `json:"diff"`
	HookRefused   int      `json:"hook_refused,omitempty"`
	HookReads     int      `json:"hook_reads,omitempty"`
	SlippedWrites []string `json:"slipped_writes,omitempty"`
	Exclusion     string   `json:"exclusion,omitempty"`
	Note          string   `json:"note,omitempty"`

	interrupted bool // stopped by a signal to bench: no receipt is written for it
}

// DiffStat is what the attempt changed, counted against its base.
type DiffStat struct {
	Files      int `json:"files"`
	Insertions int `json:"insertions"`
	Deletions  int `json:"deletions"`
}

// RunOptions is one stage's plan. Containment, ConfigDir and Canary are the
// gates ADR-156 leaves to Zy: Run starts nothing until each is given.
type RunOptions struct {
	Tasks       []Task
	Stage       string
	Out         string
	Models      []string
	Configs     []string
	Repeats     int
	Seed        int64
	Claude      string // the CLI, by absolute path
	Mrw         string // the pinned mrw build configuration B gets
	Bench       string // this binary, which configuration B's hook runs
	ConfigDir   string // the dedicated CLAUDE_CONFIG_DIR
	Containment string // "bypass"; "user:<name>" is Zy's other choice, not built yet
	Canary      string // a passing canary receipt for this CLI version
	Budget      time.Duration
	Scratch     string // where attempt trees are built; os.TempDir() when empty
}

// plan is what run.json records; a second Run into the same directory must
// carry the same plan, or it is an earlier run's directory and is refused.
type plan struct {
	Stage       string   `json:"stage"`
	Tasks       []string `json:"tasks"`
	Models      []string `json:"models"`
	Configs     []string `json:"configs"`
	Repeats     int      `json:"repeats"`
	Seed        int64    `json:"seed"`
	CLIVersion  string   `json:"cli_version"`
	MrwVersion  string   `json:"mrw_version"`
	Containment string   `json:"containment"`
}

// attempt is one run of one task in one configuration on one model.
type attempt struct {
	ID     string
	Task   Task
	Model  string
	Config Config
	Repeat int
}

// Run carries out a stage: every attempt of the plan in a seeded random order,
// one at a time, each in a fresh tree, each recorded in its own receipt. An
// attempt with a receipt, or one that was started and never finished, is not
// run again (ADR-156 Decision 6). An attempt the registration excludes is
// replaced by a new one under its own id — a start error at most twice, a
// setup failure once — and every excluded receipt is kept.
func Run(o RunOptions) error {
	cd, err := gates(o)
	if err != nil {
		return err
	}
	o.ConfigDir = cd
	if o.Budget == 0 {
		o.Budget = DefaultBudget
	}
	cli, err := firstLine(o.Claude, "--version")
	if err != nil {
		return fmt.Errorf("%s --version: %w", o.Claude, err)
	}
	if err := checkCanary(o.Canary, cli); err != nil {
		return err
	}
	mrwVersion := ""
	if slices.Contains(o.Configs, "B") {
		if mrwVersion, err = firstLine(o.Mrw, "version"); err != nil {
			return fmt.Errorf("%s version: %w", o.Mrw, err)
		}
	}
	if p := onPath("mrw", systemPath); p != "" {
		return fmt.Errorf("configuration A's PATH would hold %s: an attempt runs with %s, and mrw must not be on it", p, systemPath)
	}
	// A stage whose repository is not on this machine stops here, by name,
	// rather than recording every attempt as a setup exclusion.
	for _, t := range o.Tasks {
		if t.Repo != "fixture" {
			if _, err := repoPath(t); err != nil {
				return err
			}
		}
	}
	p := plan{Stage: o.Stage, Models: o.Models, Configs: o.Configs, Repeats: o.Repeats, Seed: o.Seed, CLIVersion: cli, MrwVersion: mrwVersion, Containment: o.Containment}
	for _, t := range o.Tasks {
		p.Tasks = append(p.Tasks, t.ID)
	}
	if err := claimOut(o.Out, p); err != nil {
		return err
	}
	for i, a := range planAttempts(o) {
		for try := 0; ; try++ {
			at := a
			if try > 0 {
				at.ID = fmt.Sprintf("%s.r%d", a.ID, try)
			}
			base := filepath.Join(o.Out, at.ID)
			var r Receipt
			switch {
			case exists(base + ".receipt.json"):
				b, err := os.ReadFile(base + ".receipt.json")
				if err != nil {
					return err
				}
				if err := json.Unmarshal(b, &r); err != nil {
					return fmt.Errorf("%s.receipt.json cannot be read back: %w", at.ID, err)
				}
			case exists(base + ".started"):
				r.Exclusion = "" // started and never finished: neither it nor a replacement runs
			default:
				if err := writeOnce(base+".started", []byte(time.Now().UTC().Format(time.RFC3339)+"\n")); err != nil {
					return err
				}
				r = runAttempt(o, at, i, cli, mrwVersion)
				if r.interrupted {
					return fmt.Errorf("interrupted during %s: no receipt is written, the attempt stays started and is not repeated, and the stage stops here", at.ID)
				}
				b, err := json.MarshalIndent(r, "", "  ")
				if err != nil {
					return err
				}
				if err := writeOnce(base+".receipt.json", append(b, '\n')); err != nil {
					return err
				}
			}
			if !replaced(r.Exclusion, try) {
				break
			}
		}
	}
	return nil
}

// replaced says whether an attempt excluded for this reason, on this try, gets
// a replacement: the registration re-queues a start error at most twice and a
// setup failure once.
func replaced(exclusion string, try int) bool {
	switch exclusion {
	case ExclusionStart:
		return try < 2
	case ExclusionSetup:
		return try < 1
	}
	return false
}

// systemPath is the part of an attempt's PATH bench does not build itself.
const systemPath = "/usr/bin:/bin:/usr/sbin:/sbin"

// onPath returns where name would be found on path, or "".
func onPath(name, path string) string {
	for _, dir := range filepath.SplitList(path) {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return p
		}
	}
	return ""
}

// gates refuses a stage whose choices Zy has not made, and returns the config
// directory resolved: an attempt is started with that path, never a spelling
// that could lead somewhere else.
func gates(o RunOptions) (string, error) {
	switch {
	case o.Containment == "":
		return "", errors.New("no --containment: Zy chose bypass, bypassPermissions under his own account, on 2026-10-14 (ADR-156 Decision 2); name it")
	case strings.HasPrefix(o.Containment, "user:"):
		return "", errors.New("--containment user:<name> is not built: Zy chose bypass on 2026-10-14 (ADR-156 Decision 2)")
	case o.Containment != "bypass":
		return "", fmt.Errorf("--containment %q: want bypass or user:<name>", o.Containment)
	case o.ConfigDir == "":
		return "", errors.New("no --config-dir: the dedicated CLAUDE_CONFIG_DIR Zy logs in to once (ADR-156 Decision 2)")
	case o.Canary == "":
		return "", errors.New("no --canary: a stage starts only after a passing canary for this CLI version (ADR-156 T2)")
	case o.Repeats < 1 || len(o.Models) == 0 || len(o.Configs) == 0 || len(o.Tasks) == 0:
		return "", errors.New("a stage needs tasks, models, configurations and at least one repeat")
	}
	for _, c := range o.Configs {
		if _, ok := Configs[c]; !ok {
			return "", fmt.Errorf("unknown configuration %q", c)
		}
	}
	cd, err := filepath.Abs(o.ConfigDir)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(cd); err == nil {
		cd = real
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	own := filepath.Join(home, ".claude")
	mine, err1 := os.Stat(cd)
	theirs, err2 := os.Stat(own)
	if strings.EqualFold(cd, own) || (err1 == nil && err2 == nil && os.SameFile(mine, theirs)) {
		return "", fmt.Errorf("--config-dir %s is the user's own ~/.claude: an attempt would load his instructions, hooks and MCP servers", o.ConfigDir)
	}
	return cd, nil
}

// Verify checks a stage directory against its plan: the stage and the counts
// the caller expects, and for every planned attempt a receipt that reads back
// as that attempt. It names what is missing or wrong.
func Verify(dir, stage string, tasks, repeats int) error {
	b, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		return err
	}
	var p plan
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	if p.Stage != stage || len(p.Tasks) != tasks || p.Repeats != repeats {
		return fmt.Errorf("%s holds stage %q with %d tasks × %d repeats, want %q with %d × %d", dir, p.Stage, len(p.Tasks), p.Repeats, stage, tasks, repeats)
	}
	var missing []string
	for _, t := range p.Tasks {
		for _, m := range p.Models {
			for _, c := range p.Configs {
				for rep := 1; rep <= p.Repeats; rep++ {
					id := fmt.Sprintf("%s.%s.%s.%d", t, c, safe(m), rep)
					b, err := os.ReadFile(filepath.Join(dir, id+".receipt.json"))
					if err != nil {
						missing = append(missing, id)
						continue
					}
					var r Receipt
					if err := json.Unmarshal(b, &r); err != nil || r.Attempt != id || r.Task != t || r.Config != c || r.Model != m || r.Repeat != rep {
						missing = append(missing, id+" (its receipt does not read back as this attempt)")
						continue
					}
					// An excluded attempt is complete only with the replacement
					// the registration owes it (the re-review of #411).
					for try := 0; replaced(r.Exclusion, try); try++ {
						next := fmt.Sprintf("%s.r%d", id, try+1)
						b, err := os.ReadFile(filepath.Join(dir, next+".receipt.json"))
						r = Receipt{}
						if err != nil || json.Unmarshal(b, &r) != nil || r.Attempt != next {
							missing = append(missing, next+" (the replacement an excluded attempt is owed)")
							break
						}
					}
				}
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%d attempt(s) have no receipt that verifies: %s", len(missing), strings.Join(missing, ", "))
	}
	return nil
}

// checkCanary refuses a stage whose canary did not pass for this CLI version.
func checkCanary(path, cli string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("canary: %w", err)
	}
	var c struct {
		Pass       bool   `json:"pass"`
		CLIVersion string `json:"cli_version"`
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("canary %s: %w", path, err)
	}
	if !c.Pass || c.CLIVersion != cli {
		return fmt.Errorf("canary %s: pass %v for CLI %q; this CLI is %q, and a stage needs a passing canary for it", path, c.Pass, c.CLIVersion, cli)
	}
	return nil
}

// claimOut makes the receipt directory this plan's, or finds it already is.
func claimOut(out string, p plan) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	want, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	want = append(want, '\n')
	path := filepath.Join(out, "run.json")
	got, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		ents, _ := os.ReadDir(out)
		if len(ents) > 0 {
			return fmt.Errorf("%s holds files and no run.json: it may be an earlier run's, so nothing is written there", out)
		}
		return writeOnce(path, want)
	}
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s holds an earlier run with another plan (run.json differs); its receipts are kept, use a new directory", out)
	}
	return nil
}

func planAttempts(o RunOptions) []attempt {
	var as []attempt
	for _, t := range o.Tasks {
		for _, m := range o.Models {
			for _, c := range o.Configs {
				for r := 1; r <= o.Repeats; r++ {
					as = append(as, attempt{
						ID:   fmt.Sprintf("%s.%s.%s.%d", t.ID, c, safe(m), r),
						Task: t, Model: m, Config: Configs[c], Repeat: r,
					})
				}
			}
		}
	}
	rand.New(rand.NewSource(o.Seed)).Shuffle(len(as), func(i, j int) { as[i], as[j] = as[j], as[i] })
	return as
}

func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == ' ' || r == ':' {
			return '_'
		}
		return r
	}, s)
}

// runAttempt builds the tree, starts the CLI, grades the result. A failure of
// bench's own (tree, oracle) is recorded as the registered setup exclusion.
func runAttempt(o RunOptions, a attempt, order int, cli, mrwVersion string) Receipt {
	r := Receipt{Attempt: a.ID, Stage: o.Stage, Task: a.Task.ID, Shape: a.Task.Shape, Source: "own", Model: a.Model,
		Config: a.Config.Name, Repeat: a.Repeat, Order: order, CLIVersion: cli, Started: time.Now().UTC().Format(time.RFC3339)}
	if a.Config.Mrw {
		r.MrwVersion = mrwVersion
	}
	scratch, err := os.MkdirTemp(o.Scratch, "bench-")
	if err != nil {
		_ = os.MkdirAll(o.Scratch, 0o755)
		if scratch, err = os.MkdirTemp(o.Scratch, "bench-"); err != nil {
			r.Exclusion, r.Note = ExclusionSetup, err.Error()
			return r
		}
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	tree := filepath.Join(scratch, "tree")
	if err := os.Mkdir(tree, 0o755); err != nil {
		r.Exclusion, r.Note = ExclusionSetup, err.Error()
		return r
	}
	if _, err := Materialize(a.Task, tree, nil); err != nil {
		r.Exclusion, r.Note = ExclusionSetup, err.Error()
		return r
	}
	args, env, err := invocation(o, a, scratch)
	if err != nil {
		r.Exclusion, r.Note = ExclusionSetup, err.Error()
		return r
	}
	base := filepath.Join(o.Out, a.ID)
	stdout, err := os.OpenFile(base+".transcript.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		r.Exclusion, r.Note = ExclusionSetup, err.Error()
		return r
	}
	stderr, err := os.OpenFile(base+".stderr", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		_ = stdout.Close()
		r.Exclusion, r.Note = ExclusionSetup, err.Error()
		return r
	}
	// subproc starts the CLI in a group of its own and stops what is left of
	// that group when it returns, on a deadline, a cancel or a clean exit: a
	// shell the agent left running does not outlive the attempt, or run on
	// while its tree is graded and removed (the review of #411).
	ctx, cancel := context.WithTimeout(context.Background(), o.Budget)
	ictx, stopSignals := subproc.Interruptible(ctx)
	cmd := subproc.Command(ictx, o.Claude, args...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = tree, env, stdout, stderr
	start := time.Now()
	runErr := subproc.Run(cmd)
	r.WallMS = time.Since(start).Milliseconds()
	r.TimedOut = ctx.Err() == context.DeadlineExceeded
	r.interrupted = ictx.Err() != nil && ctx.Err() == nil // a signal, not the budget
	stopSignals()
	cancel()
	_ = stdout.Close()
	_ = stderr.Close()
	if ee := (*exec.ExitError)(nil); errors.As(runErr, &ee) {
		r.ExitCode = ee.ExitCode()
	} else if runErr != nil {
		r.ExitCode = -1
	}
	b, err := os.ReadFile(base + ".transcript.jsonl")
	if err != nil {
		r.Exclusion, r.Note = ExclusionSetup, err.Error()
		return r
	}
	t := parseTranscript(b)
	// A result that says success from a CLI that then exited non-zero is an
	// error all the same (the re-review of #411).
	r.IsError, r.NumTurns, r.ToolCalls, r.CostUSD, r.Usage = t.IsError || !t.Result || r.ExitCode != 0, t.NumTurns, t.ToolCalls, t.CostUSD, t.Usage
	r.InitTools, r.InitMCP = t.Tools, t.MCPServers
	r.OverBudget = r.TimedOut || t.NumTurns > MaxTurns
	if errText, _ := os.ReadFile(base + ".stderr"); startFailed(t, r, string(errText)) {
		r.Exclusion = ExclusionStart
	}
	v, err := Oracle(a.Task, tree)
	if err != nil {
		r.Exclusion, r.Note = ExclusionSetup, "oracle: "+err.Error()
		return r
	}
	if r.OverBudget {
		v.Solved = false
	}
	r.Oracle = v
	r.SilentMisEdit = !r.IsError && (len(v.Outside) > 0 || len(v.Structure) > 0)
	if r.Diff, err = diffStat(tree); err != nil {
		r.Note = "diffstat: " + err.Error()
	}
	if a.Config.Mrw {
		r.HookRefused, r.HookReads = hookCounts(filepath.Join(scratch, "hook.log"))
		if r.SlippedWrites, err = SlippedWrites(tree, filepath.Join(scratch, "state")); err != nil {
			r.Note = "slipped writes: " + err.Error()
		}
	}
	return r
}

// startFailure is what an authentication, rate-limit or start error says.
var startFailure = regexp.MustCompile(`(?i)rate.?limit|\b429\b|overloaded|usage limit|quota|auth|log ?in|credential|unauthori[sz]ed|forbidden|\b401\b|\b403\b`)

// startFailed says whether an attempt ended before its first tool call with one
// of the start errors the registration excludes: the CLI never opened a session
// (no init line), or it erred before any tool call naming authentication, a
// rate limit or a quota. Any other error is an attempt that failed, unsolved,
// with its cost counted (BACKLOG, "Pre-registered for ADR-156").
func startFailed(t transcript, r Receipt, stderr string) bool {
	if r.ToolCalls > 0 || r.TimedOut {
		return false
	}
	if !t.InitSeen {
		return true
	}
	return (r.IsError || r.ExitCode != 0) && startFailure.MatchString(stderr+"\n"+t.ResultText)
}

// shellQuote quotes s for a POSIX shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// toolPath builds the directory of the few host tools an attempt may call by
// name — git and go — as links to each, so nothing else in their directories
// (an installed mrw beside git, say) reaches the agent's PATH.
func toolPath(scratch string) (string, error) {
	dir := filepath.Join(scratch, "tools")
	if err := os.Mkdir(dir, 0o755); err != nil {
		return "", err
	}
	for _, name := range []string{"git", "go", "gofmt"} {
		p, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if real, err := filepath.EvalSymlinks(p); err == nil {
			p = real
		}
		if err := os.Symlink(p, filepath.Join(dir, name)); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// invocation is the CLI's arguments and environment for one attempt (ADR-156
// Decision 2). PATH is bench's own: configuration A's holds no mrw.
func invocation(o RunOptions, a attempt, scratch string) ([]string, []string, error) {
	empty := filepath.Join(scratch, "mcp.json")
	if err := os.WriteFile(empty, []byte(`{"mcpServers":{}}`+"\n"), 0o644); err != nil {
		return nil, nil, err
	}
	settings := map[string]any{}
	tools, err := toolPath(scratch)
	if err != nil {
		return nil, nil, err
	}
	path := tools + ":" + systemPath
	args := []string{"-p", a.Task.Instruction, "--output-format", "stream-json", "--verbose", "--model", a.Model,
		"--tools", strings.Join(a.Config.Tools, ","), "--strict-mcp-config", "--mcp-config", empty,
		"--disable-slash-commands", "--no-session-persistence", "--no-chrome", "--permission-mode", "bypassPermissions"}
	// USER is how the CLI finds the subscription's credentials in the macOS
	// keychain: without it a logged-in config directory answers "Not logged
	// in" (the first real canary, 2026-10-14).
	env := []string{"CLAUDE_CONFIG_DIR=" + o.ConfigDir, "HOME=" + os.Getenv("HOME"), "USER=" + os.Getenv("USER"), "TMPDIR=" + scratch, "LANG=C.UTF-8"}
	if a.Config.Mrw {
		bin := filepath.Join(scratch, "bin")
		if err := os.Mkdir(bin, 0o755); err != nil {
			return nil, nil, err
		}
		if err := os.Symlink(o.Mrw, filepath.Join(bin, "mrw")); err != nil {
			return nil, nil, err
		}
		path = bin + ":" + path
		text, err := exec.Command(o.Mrw, "instructions").Output()
		if err != nil {
			return nil, nil, fmt.Errorf("mrw instructions: %w", err)
		}
		prompt := filepath.Join(scratch, "mrw-instructions.txt")
		if err := os.WriteFile(prompt, text, 0o644); err != nil {
			return nil, nil, err
		}
		args = append(args, "--append-system-prompt-file", prompt)
		settings["hooks"] = map[string]any{"PreToolUse": []any{map[string]any{
			"matcher": "Bash",
			"hooks":   []any{map[string]any{"type": "command", "command": shellQuote(o.Bench) + " hook --log " + shellQuote(filepath.Join(scratch, "hook.log"))}},
		}}}
		env = append(env, "XDG_STATE_HOME="+filepath.Join(scratch, "state"))
	}
	sb, err := json.Marshal(settings)
	if err != nil {
		return nil, nil, err
	}
	sf := filepath.Join(scratch, "settings.json")
	if err := os.WriteFile(sf, append(sb, '\n'), 0o644); err != nil {
		return nil, nil, err
	}
	args = append(args, "--settings", sf)
	if !a.Config.Mrw {
		if p := onPath("mrw", path); p != "" {
			return nil, nil, fmt.Errorf("configuration A's PATH holds %s", p)
		}
	}
	env = append(env, "PATH="+path)
	return args, env, nil
}

func diffStat(tree string) (DiffStat, error) {
	var d DiffStat
	if err := git(tree, "add", "-A"); err != nil {
		return d, err
	}
	out, err := gitOut(tree, "diff", "--cached", "--numstat", "HEAD")
	if err != nil {
		return d, err
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		var ins, del int
		var name string
		if n, _ := fmt.Sscanf(l, "%d\t%d\t%s", &ins, &del, &name); n == 3 {
			d.Files++
			d.Insertions += ins
			d.Deletions += del
		} else if strings.HasPrefix(l, "-\t-\t") {
			d.Files++
		}
	}
	return d, nil
}

func firstLine(bin string, args ...string) (string, error) {
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return "", err
	}
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(line), nil
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// writeOnce creates path with b, and refuses if anything is there already. It
// writes a temporary file first and links it into place, so an interrupted
// write leaves no half a receipt under the name a resume would trust (the
// review of #411).
func writeOnce(path string, b []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".partial-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		return err
	}
	return os.Link(tmp, path)
}
