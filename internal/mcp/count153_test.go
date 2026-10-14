package mcp

import (
	"regexp"
	"strings"
	"testing"
)

// ADR-153 Decision 5. mrw_read with `count` and `grep` answers how many lines
// match in each file, in `counts`, keyed as `observed` is; it serves no line,
// licenses none, and is refused without `grep` or beside `stat`, `max_lines`
// or `ast_grep`.
func TestMCPReadCountsWhatGrepMatches(t *testing.T) {
	root := grepTree(t, 2, 3)
	res := call(t, root, "mrw_read", map[string]any{"grep": "NEEDLE", "count": true})
	if isErr, _ := res["isError"].(bool); isErr {
		t.Fatalf("a count was refused: %s", resultText(res))
	}
	text := resultText(res)
	if regexp.MustCompile(`(?m)^ *[0-9]+\| `).MatchString(text) {
		t.Errorf("count served a line:\n%s", text)
	}
	if !strings.Contains(text, "-- 6 matching line(s) in 2 file(s)") {
		t.Errorf("the answer does not total the count:\n%s", text)
	}
	counts, _ := receipt(t, res)["counts"].(map[string]any)
	for _, f := range []string{"document00000.csv", "document00001.csv"} {
		if n, _ := counts[f].(float64); n != 3 {
			t.Errorf("counts[%q] = %v, want 3 (receipt counts: %v)", f, counts[f], counts)
		}
	}

	w := call(t, root, "mrw_write", map[string]any{"plan": "@@ document00000.csv 2 replace\nchanged\n", "check": false})
	if isErr, _ := w["isError"].(bool); !isErr || !strings.Contains(resultText(w), "has not been read") {
		t.Errorf("a write to a line only counted was not refused as unread: %s", resultText(w))
	}

	for name, args := range map[string]map[string]any{
		"no grep":   {"specs": []any{"document00000.csv"}, "count": true},
		"stat":      {"grep": "NEEDLE", "count": true, "stat": true},
		"max_lines": {"grep": "NEEDLE", "count": true, "max_lines": 2},
		"ast_grep":  {"ast_grep": "x", "count": true},
	} {
		r := call(t, root, "mrw_read", args)
		if isErr, _ := r["isError"].(bool); !isErr || !strings.Contains(resultText(r), "count") {
			t.Errorf("count with %s was not refused by name: %s", name, resultText(r))
		}
	}
}
