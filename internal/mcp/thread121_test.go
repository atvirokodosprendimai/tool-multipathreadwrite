package mcp

import (
	"bufio"
	"encoding/json"
	"io"
	"runtime"
	"testing"
	"time"
)

// pipeServe is a Serve driven through pipes, as a host drives it.
type pipeServe struct {
	in    *io.PipeWriter
	lines chan map[string]any
	done  chan error
}

func startServe(t *testing.T, root string) *pipeServe {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ps := &pipeServe{in: inW, lines: make(chan map[string]any, 64), done: make(chan error, 1)}
	go func() {
		err := Serve(inR, outW, root)
		_ = outW.Close()
		ps.done <- err
	}()
	go func() {
		sc := bufio.NewScanner(outR)
		sc.Buffer(make([]byte, 1<<20), 64<<20)
		for sc.Scan() {
			var m map[string]any
			if err := json.Unmarshal(sc.Bytes(), &m); err == nil {
				ps.lines <- m
			}
		}
		close(ps.lines)
	}()
	t.Cleanup(func() { _ = inW.Close() })
	return ps
}

func (ps *pipeServe) send(t *testing.T, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ps.in.Write(append(b, '\n')); err != nil {
		t.Fatal(err)
	}
}

// until reads lines until every id in want has an answer, or 30 s pass.
func (ps *pipeServe) until(t *testing.T, want ...float64) []map[string]any {
	t.Helper()
	var got []map[string]any
	left := map[float64]bool{}
	for _, id := range want {
		left[id] = true
	}
	deadline := time.After(30 * time.Second)
	for len(left) > 0 {
		select {
		case m, ok := <-ps.lines:
			if !ok {
				t.Fatalf("output ended with %v unanswered", left)
			}
			got = append(got, m)
			if id, ok := m["id"].(float64); ok {
				delete(left, id)
			}
		case <-deadline:
			t.Fatalf("no answer for %v in 30 s; got %v", left, got)
		}
	}
	return got
}

func idAt(lines []map[string]any, id float64) int {
	for i, m := range lines {
		if m["id"] == id {
			return i
		}
	}
	return -1
}

func writeCall(id float64, plan string, meta map[string]any) map[string]any {
	params := map[string]any{"name": "mrw_write", "arguments": map[string]any{"plan": plan}}
	if meta != nil {
		params["_meta"] = meta
	}
	return map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": params}
}

// ADR-121. A write's check held the server's only thread: a ping sent after it
// waited for the whole check, though the spec says a ping is answered promptly.
func TestAPingIsAnsweredWhileAWritesCheckRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	ps := startServe(t, checkTree113(t, `{"check":"sleep 2"}`))
	ps.send(t, writeCall(1, goEdit113, nil))
	time.Sleep(300 * time.Millisecond) // the write has reached its check
	ps.send(t, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "ping"})
	lines := ps.until(t, 1, 2)
	if idAt(lines, 2) > idAt(lines, 1) {
		t.Fatalf("the ping was answered after the write's check: %v", lines)
	}
}

// A call that carries a progress token hears progress while it runs, and none
// after its answer (the host's idle window resets on progress, BACKLOG "From
// ADR-113").
func TestProgressIsSentWhileACallRunsAndNotAfter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	old := progressEvery
	progressEvery = 100 * time.Millisecond
	t.Cleanup(func() { progressEvery = old })
	ps := startServe(t, checkTree113(t, `{"check":"sleep 1"}`))
	ps.send(t, writeCall(1, goEdit113, map[string]any{"progressToken": "tok"}))
	lines := ps.until(t, 1)
	before := 0
	for _, m := range lines {
		if p, _ := m["params"].(map[string]any); m["method"] == "notifications/progress" && p["progressToken"] == "tok" {
			before++
		}
	}
	if before == 0 {
		t.Fatalf("no progress while the write ran: %v", lines)
	}
	time.Sleep(300 * time.Millisecond)
	_ = ps.in.Close()
	for m := range ps.lines {
		if m["method"] == "notifications/progress" {
			t.Fatalf("progress after the answer: %v", m)
		}
	}
}

// Requests with no check are answered in the order they arrived.
func TestQuickAnswersKeepTheirOrder(t *testing.T) {
	ps := startServe(t, checkTree113(t, ""))
	for id := 1; id <= 6; id++ {
		if id == 3 {
			ps.send(t, map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/list"})
			continue
		}
		ps.send(t, map[string]any{"jsonrpc": "2.0", "id": id, "method": "ping"})
	}
	lines := ps.until(t, 1, 2, 3, 4, 5, 6)
	for i := 1; i < 6; i++ {
		if idAt(lines, float64(i)) > idAt(lines, float64(i+1)) {
			t.Fatalf("answers out of order: %v", lines)
		}
	}
}

// Input closed while a write's check runs: the write is still answered before
// Serve returns.
func TestServeAnswersACallInFlightAtEndOfInput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	ps := startServe(t, checkTree113(t, `{"check":"sleep 1"}`))
	ps.send(t, writeCall(1, goEdit113, nil))
	time.Sleep(300 * time.Millisecond)
	_ = ps.in.Close()
	ps.until(t, 1)
	select {
	case err := <-ps.done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve did not return")
	}
}

// A call that runs during a modern write's check sets and clears the per-call
// era; the write takes its own back when it resumes, so its answer still
// carries the modern fields (ADR-067, ADR-121).
func TestAModernWriteKeepsItsDecorationWhenACallRunsDuringItsCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	ps := startServe(t, checkTree113(t, `{"check":"sleep 2"}`))
	ps.send(t, writeCall(1, goEdit113, map[string]any{
		metaVersion: modernVersion, metaCapabilities: map[string]any{},
	}))
	time.Sleep(300 * time.Millisecond)
	ps.send(t, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{"name": "mrw_read", "arguments": map[string]any{"specs": []any{"notes.md"}}}})
	lines := ps.until(t, 1, 2)
	w := lines[idAt(lines, 1)]
	res, _ := w["result"].(map[string]any)
	if res == nil || res["resultType"] != "complete" {
		t.Fatalf("the modern write's answer lost its decoration after a call ran during its check: %v", w)
	}
}

// The loop is released only by a write whose check or step will run: a write
// with neither keeps its place in the answer order (the review of #327).
func TestTheLoopIsReleasedOnlyWhenACheckOrStepRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	for _, tc := range []struct {
		name    string
		harness string
		args    map[string]any
		want    bool
	}{
		{"a code write whose check runs", `{"check":"true"}`, map[string]any{"plan": goEdit113}, true},
		{"check: false", `{"check":"true"}`, map[string]any{"plan": goEdit113, "check": false}, false},
		{"a prose write", `{"check":"true"}`, map[string]any{"plan": "@@ notes.md 2 replace\nline 2\n"}, false},
		{"a dry run", `{"check":"true"}`, map[string]any{"plan": goEdit113, "dry_run": true}, false},
	} {
		root := checkTree113(t, tc.harness)
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
