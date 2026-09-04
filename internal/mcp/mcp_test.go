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

func TestPingIsAnsweredWithAnEmptyResult(t *testing.T) {
	// The spec's ping utility: "The receiver MUST respond promptly with an
	// empty response". Hosts send these on a timer, and a server that errors
	// every health check is one a host is entitled to drop — so this is a
	// base-protocol obligation, not an optional capability.
	lines := serve(t, `{"jsonrpc":"2.0","id":5,"method":"ping"}`)
	if len(lines) != 1 {
		t.Fatalf("got %d response line(s), want 1: %q", len(lines), lines)
	}
	got := decode(t, lines[0])
	if got["id"] != float64(5) {
		t.Errorf("id = %v, want 5", got["id"])
	}
	if _, isErr := got["error"]; isErr {
		t.Fatalf("ping was answered with an error: %s", lines[0])
	}
	res, ok := got["result"].(map[string]any)
	if !ok {
		t.Fatalf("ping has no result object: %s", lines[0])
	}
	if len(res) != 0 {
		t.Errorf("result = %v, want an empty object", res)
	}
}

func TestAJSONArrayIsAnInvalidRequestNotAParseError(t *testing.T) {
	// An array is valid JSON and is not a message. Refusing it is right — this
	// revision dropped batching — but "parse error" misdescribes the input.
	lines := serve(t, `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`)
	if len(lines) != 1 {
		t.Fatalf("got %d response line(s), want 1: %q", len(lines), lines)
	}
	e, ok := decode(t, lines[0])["error"].(map[string]any)
	if !ok {
		t.Fatalf("an array was not refused: %s", lines[0])
	}
	if e["code"] != float64(-32600) {
		t.Errorf("code = %v, want -32600 (invalid request), not a parse error", e["code"])
	}
}

// TestTheInstructionsTellAHostHowToAuthorAPlan asserts the handshake carries
// the field the lifecycle spec provides for "how to drive this server", and
// that what it carries is the format itself rather than a pointer to a file.
//
// The pointer case is the one worth naming: an MCP-only caller is driving mrw
// in a checkout it did not clone from here, so AGENTS.md and README.md do not
// exist for it. A reference to a file the reader cannot open reads as help and
// is not, which is worse than silence.
func TestTheInstructionsTellAHostHowToAuthorAPlan(t *testing.T) {
	lines := serve(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	if len(lines) != 1 {
		t.Fatalf("got %d response line(s), want 1", len(lines))
	}
	res, ok := decode(t, lines[0])["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result object in %q", lines[0])
	}
	got, _ := res["instructions"].(string)
	if strings.TrimSpace(got) == "" {
		t.Fatal("initialize carries no instructions — the one field the lifecycle spec provides for telling a host how to drive this server")
	}
	if !strings.Contains(got, examplePlan) {
		t.Error("the instructions do not carry the worked plan; the grammar without an example is what the tool description already was")
	}
	for _, must := range []string{"@@", "replace", "insert-after"} {
		if !strings.Contains(got, must) {
			t.Errorf("the instructions never mention %q", must)
		}
	}
	for _, file := range []string{"AGENTS.md", "README.md", "CONTRIBUTING.md"} {
		if strings.Contains(got, file) {
			t.Errorf("the instructions point at %s, which an MCP-only caller cannot open", file)
		}
	}
	// Paid on every session, by every host that reads the field. A bound is
	// what keeps this from becoming a second copy of AGENTS.md.
	if len(got) > maxInstructionsChars {
		t.Errorf("instructions are %d bytes, over the %d-byte bound", len(got), maxInstructionsChars)
	}
}

// TestTheDescriptionsSayWhenToReachForTheTool asserts the trigger, not the
// behaviour. Over MCP mrw competes with the host's own Edit and Write, and the
// description is the whole pitch: a caller told what the tool does and not when
// it wins will keep using the editor it already has.
func TestTheDescriptionsSayWhenToReachForTheTool(t *testing.T) {
	for _, tl := range tools() {
		if !strings.Contains(tl.Description, triggerRule) {
			t.Errorf("tool %q does not say when to reach for it; want the threshold %q in:\n%s", tl.Name, triggerRule, tl.Description)
		}
	}
	// The counter-advice belongs with the pitch: a tool that never says when
	// NOT to use it is advertising, and a single edit really does cost more
	// through mrw than through an ordinary editor.
	if !strings.Contains(instructionsText(), "Below that") {
		t.Error("the instructions never say when NOT to reach for mrw")
	}
}

// TestTheInstructionsTeachTheContinuation asserts the wire teaches ADR-014's
// paging AND its consequence.
//
// The consequence is the part that matters and the part prose usually drops. A
// caller told "you get a page" and not told "stopping here leaves you with part
// of a file" will stop, and the field it ignored was the only thing telling it
// otherwise. ADR-012 shipped an enum the engine never sent and ADR-013 shipped
// two examples that could not match; both were prose written beside behaviour
// instead of against it, so this checks the shipped text.
func TestTheInstructionsTeachTheContinuation(t *testing.T) {
	lines := serve(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	res, ok := decode(t, lines[0])["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result object in %q", lines[0])
	}
	got, _ := res["instructions"].(string)
	for _, must := range []string{"next_read", "PAGE", "part of a file"} {
		if !strings.Contains(got, must) {
			t.Errorf("the instructions never mention %q, so a caller meets paging as a surprise", must)
		}
	}
	// The exit condition has to be stated: a caller that does not know absence
	// terminates the loop has no way to know when it is done.
	if !strings.Contains(got, "absent") {
		t.Error("the instructions do not say that the ABSENCE of next_read is how you know you have the whole file")
	}
	if len(got) > maxInstructionsChars {
		t.Errorf("instructions are %d bytes, over the %d-byte bound — shorten what is there rather than raising it", len(got), maxInstructionsChars)
	}
}
