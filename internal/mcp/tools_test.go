package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// checkout makes a throwaway root with one file and returns the root and path.
// XDG_STATE_HOME is redirected per test so the ledger is this test's alone.
func checkout(t *testing.T, name, content string) (root, path string) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	path = filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, name
}

// call drives one tools/call through the real wire and returns the result
// object. Going through Serve rather than the handler keeps every test honest
// about the framing as well as the payload.
func call(t *testing.T, root, tool string, args map[string]any) map[string]any {
	t.Helper()
	params, err := json.Marshal(map[string]any{"name": tool, "arguments": args})
	if err != nil {
		t.Fatal(err)
	}
	req, err := json.Marshal(request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/call", Params: params})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Serve(strings.NewReader(string(req)+"\n"), &out, root); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	line := strings.TrimSuffix(out.String(), "\n")
	var resp map[string]any
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, line)
	}
	if e, ok := resp["error"]; ok {
		t.Fatalf("tools/call returned a JSON-RPC error: %v", e)
	}
	res, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result object: %s", line)
	}
	return res
}

// cliWrite applies the same plan the way cmd/mrw does, so a test can compare
// the two transports rather than compare MCP against a hand-written expectation.
func cliWrite(t *testing.T, root, planText string) apply.Result {
	t.Helper()
	hunks, err := plan.Parse(strings.NewReader(planText))
	if err != nil {
		t.Fatalf("plan.Parse: %v", err)
	}
	in := make([]apply.Input, 0, len(hunks))
	for _, h := range hunks {
		in = append(in, apply.Input{
			Path: h.Path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
			Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor,
			SrcLine: h.SrcLine, Index: h.Index,
		})
	}
	ledger, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := apply.Apply(root, in, apply.Options{Seen: ledger})
	if err != nil {
		t.Fatalf("apply.Apply: %v", err)
	}
	return res
}

// structured pulls structuredContent out of a CallToolResult.
func structured(t *testing.T, res map[string]any) map[string]any {
	t.Helper()
	sc, ok := res["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("result has no structuredContent object: %v", res)
	}
	return sc
}

func TestTheToolResultCarriesContentAndStructuredContent(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\ntwo\n")
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}})

	// The spec requires a content array. A host may reject or hide a tool that
	// answers with a bare result, which is why this is its own test rather than
	// an assertion buried in another.
	content, ok := res["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("result has no content array: %v", res)
	}
	block, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] is not an object: %v", content[0])
	}
	if block["type"] != "text" {
		t.Errorf(`content[0].type = %v, want "text"`, block["type"])
	}
	if txt, _ := block["text"].(string); txt == "" {
		t.Error("content[0].text is empty")
	}
	structured(t, res) // and the machine-readable half is present too
}

func TestTheWriteToolReturnsTheSameResultAsTheCLI(t *testing.T) {
	// ADR-010's Enforced-by. One engine, one answer: the verdict a caller gets
	// over MCP is the value the CLI would have produced for the same plan.
	planText := "@@ a.txt 1 replace\nONE\n"

	// Two identical checkouts so both transports face the same starting tree.
	mcpRoot, path := checkout(t, "a.txt", "one\ntwo\n")
	call(t, mcpRoot, "mrw_read", map[string]any{"specs": []any{path}})
	got := structured(t, call(t, mcpRoot, "mrw_write", map[string]any{"plan": planText}))

	cliRoot, cliPath := checkout(t, "a.txt", "one\ntwo\n")
	if _, err := os.ReadFile(filepath.Join(cliRoot, cliPath)); err != nil {
		t.Fatal(err)
	}
	// Read first on this side too, so read-before-write is satisfied identically.
	call(t, cliRoot, "mrw_read", map[string]any{"specs": []any{cliPath}})
	want := cliWrite(t, cliRoot, planText)

	// Compare the DECODED structuredContent against the CLI's result marshalled
	// by the same encoder. Raw JSON-RPC equality would compare the envelope,
	// which is the protocol's and not mrw's.
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var wantMap map[string]any
	if err := json.Unmarshal(wantJSON, &wantMap); err != nil {
		t.Fatal(err)
	}
	// Root differs by construction (two temp dirs); every other field must match.
	delete(got, "root")
	delete(wantMap, "root")
	gotJSON, _ := json.Marshal(got)
	normJSON, _ := json.Marshal(wantMap)
	if string(gotJSON) != string(normJSON) {
		t.Errorf("the two transports disagree about what happened.\n MCP: %s\n CLI: %s", gotJSON, normJSON)
	}
	if applied, _ := got["applied"].(bool); !applied {
		t.Errorf("the plan did not apply over MCP: %s", gotJSON)
	}
}

func TestTheReadToolObservesWhatTheCLIWouldObserve(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\ntwo\nthree\n")
	call(t, root, "mrw_read", map[string]any{"specs": []any{path + ":1-2"}})

	ledger, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	obs, ok := ledger[path]
	if !ok {
		t.Fatalf("a read over MCP left no ledger entry; ledger = %v", ledger)
	}
	if obs.SHA == "" {
		t.Error("the ledger entry carries no SHA, so a later write cannot tell whether the file moved")
	}
	if len(obs.Spans) == 0 {
		t.Errorf("a partial read recorded no spans: %+v — the per-LINE guarantee needs them", obs)
	}
}

func TestAnMCPReadLicensesACLIWrite(t *testing.T) {
	// One guarantee, not one per transport. This is what makes ADR-002 hold
	// across both, and it works because there is one ledger on disk.
	root, path := checkout(t, "a.txt", "one\ntwo\n")
	call(t, root, "mrw_read", map[string]any{"specs": []any{path}})

	res := cliWrite(t, root, "@@ a.txt 1 replace\nONE\n")
	if !res.Applied || res.Failed != 0 {
		t.Fatalf("a CLI write after an MCP read was refused: %+v", res)
	}
}

func TestAWriteToAnUnreadFileIsRefusedOverMCP(t *testing.T) {
	// ADR-002 on the new path. Nothing was read, so nothing may be written.
	root, _ := checkout(t, "a.txt", "one\ntwo\n")
	got := structured(t, call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nONE\n"}))

	if applied, _ := got["applied"].(bool); applied {
		t.Fatalf("a write to an unread file was applied over MCP: %v", got)
	}
	if failed, _ := got["failed"].(float64); failed == 0 {
		t.Errorf("failed = 0, want a refusal recorded: %v", got)
	}
	// And the refusal must not have touched the file.
	b, err := os.ReadFile(filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "one\ntwo\n" {
		t.Errorf("the file changed despite the refusal: %q", b)
	}
}

func TestConcurrentToolCallsDoNotLoseALedgerEntry(t *testing.T) {
	// The concurrency gap ADR-010 closes: parallel CLI PROCESSES race because
	// the ledger is a whole-file rewrite. One server is one writer, so calls
	// made through it must serialize.
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	const n = 12
	names := make([]string, n)
	for i := range names {
		names[i] = "f" + string(rune('a'+i)) + ".txt"
		if err := os.WriteFile(filepath.Join(root, names[i]), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(spec string) {
			defer wg.Done()
			params, _ := json.Marshal(map[string]any{
				"name": "mrw_read", "arguments": map[string]any{"specs": []any{spec}},
			})
			req, _ := json.Marshal(request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/call", Params: params})
			var out bytes.Buffer
			if err := Serve(strings.NewReader(string(req)+"\n"), &out, root); err != nil {
				t.Errorf("Serve: %v", err)
			}
		}(name)
	}
	wg.Wait()

	ledger, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, name := range names {
		if _, ok := ledger[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d of %d reads left no ledger entry: %v", len(missing), n, missing)
	}
}
