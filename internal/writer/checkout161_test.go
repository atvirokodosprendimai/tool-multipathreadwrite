package writer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ADR-161 Decision 14, ADR-127 Decision 2. The write counter is the
// checkout's: a write from a subdirectory moves the count the top reads, which
// is what a check running at the top compares to name writers in the checkout.
func TestAWriteFromASubdirectoryMovesTheCheckoutsCounter(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	top := t.TempDir()
	sub := filepath.Join(top, "sub")
	if err := os.MkdirAll(filepath.Join(top, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	start := Writes(top)
	if _, err := Apply(sub, []apply.Input{{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1}}, apply.Options{Force: true}); err != nil {
		t.Fatal(err)
	}
	if got := Writes(top) - start; got != 1 {
		t.Errorf("a write from sub moved the top's counter by %d, want 1", got)
	}
	if Writes(sub) != Writes(top) {
		t.Errorf("sub counts %d writes and the top %d: the counter is not shared", Writes(sub), Writes(top))
	}
}
