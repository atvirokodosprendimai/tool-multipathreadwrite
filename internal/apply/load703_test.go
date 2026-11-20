package apply

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// dirReadCause is the error text a read of a directory ends with: Windows
// answers ReadFile on a directory "Incorrect function", unix "is a directory".
var dirReadCause = map[bool]string{true: "Incorrect function", false: "is a directory"}[runtime.GOOS == "windows"]

// TestALoadFailureKeepsEveryVerdict: a plan naming a directory beside a valid
// hunk returned its error before any verdict was assembled, so the receipt
// held no hunks at all (#703). The error and exit 2 stay; the directory's
// hunk fails with it, every other hunk is skipped, and nothing is written.
func TestALoadFailureKeepsEveryVerdict(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	in := []Input{
		{Path: "ok.txt", Op: "create", Body: []string{"ok"}, Lines: -1},
		{Path: "d", Op: "replace", Start: 1, End: 1, Body: []string{"z"}, Lines: -1},
	}
	res, err := Apply(root, in, Options{Force: true})
	if err == nil || !strings.HasPrefix(err.Error(), "d: ") {
		t.Fatalf("the error is %v, want the directory's load failure", err)
	}
	if len(res.Hunks) != 2 {
		t.Fatalf("the receipt holds %d verdicts, want 2: %+v", len(res.Hunks), res.Hunks)
	}
	if h := res.Hunks[1]; h.Path != "d" || h.Status != StatusFailed || !strings.Contains(h.Reason, dirReadCause) {
		t.Errorf("the directory's verdict is %+v, want failed with its load error", h)
	}
	if h := res.Hunks[0]; h.Status != StatusSkipped {
		t.Errorf("the sibling's verdict is %s, want skip", h.Status)
	}
	if res.Applied {
		t.Error("the receipt says applied")
	}
	if _, err := os.Stat(filepath.Join(root, "ok.txt")); !os.IsNotExist(err) {
		t.Errorf("the sibling was written: %v", err)
	}
}
