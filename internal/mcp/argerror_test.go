package mcp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// ADR-067 T3. A caller's argument mistake is a tool execution error — a result
// with isError, which a host hands the model so it can correct itself (SEP-1303,
// 2025-11-25) — and no longer a JSON-RPC -32602. A malformed call is still one.

// rawCall sends one tools/call whose params are given verbatim and returns the
// decoded response, error or result.
func rawCall(t *testing.T, params string) map[string]any {
	t.Helper()
	root, _ := checkout(t, "a.txt", "one\n")
	var out bytes.Buffer
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":%s}`, params)
	if err := Serve(strings.NewReader(req+"\n"), &out, root); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	return decode(t, strings.TrimSuffix(out.String(), "\n"))
}

func TestAnArgumentErrorIsAToolExecutionError(t *testing.T) {
	for _, c := range []struct{ params, sentence string }{
		{`{"name":"mrw_read","arguments":{"specs":42}}`, "arguments:"},
		{`{"name":"mrw_read","arguments":{}}`, "needs at least one spec"},
		{`{"name":"mrw_write","arguments":{"plan":7}}`, "arguments:"},
		{`{"name":"mrw_write","arguments":{"plan":"x","echo_pad":-1}}`, "echo_pad must be >= 0"},
		{`{"name":"mrw_write","arguments":{"plan":"   "}}`, "needs a plan"},
		{`{"name":"mrw_write","arguments":{"plan":"x","format":"git"}}`, "git patch is not"},
		{`{"name":"mrw_write","arguments":{"plan":"x","format":"nope"}}`, "unknown format"},
	} {
		got := rawCall(t, c.params)
		if e, ok := got["error"]; ok {
			t.Errorf("%s: answered a JSON-RPC error %v, want an isError result", c.params, e)
			continue
		}
		res, _ := got["result"].(map[string]any)
		if res["isError"] != true {
			t.Errorf("%s: the result is not flagged isError: %v", c.params, res)
			continue
		}
		if text := served0(t, res); !strings.Contains(text, c.sentence) {
			t.Errorf("%s: the result does not say %q: %q", c.params, c.sentence, text)
		}
	}
}

// A guard that holds today and must keep holding: a call that does not
// satisfy CallToolRequest is a protocol error, not a mistake the model made
// inside its arguments. `arguments` present but not an object reached the
// argument decoders this task moves, so it is refused before dispatch.
func TestAMalformedCallIsStillAProtocolError(t *testing.T) {
	cases := []string{`"x"`, `{"name":"no_such_tool","arguments":{}}`}
	for _, tool := range []string{"mrw_read", "mrw_write"} {
		for _, args := range []string{`42`, `"x"`, `[]`, `null`} {
			cases = append(cases, fmt.Sprintf(`{"name":%q,"arguments":%s}`, tool, args))
		}
	}
	for _, params := range cases {
		got := rawCall(t, params)
		e, ok := got["error"].(map[string]any)
		if !ok {
			t.Errorf("%s: answered a result, want a JSON-RPC error: %v", params, got["result"])
			continue
		}
		if e["code"] != float64(codeInvalidParams) {
			t.Errorf("%s: code %v, want %d", params, e["code"], codeInvalidParams)
		}
	}
}
