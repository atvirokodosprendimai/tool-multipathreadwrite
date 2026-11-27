package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
)

// ADR-215 Decision 6. mrw_write and the CLI are one engine: a rooted plan path
// is an isError answer with the structured receipt, the rooted row failed with
// the sentence and the valid sibling skipped, and nothing is written.
func TestMrwWriteRefusesARootedPathLikeTheCLI(t *testing.T) {
	for _, c := range []struct {
		name    string
		plan    func(root string) string
		pointer bool // the working set holds /dd/zz.txt and the plan names it as @1
	}{
		{"a replace", func(string) string { return "@@ /dd/zz.txt 1 replace\nX\n@@ fresh.txt 0 create\nY\n" }, false},
		{"a replace of an absolute path inside the root", func(root string) string {
			return "@@ \"" + filepath.Join(root, "dd", "zz.txt") + "\" 1 replace\nX\n@@ fresh.txt 0 create\nY\n"
		}, false},
		{"a create", func(string) string { return "@@ /new/x 0 create\nX\n@@ fresh.txt 0 create\nY\n" }, false},
		// ADR-215 Decision 5: mrw_write expands @N as the CLI does, and the refusal says which entry holds the spelling.
		{"an expanded working-set pointer", func(string) string { return "@@ @1 1 replace\nX\n@@ fresh.txt 0 create\nY\n" }, true},
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
			if c.pointer {
				var set iter.Set
				set.Add("/dd/zz.txt")
				if err := iter.Save(root, set); err != nil {
					t.Fatal(err)
				}
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
			if c.pointer && !strings.Contains(reason, "the working-set entry @1 holds it") {
				t.Errorf("the rooted row does not name the entry that holds it: %q", reason)
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

	// A rooted SECOND hunk: each row carries its own plan_line, address, verdict
	// and reason, rows in plan order, and nothing is written (the Codex review of #800).
	t.Run("a rooted second hunk", func(t *testing.T) {
		root, _ := checkout(t, "ok.txt", "a\nb\nc\n")
		zz := filepath.Join(root, "dd", "zz.txt")
		if err := os.MkdirAll(filepath.Dir(zz), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(zz, []byte("a\nb\nc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ fresh.txt 0 create\nY\n@@ /dd/zz.txt 1 replace\nX\n", "check": false})
		if res["isError"] != true {
			t.Fatalf("not an error: %v", res)
		}
		hunks, _ := structured(t, res)["hunks"].([]any)
		if len(hunks) != 2 {
			t.Fatalf("hunks %v, want two rows", hunks)
		}
		valid, _ := hunks[0].(map[string]any)
		bad, _ := hunks[1].(map[string]any)
		if valid["path"] != "fresh.txt" || valid["plan_line"] != float64(1) || valid["addr"] != "0" || valid["status"] != "skipped" {
			t.Errorf("the valid first row: %v", valid)
		}
		reason, _ := bad["reason"].(string)
		if bad["plan_line"] != float64(3) || bad["addr"] != "1" || bad["status"] != "failed" || !strings.Contains(reason, "/dd/zz.txt is absolute, and every path in a plan is relative to the root") {
			t.Errorf("the rooted second row: %v", bad)
		}
		if _, err := os.Stat(filepath.Join(root, "fresh.txt")); err == nil {
			t.Error("fresh.txt was made")
		}
		if b, _ := os.ReadFile(zz); string(b) != "a\nb\nc\n" {
			t.Errorf("dd/zz.txt changed: %q", b)
		}
	})
}
