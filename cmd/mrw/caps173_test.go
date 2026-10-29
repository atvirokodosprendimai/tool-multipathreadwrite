package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp"
)

// lastLine is the last non-empty line of s, where mrw says its kind (ADR-146).
func lastLine(s string) string {
	ls := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return ls[len(ls)-1]
}

// ADR-173. A range the caller's own --max-lines or --max-matches withheld is
// not one mrw could not serve: such a read exits 0 and says `capped:` last,
// the way `head` exits 0. A cap beside a missed range, a refused path or a
// reached --max-chars is still incomplete, a search that matched no file still
// says no match, and withheld lines still license nothing.
func TestACapTheCallerSetIsNotAnIncompleteRead(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{
		"a.txt":   "l1\nl2\nl3\nl4\nl5\n",
		"e.txt":   "hit\nx\nx\nx\nhit\n",
		"d/h.txt": "hit 1\nfill\nhit 2\nhit 3\n",
	})

	for _, c := range []struct {
		name  string
		stdin string
		args  []string
		out   string // stdout holds
		last  string // the last stderr line begins
	}{
		{"a span cut by --max-lines", "", []string{"read", "--max-lines", "3", "a.txt"}, "more line(s) withheld", "mrw: capped: 1 range(s) withheld by --max-lines"},
		{"a span cut and a span withheld whole", "", []string{"read", "--grep", "hit", "--max-lines", "1", "e.txt"}, "WITHHELD 1 line(s)", "mrw: capped: 2 range(s) withheld by --max-lines"},
		{"a grep walk under --max-lines", "", []string{"read", "--grep", "hit", "--max-lines", "1", "d"}, "more line(s) withheld", "mrw: capped: 1 range(s) withheld by --max-lines"},
		{"--max-lines 0", "", []string{"read", "--max-lines", "0", "a.txt"}, "WITHHELD 5 line(s)", "mrw: capped: 1 range(s) withheld by --max-lines"},
		{"standard input under --max-lines", "l1\nl2\n", []string{"read", "--max-lines", "1", "-"}, "    1| l1", "mrw: capped: 1 range(s) withheld by --max-lines"},
		{"standard input under --max-lines 0", "a\n", []string{"read", "--max-lines", "0", "-"}, "WITHHELD 1 line(s)", "mrw: capped: 1 range(s) withheld by --max-lines"},
		{"a walk under --max-matches", "", []string{"read", "--grep", "hit", "--max-matches", "1", "d"}, "-- max-matches 1 reached: 2 more matching line(s) in 1 file(s) not shown", "mrw: capped: --max-matches 1 reached, 2 matching line(s) not shown"},
		{"--max-matches 0", "", []string{"read", "--grep", "hit", "--max-matches", "0", "d"}, "-- max-matches 0 reached: 3 more", "mrw: capped: --max-matches 0 reached, 3 matching line(s) not shown"},
		{"standard input under --max-matches", "hit 1\nfill\nhit 2\nhit 3\n", []string{"read", "--grep", "hit", "--max-matches", "1", "-"}, "-- max-matches 1 reached", "mrw: capped: --max-matches 1 reached, 2 matching line(s) not shown"},
		{"a match withheld from standard input", "a\rb\r", []string{"read", "--grep", "^b$", "--max-lines", "0", "-"}, "WITHHELD", "mrw: capped: 1 range(s) withheld by --max-lines"},
		{"both caps", "", []string{"read", "--grep", "hit", "--max-lines", "1", "--max-matches", "1", "d"}, "-- max-matches 1 reached", "mrw: capped: 1 range(s) withheld by --max-lines; --max-matches 1 reached, 2 matching line(s) not shown"},
	} {
		out, errOut, code := run(c.stdin, c.args...)
		if code != 0 || !strings.Contains(out, c.out) || !strings.HasPrefix(lastLine(errOut), c.last) {
			t.Errorf("%s: exit %d, want 0 with %q on stdout and %q last on stderr:\n%s%s", c.name, code, c.out, c.last, out, errOut)
		}
	}

	for _, c := range []struct {
		name  string
		stdin string
		args  []string
		last  string
	}{
		{"a cap beside a missed range", "", []string{"read", "--max-lines", "2", "a.txt:/nomatch/", "a.txt:1-4"}, "mrw: incomplete: 2 range(s) could not be served"},
		{"a cap beside a refused path", "", []string{"read", "--max-lines", "2", "a.txt", "../outside.txt"}, "mrw: incomplete: 2 range(s) could not be served"},
		{"a cap that reached --max-chars", "", []string{"read", "--max-lines", "3", "--max-chars", "40", "a.txt", "b.txt"}, "mrw: incomplete: "},
		{"a cap whose --max-chars cut dropped no range of its own", "", []string{"read", "--max-lines", "2", "--max-chars", "60", "a.txt"}, "mrw: incomplete: 1 range(s) could not be served; --max-chars 60 reached"},
		{"--max-chars dropping only the match note", "hit\nhit\nhit\n", []string{"read", "--grep", "hit", "--max-matches", "0", "--max-chars", "1", "-"}, "mrw: incomplete: --max-matches 0 reached, 3 matching line(s) not shown; --max-chars 1 reached"},
		{"a capped search that matched no file", "", []string{"read", "--grep", "ABSENT", "--max-lines", "0", "d"}, "mrw: no match: "},
		{"a capped search of standard input that matched nothing", "a\n", []string{"read", "--grep", "z", "--max-lines", "0", "-"}, "mrw: no match: "},
	} {
		_, errOut, code := run(c.stdin, c.args...)
		if code != 1 || !strings.HasPrefix(lastLine(errOut), c.last) {
			t.Errorf("%s: exit %d, want 1 with %q last on stderr:\n%s", c.name, code, c.last, errOut)
		}
	}
	if _, errOut, _ := run("", "read", "--max-lines", "3", "--max-chars", "40", "a.txt", "b.txt"); !strings.Contains(lastLine(errOut), "--max-chars 40 reached") {
		t.Errorf("a cap that reached --max-chars does not name the budget:\n%s", errOut)
	}

	// Withheld and unshown lines license nothing, whatever the exit.
	if _, _, code := run("", "read", "--max-lines", "2", "a.txt"); code != 0 {
		t.Fatalf("read --max-lines 2 a.txt: exit %d, want 0", code)
	}
	if out, e, code := run("@@ a.txt 5 replace\nL5\n", "write", "--no-check", "-"); code != 1 || !strings.Contains(out+e, "has not been read") || subFile(t, root, "a.txt") != "l1\nl2\nl3\nl4\nl5\n" {
		t.Errorf("a write to a withheld line: exit %d, want 1 refused as unread:\n%s%s", code, out, e)
	}
	if _, _, code := run("", "read", "--grep", "hit", "--max-matches", "1", "d"); code != 0 {
		t.Fatalf("read --max-matches 1: exit %d, want 0", code)
	}
	if out, e, code := run("@@ d/h.txt 4 replace\nz\n", "write", "--no-check", "-"); code != 1 || !strings.Contains(out+e, "has not been read") {
		t.Errorf("a write to an unshown match: exit %d, want 1 refused as unread:\n%s%s", code, out, e)
	}

	// The receipt: problems keeps its meaning, capped says how many were the
	// caller's own, on `mrw read --json` and on mrw_read alike.
	for _, c := range []struct {
		name             string
		args             []string
		problems, capped float64
	}{
		{"a capped span", []string{"--max-lines", "2", "a.txt"}, 1, 1},
		{"a capped span and a missed range", []string{"--max-lines", "2", "a.txt:/nomatch/", "a.txt:1-4"}, 2, 1},
	} {
		out, _, _ := run("", append([]string{"read", "--json"}, c.args...)...)
		var doc struct{ Receipt map[string]any }
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("%s: read --json: %v\n%s", c.name, err, out)
		}
		if doc.Receipt["problems"] != c.problems || doc.Receipt["capped"] != c.capped {
			t.Errorf("%s: read --json receipt problems %v capped %v, want %v and %v", c.name, doc.Receipt["problems"], doc.Receipt["capped"], c.problems, c.capped)
		}
		specs := append([]string{}, c.args[2:]...)
		req, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "mrw_read", "arguments": map[string]any{"specs": specs, "max_lines": 2}}})
		var resp bytes.Buffer
		if err := mcp.Serve(bytes.NewReader(append(req, '\n')), &resp, root); err != nil {
			t.Fatal(err)
		}
		var rpc struct {
			Result struct{ Content []struct{ Text string } }
		}
		var got map[string]any
		if err := json.Unmarshal(resp.Bytes(), &rpc); err != nil || len(rpc.Result.Content) < 2 || json.Unmarshal([]byte(rpc.Result.Content[1].Text), &got) != nil {
			t.Fatalf("%s: mrw_read: %v\n%s", c.name, err, resp.Bytes())
		}
		if got["problems"] != c.problems || got["capped"] != c.capped {
			t.Errorf("%s: mrw_read receipt problems %v capped %v, want %v and %v", c.name, got["problems"], got["capped"], c.problems, c.capped)
		}
	}
}
