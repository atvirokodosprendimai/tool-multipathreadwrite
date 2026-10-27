package mcp

import "testing"

// Issue #537. A host that writes two marks before the first request is read
// as it is with one (ADR-128).
func TestTwoByteOrderMarksOnTheFirstLineStillInitialize(t *testing.T) {
	line := "\xef\xbb\xbf\xef\xbb\xbf" + `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
	resp, answer := handle(line, t.TempDir(), nil, func(any) {})
	if !answer || resp.Error != nil || resp.Result == nil {
		t.Fatalf("a double-BOM initialize was not answered with a result: %+v", resp)
	}
}
