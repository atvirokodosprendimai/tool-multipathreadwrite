package bench

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	HostReads   int      // the Read tool calls: ADR-192's measure of H's uptake
	Result      bool
	IsError     bool
	Subtype     string
	NumTurns    int
	ResultText  string
	CostUSD     *float64
	Usage       Usage
}

// Keep reading after a damaged event: a later result can still report real spend.
func parseTranscript(b []byte) (transcript, error) {
	var t transcript
	var firstErr error
	line := 0
	for raw := range bytes.Lines(b) {
		line++
		if err := t.readEvent(raw); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("transcript line %d: %w", line, err)
		}
	}
	return t, firstErr
}

func (t *transcript) readEvent(raw []byte) error {
	if len(raw) > 1<<26 {
		return errors.New("event exceeds 64 MiB")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var event struct {
		Type, Subtype string
		Cost          json.RawMessage `json:"total_cost_usd"`
	}
	if err := json.Unmarshal(raw, &event); err != nil {
		return err
	}
	switch event.Type {
	case "":
		return errors.New("expected an event with a type")
	case "system":
		switch event.Subtype {
		case "init":
			var m struct {
				Model      string   `json:"model"`
				Tools      []string `json:"tools"`
				MCPServers []struct {
					Name string `json:"name"`
				} `json:"mcp_servers"`
				Skills  []string     `json:"skills"`
				Plugins []initPlugin `json:"plugins"`
			}
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			t.InitSeen, t.Model, t.Tools, t.InitSkills = true, m.Model, m.Tools, m.Skills
			for _, s := range m.MCPServers {
				t.MCPServers = append(t.MCPServers, s.Name)
			}
			t.InitPlugins = append(t.InitPlugins, m.Plugins...)
		case "hook_started":
			var m struct {
				Event string `json:"hook_event"`
				Name  string `json:"hook_name"`
			}
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			t.HookEvents, t.HookNames = append(t.HookEvents, m.Event), append(t.HookNames, m.Name)
		}
	case "assistant":
		var m struct {
			Message struct {
				Content []struct {
					Type  string          `json:"type"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			return err
		}
		for _, c := range m.Message.Content {
			if c.Type == "tool_use" {
				t.ToolCalls++
				if c.Name == "Read" {
					t.HostReads++
				}
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
		// Spend is a budget input even if another result field cannot be read.
		var cost *float64
		if event.Cost != nil {
			if err := json.Unmarshal(event.Cost, &cost); err != nil {
				return err
			}
		}
		t.CostUSD = cost
		var m struct {
			IsError  bool   `json:"is_error"`
			NumTurns int    `json:"num_turns"`
			Text     string `json:"result"`
			Usage    Usage  `json:"usage"`
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			return err
		}
		t.Result, t.IsError, t.Subtype, t.NumTurns, t.Usage, t.ResultText = true, m.IsError, event.Subtype, m.NumTurns, m.Usage, m.Text
	}
	return nil
}

// initPlugin is a plugin the init line lists: its name, where it was loaded
// from ("quality-harness@inline" for --plugin-dir) and its directory.
type initPlugin struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path"`
}
