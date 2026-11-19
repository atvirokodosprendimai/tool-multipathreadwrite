package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-200 T1. mrw_write counts a typed plan's numbered hunks as the CLI does,
// and a plan mrw computed from the file (grep with sub, or a foreign format) is
// a landing and never a candidate.
func TestAnMcpWriteIsCountedAndAComputedPlanIsNot(t *testing.T) {
	const body = "a\nb\nc\nd\ne\n"
	// Known the way a write leaves a file: wholly licensed, nothing shown.
	fresh := func(t *testing.T) string {
		t.Helper()
		root, _ := checkout(t, "f.txt", body)
		if err := seen.Record(root, map[string]seen.Observation{"f.txt": {SHA: seen.SHA([]byte(body)), Written: true}}); err != nil {
			t.Fatal(err)
		}
		return root
	}
	write := func(t *testing.T, root string, args map[string]any) {
		t.Helper()
		args["check"] = false
		if res := call(t, root, "mrw_write", args); res["isError"] == true {
			t.Fatalf("mrw_write %v: %v", args, res)
		}
	}
	t.Run("a typed plan is counted", func(t *testing.T) {
		// Line 4 is shown since the write: an unshown one is refused (ADR-203), not counted.
		root, _ := checkout(t, "f.txt", body)
		if err := seen.Record(root, map[string]seen.Observation{"f.txt": {SHA: seen.SHA([]byte(body)), Written: true, Shown: [][2]int{{4, 4}}}}); err != nil {
			t.Fatal(err)
		}
		write(t, root, map[string]any{"plan": "@@ f.txt 4 replace\nX\n"})
		if got, want := authoring.LoadNumbering(root), (authoring.Numbering{Landed: 1, Candidates: 1}); got != want {
			t.Errorf("%+v, want %+v", got, want)
		}
	})
	t.Run("a typed plan with anchor= is a landing alone", func(t *testing.T) {
		root := fresh(t)
		write(t, root, map[string]any{"plan": "@@ f.txt 4 replace anchor=\"d\"\nX\n"})
		if got := authoring.LoadNumbering(root); got != (authoring.Numbering{Landed: 1}) {
			t.Errorf("%+v, want one landing", got)
		}
	})
	t.Run("grep with sub is a landing and no candidate", func(t *testing.T) {
		root := fresh(t)
		write(t, root, map[string]any{"grep": "^d$", "sub": "X"})
		if b, _ := os.ReadFile(filepath.Join(root, "f.txt")); string(b) != "a\nb\nc\nX\ne\n" {
			t.Fatalf("the sub did not land: %q", b)
		}
		if got := authoring.LoadNumbering(root); got != (authoring.Numbering{Landed: 1}) {
			t.Errorf("%+v, want one landing alone", got)
		}
	})
	t.Run("a foreign format is a landing and no candidate", func(t *testing.T) {
		root := fresh(t)
		write(t, root, map[string]any{"format": "apply_patch", "plan": "*** Begin Patch\n*** Update File: f.txt\n@@\n-d\n+X\n*** End Patch\n"})
		if got := authoring.LoadNumbering(root); got != (authoring.Numbering{Landed: 1}) {
			t.Errorf("apply_patch: %+v, want one landing alone", got)
		}
		write(t, root, map[string]any{"format": "search_replace", "plan": "f.txt\n<<<<<<< SEARCH\nX\n=======\nd\n>>>>>>> REPLACE\n"})
		if got := authoring.LoadNumbering(root); got != (authoring.Numbering{Landed: 2}) {
			t.Errorf("search_replace: %+v, want two landings and no candidate", got)
		}
	})
}
