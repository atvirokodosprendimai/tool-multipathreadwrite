package mcp

import (
	"context"
	"encoding/json"
)

// callTool is callToolCtx under a context nothing cancels, for the tests that
// drive one call directly. Serve routes through callToolCtx with the call's own
// context (ADR-128), so this lives with the tests (static analysis: nothing in
// production may be reached only by tests).
func callTool(root string, raw json.RawMessage, modern bool, release func()) (callToolResult, *rpcError) {
	return callToolCtx(context.Background(), root, raw, modern, release)
}
