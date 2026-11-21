package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-215 Decision 6. mrw_write and the CLI are one engine: a rooted plan path
// is an isError answer with the structured receipt, the rooted row failed with
// the sentence and the valid sibling skipped, and nothing is written.
func TestMrwWriteRefusesARootedPathLikeTheCLI(t *testing.T) {
	for _, c := range []struct {
		name string
		plan func(root string) string
	}{
		{"a replace", func(string) string { return "@@ /dd/zz.txt 1 replace\nX\n@@ fresh.txt 0 create\nY\n" }},
		{"a replace of an absolute path inside the root", func(root string) string {
			return "@@ \"" + filepath.Join(root, "dd", "zz.txt") + "\" 1 replace\nX\n@@ fresh.txt 0 create\nY\n"
		}},
		{"a create", func(string) string { return "@@ /new/x 0 create\nX\n@@ fresh.txt 0 create\nY\n" }},
	} {
		t.Run(c.name, func(t *testing.T) {
			root, _ := checkout(t, "ok.txt", "a\nb\nc\n")
			zz := filepath.Join(root, "dd", "zz.txt")
			if err := os.MkdirAll(filepath.Dir(zz), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(zz, []byte("a\nb\nc\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			res := call(t, root, "mrw_write", map[string]any{"plan": c.plan(root), "check": false})
			if res["isError"] != true {
				t.Fatalf("not an error: %v", res)
			}
			sc := structured(t, res)
			hunks, _ := sc["hunks"].([]any)
			if len(hunks) != 2 {
				t.Fatalf("hunks %v, want the rooted row and its sibling", sc["hunks"])
			}
			first, _ := hunks[0].(map[string]any)
			second, _ := hunks[1].(map[string]any)
			reason, _ := first["reason"].(string)
			if first["status"] != "failed" || !strings.Contains(reason, "is absolute, and every path in a plan is relative to the root") {
				t.Errorf("the rooted row: %v", first)
			}
			if second["status"] != "skipped" {
				t.Errorf("the sibling: %v", second)
			}
			for _, gone := range []string{"fresh.txt", "new"} {
				if _, err := os.Stat(filepath.Join(root, gone)); err == nil {
					t.Errorf("%s was made", gone)
				}
			}
			if b, _ := os.ReadFile(zz); string(b) != "a\nb\nc\n" {
				t.Errorf("dd/zz.txt changed: %q", b)
			}
		})
	}
}
