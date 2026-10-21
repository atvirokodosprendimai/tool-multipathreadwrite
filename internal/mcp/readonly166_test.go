package mcp

import (
	"encoding/json"
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

	// Every page answers without checkpoints and within the ceiling, and
	// following next_read gives back the whole file (Codex review of f2e6f008).
	spec, lines := "big.txt", 0
	for page := 0; ; page++ {
		if page > 40 {
			t.Fatalf("still paging after %d pages", page)
		}
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{spec}})
		body := served0(t, res)
		if isErr, _ := res["isError"].(bool); isErr || !strings.HasPrefix(body, line) || strings.Contains(body, "-- ck ") || strings.Contains(body, "acknowledge it") {
			t.Fatalf("page %d (%s): isError %v, text %.300q", page, spec, isErr, body)
		}
		raw, err := json.Marshal(res)
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) > MaxResultChars {
			t.Fatalf("page %d encodes to %d, over the %d ceiling", page, len(raw), MaxResultChars)
		}
		lines += len(numberedLines(t, body))
		next := nextOf(t, res)
		if page == 0 && (next == "" || !strings.Contains(body, "-- PARTIAL: lines 1-")) {
			t.Fatalf("the first answer is not a page:\n%.400s", body)
		}
		if next == "" {
			break
		}
		if next == spec {
			t.Fatalf("page %d hands back its own spec %q", page, next)
		}
		spec = next
	}
	if lines != 150000 {
		t.Errorf("reassembled %d lines, want 150000", lines)
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
