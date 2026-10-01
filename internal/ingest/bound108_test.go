package ingest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-108 T4. The foreign-format compilers read the target of a document whole
// with os.ReadFile, before apply's capped loader, so a short request could
// allocate any repository file. targetBytes refuses a file over the limit by
// its size, naming size and limit, reads through a bound, and reads a file
// under the limit whole.
func TestForeignFormatsAreBounded(t *testing.T) {
	old := maxTargetBytes
	t.Cleanup(func() { maxTargetBytes = old })
	maxTargetBytes = 16
	root := t.TempDir()
	big := filepath.Join(root, "big")
	if err := os.WriteFile(big, []byte(strings.Repeat("x\n", 20)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := targetBytes(big); err == nil || !strings.Contains(err.Error(), "40 bytes") || !strings.Contains(err.Error(), "16") {
		t.Errorf("a target over the limit was not refused naming size and limit: %v", err)
	}
	huge := filepath.Join(root, "huge")
	if err := os.WriteFile(huge, []byte(strings.Repeat("h", 4<<20)), 0o644); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, herr := targetBytes(huge)
	runtime.ReadMemStats(&after)
	if herr == nil {
		t.Error("a 4 MB target past a 16-byte limit was read")
	}
	if d := after.TotalAlloc - before.TotalAlloc; d > 256<<10 {
		t.Errorf("refusing a 4 MB target allocated %d bytes, want under 256 KB", d)
	}
	small := filepath.Join(root, "small")
	if err := os.WriteFile(small, []byte("s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, err := targetBytes(small); err != nil || string(b) != "s\n" {
		t.Errorf("a target under the limit read %q, %v", b, err)
	}
}
