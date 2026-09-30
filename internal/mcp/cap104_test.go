package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// ADR-104 T2. Serve read each request with ReadString, which grows without
// limit: one line with no newline was read whole. A line over the limit is
// answered -32600 with a null id naming the limit, its rest is discarded, and
// the next line is served.
func TestAnOversizedRequestIsRefusedAndTheServerKeepsServing(t *testing.T) {
	old := maxRequestBytes
	maxRequestBytes = 1024
	t.Cleanup(func() { maxRequestBytes = old })
	big := `{"jsonrpc":"2.0","id":7,"method":"tools/list","params":{"pad":"` + strings.Repeat("a", 4096) + `"}}`
	list := `{"jsonrpc":"2.0","id":8,"method":"tools/list"}`
	var out bytes.Buffer
	if err := Serve(strings.NewReader(big+"\n"+list+"\n"), &out, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want two answers (the refusal, then the tool list), got %d:\n%.600s", len(lines), out.String())
	}
	var first, second map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatal(err)
	}
	e, _ := first["error"].(map[string]any)
	if first["id"] != nil || e["code"] != float64(-32600) || !strings.Contains(asString(e["message"]), "1024") {
		t.Errorf("the oversized line's answer: %v", first)
	}
	if second["id"] != float64(8) || second["result"] == nil {
		t.Errorf("the next line was not served: %v", second)
	}
}
