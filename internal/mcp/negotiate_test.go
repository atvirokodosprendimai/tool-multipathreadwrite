package mcp

import (
	"encoding/json"
	"fmt"
	"testing"
)

// ADR-067 T2. initialize answers the version the host asked for when mrw
// speaks it. Claude Code 2.1.281 asks for 2025-11-25 on every start (wire
// capture, 2026-09-24); mrw answered 2025-06-18 whatever was asked.

func initResult(t *testing.T, params string) map[string]any {
	t.Helper()
	lines := serve(t, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":%s}`, params))
	if len(lines) != 1 {
		t.Fatalf("initialize answered %d lines, want 1", len(lines))
	}
	res, ok := decode(t, lines[0])["result"].(map[string]any)
	if !ok {
		t.Fatalf("initialize has no result: %s", lines[0])
	}
	return res
}

func TestInitializeEchoesASupportedRequestedVersion(t *testing.T) {
	for _, v := range []string{"2025-11-25", "2025-06-18"} {
		res := initResult(t, fmt.Sprintf(`{"protocolVersion":%q}`, v))
		if res["protocolVersion"] != v {
			t.Errorf("asked for %s, answered %v", v, res["protocolVersion"])
		}
	}
}

func TestInitializeAnswersItsLatestVersionForAnUnknownOne(t *testing.T) {
	for _, params := range []string{`{"protocolVersion":"2024-11-05"}`, `{}`, `null`} {
		res := initResult(t, params)
		if res["protocolVersion"] != "2025-11-25" {
			t.Errorf("params %s: answered %v, want the latest, 2025-11-25", params, res["protocolVersion"])
		}
	}
}

func TestServerInfoCarriesATitleAndADescription(t *testing.T) {
	info, _ := initResult(t, `{"protocolVersion":"2025-11-25"}`)["serverInfo"].(map[string]any)
	if info["name"] != "mrw" || info["version"] != Version {
		t.Errorf("serverInfo name/version = %v/%v, want mrw/%s", info["name"], info["version"], Version)
	}
	for _, k := range []string{"title", "description"} {
		if s, _ := info[k].(string); s == "" {
			t.Errorf("serverInfo carries no %s: %v", k, info)
		}
	}
}

// An encode failure is the server's, not the request's: -32603, not -32600.
func TestAResultThatCannotBeEncodedIsAnInternalError(t *testing.T) {
	resp, _ := resultResponse(json.RawMessage("7"), make(chan int))
	if resp.Error == nil || resp.Error.Code != codeInternal {
		t.Fatalf("an unencodable result answered %+v, want code %d", resp.Error, codeInternal)
	}
}

// MCP: a request id MUST NOT be null. One that is gets -32600 and is not run.
func TestARequestWithANullIdIsInvalid(t *testing.T) {
	lines := serve(t, `{"jsonrpc":"2.0","id":null,"method":"tools/list"}`)
	if len(lines) != 1 {
		t.Fatalf("a null-id request answered %d lines, want 1", len(lines))
	}
	got := decode(t, lines[0])
	if _, ran := got["result"]; ran {
		t.Fatalf("a null-id request was dispatched: %s", lines[0])
	}
	e, _ := got["error"].(map[string]any)
	if e["code"] != float64(codeInvalidRequest) || got["id"] != nil {
		t.Errorf("a null-id request answered %s, want -32600 with a null id", lines[0])
	}
}
