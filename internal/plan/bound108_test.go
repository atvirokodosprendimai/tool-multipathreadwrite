package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-108 T4. LoadBodyFiles read a body file whole with os.ReadFile, before
// apply's capped loader. A body file over the limit is refused by its size,
// naming it and the limit; one under it loads.
func TestABodyFileIsBounded(t *testing.T) {
	old := maxBodyBytes
	t.Cleanup(func() { maxBodyBytes = old })
	maxBodyBytes = 16
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "big"), []byte(strings.Repeat("x\n", 20)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "small"), []byte("s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := LoadBodyFiles(root, []Hunk{{BodyFile: "big", SrcLine: 3}})
	if err == nil || !strings.Contains(err.Error(), "40 bytes") || !strings.Contains(err.Error(), "16") {
		t.Errorf("a body file over the limit was not refused naming size and limit: %v", err)
	}
	if err := LoadBodyFiles(root, []Hunk{{BodyFile: "small", SrcLine: 1}}); err != nil {
		t.Errorf("a body file under the limit was refused: %v", err)
	}
}
