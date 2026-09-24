package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A file whose name holds a space was refused over MCP with -32603 "holding
// checkpoints: no observation for x": splitServed took the served header's
// path as everything before its first space, so `x y.txt` became `x`. Found by
// the chaos harness's MCP suite on 2026-09-24; present since ADR-039 (v1.11.0).
// The CLI served the same file fine. The pair: a name with two spaces in a row,
// which the header's own two-space separator must not split either.
func TestAFileWithASpaceInItsNameIsServedAndLicensedOverMCP(t *testing.T) {
	for _, name := range []string{"x y.txt", "My  Notes.txt"} {
		root, _ := checkout(t, name, "hello\nworld\n")
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{name}})
		text := served0(t, res)
		if !strings.Contains(text, "    1| hello") {
			t.Fatalf("%q: the read did not serve the file:\n%s", name, text)
		}
		ack := checkpointsIn(text)
		if len(ack) == 0 {
			t.Fatalf("%q: the served read carries no checkpoint to acknowledge:\n%s", name, text)
		}
		// A spaced path is double-quoted in a plan header (plan.splitHeader).
		w := call(t, root, "mrw_write", map[string]any{
			"plan": `@@ "` + name + `" 1 replace` + "\nHELLO\n", "ack": ack})
		if w["isError"] == true {
			t.Fatalf("%q: a write to an acknowledged line was refused: %v", name, w["content"])
		}
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !strings.HasPrefix(string(b), "HELLO\n") {
			t.Errorf("%q: the write did not land (err %v): %q", name, err, b)
		}
	}
}
