package bench

import (
	"bufio"
	"bytes"
	"encoding/json"
)

// Usage is the token counts a CLI result reports.
type Usage struct {
	InputTokens      int `json:"input_tokens"`
	OutputTokens     int `json:"output_tokens"`
	CacheReadTokens  int `json:"cache_read_input_tokens"`
	CacheWriteTokens int `json:"cache_creation_input_tokens"`
}

// transcript is what bench reads from a stream-json run. The message shapes
// are the ones the Claude Agent SDK documents for `--output-format
// stream-json`: a "system"/"init" line naming the model, tools and MCP
// servers; "assistant" lines whose content may hold "tool_use" items; and a
// final "result" line with total_cost_usd, usage, num_turns and is_error.
// Not yet observed from a subscription run on this machine: the T2 canary
// records the first real one, and whether total_cost_usd is present there
// (ADR-156 Decision 4).
type transcript struct {
	InitSeen   bool
	Model      string
	Tools      []string
	MCPServers []string
	// Under ADR-162's profile: the init line's skills and plugin names, and
	// each hook that started, as event and name ("PreToolUse:Bash").
	InitSkills  []string
	InitPlugins []initPlugin
	HookEvents  []string
	HookNames   []string
	ToolCalls   int
	Commands    []string // the Bash commands the agent ran
	Result      bool
	IsError     bool
	Subtype     string
	NumTurns    int
	ResultText  string
	CostUSD     *float64
	Usage       Usage
}

func parseTranscript(b []byte) transcript {
	var t transcript
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		var m struct {
			Type       string   `json:"type"`
			Subtype    string   `json:"subtype"`
			Model      string   `json:"model"`
			Tools      []string `json:"tools"`
			MCPServers []struct {
				Name string `json:"name"`
			} `json:"mcp_servers"`
			Skills    []string     `json:"skills"`
			Plugins   []initPlugin `json:"plugins"`
			HookEvent string       `json:"hook_event"`
			HookName  string       `json:"hook_name"`
			Message   struct {
				Content []struct {
					Type  string          `json:"type"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				} `json:"content"`
			} `json:"message"`
			IsError  bool     `json:"is_error"`
			NumTurns int      `json:"num_turns"`
			Text     string   `json:"result"`
			Cost     *float64 `json:"total_cost_usd"`
			Usage    Usage    `json:"usage"`
		}
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		switch m.Type {
		case "system":
			switch m.Subtype {
			case "init":
				t.InitSeen, t.Model, t.Tools, t.InitSkills = true, m.Model, m.Tools, m.Skills
				for _, s := range m.MCPServers {
					t.MCPServers = append(t.MCPServers, s.Name)
				}
				t.InitPlugins = append(t.InitPlugins, m.Plugins...)
			case "hook_started":
				t.HookEvents, t.HookNames = append(t.HookEvents, m.HookEvent), append(t.HookNames, m.HookName)
			}
		case "assistant":
			for _, c := range m.Message.Content {
				if c.Type == "tool_use" {
					t.ToolCalls++
					if c.Name == "Bash" {
						var in struct {
							Command string `json:"command"`
						}
						if json.Unmarshal(c.Input, &in) == nil && in.Command != "" {
							t.Commands = append(t.Commands, in.Command)
						}
					}
				}
			}
		case "result":
			t.Result, t.IsError, t.Subtype, t.NumTurns, t.CostUSD, t.Usage, t.ResultText = true, m.IsError, m.Subtype, m.NumTurns, m.Cost, m.Usage, m.Text
		}
	}
	return t
}

// initPlugin is a plugin the init line lists: its name, where it was loaded
// from ("quality-harness@inline" for --plugin-dir) and its directory.
type initPlugin struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path"`
}
