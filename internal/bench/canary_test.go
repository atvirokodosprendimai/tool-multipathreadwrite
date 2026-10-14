package bench

import (
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
