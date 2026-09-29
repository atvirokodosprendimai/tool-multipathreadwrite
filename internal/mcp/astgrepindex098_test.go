package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ADR-098 T1. An ast_grep answer too large to serve is an INDEX, like a grep's,
// and the index told the caller to "send the SAME grep again with after=" —
// while `after` without grep was refused and astGrepSpecs took no cursor, so
// page two of a structural search could not be reached (the 2026-09-29 audit,
// row H8).

// astGrepTree plants n grepTree files and a fake ast-grep reporting one hit in
// each, emitted in REVERSE path order: a cursor that trusted the binary's order
// would skip or repeat files, and the paging test would say so.
func astGrepTree(t *testing.T, n int) string {
	t.Helper()
	root := grepTree(t, n, 1)
	var hits []map[string]any
	for i := n - 1; i >= 0; i-- {
		hits = append(hits, map[string]any{"file": fmt.Sprintf("document%05d.csv", i),
			"range": map[string]any{"start": map[string]any{"line": 1}, "end": map[string]any{"line": 1}}})
	}
	raw, err := json.Marshal(hits)
	if err != nil {
		t.Fatal(err)
	}
	installFakeAstGrep(t, string(raw))
	return root
}

// pageTally follows one read's pages: an index names its files, a served page
// observes them. It fails on a file seen twice and returns the next cursor.
func pageTally(t *testing.T, res map[string]any, seen map[string]bool) string {
	t.Helper()
	if res["isError"] == true {
		t.Fatalf("a page was refused:\n%.600s", served0(t, res))
	}
	st := receipt(t, res)
	if idx, ok := st["index"].([]any); ok {
		for _, e := range idx {
			s := fmt.Sprint(e)
			if i := strings.LastIndex(s, ":"); i > 0 {
				s = s[:i]
			}
			if seen[s] {
				t.Errorf("%q appears on two pages — a cursor that overlaps loses the caller's place", s)
			}
			seen[s] = true
		}
		next, _ := st["next_index"].(string)
		return next
	}
	for p := range st["observed"].(map[string]any) {
		if seen[p] {
			t.Errorf("%q appears on two pages", p)
		}
		seen[p] = true
	}
	return ""
}

func TestAnAstGrepIndexPagesToTheEnd(t *testing.T) {
	const files = 400
	root := astGrepTree(t, files)
	withCeiling(t, 6_000)

	res := call(t, root, "mrw_read", map[string]any{"ast_grep": "NEEDLE"})
	indexOf(t, res, "an oversized ast_grep read")
	seen := map[string]bool{}
	next := pageTally(t, res, seen)
	if next == "" {
		t.Fatal("the first index was not cut short, so this fixture pages nothing")
	}
	for pages := 1; next != ""; pages++ {
		if pages > 50 {
			t.Fatalf("following next_index did not end after %d pages", pages)
		}
		next = pageTally(t, call(t, root, "mrw_read", map[string]any{"ast_grep": "NEEDLE", "after": next}), seen)
	}
	if len(seen) != files {
		t.Errorf("paging to the end yielded %d distinct files, want %d", len(seen), files)
	}
}

func TestAfterWithoutAFinderIsRefusedNamingBoth(t *testing.T) {
	root := astGrepTree(t, 3)
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"document00000.csv"}, "after": "x"})
	text := fmt.Sprint(res["content"])
	if res["isError"] != true || !strings.Contains(text, "grep") || !strings.Contains(text, "ast_grep") {
		t.Errorf("after with plain specs: want a refusal naming grep and ast_grep, got %v", res)
	}
	if res := call(t, root, "mrw_read", map[string]any{"ast_grep": "NEEDLE", "after": "document00000.csv"}); res["isError"] == true {
		t.Errorf("after with ast_grep was refused:\n%.600s", served0(t, res))
	}
}

func TestTheIndexNamesTheFinderThatMadeIt(t *testing.T) {
	withCeiling(t, 6_000)
	for _, c := range []struct{ finder, other string }{{"ast_grep", "grep"}, {"grep", "ast_grep"}} {
		root := astGrepTree(t, 400)
		text := indexOf(t, call(t, root, "mrw_read", map[string]any{c.finder: "NEEDLE"}), "an oversized "+c.finder)
		if !strings.Contains(text, "SAME "+c.finder+" again") || strings.Contains(text, "SAME "+c.other+" again") {
			t.Errorf("a %s index does not name its own finder:\n%.800s", c.finder, text)
		}
		if !strings.Contains(text, "next_index is empty") || strings.Contains(text, "next_index is absent") {
			t.Errorf("a %s index does not say to repeat until next_index is empty:\n%.800s", c.finder, text)
		}
	}
}
