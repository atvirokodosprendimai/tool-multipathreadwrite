package bench

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// hookMarkers are strings the user's own hooks on this machine print into a
// session (memoryd, agentsmemory, quality-harness); any of them in a
// transcript means an attempt ran under their setup.
var hookMarkers = []string{"<<<memoryd data", "memoryd ·", "agentsmemory", "quality-harness:", "SessionStart hook", "UserPromptSubmit hook"}

// CheckCanary returns what a canary transcript shows leaking into an attempt
// (ADR-156 Decision 2), empty when it is clean: no init line; tools other than
// the configuration's; any MCP server; mrw named in configuration A; a
// sentence of the user's CLAUDE.md; a user hook's output. configDir is the
// attempt's CLAUDE_CONFIG_DIR: the CLI prints it in the init line's paths, and
// it is bench's own name for a directory, not context the model was given, so
// it is taken out of the text the mrw and hook checks read; the CLAUDE.md
// check reads the text whole, so a sentence naming that directory still counts.
func CheckCanary(transcript []byte, c Config, claudeMD, configDir string) []string {
	var problems []string
	t := parseTranscript(transcript)
	if !t.InitSeen {
		problems = append(problems, "no system/init line: the tools and MCP servers cannot be checked")
	} else {
		if !slices.Equal(sorted(t.Tools), sorted(c.Tools)) {
			problems = append(problems, fmt.Sprintf("tools %v, want %v", t.Tools, c.Tools))
		}
		if len(t.MCPServers) > 0 {
			problems = append(problems, fmt.Sprintf("MCP servers %v, want none", t.MCPServers))
		}
	}
	// The raw lines and every JSON string in them, decoded: a sentence with a
	// quote in it, or a marker sent as \u escapes, is in the transcript only
	// escaped (the review of #411).
	text := string(transcript) + "\n" + decodedStrings(transcript)
	named := text
	if configDir != "" {
		named = strings.ReplaceAll(text, configDir, "<config-dir>")
	}
	if !c.Mrw && regexp.MustCompile(`(?i)\bmrw\b`).MatchString(named) {
		problems = append(problems, "configuration A's transcript names mrw")
	}
	for _, s := range distinctive(claudeMD) {
		if strings.Contains(text, s) {
			problems = append(problems, fmt.Sprintf("the user's CLAUDE.md is quoted: %q", s))
			break
		}
	}
	for _, m := range hookMarkers {
		if strings.Contains(named, m) {
			problems = append(problems, fmt.Sprintf("a user hook's output is present: %q", m))
		}
	}
	return problems
}

// CanaryReceipt is the verdict of one canary run: one attempt per
// configuration on a fixture task, checked for leaks, with a sentinel file
// outside the attempt's tree that must come through untouched.
type CanaryReceipt struct {
	Pass       bool                `json:"pass"`
	CLIVersion string              `json:"cli_version"`
	Model      string              `json:"model"`
	Started    string              `json:"started"`
	Problems   map[string][]string `json:"problems"`
	// Under ADR-162's profile: the profile, the configurations it tested and the
	// plugin it loaded, which a profiled stage must match.
	Profile string              `json:"profile,omitempty"`
	Configs []string            `json:"configs,omitempty"`
	Plugin  *PluginRef          `json:"plugin,omitempty"`
	Costs   map[string]*float64 `json:"total_cost_usd"`
}

// RunCanary runs the task once in each configuration and writes canary.json
// into o.Out. It needs the same gates as a stage, except a canary.
func RunCanary(o RunOptions, task Task, claudeMD string) (CanaryReceipt, error) {
	var c CanaryReceipt
	o.Canary, o.canary = "-", true
	cd, err := gates(o)
	if err != nil {
		return c, err
	}
	o.ConfigDir = cd
	if err := resolveProfile(&o); err != nil {
		return c, err
	}
	if o.Budget == 0 {
		o.Budget = DefaultBudget
	}
	cli, err := firstLine(o.Claude, "--version")
	if err != nil {
		return c, err
	}
	mrwVersion, err := firstLine(o.Mrw, "version")
	if err != nil {
		return c, err
	}
	if err := os.MkdirAll(o.Out, 0o755); err != nil {
		return c, err
	}
	if err := os.MkdirAll(o.Scratch, 0o755); err != nil {
		return c, err
	}
	sentinel := filepath.Join(o.Scratch, "sentinel-outside-the-tree.txt")
	want := []byte("the canary's sentinel " + time.Now().UTC().Format(time.RFC3339Nano) + "\n")
	if err := os.WriteFile(sentinel, want, 0o644); err != nil {
		return c, err
	}
	c = CanaryReceipt{Pass: true, CLIVersion: cli, Model: o.Models[0], Started: time.Now().UTC().Format(time.RFC3339),
		Problems: map[string][]string{}, Costs: map[string]*float64{}, Profile: o.Profile, Plugin: o.plugin}
	if o.Profile != "" {
		c.Configs = o.Configs
	}
	for i, name := range o.Configs {
		a := attempt{ID: "canary." + name, Task: task, Model: o.Models[0], Config: Configs[name], Repeat: 1}
		r := runAttempt(o, a, i, cli, mrwVersion)
		if r.interrupted {
			return c, fmt.Errorf("interrupted during the canary's configuration %s: no canary.json is written and nothing more runs", name)
		}
		c.Costs[name] = r.CostUSD
		var problems []string
		if r.Exclusion != "" || r.IsError {
			problems = append(problems, fmt.Sprintf("the attempt did not run cleanly: exclusion %q, error %v, exit %d %s", r.Exclusion, r.IsError, r.ExitCode, r.Note))
		}
		b, err := os.ReadFile(filepath.Join(o.Out, a.ID+".transcript.jsonl"))
		if err != nil {
			return c, err
		}
		if o.Profile != "" {
			problems = append(problems, CheckProfileCanary(b, Configs[name], claudeMD, o.ConfigDir, o.PluginDir)...)
		} else {
			problems = append(problems, CheckCanary(b, Configs[name], claudeMD, o.ConfigDir)...)
		}
		if len(problems) > 0 {
			c.Pass = false
			c.Problems[name] = problems
		}
	}
	// Under stage6 the container itself is asked what an attempt in it would
	// see, apart from the attempts above (ADR-163 Decision 6).
	if o.Profile == ProfileStage6 {
		if p := probeContainer(o, task); len(p) > 0 {
			c.Pass = false
			c.Problems["container"] = p
		}
	}
	if got, err := os.ReadFile(sentinel); err != nil || !bytes.Equal(got, want) {
		c.Pass = false
		c.Problems["sentinel"] = []string{"a file outside the attempt's tree changed or vanished"}
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return c, err
	}
	return c, writeOnce(filepath.Join(o.Out, "canary.json"), append(b, '\n'))
}

// CheckProfileCanary is the canary under ADR-162's stage5 profile (Decision 2):
// CheckCanary over the transcript with the configuration's profiled tools, and
// then what the profile adds. A's and B's init line lists exactly the built-in
// skills and no plugin; P's lists those and the plugin's own, and the plugin
// alone. A fires no hook; B only bench's own, PreToolUse:Bash; P at least one
// of the plugin's session hooks — the events carry no source, and in a profiled
// attempt the plugin is the only hook source P has. In P the plugin's own text
// — its hook output, its skills in init, the Skill calls to them, their results
// and the skill text they inject — is taken out before the mrw and marker
// checks read it, since the plugin itself names mrw and the markers; the
// agent's own text is read as in A.
func CheckProfileCanary(transcript []byte, c Config, claudeMD, configDir, pluginDir string) []string {
	text := transcript
	if c.Plugin {
		text = withoutPluginText(transcript, pluginDir)
	}
	problems := CheckCanary(text, profiled(c), claudeMD, configDir)
	t := parseTranscript(transcript)
	if !t.InitSeen {
		return problems
	}
	var own, other []string
	for _, s := range t.InitSkills {
		if strings.HasPrefix(s, pluginName+":") {
			own = append(own, s)
		} else {
			other = append(other, s)
		}
	}
	if !slices.Equal(sorted(other), sorted(builtinSkills)) {
		problems = append(problems, fmt.Sprintf("skills %v beside the plugin's, want exactly the built-in %v", other, builtinSkills))
	}
	var plugins []string
	if c.Plugin {
		plugins = []string{pluginName}
		if len(own) == 0 {
			problems = append(problems, "configuration P lists none of the plugin's skills: the plugin is not engaged")
		}
	} else if len(own) > 0 {
		problems = append(problems, fmt.Sprintf("configuration %s lists the plugin's skills %v", c.Name, own))
	}
	var names []string
	for _, p := range t.InitPlugins {
		names = append(names, p.Name)
		// A plugin of the right name from elsewhere — a synced copy, another
		// checkout — is not the one the receipt names (the review of T1).
		if c.Plugin && p.Name == pluginName && (p.Source != pluginName+"@inline" || filepath.Clean(p.Path) != filepath.Clean(pluginDir)) {
			problems = append(problems, fmt.Sprintf("the plugin was loaded from %s (%s), want %s@inline from %s", p.Path, p.Source, pluginName, pluginDir))
		}
	}
	if !slices.Equal(sorted(names), plugins) {
		problems = append(problems, fmt.Sprintf("plugins %v, want %v", names, plugins))
	}
	switch {
	case c.Plugin:
		if !slices.ContainsFunc(t.HookEvents, func(e string) bool { return e == "SessionStart" || e == "UserPromptSubmit" || e == "Stop" }) {
			problems = append(problems, "no session hook of the plugin fired: the plugin is not engaged")
		}
	case c.Mrw:
		for _, h := range t.HookNames {
			if h != "PreToolUse:Bash" {
				problems = append(problems, fmt.Sprintf("a hook other than bench's own fired: %s", h))
			}
		}
	default:
		if len(t.HookNames) > 0 {
			problems = append(problems, fmt.Sprintf("configuration %s fired hooks %v", c.Name, t.HookNames))
		}
	}
	return problems
}

// withoutPluginText is a P transcript without the plugin's own text: its hook
// lines, its skills, agents and plugin entry in init, the Skill calls to its
// skills, their results, and the skill text injected from pluginDir.
func withoutPluginText(transcript []byte, pluginDir string) []byte {
	calls := map[any]bool{}
	var out [][]byte
	for _, line := range bytes.Split(transcript, []byte("\n")) {
		var m map[string]any
		if json.Unmarshal(line, &m) != nil {
			out = append(out, line)
			continue
		}
		switch m["type"] {
		case "system":
			switch m["subtype"] {
			case "hook_started", "hook_response":
				continue
			case "init":
				for _, k := range []string{"skills", "slash_commands", "agents", "plugins"} {
					delete(m, k)
				}
			}
		case "assistant", "user":
			msg, _ := m["message"].(map[string]any)
			content, ok := msg["content"].([]any)
			if !ok {
				break
			}
			var kept []any
			for _, item := range content {
				b, _ := item.(map[string]any)
				if b["type"] == "tool_use" && b["name"] == "Skill" {
					if in, _ := b["input"].(map[string]any); in != nil {
						if s, _ := in["skill"].(string); strings.HasPrefix(s, pluginName+":") {
							calls[b["id"]] = true
							continue
						}
					}
				}
				if b["type"] == "tool_result" && calls[b["tool_use_id"]] {
					continue
				}
				// Only the CLI injects a skill's text, as a user message; the
				// agent's own text is never exempt (the review of T1).
				if s, _ := b["text"].(string); m["type"] == "user" && b["type"] == "text" && pluginDir != "" && strings.HasPrefix(s, "Base directory for this skill: "+pluginDir+"/") {
					continue
				}
				kept = append(kept, item)
			}
			msg["content"] = kept
		}
		b, err := json.Marshal(m)
		if err != nil {
			continue
		}
		out = append(out, b)
	}
	return bytes.Join(out, []byte("\n"))
}

// decodedStrings joins every string value of every JSON line of a transcript.
func decodedStrings(transcript []byte) string {
	var out []string
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			out = append(out, x)
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	for _, line := range bytes.Split(transcript, []byte("\n")) {
		var v any
		if json.Unmarshal(line, &v) == nil {
			walk(v)
		}
	}
	return strings.Join(out, "\n")
}

// distinctive are the lines of a CLAUDE.md long enough that finding one in a
// transcript cannot be chance.
func distinctive(md string) []string {
	var out []string
	for _, l := range strings.Split(md, "\n") {
		l = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l), "#*->"))
		if len(l) >= 40 {
			out = append(out, l)
		}
	}
	return out
}

func sorted(s []string) []string {
	c := slices.Clone(s)
	slices.Sort(c)
	return c
}

// netProbe is the Python the canary asks the container: it prints blocked when
// no connection can be made, and open when one is.
const netProbe = `import socket
try:
    socket.create_connection(("1.1.1.1", 53), timeout=5)
    print("open")
except OSError:
    print("blocked")
`

// CheckContainerCanary is the canary of the container under stage6 (ADR-163
// Decision 6): python printed linux for sys.platform, so the shim reaches the
// image; the network call failed, so there is no network; and the log of every
// ref in /testbed shows one commit, so there is no history. Each answer is
// judged alone.
func CheckContainerCanary(platform, network, commits string) []string {
	var problems []string
	if p := strings.TrimSpace(platform); p != "linux" {
		problems = append(problems, fmt.Sprintf("python printed %q for sys.platform, want linux: the shim does not reach the image", p))
	}
	if n := strings.TrimSpace(network); n != "blocked" {
		problems = append(problems, fmt.Sprintf("the network check printed %q, want blocked: the container has a network, or the check did not run", n))
	}
	count := 0
	for _, l := range strings.Split(commits, "\n") {
		if strings.TrimSpace(l) != "" {
			count++
		}
	}
	if count != 1 {
		problems = append(problems, fmt.Sprintf("the log of every ref in /testbed shows %d commits, want 1: the history was not removed", count))
	}
	return problems
}

// probeContainer starts a container of the image for the task, asks it the
// three questions of CheckContainerCanary through the shim, and removes it. A
// failure to ask is a problem, never a pass.
func probeContainer(o RunOptions, task Task) (problems []string) {
	_ = os.MkdirAll(o.Scratch, 0o755)
	scratch, err := os.MkdirTemp(o.Scratch, "bench-probe-")
	if err != nil {
		return []string{err.Error()}
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	tree := attemptTree(scratch)
	if err := os.Mkdir(tree, 0o755); err != nil {
		return []string{err.Error()}
	}
	base, err := Materialize(task, tree, nil)
	if err != nil {
		return []string{err.Error()}
	}
	name := containerName("canary")
	box, err := startContainer(o.Docker, imageFor(o, task), name, o.Stage, o.Budget+300*time.Second, tree, base)
	if box != nil {
		defer func() {
			if err := box.remove(); err != nil {
				problems = append(problems, err.Error())
			}
		}()
	}
	if err != nil {
		return []string{"the container did not start: " + err.Error()}
	}
	cfg := shimConfig{Bench: o.Bench, Docker: o.Docker, Container: name, Tree: tree, State: filepath.Join(scratch, "shim"), Base: base}
	// The questions go through the shims of an attempt, written into a tools
	// directory of their own and run as the agent would run them, so bench
	// itself and the script are proven as well as the container.
	tools := filepath.Join(scratch, "tools")
	if err := os.Mkdir(tools, 0o755); err != nil {
		return []string{err.Error()}
	}
	if err := writeShims(tools, cfg); err != nil {
		return []string{err.Error()}
	}
	// The probe is the start of the environment: no variable of it is forwarded.
	if err := writeStartEnv(cfg.State, os.Environ()); err != nil {
		return []string{err.Error()}
	}
	// A question the shim answered with the right output and a failing exit is
	// a failure all the same.
	var failures []string
	ask := func(argv ...string) string {
		cmd := exec.Command(filepath.Join(tools, "python"), argv...)
		cmd.Dir = tree
		out, err := cmd.Output()
		if err != nil {
			var ee *exec.ExitError
			var stderr string
			if errors.As(err, &ee) {
				stderr = strings.TrimSpace(string(ee.Stderr))
			}
			failures = append(failures, fmt.Sprintf("a question to the container failed: %v: %s", err, stderr))
		}
		return string(out)
	}
	commits, err := dockerRun(o.Docker, nil, "exec", name, "git", "-C", testbed, "log", "--all", "--oneline")
	if err != nil {
		return []string{err.Error()}
	}
	platform := ask("-c", "import sys; print(sys.platform)")
	network := ask("-c", netProbe)
	return append(CheckContainerCanary(platform, network, string(commits)), failures...)
}
