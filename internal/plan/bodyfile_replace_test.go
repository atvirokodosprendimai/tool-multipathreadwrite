package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-060 T7: replace/insert body=@ must parse; Load fills Body. An inline
// empty replace still refuses — BodyFile is the exception, not CountedBody
// (replace body=0 would otherwise parse).
func TestBodyAtPathReplaceParsesAndLoads(t *testing.T) {
	writeSrc := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "src.txt"), []byte("alpha\nbeta\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}

	t.Run("replace", func(t *testing.T) {
		root := writeSrc(t)
		hunks, err := Parse(strings.NewReader("@@ dest.txt 1 replace body=@src.txt\n"))
		if err != nil {
			t.Fatalf("replace body=@src.txt must parse: %v", err)
		}
		if hunks[0].BodyFile != "src.txt" {
			t.Fatalf("BodyFile = %q, want src.txt", hunks[0].BodyFile)
		}
		if err := LoadBodyFiles(root, hunks); err != nil {
			t.Fatalf("LoadBodyFiles: %v", err)
		}
		if strings.Join(hunks[0].Body, ",") != "alpha,beta" {
			t.Errorf("Body = %q, want alpha,beta from src.txt", hunks[0].Body)
		}
	})
	t.Run("insert-after", func(t *testing.T) {
		root := writeSrc(t)
		hunks, err := Parse(strings.NewReader("@@ dest.txt 1 insert-after body=@src.txt\n"))
		if err != nil {
			t.Fatalf("insert-after body=@src.txt must parse: %v", err)
		}
		if err := LoadBodyFiles(root, hunks); err != nil {
			t.Fatalf("LoadBodyFiles: %v", err)
		}
		if strings.Join(hunks[0].Body, ",") != "alpha,beta" {
			t.Errorf("Body = %q, want alpha,beta from src.txt", hunks[0].Body)
		}
	})
	t.Run("insert-before", func(t *testing.T) {
		root := writeSrc(t)
		hunks, err := Parse(strings.NewReader("@@ dest.txt 1 insert-before body=@src.txt\n"))
		if err != nil {
			t.Fatalf("insert-before body=@src.txt must parse: %v", err)
		}
		if err := LoadBodyFiles(root, hunks); err != nil {
			t.Fatalf("LoadBodyFiles: %v", err)
		}
		if strings.Join(hunks[0].Body, ",") != "alpha,beta" {
			t.Errorf("Body = %q, want alpha,beta from src.txt", hunks[0].Body)
		}
	})
	t.Run("empty_inline", func(t *testing.T) {
		if _, err := Parse(strings.NewReader("@@ dest.txt 1 replace\n")); err == nil {
			t.Fatal("replace with no body parsed; BodyFile is the exception, not any empty Body")
		}
	})
	t.Run("body_zero", func(t *testing.T) {
		if _, err := Parse(strings.NewReader("@@ dest.txt 1 replace body=0\n")); err == nil {
			t.Fatal("replace body=0 parsed; CountedBody must not skip the empty-body refuse")
		}
	})
}
