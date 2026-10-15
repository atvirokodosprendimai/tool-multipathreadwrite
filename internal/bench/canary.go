package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
	Costs      map[string]*float64 `json:"total_cost_usd"`
}

// RunCanary runs the task once in each configuration and writes canary.json
// into o.Out. It needs the same gates as a stage, except a canary.
func RunCanary(o RunOptions, task Task, claudeMD string) (CanaryReceipt, error) {
	var c CanaryReceipt
	o.Canary = "-"
	cd, err := gates(o)
	if err != nil {
		return c, err
	}
	o.ConfigDir = cd
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
		Problems: map[string][]string{}, Costs: map[string]*float64{}}
	for i, name := range []string{"A", "B"} {
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
		problems = append(problems, CheckCanary(b, Configs[name], claudeMD, o.ConfigDir)...)
		if len(problems) > 0 {
			c.Pass = false
			c.Problems[name] = problems
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
