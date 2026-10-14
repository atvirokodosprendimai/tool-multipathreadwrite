package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-137. A line read cut to a window was not shown, so a plan that replaces it
// is refused as unread; a line shown whole in the same read is licensed.
func TestAPlanToACutLineIsRefusedAsUnread(t *testing.T) {
	root := t.TempDir()
	long := strings.Repeat("x", 300) + "NEEDLE" + strings.Repeat("y", 94)
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("short one\n"+long+"\nshort three\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runIn(t, root, "read", "--max-cols", "40", "--context", "1", "f.txt:/NEEDLE/")
	if code != 0 || !strings.Contains(out, "[cols ") {
		t.Fatalf("the cut read: exit %d\n%s", code, out)
	}
	planDir := t.TempDir()
	cutPlan := filepath.Join(planDir, "cut.plan")
	if err := os.WriteFile(cutPlan, []byte("@@ f.txt 2 replace\nREPLACED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wout, wcode := runIn(t, root, "write", "--no-check", cutPlan)
	if wcode == 0 || !strings.Contains(wout, "has not been read") {
		t.Errorf("a plan to the cut line: exit %d, want it refused as unread\n%s", wcode, wout)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "f.txt")); !strings.Contains(string(b), "NEEDLE") {
		t.Errorf("the cut line was replaced:\n%s", b)
	}
	wholePlan := filepath.Join(planDir, "whole.plan")
	if err := os.WriteFile(wholePlan, []byte("@@ f.txt 3 replace\nNEW THREE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if wout, wcode := runIn(t, root, "write", "--no-check", wholePlan); wcode != 0 {
		t.Errorf("a plan to a line shown whole: exit %d\n%s", wcode, wout)
	}
	if out, code := runIn(t, root, "read", "--max-cols", "-5", "f.txt"); code != 2 || !strings.Contains(out, "cannot be negative") {
		t.Errorf("a negative width: exit %d, want 2 naming it\n%s", code, out)
	}
}
