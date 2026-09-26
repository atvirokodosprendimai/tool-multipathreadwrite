package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ADR-078 T2. A request id is a string or an integer; true, 1.5 or an object
// was dispatched and echoed back. Each is refused as an invalid request with a
// null id; strings and integers are served.
func TestARequestIdThatIsNeitherAStringNorAnIntegerIsInvalid(t *testing.T) {
	for _, id := range []string{`true`, `1.5`, `{}`, `[]`, `1e3`, `-0.0`} {
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
	for _, id := range []string{`7`, `-3`, `0`, `"a"`} {
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
	for glob, want := range map[string]string{"[": "syntax error", "/vendor": "starts with /"} {
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
// kept reading every spec to the end first: 100,000 held the server past two
// minutes. It stops at the overflow and says so.
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
	if res["isError"] != true || !strings.Contains(txt, "would have returned more than") || !strings.Contains(txt, "name fewer files") || strings.Contains(txt, "One line of this file") {
		t.Errorf("want a refusal that stopped early and advises fewer files, not a long line: %.400s", txt)
	}
}

// ADR-078 T3, the waiver on #232. A line that alone encodes past the ceiling
// was advised a range that held it, and the next call said no narrower range
// could help, though one after the line did. The line is named, with the ranges
// around it, and the range offered is served.
func TestTheRefusalNamesTheLineThatEncodesPastTheCeiling(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	body := strings.Repeat("<", 150000) + "\n" + strings.Repeat(strings.Repeat("y", 70)+"\n", 1000)
	if err := os.WriteFile(filepath.Join(root, "f.svg"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	txt := resultText(call(t, root, "mrw_read", map[string]any{"specs": []any{"f.svg"}}))
	if !strings.Contains(txt, "Line 1 of f.svg") || !strings.Contains(txt, "`f.svg:2-`") || strings.Contains(txt, ":1-1") {
		t.Fatalf("the refusal does not name line 1 and the range after it:\n%s", txt)
	}
	if res := call(t, root, "mrw_read", map[string]any{"specs": []any{"f.svg:2-"}}); res["isError"] == true {
		t.Errorf("the range the refusal offered was refused:\n%.300s", resultText(res))
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
	for _, want := range []string{"Line 5 of f.svg", "`f.svg:1-4`", "`f.svg:6-`", "mrw read f.svg:5"} {
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
