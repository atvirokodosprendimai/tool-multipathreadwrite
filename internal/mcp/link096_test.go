package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-096 T1 over MCP `grep`: a named link to a directory is named in the
// answer with the directory to name instead. Alone it is an error; beside a
// path that is served it leaves the answer unflagged (ADR-024).
func TestMcpGrepRefusesANamedDirectoryLinkByName(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "d", "f.go"), []byte("package d\nWANTED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("d", filepath.Join(root, "dlink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"dlink"}, "grep": "WANTED"})
	all := fmt.Sprint(res["content"])
	if res["isError"] != true {
		t.Errorf("a named directory link alone is not isError: %v", res)
	}
	if !strings.Contains(all, "dlink") || !strings.Contains(all, "name d") {
		t.Errorf("the answer does not name dlink and the directory d:\n%s", all)
	}

	res = call(t, root, "mrw_read", map[string]any{"specs": []any{"d", "dlink"}, "grep": "WANTED"})
	all = fmt.Sprint(res["content"])
	if res["isError"] == true {
		t.Errorf("an answer that served d/f.go was flagged isError: %v", res)
	}
	if !strings.Contains(all, "d/f.go") {
		t.Errorf("d/f.go was not served:\n%s", all)
	}
	if !strings.Contains(all, "-- dlink:") || !strings.Contains(all, "name d") {
		t.Errorf("the served answer does not name dlink's refusal:\n%s", all)
	}
}
