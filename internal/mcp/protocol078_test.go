package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ADR-078 T2. A request id is a string or an integer; true, 1.5 or an object
// was dispatched and echoed back. Each is refused as an invalid request with a
// null id. An integer is a value, not a spelling — MCP's RequestId is a JSON
// Schema integer — so 1.0, 1e3, -0.0 and one past 64 bits are served, as a
// string is; refusing them lost a conforming client's request (the review of
// #239).
func TestARequestIdThatIsNeitherAStringNorAnIntegerIsInvalid(t *testing.T) {
	for _, id := range []string{`true`, `1.5`, `{}`, `[]`, `1e-1`, `15e-1`, `1.25e1`} {
		lines := serve(t, `{"jsonrpc":"2.0","id":`+id+`,"method":"ping"}`)
		if len(lines) != 1 {
			t.Fatalf("id %s: %d responses", id, len(lines))
		}
		m := decode(t, lines[0])
		e, _ := m["error"].(map[string]any)
		if e == nil || e["code"] != float64(codeInvalidRequest) || m["id"] != nil {
			t.Errorf("id %s was not refused as an invalid request with a null id: %v", id, m)
		}
	}
	for _, id := range []string{`7`, `-3`, `0`, `"a"`, `1.0`, `1e3`, `-0.0`, `10e-1`, `1.25e2`, `123456789012345678901234567890`} {
		m := decode(t, serve(t, `{"jsonrpc":"2.0","id":`+id+`,"method":"ping"}`)[0])
		if _, bad := m["error"]; bad {
			t.Errorf("id %s was refused: %v", id, m)
		}
	}
}

// rawToolCall sends one tools/call whose arguments are raw JSON bytes, which
// may be invalid UTF-8 — the thing json.Marshal would have repaired.
func rawToolCall(t *testing.T, root, tool string, args []byte) map[string]any {
	t.Helper()
	line := append([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"`+tool+`","arguments":`), args...)
	line = append(line, "}}\n"...)
	var out bytes.Buffer
	if err := Serve(bytes.NewReader(line), &out, root); err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, out.Bytes())
	}
	res, _ := resp["result"].(map[string]any)
	if res == nil {
		t.Fatalf("no result: %v", resp)
	}
	return res
}

func resultText(res map[string]any) string {
	c, _ := res["content"].([]any)
	if len(c) == 0 {
		return ""
	}
	s, _ := c[0].(map[string]any)["text"].(string)
	return s
}

// ADR-078 T2. encoding/json replaced invalid UTF-8 with U+FFFD, so a spec sent
// as "\xff.go" reached the engine as a path nobody sent. The argument is
// refused by name, on both tools; UTF-8 passes as written.
func TestArgumentsThatAreNotUTF8AreRefusedByName(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	for tool, c := range map[string]struct {
		args []byte
		name string
	}{
		"mrw_read":  {[]byte("{\"specs\":[\"\xff.go\"]}"), "specs"},
		"mrw_write": {[]byte("{\"plan\":\"@@ \xff.go 0 create\\nx\\n\"}"), "plan"},
	} {
		res := rawToolCall(t, root, tool, c.args)
		if res["isError"] != true || !strings.Contains(resultText(res), c.name+" is not valid UTF-8") {
			t.Errorf("%s: want %s refused as not UTF-8: %v", tool, c.name, res)
		}
	}
	// A key spelled "" hid its invalid value, since "" was also the answer for
	// valid input (the review of #239).
	for tool, args := range map[string][]byte{
		"mrw_read":  []byte("{\"\":\"\xff\",\"specs\":[\"a.txt\"]}"),
		"mrw_write": []byte("{\"\":\"\xff\",\"plan\":\"@@ b.txt 0 create\\nx\\n\"}"),
	} {
		if res := rawToolCall(t, root, tool, args); res["isError"] != true || !strings.Contains(resultText(res), "not valid UTF-8") {
			t.Errorf("%s: an invalid value under an empty key was taken as valid: %v", tool, res)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "b.txt")); err == nil {
		t.Error("a plan sent beside an invalid empty-named argument was applied")
	}
	if _, err := os.Stat(filepath.Join(root, "\ufffd.go")); err == nil {
		t.Error("a file named with U+FFFD was created")
	}
	if strings.Contains(resultText(call(t, root, "mrw_read", map[string]any{"specs": []any{"é.go"}})), "not valid UTF-8") {
		t.Error("a UTF-8 name was refused as not UTF-8")
	}
}

// ADR-078 T2. `exclude: ["["]` was ignored over MCP while the CLI refused it,
// and a glob starting with / can never match a root-relative path.
func TestABadExcludeGlobIsRefusedOverMCP(t *testing.T) {
	root, _ := checkout(t, "a.txt", "a\n")
	for glob, want := range map[string]string{"[": "syntax error", "/vendor": "rooted", "vendor/": "ends in /", "./vendor": "starts with ./"} {
		res := call(t, root, "mrw_read", map[string]any{"grep": "a", "exclude": []any{glob}})
		if res["isError"] != true || !strings.Contains(resultText(res), want) {
			t.Errorf("exclude %q: want it refused with %q: %v", glob, want, res)
		}
	}
	if res := call(t, root, "mrw_read", map[string]any{"grep": "a", "exclude": []any{"vendor"}}); res["isError"] == true {
		t.Errorf("a well-formed exclude was refused: %v", res)
	}
}

// ADR-078 T3. A named read of many specs is refused once it overflows, and it
// kept reading every spec to the end first: 100,000 held the server past 120 s
// in the v1.25.1 round. It stops at the overflow and says so.
func TestAManySpecReadStopsOnceItIsOverTheCeiling(t *testing.T) {
	root, _ := checkout(t, "a.txt", strings.Repeat("x", 200)+"\n")
	specs := make([]any, 100000)
	for i := range specs {
		specs[i] = "a.txt"
	}
	start := time.Now()
	res := call(t, root, "mrw_read", map[string]any{"specs": specs})
	if d := time.Since(start); d > 30*time.Second {
		t.Errorf("100,000 specs took %v", d)
	}
	txt := resultText(res)
	if res["isError"] != true || !strings.Contains(txt, "name fewer files") || strings.Contains(txt, "One line of this file") {
		t.Errorf("want a refusal that advises fewer files, not a long line: %.400s", txt)
	}
	// The size, not the wording, shows the read stopped: a closure that set
	// "more than" and let the read run on said "more than 24900000" and passed
	// (the review of #239). Stopped at the first spec past the limit, the size
	// is under twice it.
	m := regexp.MustCompile(`would have returned more than (\d+) bytes`).FindStringSubmatch(txt)
	if m == nil {
		t.Fatalf("the refusal does not say it stopped: %.400s", txt)
	}
	if n, _ := strconv.Atoi(m[1]); n >= 2*MaxResultChars {
		t.Errorf("the read ran on past the overflow: more than %d bytes", n)
	}
}

// ADR-078 T3, the waiver on #232. A line past the ceiling was advised a range
// that held it, and the next call said no narrower range could help, though
// one after the line did. The line is named, with the ranges around it, and the
// open range offered is served: for a line escaping pushes past the ceiling,
// for a plain one longer than it — which the capped sample never holds whole,
// so it took the old sentence (the review of #239) — and down a file with two
// such lines, one refusal at a time.
func TestTheRefusalNamesTheLineThatEncodesPastTheCeiling(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for name, body := range map[string]string{
		"f.svg": strings.Repeat("<", 150000) + "\n" + strings.Repeat(strings.Repeat("y", 70)+"\n", 1000),
		"h.js":  strings.Repeat("x", MaxResultChars+1000) + "\nnext\n",
		"g.svg": strings.Repeat("<", 150000) + "\n" + strings.Repeat("<", 150000) + "\nlast\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []struct{ spec, named, next string }{
		{"f.svg", "Line 1 of f.svg", "`f.svg:2-`"},
		{"h.js", "Line 1 of h.js", "`h.js:2-`"},
		{"g.svg", "Line 1 of g.svg", "`g.svg:2-`"},
		{"g.svg:2-", "Line 2 of g.svg", "`g.svg:3-`"},
	} {
		txt := resultText(call(t, root, "mrw_read", map[string]any{"specs": []any{s.spec}}))
		if !strings.Contains(txt, s.named) || !strings.Contains(txt, s.next) || strings.Contains(txt, ":1-0") {
			t.Errorf("%s: the refusal does not name %q and %s:\n%s", s.spec, s.named, s.next, txt)
		}
	}
	for _, spec := range []string{"f.svg:2-", "h.js:2-", "g.svg:3-"} {
		if res := call(t, root, "mrw_read", map[string]any{"specs": []any{spec}}); res["isError"] == true {
			t.Errorf("%s, the range a refusal offered, was refused:\n%.300s", spec, resultText(res))
		}
	}
}

// ADR-078 T3. A long line in the middle is named with the ranges before and
// after it.
func TestALongLineInTheMiddleIsNamedWithTheRangesAroundIt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	body := "a\nb\nc\nd\n" + strings.Repeat("<", 150000) + "\n" + strings.Repeat(strings.Repeat("y", 70)+"\n", 1000)
	if err := os.WriteFile(filepath.Join(root, "f.svg"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	txt := resultText(call(t, root, "mrw_read", map[string]any{"specs": []any{"f.svg:5-"}}))
	// Neither range is promised: "still serve here" was said of a prefix range
	// that was then refused (the review of #239). They are named for what they
	// hold.
	for _, want := range []string{"Line 5 of f.svg", "`f.svg:1-4` holds the lines before it", "`f.svg:6-` reads on from the line after it", "mrw read f.svg:5"} {
		if !strings.Contains(txt, want) {
			t.Errorf("the refusal lacks %q:\n%s", want, txt)
		}
	}
}

// ADR-078 T3. A file whose text renders inside the ceiling but encodes past it
// took the rendered-fit refusal, which said no narrower range could be served
// though the range after the long line could. It names the line and that range.
func TestARenderedFitRefusalNamesTheLine(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "f.svg"), []byte(strings.Repeat("<", 150000)+"\nsecond\nthird\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	txt := resultText(call(t, root, "mrw_read", map[string]any{"specs": []any{"f.svg"}}))
	if !strings.Contains(txt, "Line 1 of f.svg") || !strings.Contains(txt, "`f.svg:2-`") {
		t.Errorf("the rendered-fit refusal does not name line 1 and the range after it:\n%s", txt)
	}
}
