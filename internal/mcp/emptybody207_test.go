package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-207. mrw_write refuses a delete whose body=@ file is empty with an isError
// answer naming the file, and writes nothing.
func TestMrwWriteRefusesAnEmptyBodyFileOnADelete(t *testing.T) {
	root := licensed(t, map[string]string{"f.txt": "a\nb\nc\n", "g.txt": "x\n", "empty.txt": ""})
	read := func(n string) string {
		b, _ := os.ReadFile(filepath.Join(root, n))
		return string(b)
	}
	before := read("f.txt") + "|" + read("g.txt")
	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ g.txt 1 replace\ny\n@@ f.txt 2 delete body=@empty.txt\n"})
	content, _ := res["content"].([]any)
	var text string
	if len(content) > 0 {
		first, _ := content[0].(map[string]any)
		text = asString(first["text"])
	}
	if res["isError"] != true || !strings.Contains(text, "line 3: body=@empty.txt") || read("f.txt")+"|"+read("g.txt") != before {
		t.Errorf("isError %v, text %q; want an error naming \"line 3: body=@empty.txt\" and both files untouched", res["isError"], text)
	}
}
