package mcp

import (
	"fmt"
	"os"
	"path/filepath"
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

// ADR-203 Decision 9. The refusal names a pending page only when it is for the
// spelling the retry uses and covers the first line the refusal names, not
// merely some line of the hunk's address.
func TestMrwWriteNamesAPendingPageOnlyWhenItCoversTheRefusedLine(t *testing.T) {
	const body = "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\n"
	seeded := func(t *testing.T) string {
		t.Helper()
		root, _ := checkout(t, "f.txt", body)
		acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"f.txt"}})))
		if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 3 insert-after\nNEW\n", "ack": acks, "check": false}); res["isError"] == true {
			t.Fatalf("the seed write was refused: %s", firstText(t, res))
		}
		return root
	}
	t.Run("a page short of the first unshown line is not named", func(t *testing.T) {
		root := seeded(t)
		call(t, root, "mrw_read", map[string]any{"specs": []any{"f.txt:6-7"}}) // pending: lines 6 to 8
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 4-8 delete\n", "check": false})
		text := firstText(t, res)
		if res["isError"] != true || !strings.Contains(text, "line 4 of f.txt has no recorded showing") || strings.Contains(text, AckRule) {
			t.Errorf("want the refusal naming line 4 and no ack advice: %v", res)
		}
	})
	t.Run("a page for another spelling is not named", func(t *testing.T) {
		root := seeded(t)
		if err := os.Symlink("f.txt", filepath.Join(root, "link.txt")); err != nil {
			t.Skipf("symlinks are refused here: %v", err)
		}
		call(t, root, "mrw_read", map[string]any{"specs": []any{"link.txt:8"}})
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 8 replace\nX\n", "check": false})
		if text := firstText(t, res); res["isError"] != true || strings.Contains(text, AckRule) {
			t.Errorf("a page under the link's name was offered for f.txt: %v", res)
		}
	})
	t.Run("a page that covers the first unshown line is named, and its ack lands", func(t *testing.T) {
		root := seeded(t)
		page := call(t, root, "mrw_read", map[string]any{"specs": []any{"f.txt:4-8"}})
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 4-8 delete\n", "check": false})
		if text := firstText(t, res); res["isError"] != true || !strings.Contains(text, AckRule) {
			t.Fatalf("the covering page is not named: %v", res)
		}
		res = call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 4-8 delete\n", "ack": checkpointsIn(served0(t, page)), "check": false})
		if res["isError"] == true || fileText167(t, root, "f.txt") != "L1\nL2\nL3\nL8\nL9\nL10\n" {
			t.Errorf("the acked retry did not land: %v", res)
		}
	})
}

// ADR-203 Decision 8 over MCP: the plan mrw_read makes carries anchor= on each
// hunk, and a hunk with no line to anchor on is named in the receipt.
func TestAnMCPSubReadAnchorsItsHunksAndNamesTheOnesItCannot(t *testing.T) {
	root := tree161(t, map[string]string{"a.txt": "Old one\n\nOld two\n"})
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "grep": "Old", "sub": "New"})
	plan := served0(t, res)
	if strings.Count(plan, " anchor=\"Old ") != 2 || strings.Contains(fmt.Sprint(res["content"]), "carry no anchor=") {
		t.Errorf("want two anchored hunks and no note:\n%s", plan)
	}
	res = call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "grep": "^$", "sub": "BLANK"})
	if plan := served0(t, res); strings.Contains(plan, "anchor=") || !strings.Contains(fmt.Sprint(res["content"]), "hunk(s) carry no anchor=") {
		t.Errorf("want an unanchored hunk and the note in the receipt: %v", res)
	}
}
