package mcp

import "testing"

// Zy, 2026-10-22 (ADR-167): an mrw_read with quiet licenses lines and shows
// none, so a from= copy of them is refused; a plain acked read of the same
// lines is the pair.
func TestAQuietMCPReadDoesNotLicenseACopy(t *testing.T) {
	const plan = "@@ b.txt 1 insert-after from=a.txt:3-5\n"
	root := tree167(t)
	call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "quiet": true})
	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"b.txt"}})))
	res := call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks, "check": false})
	if res["isError"] != true || fileText167(t, root, "b.txt") != "x\ny\n" {
		t.Errorf("a quiet-read source was copied: %v / %q", res, fileText167(t, root, "b.txt"))
	}

	root = tree167(t)
	acks = checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt", "b.txt"}})))
	if res := call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks, "check": false}); res["isError"] == true {
		t.Errorf("the pair: a plain acked read was refused: %s", firstText(t, res))
	}
}
