package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// serve drives Serve over a canned session and returns the lines it wrote.
// Every test goes through the real reader/writer path rather than calling a
// handler, because the framing is the part a host trips over.
func serve(t *testing.T, requests ...string) []string {
	t.Helper()
	var out bytes.Buffer
	in := strings.NewReader(strings.Join(requests, "\n") + "\n")
	if err := Serve(in, &out, t.TempDir()); err != nil {
		t.Fatalf("Serve returned %v, want nil on a clean EOF", err)
	}
	s := out.String()
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// decode parses one response line, failing the test if it is not an object.
func decode(t *testing.T, line string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("response line is not JSON: %v\nline: %q", err, line)
	}
	return m
}

func TestInitializeCompletesTheHandshake(t *testing.T) {
	lines := serve(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	if len(lines) != 1 {
		t.Fatalf("got %d response line(s), want 1: %q", len(lines), lines)
	}
	got := decode(t, lines[0])
	if got["jsonrpc"] != "2.0" {
		t.Errorf(`jsonrpc = %v, want "2.0"`, got["jsonrpc"])
	}
	if got["id"] != float64(1) {
		t.Errorf("id = %v, want 1 — a response must carry the request's id", got["id"])
	}
	res, ok := got["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result object in %q", lines[0])
	}
	// A response carrying only a protocol version parses as JSON and still
	// fails negotiation. Each of these is required by the lifecycle spec.
	if res["protocolVersion"] == nil {
		t.Error("result has no protocolVersion")
	}
	info, ok := res["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("result has no serverInfo object")
	}
	if info["name"] == nil || info["version"] == nil {
		t.Errorf("serverInfo = %v, want a name and a version", info)
	}
	caps, ok := res["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("result has no capabilities object")
	}
	if _, ok := caps["tools"]; !ok {
		t.Errorf("capabilities = %v, want a tools capability — a host that does not see one will not call tools/list", caps)
	}
}

func TestOneMessagePerLineRoundTrips(t *testing.T) {
	// Two requests on two lines must produce two responses on two lines. MCP
	// stdio delimits by newline and forbids an embedded one; this is NOT the
	// Language Server Protocol's Content-Length framing.
	lines := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	)
	if len(lines) != 2 {
		t.Fatalf("got %d response line(s), want 2 — one message per line: %q", len(lines), lines)
	}
	for i, line := range lines {
		if strings.ContainsAny(line, "\n\r") {
			t.Errorf("response %d contains an embedded newline, which the spec forbids: %q", i, line)
		}
		decode(t, line) // each line must stand alone as a complete message
	}
	if decode(t, lines[0])["id"] != float64(1) || decode(t, lines[1])["id"] != float64(2) {
		t.Errorf("responses are not in request order: %q", lines)
	}
}

func TestAMalformedFrameIsAnErrorNotAClose(t *testing.T) {
	// The second request proves the connection stayed open: a host that sees a
	// closed pipe cannot tell a crash from a refusal.
	lines := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize"`, // truncated, unparseable
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	)
	if len(lines) != 2 {
		t.Fatalf("got %d response line(s), want 2 — a parse error is answered and the session continues: %q", len(lines), lines)
	}
	bad := decode(t, lines[0])
	e, ok := bad["error"].(map[string]any)
	if !ok {
		t.Fatalf("malformed request did not get an error object: %q", lines[0])
	}
	if e["code"] != float64(-32700) {
		t.Errorf("code = %v, want -32700 (parse error)", e["code"])
	}
	if _, ok := bad["result"]; ok {
		t.Error("a response carries either result or error, never both")
	}
	if decode(t, lines[1])["id"] != float64(2) {
		t.Error("the request after the malformed one was not answered — the session closed")
	}
}

func TestToolsListNamesBothTools(t *testing.T) {
	lines := serve(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if len(lines) != 1 {
		t.Fatalf("got %d response line(s), want 1: %q", len(lines), lines)
	}
	res, ok := decode(t, lines[0])["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result object in %q", lines[0])
	}
	tools, ok := res["tools"].([]any)
	if !ok {
		t.Fatalf("result has no tools array: %v", res)
	}
	seen := map[string]bool{}
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("tools entry is not an object: %v", raw)
		}
		name, _ := tool["name"].(string)
		seen[name] = true
		// A host reads this list rather than the source, so a tool without a
		// schema is a tool it cannot call.
		if tool["inputSchema"] == nil {
			t.Errorf("tool %q has no inputSchema", name)
		}
	}
	if !seen["mrw_read"] || !seen["mrw_write"] {
		t.Errorf("tools = %v, want both mrw_read and mrw_write", seen)
	}
}

func TestAnUnknownMethodIsAnErrorResponse(t *testing.T) {
	lines := serve(t, `{"jsonrpc":"2.0","id":7,"method":"nonesuch/method","params":{}}`)
	if len(lines) != 1 {
		t.Fatalf("got %d response line(s), want 1 — an unknown method is answered, not dropped: %q", len(lines), lines)
	}
	got := decode(t, lines[0])
	if got["id"] != float64(7) {
		t.Errorf("id = %v, want 7", got["id"])
	}
	e, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error object in %q", lines[0])
	}
	if e["code"] != float64(-32601) {
		t.Errorf("code = %v, want -32601 (method not found)", e["code"])
	}
	// An error object carries BOTH a code and a message; a code alone is not
	// a valid JSON-RPC error.
	if msg, _ := e["message"].(string); msg == "" {
		t.Errorf("error = %v, want a non-empty message beside the code", e)
	}
}

func TestTheInitializedNotificationGetsNoResponse(t *testing.T) {
	// A notification has no id and MUST NOT be answered. Answering it is a
	// protocol violation some hosts treat as fatal.
	lines := serve(t,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	)
	if len(lines) != 1 {
		t.Fatalf("got %d response line(s), want 1 — the notification must be silent: %q", len(lines), lines)
	}
	if decode(t, lines[0])["id"] != float64(1) {
		t.Errorf("the one response is not the tools/list answer: %q", lines[0])
	}
}

func TestOnlyMCPMessagesReachStdout(t *testing.T) {
	// The spec: the server MUST NOT write anything to stdout that is not a
	// valid MCP message. This binary prints to stdout everywhere else, so the
	// rule is a live constraint rather than boilerplate.
	lines := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`not json at all`,
		``,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	)
	for i, line := range lines {
		m := decode(t, line)
		if m["jsonrpc"] != "2.0" {
			t.Errorf("stdout line %d is not an MCP message: %q", i, line)
		}
		_, hasResult := m["result"]
		_, hasErr := m["error"]
		if hasResult == hasErr {
			t.Errorf("stdout line %d has neither or both of result/error: %q", i, line)
		}
	}
}
