package mcp

import (
	"strings"
	"testing"
)

// ADR-198 Decision 7, on mrw_write. A write that stopped at its commit returns
// an error that begins with the file as the platform spells it; the structured
// `error` and the text's `error:` line both begin as the row does. The
// separator is set to `\` and the plan names a file with a backslash in it
// (q\n.txt) beside a path under it, so the commit meets `q\n.txt` already made.
func TestAStoppedWritesErrorBeginsAsTheRowDoesOnMCP(t *testing.T) {
	prev := receiptSep
	receiptSep = '\\'
	t.Cleanup(func() { receiptSep = prev })
	root, _ := checkout(t, "a.txt", "one\n")

	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ q\\n.txt 0 create\nx\n@@ q\\n.txt/x 0 create\ny\n"})
	got := structured(t, res)
	const want = "q/n.txt appeared before commit"
	if e, _ := got["error"].(string); !strings.HasPrefix(e, want) {
		t.Errorf("the structured error %q, want it to begin %q", e, want)
	}
	content, _ := res["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("no content: %v", res)
	}
	block, _ := content[0].(map[string]any)
	text, _ := block["text"].(string)
	var line string
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, "error: ") {
			line = strings.TrimPrefix(l, "error: ")
		}
	}
	if !strings.HasPrefix(line, want) {
		t.Errorf("the text's error line %q, want it to begin %q\n%s", line, want, text)
	}
}
