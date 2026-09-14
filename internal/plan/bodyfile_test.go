package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-060 T4: body=@path loads lines; rooted and outside-root paths refuse.
func TestBodyAtPathLoadsTheFile(t *testing.T) {
	t.Run("loads", func(t *testing.T) {
		root := t.TempDir()
		src := filepath.Join(root, "src.txt")
		if err := os.WriteFile(src, []byte("alpha\nbeta\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		hunks, err := Parse(strings.NewReader("@@ dest.txt 0 create body=@src.txt\n"))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if err := LoadBodyFiles(root, hunks); err != nil {
			t.Fatalf("LoadBodyFiles: %v", err)
		}
		want := []string{"alpha", "beta"}
		if strings.Join(hunks[0].Body, ",") != strings.Join(want, ",") {
			t.Errorf("Body = %q, want %q", hunks[0].Body, want)
		}
	})
	t.Run("rooted", func(t *testing.T) {
		root := t.TempDir()
		hunks, err := Parse(strings.NewReader("@@ dest.txt 0 create body=@/etc/hosts\n"))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		err = LoadBodyFiles(root, hunks)
		if err == nil {
			t.Fatal("rooted body=@/etc/hosts loaded")
		}
		if !strings.Contains(err.Error(), "/etc/hosts") {
			t.Errorf("error does not name the path:\n%s", err)
		}
	})
	t.Run("outside", func(t *testing.T) {
		root := t.TempDir()
		rel := filepath.Join("..", "out.txt")
		hunks, err := Parse(strings.NewReader("@@ dest.txt 0 create body=@" + rel + "\n"))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		err = LoadBodyFiles(root, hunks)
		if err == nil {
			t.Fatal("outside body=@../out loaded")
		}
		if !strings.Contains(err.Error(), rel) && !strings.Contains(err.Error(), "outside") {
			t.Errorf("error does not name the outside path:\n%s", err)
		}
	})
}
