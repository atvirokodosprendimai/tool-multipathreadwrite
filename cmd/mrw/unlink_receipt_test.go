package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

func TestUnlinkReceiptNamesRemoved(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "gone.txt"), []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readIn(t, root, "gone.txt"); err != nil {
		t.Fatalf("read gone.txt: %v", err)
	}
	plan := planFile(t, "@@ gone.txt - unlink\n")
	out, code := writeIn(t, root, "--no-check", plan)
	if code != 0 {
		t.Fatalf("unlink write exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "removed gone.txt") {
		t.Errorf("receipt does not name removed:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "gone.txt")); !os.IsNotExist(err) {
		t.Fatalf("path still exists: %v", err)
	}
	led, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := led["gone.txt"]; ok {
		t.Errorf("ledger still holds gone.txt: %+v", led)
	}
}
