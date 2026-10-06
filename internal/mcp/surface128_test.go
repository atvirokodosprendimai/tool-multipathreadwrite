package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ADR-128. A UTF-8 byte-order mark before the first request — a Windows host's
// default for a text stream — made initialize a parse error, -32700, and the
// session never started.
func TestABOMOnTheFirstLineStillInitializes(t *testing.T) {
	line := string([]byte{0xEF, 0xBB, 0xBF}) + `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
	resp, answer := handle(line, t.TempDir(), nil, func(any) {})
	if !answer || resp.Error != nil || resp.Result == nil {
		t.Fatalf("a BOM'd initialize was not answered with a result: %+v", resp)
	}
}

// ADR-128. mrw_write has no force argument, and its read-before-modify
// refusals told the caller to "pass --force", advice that can never be taken.
func TestAnMCPRefusalDoesNotAdviseForce(t *testing.T) {
	root := checkTree113(t, "")
	if err := os.WriteFile(filepath.Join(root, "unread.md"), []byte("never served\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ unread.md 1 replace\nx\n"})
	text := firstText(t, res)
	if !strings.Contains(text, "has not been read") || strings.Contains(text, "--force") {
		t.Errorf("an MCP refusal advised --force, or was not the read refusal:\n%s", text)
	}
}

// ADR-128. An ack id that matched no pending checkpoint was skipped in
// silence, so the caller learned nothing until a later write was refused.
func TestAnUnknownAckIsNamed(t *testing.T) {
	root := checkTree113(t, "")
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"notes.md"}, "ack": []any{"nosuchid1"}})
	if text := firstText(t, res); !strings.Contains(text, "matched no checkpoint") || !strings.Contains(text, "nosuchid1") {
		t.Errorf("the unknown ack id was not named:\n%s", text)
	}
}

// ADR-128, BACKLOG "From ADR-121". A host's notifications/cancelled was
// dropped, so a write's long check ran to its own bound. The cancel stops it,
// and the check is reported interrupted.
func TestACancelStopsAWritesRunningCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	ps := startServe(t, checkTree113(t, `{"check":"sleep 30"}`))
	start := time.Now()
	ps.send(t, writeCall(7, goEdit113, nil))
	time.Sleep(500 * time.Millisecond) // the write has reached its check
	ps.send(t, map[string]any{"jsonrpc": "2.0", "method": "notifications/cancelled", "params": map[string]any{"requestId": 7, "reason": "test"}})
	lines := ps.until(t, 7)
	if took := time.Since(start); took > 15*time.Second {
		t.Fatalf("the cancelled write answered after %s: the check ran to its end", took)
	}
	w := lines[idAt(lines, 7)]
	res, _ := w["result"].(map[string]any)
	sc, _ := res["structuredContent"].(map[string]any)
	chk, _ := sc["check"].(map[string]any)
	if chk == nil || chk["skipped"] != "interrupted" {
		t.Errorf("the cancelled check was not reported interrupted: %v", w)
	}
}
