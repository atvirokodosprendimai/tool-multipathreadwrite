package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-108 T8. firstPage counts a file's lines before it pages a read too large
// for the ceiling, and the count read the joined path whole: any size, any
// link. A read that read itself refused can reach it, because the refusal can
// overflow a small ceiling too. The count now opens what read would serve.
func TestAPageCountIsBoundedAndConfined(t *testing.T) {
	root, _ := checkout(t, "big.txt", strings.Repeat("abcd\n", 8))
	old := maxCountBytes
	t.Cleanup(func() { maxCountBytes = old })
	maxCountBytes = 16
	if _, err := countFileLines(root, "big.txt"); err == nil || !strings.Contains(err.Error(), "40 bytes") {
		t.Errorf("a file over the bound was counted, or refused without its size: %v", err)
	}
	maxCountBytes = old

	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err == nil {
		if _, err := countFileLines(root, "link.txt"); err == nil {
			t.Error("a link out of the root was counted")
		}
	}

	if err := os.WriteFile(filepath.Join(root, "cr.txt"), []byte("a\rb\rc\r"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n, err := countFileLines(root, "cr.txt"); err != nil || n != 3 {
		t.Errorf("a CR-only file of three lines counted %d, %v", n, err)
	}
}
