package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// answer is how one tools/call came back: a JSON-RPC error or a result, with
// the words a caller reads (the content text, and a receipt's error).
type answer struct {
	rpc     bool
	code    float64
	isError bool
	text    string
}

func ask(t *testing.T, root, tool string, args map[string]any) answer {
	t.Helper()
	params, err := json.Marshal(map[string]any{"name": tool, "arguments": args})
	if err != nil {
		t.Fatal(err)
	}
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":%s}`, params)
	var out bytes.Buffer
	if err := Serve(strings.NewReader(req+"\n"), &out, root); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := decode(t, strings.TrimSuffix(out.String(), "\n"))
	if e, ok := resp["error"].(map[string]any); ok {
		code, _ := e["code"].(float64)
		return answer{rpc: true, code: code, text: asString(e["message"])}
	}
	res, _ := resp["result"].(map[string]any)
	a := answer{isError: res["isError"] == true}
	if c, ok := res["content"].([]any); ok && len(c) > 0 {
		b, _ := c[0].(map[string]any)
		a.text = asString(b["text"])
	}
	if sc, ok := res["structuredContent"].(map[string]any); ok {
		a.text += "\n" + asString(sc["error"])
	}
	return a
}

// With the state directory unusable, every MCP route that fails in it says so,
// as the CLI's read and write do (ADR-164 Decision 3): mrw's state directory and
// the variable that moves it. XDG_STATE_HOME a regular file fails for any user,
// root included. The shape is the route's own and does not move: a JSON-RPC
// -32603 stays one, an isError result stays one. The pair: an error that is not
// about the state directory is as it was.
func TestTheMCPSurfaceNamesItsStateDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Past one answer's ceiling, so the read pages and holds a checkpoint from firstPage.
	if err := os.WriteFile(filepath.Join(root, "big.txt"), []byte(strings.Repeat("x\n", 150000)), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", file)
	plan := "@@ a.txt 1 replace\nuno\n"
	for _, c := range []struct {
		name string
		tool string
		args map[string]any
		rpc  bool // today's shape: a JSON-RPC error, else an isError result
	}{
		{"a read holds its checkpoints", "mrw_read", map[string]any{"specs": []any{"a.txt"}}, true},
		{"a paged read holds its checkpoints", "mrw_read", map[string]any{"specs": []any{"big.txt"}}, false},
		{"a quiet read records", "mrw_read", map[string]any{"specs": []any{"a.txt"}, "quiet": true}, true},
		{"a read promotes an ack", "mrw_read", map[string]any{"specs": []any{"a.txt"}, "ack": []any{"x"}}, true},
		{"a read plan with a path loads the working set", "mrw_read", map[string]any{"grep": "one", "sub": "uno", "specs": []any{"a.txt"}}, false},
		{"a read plan records the planned lines", "mrw_read", map[string]any{"grep": "one", "sub": "uno"}, true},
		{"a write loads the ledger", "mrw_write", map[string]any{"plan": plan}, true},
		{"a write promotes an ack", "mrw_write", map[string]any{"plan": plan, "ack": []any{"x"}}, true},
		{"a write plan with a path loads the working set", "mrw_write", map[string]any{"grep": "one", "sub": "uno", "paths": []any{"a.txt"}}, false},
		{"a write plan records the planned lines", "mrw_write", map[string]any{"grep": "one", "sub": "uno"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := ask(t, root, c.tool, c.args)
			if got.rpc != c.rpc || (c.rpc && got.code != codeInternal) || (!c.rpc && !got.isError) {
				t.Errorf("shape moved: %+v, want rpc %v (code %d) or an isError result", got, c.rpc, codeInternal)
			}
			for _, w := range []string{"mrw's state directory", "XDG_STATE_HOME=" + file} {
				if !strings.Contains(got.text, w) {
					t.Errorf("%q lacks %q", got.text, w)
				}
			}
		})
	}

	// The pair: a refusal that has nothing to do with the state directory keeps
	// its words and its shape, here and in the same unusable state.
	got := ask(t, root, "mrw_write", map[string]any{"plan": "   "})
	if got.rpc || !got.isError || strings.Contains(got.text, "state directory") {
		t.Errorf("an error not about the state directory changed: %+v", got)
	}
}

// A hint that does not fit must not turn an answer into another kind: a plan
// maker's isError refusal that fit the ceiling before the hint keeps its kind
// and its words when the hinted one would not fit, rather than falling through
// withinCeiling to a JSON-RPC error (Codex, review of #470).
func TestAHintThatDoesNotFitLeavesTheAnswerAnIsErrorResult(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", file)
	args := map[string]any{"grep": "one", "sub": "uno", "specs": []any{"a.txt"}}
	hinted := ask(t, root, "mrw_read", args)
	i := strings.Index(hinted.text, " — mrw's state directory")
	if hinted.rpc || !hinted.isError || i < 0 {
		t.Fatalf("the fixture: %+v", hinted)
	}
	plain := hinted.text[:i]
	fit := encodedSize(decorateCall(errorResult(plain)))
	if fit >= encodedSize(decorateCall(errorResult(hinted.text))) {
		t.Fatalf("the fixture: the hint adds nothing to measure")
	}
	withCeiling(t, fit+8)
	for _, tc := range []struct {
		tool string
		args map[string]any
	}{
		{"mrw_read", args},
		{"mrw_write", map[string]any{"grep": "one", "sub": "uno", "paths": []any{"a.txt"}}},
	} {
		got := ask(t, root, tc.tool, tc.args)
		if got.rpc || !got.isError || got.text != plain {
			t.Errorf("%s at a ceiling the hinted answer overflows: %+v, want the plain isError %q", tc.tool, got, plain)
		}
	}
}

// The landing is the other half: the write lock a directory blocks, and a
// ledger that cannot be updated after the tree changed. Both answer as they did
// (an isError receipt, the ledger's carrying the landed write) and name the
// state directory.
func TestTheMCPLandingNamesItsStateDirectory(t *testing.T) {
	root := licensed(t, map[string]string{"a.txt": "one\n"})
	dir, err := state.Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(dir, "seen.write.lock")
	// The ledger's own lock a directory: Prepare does not load the ledger, Land does
	// (seen.Snapshot, state.Hold), and answers a JSON-RPC error, nothing written.
	sl := filepath.Join(dir, "seen.lock")
	_ = os.Remove(sl) // seen.Record left the lock file; a directory goes in its place
	if err := os.Mkdir(sl, 0o700); err != nil {
		t.Fatal(err)
	}
	got := ask(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nuno\n"})
	if !got.rpc || got.code != codeInternal || !strings.Contains(got.text, "mrw's state directory") {
		t.Errorf("a ledger lock that is a directory: %+v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.txt")); string(b) != "one\n" {
		t.Errorf("a.txt changed: %q", b)
	}
	if err := os.Remove(sl); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	got = ask(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nuno\n"})
	if got.rpc || !got.isError || !strings.Contains(got.text, "mrw's state directory") {
		t.Errorf("a write lock that is a directory: %+v", got)
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}

	skipUnlessPermissionsBind(t)
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ledger, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
	got = ask(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nuno\n"})
	if got.rpc || !got.isError || !strings.Contains(got.text, "mrw's state directory") {
		t.Errorf("a ledger that cannot be updated: %+v", got)
	}
}
