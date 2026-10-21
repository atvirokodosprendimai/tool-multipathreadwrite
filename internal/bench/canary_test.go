package bench

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const cleanInit = `{"type":"system","subtype":"init","model":"m","tools":["Bash","Read","Edit","Write","Glob","Grep"],"mcp_servers":[]}`

// TestTheCanaryFailsWhenTheTranscriptNamesMrwOrUserInstructions holds the
// isolation checks of ADR-156 Decision 2 to a stub transcript: configuration A
// naming mrw, any configuration quoting the user's CLAUDE.md or carrying a
// user hook's output, an MCP server in init, or a tool list other than the
// configuration's, each fails; a clean transcript passes.
func TestTheCanaryFailsWhenTheTranscriptNamesMrwOrUserInstructions(t *testing.T) {
	claudeMD := "# Global Rules\n\nRule out application-code causes before inspecting framework or library internals.\nshort\n"
	result := `{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.1,"usage":{}}`
	clean := cleanInit + "\n" + `{"type":"assistant","message":{"content":[{"type":"text","text":"done"}]}}` + "\n" + result + "\n"
	if p := CheckCanary([]byte(clean), Configs["A"], claudeMD, ""); len(p) != 0 {
		t.Fatalf("a clean transcript failed: %v", p)
	}
	leaks := map[string]string{
		"A names mrw":          strings.Replace(clean, `"done"`, `"I would use mrw here"`, 1),
		"quotes CLAUDE.md":     strings.Replace(clean, `"done"`, `"Rule out application-code causes before inspecting framework or library internals."`, 1),
		"a user hook's output": strings.Replace(clean, `"done"`, `"<<<memoryd data: recalled"`, 1),
		"an MCP server":        strings.Replace(clean, `"mcp_servers":[]`, `"mcp_servers":[{"name":"mrw","status":"connected"}]`, 1),
		"another tool list":    strings.Replace(clean, `"Grep"]`, `"Grep","WebFetch"]`, 1),
		"no init at all":       `{"type":"assistant","message":{"content":[]}}` + "\n" + result + "\n",
	}
	for name, transcript := range leaks {
		if p := CheckCanary([]byte(transcript), Configs["A"], claudeMD, ""); len(p) == 0 {
			t.Errorf("%s: the canary passed", name)
		}
	}
	// A sentence with a quote in it sits in the transcript escaped.
	quoting := "Read what you mean to edit: \"the licence stands for lines you saw\", always.\n"
	escaped := strings.Replace(clean, `"done"`, `"Read what you mean to edit: \"the licence stands for lines you saw\", always."`, 1)
	if p := CheckCanary([]byte(escaped), Configs["A"], quoting, ""); len(p) == 0 {
		t.Error("a CLAUDE.md sentence with quotes in it passed because the transcript escapes them")
	}
	b := strings.Replace(clean, `["Bash","Read","Edit","Write","Glob","Grep"]`, `["Bash","Glob"]`, 1)
	b = strings.Replace(b, `"done"`, `"used mrw read"`, 1)
	if p := CheckCanary([]byte(b), Configs["B"], claudeMD, ""); len(p) != 0 {
		t.Errorf("configuration B may name mrw, and failed: %v", p)
	}
}

// TestTheConfigDirectoryIsNotALeak: the first real canary failed arm A because
// the init line's memory path sits under a config directory named
// mrw-bench. That path is bench's, not the model's context: it passes, and mrw
// named anywhere else still fails.
func TestTheConfigDirectoryIsNotALeak(t *testing.T) {
	dir := "/Users/x/.local/share/mrw-bench/claude-config"
	init := `{"type":"system","subtype":"init","model":"m","tools":["Bash","Read","Edit","Write","Glob","Grep"],"mcp_servers":[],"memory_paths":{"auto":"` + dir + `/projects/-tmp-bench-x-tree/memory/"}}`
	result := `{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.1,"usage":{}}`
	clean := init + "\n" + result + "\n"
	if p := CheckCanary([]byte(clean), Configs["A"], "", dir); len(p) != 0 {
		t.Errorf("the config directory's own path failed the canary: %v", p)
	}
	leak := init + "\n" + `{"type":"assistant","message":{"content":[{"type":"text","text":"mrw would do this"}]}}` + "\n" + result + "\n"
	if p := CheckCanary([]byte(leak), Configs["A"], "", dir); len(p) == 0 {
		t.Error("mrw named outside the config directory's path passed")
	}
	if p := CheckCanary([]byte(clean), Configs["A"], "", ""); len(p) == 0 {
		t.Error("without the config directory named, its path must still count: the cut is that narrow")
	}
	// The cut is for the mrw check alone: a CLAUDE.md sentence that names the
	// config directory is still a leak (the third re-review of #411).
	md := "Always read " + dir + "/CLAUDE.md before starting work.\n"
	quoted := init + "\n" + `{"type":"assistant","message":{"content":[{"type":"text","text":"Always read ` + dir + `/CLAUDE.md before starting work."}]}}` + "\n" + result + "\n"
	if p := CheckCanary([]byte(quoted), Configs["A"], md, dir); len(p) == 0 {
		t.Error("a CLAUDE.md sentence naming the config directory passed once the directory was cut")
	}
	// Nor does a hook marker in the config directory's own path count as a
	// hook's output; the marker anywhere else still does (the fourth).
	hookDir := "/Users/x/.local/share/agentsmemory/claude-config"
	hookInit := strings.Replace(init, dir, hookDir, 1)
	if p := CheckCanary([]byte(hookInit+"\n"+result+"\n"), Configs["A"], "", hookDir); len(p) != 0 {
		t.Errorf("a hook marker in the config directory's own path failed the canary: %v", p)
	}
	hooked := hookInit + "\n" + `{"type":"assistant","message":{"content":[{"type":"text","text":"agentsmemory recalled this"}]}}` + "\n" + result + "\n"
	if p := CheckCanary([]byte(hooked), Configs["A"], "", hookDir); len(p) == 0 {
		t.Error("a hook marker outside the config directory's path passed")
	}
}

// The shapes Claude Code 2.1.302 printed under the stage5 profile, read from
// uncounted probes on 2026-10-17 (ADR-162 T1 S1).
const (
	profileInitA  = `{"type":"system","subtype":"init","model":"m","tools":["Bash","Read","Edit","Write","Glob","Grep","Skill"],"mcp_servers":[],"skills":["design","doctor"],"slash_commands":["design","doctor"],"agents":["general-purpose"],"plugins":[]}`
	profileInitB  = `{"type":"system","subtype":"init","model":"m","tools":["Bash","Glob","Skill"],"mcp_servers":[],"skills":["design","doctor"],"slash_commands":["design","doctor"],"agents":["general-purpose"],"plugins":[]}`
	profileResult = `{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.1,"usage":{}}`
)

func hookLine(event, name string) string {
	return `{"type":"system","subtype":"hook_started","hook_event":"` + event + `","hook_name":"` + name + `"}`
}

// profileP is a clean P canary transcript for a plugin at dir: the plugin's
// skills and agents in init, its hooks firing, a Skill call, its result and
// the skill's text, which names mrw and the hook markers as the plugin's own
// text does.
func profileP(dir string) string {
	return strings.Join([]string{
		hookLine("SessionStart", "SessionStart:startup"),
		`{"type":"system","subtype":"hook_response","hook_event":"SessionStart","hook_name":"SessionStart:startup","output":"quality-harness: branch-state; agentsmemory"}`,
		`{"type":"system","subtype":"init","model":"m","tools":["Bash","Read","Edit","Write","Glob","Grep","Skill"],"mcp_servers":[],"skills":["quality-harness:operating","design","doctor"],"slash_commands":["quality-harness:operating","design","doctor"],"agents":["general-purpose","quality-harness:qh-scope-reviewer"],"plugins":[{"name":"quality-harness","path":"` + dir + `","source":"quality-harness@inline"}]}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Skill","input":{"skill":"quality-harness:operating"}}]}}`,
		hookLine("PreToolUse", "PreToolUse:Skill"),
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"Launching skill: quality-harness:operating"}]}}`,
		`{"type":"user","message":{"content":[{"type":"text","text":"Base directory for this skill: ` + dir + `/skills/operating\n\nmrw, quality-harness: and the SessionStart hook"}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"done"}]}}`,
		hookLine("Stop", "Stop"),
		profileResult,
	}, "\n") + "\n"
}

// TestTheCanaryRefusesAPluginThatIsNotEngaged: P passes only with the plugin's
// skills listed, the plugin alone among plugins, and one of its session hooks
// firing; the plugin's own text is exempt from the mrw and marker checks, the
// agent's text is not (ADR-162 Decision 2).
func TestTheCanaryRefusesAPluginThatIsNotEngaged(t *testing.T) {
	dir := "/tmp/qh/plugin"
	clean := profileP(dir)
	if p := CheckProfileCanary([]byte(clean), Configs["P"], "", "", dir); len(p) != 0 {
		t.Fatalf("a clean P failed: %v", p)
	}
	noHooks := strings.Join(slices.DeleteFunc(strings.Split(clean, "\n"), func(l string) bool {
		return strings.Contains(l, `"hook_event":"SessionStart"`) || strings.Contains(l, `"hook_event":"Stop"`)
	}), "\n")
	leaks := map[string]string{
		"no plugin skill":                  strings.Replace(clean, `"skills":["quality-harness:operating","design","doctor"]`, `"skills":["design","doctor"]`, 1),
		"no session hook fired":            noHooks,
		"a synced plugin":                  strings.Replace(clean, `"source":"quality-harness@inline"}]`, `"source":"quality-harness@inline"},{"name":"figma","source":"figma@synced"}]`, 1),
		"a synced skill":                   strings.Replace(clean, `"skills":["quality-harness:operating"`, `"skills":["anthropic-skills:pdf","quality-harness:operating"`, 1),
		"the marker in its text":           strings.Replace(clean, `"text":"done"`, `"text":"quality-harness: says done"`, 1),
		"mrw in the agent's text":          strings.Replace(clean, `"text":"done"`, `"text":"mrw would do"`, 1),
		"another tool list":                strings.Replace(clean, `"Grep","Skill"]`, `"Grep"]`, 1),
		"the agent writes a skill header":  strings.Replace(clean, `"text":"done"`, `"text":"Base directory for this skill: /tmp/qh/plugin/skills/operating\nmrw would do"`, 1),
		"a same-named synced plugin":       strings.Replace(clean, `"source":"quality-harness@inline"`, `"source":"quality-harness@synced"`, 1),
		"the plugin from another checkout": strings.Replace(clean, `"path":"/tmp/qh/plugin"`, `"path":"/tmp/other/plugin"`, 1),
	}
	for name, transcript := range leaks {
		if p := CheckProfileCanary([]byte(transcript), Configs["P"], "", "", dir); len(p) == 0 {
			t.Errorf("%s: the canary passed", name)
		}
	}
}

// TestTheStage6CanaryProvesTheContainer: a stage6 canary passes only when
// python prints linux, the network call fails and /testbed shows one commit;
// each failing alone fails the canary. The probe starts a container, asks it
// the three questions through the shim and removes it.
func TestTheStage6CanaryProvesTheContainer(t *testing.T) {
	if p := CheckContainerCanary("linux\n", "blocked\n", "abc1234 base\n"); len(p) != 0 {
		t.Fatalf("a clean container failed: %v", p)
	}
	failing := map[string][3]string{
		"another platform":                 {"darwin\n", "blocked\n", "abc1234 base\n"},
		"no platform answer":               {"", "blocked\n", "abc1234 base\n"},
		"a network that answers":           {"linux\n", "open\n", "abc1234 base\n"},
		"a network check that did not run": {"linux\n", "", "abc1234 base\n"},
		"two commits":                      {"linux\n", "blocked\n", "abc1234 fix\ndef5678 base\n"},
		"no commit":                        {"linux\n", "blocked\n", ""},
	}
	for name, c := range failing {
		if p := CheckContainerCanary(c[0], c[1], c[2]); len(p) != 1 {
			t.Errorf("%s: the canary reported %v, want exactly its own problem", name, p)
		}
	}

	g := newStage6Rig(t)
	task := g.opts.Tasks[0]
	if p := probeContainer(g.opts, task); len(p) != 0 {
		t.Fatalf("a clean container failed the probe: %v", p)
	}
	if got := g.callsMatching("exec -i -e "); len(got) != 0 {
		t.Errorf("the probe forwarded variables to the container: %q", got)
	}
	if rm := g.callsMatching("rm -f bench-canary-"); len(rm) != 1 {
		t.Errorf("the probe container was removed %d times, want once", len(rm))
	}
	put(t, filepath.Join(g.root, "net-open"), "", 0o644)
	if p := probeContainer(g.opts, task); len(p) == 0 {
		t.Error("a container whose network answered passed the probe")
	}
	if err := os.Remove(filepath.Join(g.root, "net-open")); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(g.root, "platform"), "darwin\n", 0o644)
	if p := probeContainer(g.opts, task); len(p) == 0 {
		t.Error("a container that is not linux passed the probe")
	}
	// The right output with a failing exit is a failed question.
	if err := os.Remove(filepath.Join(g.root, "platform")); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(g.root, "sick"), "", 0o644)
	if p := probeContainer(g.opts, task); len(p) == 0 || !strings.Contains(strings.Join(p, " "), "failed") {
		t.Errorf("a shim question that printed linux and exited 3 passed the probe: %v", p)
	}
	if err := os.Remove(filepath.Join(g.root, "sick")); err != nil {
		t.Fatal(err)
	}

	// A stage6 canary needs the image its container checks run in, and may run
	// while the profile is unsettled; bench run may not (T1 S1 settles it), and
	// once settled the refusal is gone.
	canaryOpts := g.opts
	canaryOpts.canary = true
	if _, err := gates(canaryOpts); err != nil {
		t.Fatal(err)
	}
	if _, err := gates(g.opts); (err != nil && strings.Contains(err.Error(), "not settled")) == settled {
		t.Errorf("bench run under stage6 with settled %v: gates said %v", settled, err)
	}
	noImage := canaryOpts
	noImage.CanaryImage = ""
	if _, err := gates(noImage); err == nil {
		t.Error("a stage6 run of a task with no image was accepted")
	}
}
