package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Review of #736. mrw_write with format apply_patch compiles before the lock; if
// another writer's change lands in between (ingest.AfterCompile), the compiled
// sha= must reach the validator and refuse, rather than replacing the wrong line.
func TestTheWriteToolForwardsTheCompiledShaToTheLock(t *testing.T) {
	root := licensed(t, map[string]string{"f.txt": "l1\nl2\nl3\nl4\nl5\n"})
	f := filepath.Join(root, "f.txt")
	const other = "NEW\nl1\nl2\nl3\nl4\nl5\n"
	ingest.AfterCompile = func() {
		if err := os.WriteFile(f, []byte(other), 0o644); err != nil {
			t.Error(err)
		}
		if err := seen.Record(root, map[string]seen.Observation{"f.txt": {SHA: seen.SHA([]byte(other)), Written: true}}); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { ingest.AfterCompile = nil })
	res := call(t, root, "mrw_write", map[string]any{"format": "apply_patch", "plan": "*** Begin Patch\n*** Update File: f.txt\n@@\n-l3\n+L3\n*** End Patch\n"})
	got, _ := os.ReadFile(f)
	content, _ := res["content"].([]any)
	var text string
	if len(content) > 0 {
		first, _ := content[0].(map[string]any)
		text = asString(first["text"])
	}
	if res["isError"] != true || string(got) != other || !strings.Contains(text, "file changed") {
		t.Errorf("isError %v, file now %q; want a refusal naming the change and the other writer's file untouched:\n%s", res["isError"], got, text)
	}
}
