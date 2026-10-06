package mcp

import (
	"encoding/json"
	"runtime"
	"testing"
)

// The Codex review of v1.42.0..v1.47.0, finding 2. A write that asked for a
// step released Serve's loop even when no step would run — a dry run, a plan
// refused at validation — so a later quick call could be answered before it,
// against ADR-121's rule that only a running check or step gives up its place.
func TestARequestedStepReleasesTheLoopOnlyWhenTheWriteLanded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the step is a POSIX shell line")
	}
	harness := `{"check":"true","steps":{"ok":"true"}}`
	for _, tc := range []struct {
		name string
		args map[string]any
		want bool
	}{
		{"a prose write with a step", map[string]any{"plan": "@@ notes.md 2 replace\nline 2\n", "then": []any{"ok"}}, true},
		{"a dry run with a step", map[string]any{"plan": "@@ notes.md 2 replace\nline 2\n", "then": []any{"ok"}, "dry_run": true}, false},
		{"a refused plan with a step", map[string]any{"plan": "@@ notes.md 99 replace\nline 2\n", "then": []any{"ok"}}, false},
	} {
		root := checkTree113(t, harness)
		raw, _ := json.Marshal(map[string]any{"name": "mrw_write", "arguments": tc.args})
		released := false
		if _, rpcErr := callTool(root, raw, false, func() { released = true }); rpcErr != nil {
			t.Fatalf("%s: %v", tc.name, rpcErr.Message)
		}
		if released != tc.want {
			t.Errorf("%s: released %v, want %v", tc.name, released, tc.want)
		}
	}
}
