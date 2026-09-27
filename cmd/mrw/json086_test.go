package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-086 T4. encoding/json turns a byte that is not valid UTF-8 into U+FFFD,
// so once the plan header kept the byte, a --json receipt on a filesystem that
// holds it named a file other than the one written. A --json plan whose path or
// rename destination is not valid UTF-8 is refused before anything lands; the
// same plan without the invalid byte applies.
func TestAJSONPlanWithANameItCannotRepresentWritesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readIn(t, root, "b.txt"); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []string{"@@ bad\xffname.txt 0 create\nx\n", "@@ b.txt - rename\nbad\xffname.txt\n"} {
		out, code := writeIn(t, root, "--no-check", "--json", planFile(t, doc))
		if code != 2 || !strings.Contains(out, "not valid UTF-8") {
			t.Errorf("--json plan %q: exit %d, want 2 naming the byte:\n%s", doc, code, out)
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "b.txt" {
			t.Errorf("--json plan %q changed the tree: %v", doc, entries)
		}
	}
	if out, code := writeIn(t, root, "--no-check", "--json", planFile(t, "@@ ok.txt 0 create\nx\n")); code != 0 {
		t.Errorf("a --json create with a valid name: exit %d, want 0\n%s", code, out)
	}
}
