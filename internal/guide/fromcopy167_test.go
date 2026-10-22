package guide_test

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp"
)

// ADR-167 T3. A copy and a move are taught as a recipe (commands only, as
// ADR-161's review required) and by one sentence on every surface that states
// the plan format.
func TestInstructionsTeachCopyAndMove(t *testing.T) {
	full := guide.CLI()

	t.Run("a recipe copies a range and deletes the same range", func(t *testing.T) {
		m := regexp.MustCompile(`(?m)^10\. [^\n]*:\n  (mrw [^\n]*)$`).FindStringSubmatch(full)
		if m == nil {
			t.Fatalf("mrw instructions has no recipe 10 shaped label line, then an indented mrw command:\n%s", full)
		}
		cmd := m[1]
		from := regexp.MustCompile(`from=(\S+):(\d+-\d+)`).FindStringSubmatch(cmd)
		if from == nil {
			t.Fatalf("recipe 10 carries no from=PATH:N-M: %s", cmd)
		}
		if !strings.Contains(cmd, "@@ "+from[1]+" "+from[2]+" delete") {
			t.Errorf("recipe 10 does not delete the range it copies (%s %s): %s", from[1], from[2], cmd)
		}
		if !strings.Contains(cmd, "mrw read") || !strings.Contains(cmd, "mrw write -") {
			t.Errorf("recipe 10 does not read the lines before the write that copies them: %s", cmd)
		}
	})

	t.Run("the plan reference names from=PATH:ADDR and the read it needs", func(t *testing.T) {
		if !strings.Contains(full, "from=PATH:ADDR") || !strings.Contains(full, "refused unless you were served them") {
			t.Errorf("mrw instructions does not teach from=PATH:ADDR and its licence")
		}
	})

	t.Run("the core stays under ADR-141's bound", func(t *testing.T) {
		if n := len(strings.Fields(guide.Core())); n > 300 {
			t.Errorf("the core is %d words, want at most 300", n)
		}
	})

	t.Run("the MCP plan description names it", func(t *testing.T) {
		var out bytes.Buffer
		req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"
		if err := mcp.Serve(strings.NewReader(req), &out, t.TempDir()); err != nil {
			t.Fatal(err)
		}
		// The plan argument's own description: the output schema mentions the
		// receipt keys, so a bare search of the whole answer would pass without it.
		var resp struct {
			Result struct {
				Tools []struct {
					Name        string `json:"name"`
					InputSchema struct {
						Properties map[string]struct {
							Description string `json:"description"`
						} `json:"properties"`
					} `json:"inputSchema"`
				} `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
			t.Fatalf("tools/list: %v\n%s", err, out.String())
		}
		for _, tool := range resp.Result.Tools {
			if tool.Name == "mrw_write" {
				if d := tool.InputSchema.Properties["plan"].Description; !strings.Contains(d, "from=PATH:ADDR") {
					t.Errorf("the mrw_write plan description does not name from=PATH:ADDR")
				}
				return
			}
		}
		t.Error("tools/list has no mrw_write")
	})
}
