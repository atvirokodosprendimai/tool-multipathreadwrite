package mcp

import (
	"encoding/json"
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

// The reviews of #343. verifyUnlocked releases gate during a check (ADR-121)
// and restored every per-call value but the unknown-ack note, so with two
// calls overlapping the note was lost, or put on the other call's answer.
func TestAnAckNoteStaysWithItsCallWhenCallsOverlap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	ackWrite := func(id float64, plan string, ack []any) map[string]any {
		args := map[string]any{"plan": plan}
		if ack != nil {
			args["ack"] = ack
		}
		return map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": "mrw_write", "arguments": args}}
	}
	text0 := func(m map[string]any) string {
		res, _ := m["result"].(map[string]any)
		c, _ := res["content"].([]any)
		if len(c) == 0 {
			return ""
		}
		b, _ := c[0].(map[string]any)
		s, _ := b["text"].(string)
		return s
	}
	ps := startServe(t, checkTree113(t, `{"check":"sleep 2"}`))
	ps.send(t, ackWrite(1, goEdit113, []any{"bogusA"}))
	time.Sleep(500 * time.Millisecond)
	ps.send(t, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "mrw_read", "arguments": map[string]any{"specs": []any{"notes.md"}}}})
	lines := ps.until(t, 1, 2)
	if a := text0(lines[idAt(lines, 1)]); !strings.Contains(a, "bogusA") {
		t.Errorf("a read during the write's check cleared the write's note: %.200q", a)
	}

	ps = startServe(t, checkTree113(t, `{"check":"sleep 2"}`))
	ps.send(t, ackWrite(1, goEdit113, nil))
	time.Sleep(500 * time.Millisecond)
	ps.send(t, ackWrite(2, "@@ a.go 2 replace anchor=\"func A\"\nfunc A() { _ = 2 }\n", []any{"bogusB"}))
	lines = ps.until(t, 1, 2)
	if a := text0(lines[idAt(lines, 1)]); strings.Contains(a, "bogusB") {
		t.Errorf("the first write's answer carries the second write's note: %.200q", a)
	}
	if b := text0(lines[idAt(lines, 2)]); !strings.Contains(b, "bogusB") {
		t.Errorf("the second write's note is missing: %.200q", b)
	}
}

// The Codex review of #343. A cancel names its request by value: 7.0 is the
// request 7, as validRequestID judges it, and a cancel spelled so stops it.
func TestACancelReachesACallWhoseIDIsSpelledDifferently(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	ps := startServe(t, checkTree113(t, `{"check":"sleep 30"}`))
	start := time.Now()
	ps.send(t, writeCall(7, goEdit113, nil))
	time.Sleep(500 * time.Millisecond)
	ps.send(t, map[string]any{"jsonrpc": "2.0", "method": "notifications/cancelled", "params": map[string]any{"requestId": json.RawMessage("7.0")}})
	ps.until(t, 7)
	if took := time.Since(start); took > 15*time.Second {
		t.Fatalf("a cancel for 7.0 did not stop the call 7: %s", took)
	}
	if callKey(json.RawMessage(`"a"`)) != callKey(json.RawMessage(`"\u0061"`)) || callKey(json.RawMessage("7")) == callKey(json.RawMessage(`"7"`)) {
		t.Error("a string id and its escaped spelling differ, or a number and a string of it are one")
	}
}
