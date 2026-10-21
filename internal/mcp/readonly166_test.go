package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-166 T2. mrw_read with an unusable state directory failed a content read
// outright, JSON-RPC -32603 "holding checkpoints: …", so a sandboxed host got
// nothing. A content read now answers without checkpoints, a line at the top
// saying so, and a page answers the same way with its next_read; reads whose
// purpose is the licence (quiet, ack, a sub plan) answer the error as before.
// XDG_STATE_HOME a regular file fails for any user, root included.
func TestMrwReadWithAnUnwritableStateServesWithoutCheckpoints(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "big.txt"), []byte(strings.Repeat("x\n", 150000)), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", file)
	const line = "-- no checkpoint was held, so nothing here can be acknowledged and a write to it is refused:"

	got := ask(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}})
	if got.rpc || got.isError || !strings.HasPrefix(got.text, line) || !strings.Contains(got.text, "1| one") || strings.Contains(got.text, "-- ck ") {
		t.Errorf("a whole content read: %+v", got)
	}
	if !strings.Contains(got.text, "XDG_STATE_HOME="+file) {
		t.Errorf("the line does not name the state directory: %q", got.text)
	}

	page := ask(t, root, "mrw_read", map[string]any{"specs": []any{"big.txt"}})
	if page.rpc || page.isError || !strings.HasPrefix(page.text, line) || strings.Contains(page.text, "-- ck ") || !strings.Contains(page.text, "-- PARTIAL: lines 1-") || !strings.Contains(page.text, `big.txt:`) {
		t.Errorf("a paged content read: rpc %v isError %v, text %.300q", page.rpc, page.isError, page.text)
	}
	if strings.Contains(page.text, "acknowledge it") {
		t.Errorf("a page with no checkpoint still asks to be acknowledged: %.300q", page.text)
	}

	for name, args := range map[string]map[string]any{
		"quiet": {"specs": []any{"a.txt"}, "quiet": true},
		"ack":   {"specs": []any{"a.txt"}, "ack": []any{"x"}},
		"a sub": {"grep": "one", "sub": "uno"},
	} {
		if a := ask(t, root, "mrw_read", args); !a.rpc && !a.isError {
			t.Errorf("%s, a licence read, answered without an error: %+v", name, a)
		}
	}
}
