package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-060 T7: CLI replace/insert body=@ writes those bytes. Today parse
// refuses empty Body before Load.
func TestWriteBodyAtPathEditsFromFile(t *testing.T) {
	t.Run("replace", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := grepTree(t, map[string]string{
			"notes.md": "# notes\nline two\n",
			"src.txt":  "line 2\n",
		})
		if _, err := readIn(t, root, "notes.md"); err != nil {
			t.Fatal(err)
		}
		doc := "@@ notes.md 2 replace anchor=\"line two\" body=@src.txt\n"
		out, code := writeIn(t, root, "--no-check", planFile(t, doc))
		if code != 0 {
			t.Fatalf("replace body=@src.txt exited %d:\n%s", code, out)
		}
		got, err := os.ReadFile(filepath.Join(root, "notes.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "line 2") || strings.Contains(string(got), "line two") {
			t.Errorf("notes.md = %q, want line 2 from src.txt", got)
		}
	})
	t.Run("insert-after", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := grepTree(t, map[string]string{
			"notes.md": "alpha\nbeta\n",
			"src.txt":  "INSERTED\n",
		})
		if _, err := readIn(t, root, "notes.md"); err != nil {
			t.Fatal(err)
		}
		doc := "@@ notes.md 1 insert-after body=@src.txt\n"
		out, code := writeIn(t, root, "--no-check", planFile(t, doc))
		if code != 0 {
			t.Fatalf("insert-after body=@src.txt exited %d:\n%s", code, out)
		}
		got, err := os.ReadFile(filepath.Join(root, "notes.md"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "alpha\nINSERTED\nbeta\n" && string(got) != "alpha\nINSERTED\nbeta" {
			t.Errorf("notes.md = %q, want INSERTED after alpha", got)
		}
	})
}
