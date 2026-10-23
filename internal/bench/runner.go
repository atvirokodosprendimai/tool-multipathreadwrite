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

// Config is one of the configurations ADR-156 Decision 1 compares, and
// ADR-162's third.
type Config struct {
	Name   string
	Tools  []string
	Mrw    bool // a pinned mrw on PATH, its instructions appended, the hook installed
	Plugin bool // the quality-harness plugin loaded with --plugin-dir (ADR-162)
	Work   bool // W: the instruction carries the pre-registered work line (ADR-163)
}

// Configs are A (the baseline), B (mrw forced), P (A with the
// quality-harness plugin, run only under the stage5 profile) and W (P with one
// more line in its instruction, run only under stage6). Task, Agent,
// WebFetch and WebSearch are offered to none.
var Configs = map[string]Config{
	"A": {Name: "A", Tools: []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep"}},
	"B": {Name: "B", Tools: []string{"Bash", "Glob"}, Mrw: true},
	"P": {Name: "P", Tools: []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep"}, Plugin: true},
	"W": {Name: "W", Tools: []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep"}, Plugin: true, Work: true},
}

// ErrStartErrors and ErrDiskFloor stop a run at an attempt boundary, resumable
// (ADR-162 Decisions 5 and 9): three start errors in a row, rather than
// retries spent on an outage, and free disk below the floor before an attempt.
var (
	ErrStartErrors = errors.New("three start errors in a row: the run stops rather than spend its retries")
	ErrDiskFloor   = errors.New("free disk is below the floor")
)

// The registered wall-time budget (BACKLOG, "Pre-registered for ADR-156"): an
// attempt past it, or past the turn cap (DefaultMaxTurns), is unsolved, its
// cost counted.
const DefaultBudget = 20 * time.Minute

// Exclusions the registration admits; no other reason leaves an attempt out.
const (
	ExclusionStart = "start-error"    // ended before its first tool call: auth, rate limit, CLI start
	ExclusionSetup = "setup"          // bench could not build the tree or grade it
	ExclusionInfra = "infrastructure" // the machine ended it, by a ruling in the stage's exclusions.json
)

// Receipt is one attempt's record, written once and never replaced.
type Receipt struct {
	Attempt       string     `json:"attempt"`
	Stage         string     `json:"stage"`
	Task          string     `json:"task"`
	Shape         string     `json:"shape"`
	Source        string     `json:"source"` // "own" or "swe-bench"
	Model         string     `json:"model"`
	Config        string     `json:"config"`
	Repeat        int        `json:"repeat"`
	Order         int        `json:"order"`
	Started       string     `json:"started"`
	WallMS        int64      `json:"wall_ms"`
	CLIVersion    string     `json:"cli_version"`
	MrwVersion    string     `json:"mrw_version,omitempty"`
	ExitCode      int        `json:"exit_code"`
	TimedOut      bool       `json:"timed_out,omitempty"`
	OverBudget    bool       `json:"over_budget,omitempty"`
	IsError       bool       `json:"is_error"`
	NumTurns      int        `json:"num_turns"`
	ToolCalls     int        `json:"tool_calls"`
	CostUSD       *float64   `json:"total_cost_usd"`
	Usage         Usage      `json:"usage"`
	InitTools     []string   `json:"init_tools"`
	InitMCP       []string   `json:"init_mcp_servers"`
	Oracle        Verdict    `json:"oracle"`
	SilentMisEdit bool       `json:"silent_mis_edit"`
	Diff          DiffStat   `json:"diff"`
	HookRefused   int        `json:"hook_refused,omitempty"`
	HookReads     int        `json:"hook_reads,omitempty"`
	SlippedWrites []string   `json:"slipped_writes,omitempty"`
	Status        string     `json:"status,omitempty"` // the STATUS line's claim, with --status-line (ADR-162)
	Plugin        *PluginRef `json:"plugin,omitempty"` // configuration P's plugin (ADR-162)
	Exclusion     string     `json:"exclusion,omitempty"`
	Note          string     `json:"note,omitempty"`
	// CheckRedAtBase is the task's flag (Task.CheckRedAtBase), copied so the
	// receipt alone says the attempt ran against a check that was red at base.
	CheckRedAtBase bool `json:"check_red_at_base,omitempty"`

	// ShimCounts are the container's, under ADR-163 stage6 profile alone: a
	// stage without it has none of these keys.
	*ShimCounts

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
	Tasks        []Task
	Stage        string
	Out          string
	Models       []string
	Configs      []string
	Repeats      int
	Seed         int64
	Claude       string // the CLI, by absolute path
	Mrw          string // the pinned mrw build configuration B gets
	Bench        string // this binary, which configuration B's hook runs
	ConfigDir    string // the dedicated CLAUDE_CONFIG_DIR
	Containment  string // "bypass"; "user:<name>" is Zy's other choice, not built yet
	Canary       string // a passing canary receipt for this CLI version
	canary       bool   // set by RunCanary alone: a canary may run a profile that bench run still refuses
	Budget       time.Duration
	Scratch      string              // where attempt trees are built; os.TempDir() when empty
	SWE          *SWEGrader          // grades the SWE-bench slice's attempts; nil for the corpus's own tasks
	Profile      string              // "" or ProfileStage5: ADR-162's profile, applied to every configuration
	PluginDir    string              // the plugin configuration P loads, a directory of a tagged checkout
	Node         string              // the node the profile puts on every attempt's PATH; found on PATH when empty
	plugin       *PluginRef          // the plugin's tag and commit, read by resolveProfile
	StatusLine   bool                // append StatusSentence to every instruction and read the claim (ADR-162)
	Block        int                 // ADR-162: the block this run is, into a directory named block-K; 0 for a stage without blocks
	ConfigsFor   map[string][]string // a model's own configurations, in place of Configs (a closed comparison's configuration is not run)
	DiskPath     string              // the volume the disk floor is checked on: the one holding Docker's disk image
	DiskFloorGiB float64             // the free space, in GiB, below which no attempt starts; 0 for no floor
	Docker       string              // the docker CLI; the stage6 profile starts a container with it (ADR-163)
	CanaryImage  string              // the image a task with none of its own (the canary fixture) runs its container in under stage6
	Checks       string              // stage6: the declared-check table, bench/swe-bench/checks.json (ADR-163 Decision 2)
	WorkLine     string              // stage6: the file holding the line configuration W adds to its instruction (ADR-163 Decision 3)
	MaxTurns     int                 // the stage's turn cap; 0 for the registered DefaultMaxTurns
	checks       *Checks             // read by resolveDeclared
	workLine     string              // the line, read by resolveDeclared
	workLineSHA  string              // the SHA-256 of the file it was read from
}

// plan is what run.json records; a second Run into the same directory must
// carry the same plan, or it is an earlier run's directory and is refused.
type plan struct {
	Stage            string              `json:"stage"`
	Tasks            []string            `json:"tasks"`
	Models           []string            `json:"models"`
	Configs          []string            `json:"configs"`
	Repeats          int                 `json:"repeats"`
	Seed             int64               `json:"seed"`
	CLIVersion       string              `json:"cli_version"`
	MrwVersion       string              `json:"mrw_version"`
	Containment      string              `json:"containment"`
	Profile          string              `json:"profile,omitempty"`
	Plugin           *PluginRef          `json:"plugin,omitempty"`
	StatusLineSHA256 string              `json:"status_line_sha256,omitempty"`
	BudgetSeconds    int                 `json:"budget_seconds,omitempty"`
	PromptSHA256     string              `json:"prompt_sha256,omitempty"`
	Block            int                 `json:"block,omitempty"`
	Looks            int                 `json:"looks,omitempty"`
	ConfigsFor       map[string][]string `json:"configs_for,omitempty"`
	MaxTurns         int                 `json:"max_turns,omitempty"`
	ChecksSHA256     string              `json:"checks_sha256,omitempty"`
	WorkLineSHA256   string              `json:"work_line_sha256,omitempty"`
}

// attempt is one run of one task in one configuration on one model.
type attempt struct {
	ID     string
	Task   Task
	Model  string
	Config Config
	Repeat int
	shim   *shimConfig // ADR-163: the shims of the attempt container; nil without the stage6 profile
}

// Run carries out a stage: every attempt of the plan in a seeded random order,
// one at a time, each in a fresh tree, each recorded in its own receipt. An
// attempt with a receipt, or one that was started and never finished, is not
// run again (ADR-156 Decision 6). An attempt the registration excludes is
// replaced by a new one under its own id — a start error at most twice, a
// setup failure once — and every excluded receipt is kept.
func Run(o RunOptions) (err error) {
	cd, err := gates(o)
	if err != nil {
		return err
	}
	o.ConfigDir = cd
	if err := resolveProfile(&o); err != nil {
		return err
	}
	if o.Budget == 0 {
		o.Budget = DefaultBudget
	}
	cli, err := firstLine(o.Claude, "--version")
	if err != nil {
		return fmt.Errorf("%s --version: %w", o.Claude, err)
	}
	if err := checkCanary(o.Canary, cli, o); err != nil {
		return err
	}
	if o.Block > 0 {
		if want := fmt.Sprintf("block-%d", o.Block); filepath.Base(o.Out) != want {
			return fmt.Errorf("block %d runs into a directory named %s, not %s (ADR-162 Decision 7)", o.Block, filepath.Base(o.Out), want)
		}
		if o.Repeats != 1 {
			return fmt.Errorf("a Stage 5 block runs one repeat per configuration, which Tango's interval reads (ADR-162 Decision 7); --repeats is %d", o.Repeats)
		}
		if o.Block > 1 {
			var first plan
			if err := readJSON(filepath.Join(filepath.Dir(o.Out), "block-1", "run.json"), &first); err != nil {
				return fmt.Errorf("block %d needs block 1's run.json beside it: %w", o.Block, err)
			}
			if first.CLIVersion != cli {
				return fmt.Errorf("block 1 ran CLI %q and this one is %q: a stage keeps one CLI, and an upgrade starts a new stage (ADR-162 Decision 7)", first.CLIVersion, cli)
			}
		}
	}
	mrwVersion := ""
	// A Stage 5 block names the stage's mrw whether or not it still runs B, so
	// a block after B's comparison closed is the same release (the review of T3).
	if hasConfig(o, "B") || (o.Block > 0 && o.Mrw != "") {
		if mrwVersion, err = firstLine(o.Mrw, "version"); err != nil {
			return fmt.Errorf("%s version: %w", o.Mrw, err)
		}
	}
	// A later block carries block 1's mrw when it names none itself, and may
	// not name another (the re-review of T3): Look compares every block's with
	// block 1's, so a block that drops the pin would poison the stage.
	if o.Block > 1 {
		var first plan
		if err := readJSON(filepath.Join(filepath.Dir(o.Out), "block-1", "run.json"), &first); err != nil {
			return fmt.Errorf("block %d needs block 1's run.json beside it: %w", o.Block, err)
		}
		if mrwVersion == "" {
			mrwVersion = first.MrwVersion
		} else if mrwVersion != first.MrwVersion {
			return fmt.Errorf("block 1 ran mrw %q and this one names %q: a stage keeps one release (ADR-162 Decision 7)", first.MrwVersion, mrwVersion)
		}
	}
	if p := onPath("mrw", systemPath); p != "" {
		return fmt.Errorf("configuration A's PATH would hold %s: an attempt runs with %s, and mrw must not be on it", p, systemPath)
	}
	// A stage whose repository is not on this machine stops here, by name,
	// rather than recording every attempt as a setup exclusion.
	for _, t := range o.Tasks {
		if t.swe != nil && o.SWE == nil {
			return fmt.Errorf("task %s is from the SWE-bench slice and the stage has no grader for it", t.ID)
		}
		if t.Repo != "fixture" {
			src, err := repoPath(t)
			if err != nil {
				return err
			}
			if err := checkSections(t, src); err != nil {
				return err
			}
		}
	}
	p := plan{Stage: o.Stage, Models: o.Models, Configs: o.Configs, Repeats: o.Repeats, Seed: o.Seed, CLIVersion: cli, MrwVersion: mrwVersion, Containment: o.Containment, Profile: o.Profile, Plugin: o.plugin}
	p.ConfigsFor = o.ConfigsFor
	p.WorkLineSHA256 = o.workLineSHA
	if o.checks != nil {
		p.ChecksSHA256 = o.checks.SHA256
	}
	// Stage 5's run.json keeps its bytes: the cap is written once a stage has one of its own.
	if o.Profile == ProfileStage6 || o.MaxTurns != 0 {
		p.MaxTurns = o.maxTurns()
	}
	if o.Block > 0 {
		p.Block, p.Looks = o.Block, Looks
	}
	if o.StatusLine {
		p.StatusLineSHA256, p.PromptSHA256, p.BudgetSeconds = statusLineSHA256(), promptSHA256(), int(o.Budget.Seconds())
	}
	for _, t := range o.Tasks {
		p.Tasks = append(p.Tasks, t.ID)
	}
	if err := claimOut(o.Out, p); err != nil {
		return err
	}
	// image is the SWE-bench image of the instance whose attempts are now
	// planned. It is dropped at every boundary between instances, whether that
	// instance's attempts ran now or before a resume, so an image left by an
	// interrupted run goes too; a stage that stops on an error names the image
	// it leaves in place for the resume (the review of #422).
	image := ""
	defer func() {
		if err != nil && image != "" {
			err = fmt.Errorf("%w (the image %s, if it was pulled, is left in place: the resume drops it once that instance is done)", err, image)
		}
	}()
	streak := 0
	for i, a := range planAttempts(o) {
		if a.Task.swe != nil && a.Task.swe.image != image {
			if image != "" {
				if err := o.SWE.drop(image); err != nil {
					return err
				}
			}
			image = a.Task.swe.image
		}
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
				if o.DiskFloorGiB > 0 {
					free, err := freeGiB(o.DiskPath)
					if err != nil {
						return fmt.Errorf("the disk floor: %s: %w", o.DiskPath, err)
					}
					if free < o.DiskFloorGiB {
						return fmt.Errorf("%w: %.1f GiB free on %s, the floor is %.1f GiB; free %.1f GiB and resume", ErrDiskFloor, free, o.DiskPath, o.DiskFloorGiB, o.DiskFloorGiB-free)
					}
				}
				if a.Task.swe != nil {
					if err := o.SWE.ensure(image); err != nil {
						return err
					}
				}
				if err := writeOnce(base+".started", []byte(time.Now().UTC().Format(time.RFC3339)+"\n")); err != nil {
					return err
				}
				r = runAttempt(o, at, i, cli, mrwVersion)
				if r.interrupted {
					return fmt.Errorf("interrupted during %s: no receipt is written, the attempt stays started and is not repeated, and the stage stops here%s", at.ID, r.Note)
				}
				b, err := json.MarshalIndent(r, "", "  ")
				if err != nil {
					return err
				}
				if err := writeOnce(base+".receipt.json", append(b, '\n')); err != nil {
					return err
				}
				// Stage 5 alone (ADR-162 Decision 5): earlier stages keep ADR-156's
				// replacements and run on.
				if r.Exclusion == ExclusionStart && o.Block > 0 {
					if streak++; streak >= 3 {
						return fmt.Errorf("%w (the last at %s); resume once the CLI starts again", ErrStartErrors, at.ID)
					}
				} else {
					streak = 0
				}
			}
			if !replaced(r.Exclusion, try) {
				break
			}
		}
	}
	if image != "" {
		if err := o.SWE.drop(image); err != nil {
			return err
		}
		image = ""
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
		return "", errors.New("no --containment: Zy chose bypass, bypassPermissions under their own account, on 2026-10-14 (ADR-156 Decision 2); name it")
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
	for _, cs := range append([][]string{o.Configs}, mapValues(o.ConfigsFor)...) {
		for _, c := range cs {
			if _, ok := Configs[c]; !ok {
				return "", fmt.Errorf("unknown configuration %q", c)
			}
		}
	}
	for m := range o.ConfigsFor {
		if !slices.Contains(o.Models, m) {
			return "", fmt.Errorf("--configs-for names %s, which the stage does not run", m)
		}
	}
	switch {
	case o.Profile != "" && o.Profile != ProfileStage5 && o.Profile != ProfileStage6:
		return "", fmt.Errorf("--profile %q: want %s or %s (ADR-162, ADR-163)", o.Profile, ProfileStage5, ProfileStage6)
	case o.Profile == ProfileStage6 && !settled && !o.canary:
		return "", errors.New("the stage6 profile is not settled: ADR-163 T1 S1 has not recorded the inspection of a real image, so bench run refuses it; bench canary may run")
	case o.Profile == ProfileStage6 && o.CanaryImage == "" && slices.ContainsFunc(o.Tasks, func(t Task) bool { return t.swe == nil }):
		return "", errors.New("the stage6 profile starts a container of the image of each task, and a task with none needs --canary-image")
	case hasConfig(o, "W") && o.Profile != ProfileStage6:
		return "", fmt.Errorf("configuration W runs only under --profile %s (ADR-163 Decision 3)", ProfileStage6)
	case hasConfig(o, "W") && o.WorkLine == "":
		return "", errors.New("configuration W needs --work-line, the file holding the line quality-harness's ADR-112 pre-registers (ADR-163 Decision 3)")
	case o.WorkLine != "" && o.Profile != ProfileStage6:
		return "", fmt.Errorf("--work-line is configuration W's, and runs only under --profile %s (ADR-163 Decision 3)", ProfileStage6)
	case o.Profile == ProfileStage6 && o.Checks == "" && slices.ContainsFunc(o.Tasks, func(t Task) bool { return t.swe != nil }):
		return "", errors.New("the stage6 profile writes each repository's declared check into the tree, and a SWE-bench task needs --checks (ADR-163 Decision 2)")
	case hasConfig(o, "P") && o.Profile == "":
		return "", fmt.Errorf("configuration P runs only under --profile %s, which every configuration shares (ADR-162 Decision 1)", ProfileStage5)
	case (hasConfig(o, "P") || hasConfig(o, "W")) && o.PluginDir == "":
		return "", errors.New("configuration P and W need --plugin-dir, the plugin's directory in a tagged checkout (ADR-162 Decision 1)")
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
		return "", fmt.Errorf("--config-dir %s is the user's own ~/.claude: an attempt would load their instructions, hooks and MCP servers", o.ConfigDir)
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
			for _, c := range plannedConfigs(p, m) {
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
						if err != nil || json.Unmarshal(b, &r) != nil || r.Attempt != next || r.Task != t || r.Config != c || r.Model != m || r.Repeat != rep {
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
func checkCanary(path, cli string, o RunOptions) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("canary: %w", err)
	}
	var c struct {
		Pass       bool       `json:"pass"`
		CLIVersion string     `json:"cli_version"`
		Profile    string     `json:"profile"`
		Configs    []string   `json:"configs"`
		Plugin     *PluginRef `json:"plugin"`
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("canary %s: %w", path, err)
	}
	if !c.Pass || c.CLIVersion != cli {
		return fmt.Errorf("canary %s: pass %v for CLI %q; this CLI is %q, and a stage needs a passing canary for it", path, c.Pass, c.CLIVersion, cli)
	}
	// A profiled stage needs a canary that ran under the profile, tested each
	// of its configurations and loaded its plugin (ADR-162 Decision 2; the
	// review of T1): an A/B canary never saw P or the profile's settings.
	if o.Profile != "" {
		if c.Profile != o.Profile {
			return fmt.Errorf("canary %s ran without the %s profile this stage runs", path, o.Profile)
		}
		// Every configuration any model runs, its own list included (the
		// re-review of T3): a canary that never ran P authorizes no P.
		for _, m := range o.Models {
			for _, name := range configsFor(o, m) {
				if !slices.Contains(c.Configs, name) {
					return fmt.Errorf("canary %s did not test configuration %s, which %s runs", path, name, m)
				}
			}
		}
		if o.plugin != nil && (c.Plugin == nil || *c.Plugin != *o.plugin) {
			return fmt.Errorf("canary %s loaded plugin %v; this stage runs %v", path, c.Plugin, *o.plugin)
		}
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

// configsFor is what a model runs: its own list from --configs-for, else the
// stage's.
func configsFor(o RunOptions, model string) []string {
	if cs, ok := o.ConfigsFor[model]; ok {
		return cs
	}
	return o.Configs
}

// hasConfig says whether any model of the stage runs configuration c.
func hasConfig(o RunOptions, c string) bool {
	for _, m := range o.Models {
		if slices.Contains(configsFor(o, m), c) {
			return true
		}
	}
	return slices.Contains(o.Configs, c) && len(o.Models) == 0
}

func mapValues(m map[string][]string) [][]string {
	var out [][]string
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func planAttempts(o RunOptions) []attempt {
	var as []attempt
	for _, t := range o.Tasks {
		for _, m := range o.Models {
			for _, c := range configsFor(o, m) {
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
	// A SWE-bench instance's attempts run one after another, in the seeded
	// order among themselves, so its image is pulled once and removed once
	// they are graded (T5 S2: twenty images do not fit on this disk).
	if len(o.Tasks) > 0 && o.Tasks[0].swe != nil {
		at := map[string]int{}
		for i, t := range o.Tasks {
			at[t.ID] = i
		}
		slices.SortStableFunc(as, func(x, y attempt) int { return at[x.Task.ID] - at[y.Task.ID] })
	}
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
func runAttempt(o RunOptions, a attempt, order int, cli, mrwVersion string) (r Receipt) {
	source := "own"
	if a.Task.swe != nil {
		source = "swe-bench"
	}
	r = Receipt{Attempt: a.ID, Stage: o.Stage, Task: a.Task.ID, Shape: a.Task.Shape, Source: source, Model: a.Model,
		Config: a.Config.Name, Repeat: a.Repeat, Order: order, CLIVersion: cli, Started: time.Now().UTC().Format(time.RFC3339),
		CheckRedAtBase: a.Task.CheckRedAtBase != ""}
	if a.Config.Mrw {
		r.MrwVersion = mrwVersion
	}
	if a.Config.Plugin {
		r.Plugin = o.plugin
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
	tree := attemptTree(scratch)
	if err := os.Mkdir(tree, 0o755); err != nil {
		r.Exclusion, r.Note = ExclusionSetup, err.Error()
		return r
	}
	baseSHA, err := materialize(a.Task, tree, nil, o.declaredFor(a.Task))
	if err != nil {
		r.Exclusion, r.Note = ExclusionSetup, err.Error()
		return r
	}
	// Under stage6 the attempt container is started from the tree just built,
	// and removed on every way out: a clean end, an error, the budget and an
	// interrupt all return through this function (ADR-163 Decision 1).
	if o.Profile == ProfileStage6 {
		name := containerName(a.ID)
		box, err := startContainer(o.Docker, imageFor(o, a.Task), name, o.Stage, o.Budget+300*time.Second, tree, baseSHA)
		if box != nil {
			defer func() {
				if err := box.remove(); err != nil {
					r.Note += " (" + err.Error() + ")"
				}
			}()
		}
		if err != nil {
			r.Exclusion, r.Note = ExclusionSetup, err.Error()
			return r
		}
		a.shim = &shimConfig{Bench: o.Bench, Docker: o.Docker, Container: name, Tree: tree, State: filepath.Join(scratch, "shim"), Base: baseSHA}
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
	// An interrupted attempt is neither graded nor recorded: Run stops the
	// stage on it, and a harness run would hold that stop for minutes (the
	// review of #422).
	if r.interrupted { // nothing grades it
		return r
	}
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
	r.OverBudget = r.TimedOut || t.NumTurns > o.maxTurns()
	if a.shim != nil {
		var note string
		r.ShimCounts, note = shimCounts(a.shim.logPath(), t.Commands, imageFor(o, a.Task), a.shim.Container, filepath.Join(scratch, "tools"))
		if note != "" {
			r.Note += " (shim log: " + note + ")"
		}
	}
	if o.StatusLine {
		r.Status = claimStatus(t.ResultText)
	}
	if errText, _ := os.ReadFile(base + ".stderr"); startFailed(t, r, string(errText)) {
		r.Exclusion = ExclusionStart
	}
	var v Verdict
	if a.Task.swe != nil {
		v, err = o.SWE.grade(a.Task, tree, scratch, a.ID)
	} else {
		v, err = Oracle(a.Task, tree)
	}
	if errors.Is(err, errGradingInterrupted) { // not graded, no receipt
		r.interrupted = true
		if err.Error() != errGradingInterrupted.Error() {
			// what the harness or its cleanup said, a container left included
			r.Note = " (" + err.Error() + ")"
		}
		return r
	}
	if err != nil {
		r.Exclusion, r.Note = ExclusionSetup, "oracle: "+err.Error()
		return r
	}
	v = settle(v, r.OverBudget, a.Task.swe != nil)
	r.Oracle = v
	r.SilentMisEdit = !r.IsError && (len(v.Outside) > 0 || len(v.Structure) > 0)
	if r.Diff, err = diffStat(tree); err != nil {
		r.Note = "diffstat: " + err.Error()
	}
	if a.Config.Mrw {
		r.HookRefused, r.HookReads = hookCounts(filepath.Join(scratch, "hook.log"))
		// A backstop that could not run is not a count of zero: the
		// attempt is a setup exclusion (the final review of #411).
		if r.SlippedWrites, err = SlippedWrites(tree, filepath.Join(scratch, "state")); err != nil {
			r.Exclusion, r.Note = ExclusionSetup, "slipped writes: "+err.Error()
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
// name, as links to each, so nothing else in their directories (an installed
// mrw beside git, say) reaches the agent's PATH: git, which bench and every
// repository's check use; go and gofmt, for the Go repositories' checks; node,
// npm and npx, which quality-harness's declared check runs (without them it
// stops at once with `node: command not found`, a failure the harness made);
// and python3, so a script finds the host's own rather than an older system
// one (Zy, 2026-10-14: "Keep, add node, declare it"). node, when given, is
// linked in place of the one PATH finds (ADR-162: on this Mac PATH's node is
// a shell function).
func toolPath(scratch, node string) (string, error) {
	dir := filepath.Join(scratch, "tools")
	if err := os.Mkdir(dir, 0o755); err != nil {
		return "", err
	}
	for _, name := range []string{"git", "go", "gofmt", "node", "npm", "npx", "python3"} {
		p, err := exec.LookPath(name)
		if name == "node" && node != "" {
			p, err = node, nil
		}
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
	tools, err := toolPath(scratch, o.Node)
	if err != nil {
		return nil, nil, err
	}
	// Under the stage6 profile the shims of ADR-163 stand where the host python3
	// was linked, beside python, pytest and py.test. The link goes first: writing
	// a shim through it would write the host interpreter.
	if a.shim != nil {
		if err := os.Remove(filepath.Join(tools, "python3")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, nil, err
		}
		if err := writeShims(tools, *a.shim); err != nil {
			return nil, nil, err
		}
	}
	path := tools + ":" + systemPath
	// Under ADR-162's profile every configuration is offered Skill, keeps its
	// skills, and reports its hooks; --include-hook-events takes the place
	// --disable-slash-commands has, so P's arguments are A's and the plugin.
	c, skills := a.Config, "--disable-slash-commands"
	if o.Profile != "" {
		c, skills = profiled(c), "--include-hook-events"
	}
	instruction := instructionFor(o, a.Config, a.Task)
	args := []string{"-p", instruction, "--output-format", "stream-json", "--verbose", "--model", a.Model,
		"--tools", strings.Join(c.Tools, ","), "--strict-mcp-config", "--mcp-config", empty,
		skills, "--no-session-persistence", "--no-chrome", "--permission-mode", "bypassPermissions",
		// The attempt's own directory would otherwise sit in the system prompt
		// ahead of B's appended instructions, so B wrote them to the cache in
		// every attempt; with the flag they are read from it, as a user's own
		// checkout reads them (ADR-156 Decision 2, measured 2026-10-14).
		"--exclude-dynamic-system-prompt-sections"}
	if a.Config.Plugin {
		args = append(args, "--plugin-dir", o.PluginDir)
	}
	// USER is how the CLI finds the subscription's credentials in the macOS
	// keychain: without it a logged-in config directory answers "Not logged
	// in" (the first real canary, 2026-10-14).
	env := []string{"CLAUDE_CONFIG_DIR=" + o.ConfigDir, "HOME=" + os.Getenv("HOME"), "USER=" + os.Getenv("USER"), "TMPDIR=" + scratch, "LANG=C.UTF-8"}
	// HOME stays for the credentials, so what it would bring in is shut out
	// one by one (ADR-158 Decision 1): the user's zsh and its aliases, the CLI
	// task directory every Claude session here shares, git's global
	// configuration and ignore, and Python's user site.
	own := map[string]string{"CLAUDE_CODE_TMPDIR": filepath.Join(scratch, "cctmp"), "XDG_CONFIG_HOME": filepath.Join(scratch, "config")}
	for _, k := range []string{"CLAUDE_CODE_TMPDIR", "XDG_CONFIG_HOME"} {
		if err := os.Mkdir(own[k], 0o755); err != nil {
			return nil, nil, err
		}
		env = append(env, k+"="+own[k])
	}
	env = append(env, "CLAUDE_CODE_SHELL=/bin/bash", "GIT_CONFIG_GLOBAL=/dev/null", "PYTHONNOUSERSITE=1")
	if o.Profile != "" {
		ps, err := profileSettings(o.ConfigDir)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range ps {
			settings[k] = v
		}
		if onPath("node", tools) == "" {
			return nil, nil, errors.New("the stage5 profile needs node on the attempt's PATH: the plugin's hooks run it")
		}
		if p := onPath("codex", tools+":"+systemPath); p != "" {
			return nil, nil, fmt.Errorf("the stage5 profile reaches no external model, and %s is on the attempt's PATH", p)
		}
		stub, err := writeCodexStub(scratch)
		if err != nil {
			return nil, nil, err
		}
		env = append(env, "CODEX_CLI_PATH="+stub)
	}
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
			"hooks":   []any{map[string]any{"type": "command", "command": shellQuote(o.Bench) + " hook --log " + shellQuote(filepath.Join(scratch, "hook.log")) + " --tree " + shellQuote(attemptTree(scratch))}},
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
	if a.shim != nil {
		if err := writeStartEnv(a.shim.State, env); err != nil {
			return nil, nil, err
		}
	}
	env = append(env, "PATH="+path)
	return args, env, nil
}

// attemptTree is where an attempt's task tree lives in its scratch directory;
// the hook is told it, so a write outside it passes.
func attemptTree(scratch string) string { return filepath.Join(scratch, "tree") }

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
