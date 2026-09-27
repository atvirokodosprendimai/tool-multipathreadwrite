package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
)

// ADR-083 T2. mrw_write returned before its tally when a pointer hunk path did
// not resolve to exactly one file, so the plan went uncounted while the CLI
// counts it as one refusal. Each refusal is one refused_apply, dry run or not,
// and a pointer that resolves lands as applied.
func TestAnMCPPlanRefusedAfterItParsedIsOneRefusal(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var set iter.Set
	set.Add("a.txt", "b.txt")
	if err := iter.Save(root, set); err != nil {
		t.Fatal(err)
	}
	want := func(step string, refused, applied, plans int) {
		t.Helper()
		tally, err := authoring.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if tally["refused_apply"] != refused || tally["applied"] != applied || tally.Plans() != plans {
			t.Errorf("%s: tally %v, want refused_apply %d, applied %d, plans %d", step, tally, refused, applied, plans)
		}
	}

	call(t, root, "mrw_write", map[string]any{"plan": "@@ @1-2 1 replace\nx\n"})
	want("a pointer naming two entries", 1, 0, 1)
	call(t, root, "mrw_write", map[string]any{"plan": "@@ @1-2 1 replace\nx\n", "dry_run": true})
	want("the same plan as a dry run", 2, 0, 2)
	call(t, root, "mrw_write", map[string]any{"plan": "@@ @9 1 replace\nx\n"})
	want("a pointer past the working set", 3, 0, 3)

	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}})))
	call(t, root, "mrw_write", map[string]any{"plan": "@@ @1 1 replace\nA\n", "ack": acks})
	b, err := os.ReadFile(filepath.Join(root, "a.txt"))
	if err != nil || string(b) != "A\n" {
		t.Fatalf("@1 did not land on a.txt: %q, %v", b, err)
	}
	want("a pointer that resolves", 3, 1, 4)
}
