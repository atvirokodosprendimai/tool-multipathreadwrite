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
	ToolCalls  int
	Result     bool
	IsError    bool
	Subtype    string
	NumTurns   int
	CostUSD    *float64
	Usage      Usage
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
			Message struct {
				Content []struct {
					Type string `json:"type"`
				} `json:"content"`
			} `json:"message"`
			IsError  bool     `json:"is_error"`
			NumTurns int      `json:"num_turns"`
			Cost     *float64 `json:"total_cost_usd"`
			Usage    Usage    `json:"usage"`
		}
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		switch m.Type {
		case "system":
			if m.Subtype == "init" {
				t.InitSeen, t.Model, t.Tools = true, m.Model, m.Tools
				for _, s := range m.MCPServers {
					t.MCPServers = append(t.MCPServers, s.Name)
				}
			}
		case "assistant":
			for _, c := range m.Message.Content {
				if c.Type == "tool_use" {
					t.ToolCalls++
				}
			}
		case "result":
			t.Result, t.IsError, t.Subtype, t.NumTurns, t.CostUSD, t.Usage = true, m.IsError, m.Subtype, m.NumTurns, m.Cost, m.Usage
		}
	}
	return t
}
