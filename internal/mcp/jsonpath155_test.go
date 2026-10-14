package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-155. mrw_read takes a JSON path in its specs, through the same parser,
// and serves the same span.
func TestAnMCPReadTakesAJSONPath(t *testing.T) {
	root := t.TempDir()
	doc := "{\n  \"name\": \"demo\",\n  \"scripts\": {\n    \"build\": \"go build\"\n  }\n}\n"
	if err := os.WriteFile(filepath.Join(root, "f.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"f.json:$.scripts.build"}})
	text := resultText(res)
	if isErr, _ := res["isError"].(bool); isErr || !strings.Contains(text, "@@ 4-4") || strings.Contains(text, "demo") {
		t.Errorf("mrw_read of f.json:$.scripts.build did not serve line 4 alone:\n%s", text)
	}
}
