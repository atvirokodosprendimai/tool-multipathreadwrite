package mcp

import (
	"fmt"
	"strings"
	"testing"
)

// ADR-203 over mrw_write. f.txt holds L1 to L10, is served and acknowledged, and
// is written with NEW after line 3; a second plan by number alone, on a line no
// acknowledged read has shown, is refused with isError and the reason in the
// hunk, naming this surface's remedy, and lands once the line is read and acked.
func TestMrwWriteRefusesAStaleNumber(t *testing.T) {
	const body = "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\n"
	const seeded = "L1\nL2\nL3\nNEW\nL4\nL5\nL6\nL7\nL8\nL9\nL10\n"
	root, _ := checkout(t, "f.txt", body)
	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"f.txt"}})))
	if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 3 insert-after\nNEW\n", "ack": acks, "check": false}); res["isError"] == true {
		t.Fatalf("the seed write was refused: %s", firstText(t, res))
	}
	const plan = "@@ f.txt 8 replace\nX\n"

	res := call(t, root, "mrw_write", map[string]any{"plan": plan, "check": false})
	text := firstText(t, res)
	if res["isError"] != true || !strings.Contains(text, "line 8 of f.txt has no recorded showing in this version") || !strings.Contains(text, "mrw_read") || !strings.Contains(text, "ack") {
		t.Fatalf("want isError and the reason with this surface's remedy: %v", res)
	}
	if strings.Contains(fmt.Sprint(res), "stale-number") || strings.Contains(text, "never acknowledged") {
		t.Errorf("the kind leaked, or an ack was named with none pending: %v", res)
	}
	if got := fileText167(t, root, "f.txt"); got != seeded {
		t.Fatalf("a refused plan changed the file:\n%s", got)
	}

	// Served and not acknowledged: the refusal names that ack.
	page := call(t, root, "mrw_read", map[string]any{"specs": []any{"f.txt:8"}})
	res = call(t, root, "mrw_write", map[string]any{"plan": plan, "check": false})
	if text := firstText(t, res); res["isError"] != true || !strings.Contains(text, AckRule) {
		t.Errorf("a pending page that covers line 8 is not named: %v", res)
	}
	// Acknowledged with the retry: it lands on line 8, which holds L7 now.
	res = call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": checkpointsIn(served0(t, page)), "check": false})
	if res["isError"] == true || fileText167(t, root, "f.txt") != strings.Replace(seeded, "L7\n", "X\n", 1) {
		t.Errorf("the acked retry did not land: %v\n%s", res, fileText167(t, root, "f.txt"))
	}
}
