package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
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

// bigCheckout writes a file of n lines and returns the root and its name.
func bigCheckout(t *testing.T, n int) (root, path string) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("// padding padding padding padding padding padding padding\n")
	}
	path = "big.txt"
	if err := os.WriteFile(filepath.Join(root, path), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, path
}

func TestAReadOverTheLimitIsRefusedNotTruncated(t *testing.T) {
	// The whole point. ADR-007's cap reports itself, which is right for a human
	// reading a terminal; over MCP the consumer is a model, and a truncated file
	// that arrives looking like the file is the silent wrong answer.
	root, path := bigCheckout(t, MaxResultChars/40)
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}})

	if isErr, _ := res["isError"].(bool); !isErr {
		t.Fatalf("a read over the limit was not refused: isError=%v", res["isError"])
	}
	content, _ := res["content"].([]any)
	if len(content) == 0 {
		t.Fatal("the refusal carries no content")
	}
	txt, _ := content[0].(map[string]any)["text"].(string)
	if len(txt) > MaxResultChars {
		t.Errorf("the refusal itself is %d chars, over the %d limit", len(txt), MaxResultChars)
	}
	// It must not have served a truncated prefix dressed as the file.
	if strings.Count(txt, "padding") > 2 {
		t.Errorf("the refusal carries file content; it truncated rather than refused:\n%.200s", txt)
	}
	// And a refused read must leave no ledger entry claiming the caller saw it.
	ledger, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ledger[path]; ok {
		t.Error("a refused read recorded a ledger entry; the caller did not see this file")
	}
}

func TestTheRefusalNamesTheLimitAndARangeToRetry(t *testing.T) {
	// A refusal a caller cannot act on turns one bad call into a loop of them.
	root, path := bigCheckout(t, MaxResultChars/40)
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}})
	content, _ := res["content"].([]any)
	txt, _ := content[0].(map[string]any)["text"].(string)

	if !strings.Contains(txt, fmt.Sprint(MaxResultChars)) {
		t.Errorf("the refusal does not name the limit %d:\n%s", MaxResultChars, txt)
	}
	if !strings.Contains(txt, path) {
		t.Errorf("the refusal does not name the file:\n%s", txt)
	}
	// The range form that would fit — the caller's next move, spelled out.
	// The NUMBER matters, not just the shape: an early version derived the
	// per-line average from two different samples and suggested "big.txt:1-2"
	// for a 193 MB file, which passed a prefix check and wasted the retry it
	// exists to save.
	m := regexp.MustCompile(regexp.QuoteMeta(path) + `:1-(\d+)`).FindStringSubmatch(txt)
	if m == nil {
		t.Fatalf("the refusal does not show a range to retry with:\n%s", txt)
	}
	n, _ := strconv.Atoi(m[1])
	if n < 100 {
		t.Errorf("the suggested range is %s:1-%d, which is too small to be a useful retry", path, n)
	}
}

func TestAReadUnderTheLimitIsUnchanged(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\ntwo\nthree\n")
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}})
	if isErr, _ := res["isError"].(bool); isErr {
		t.Fatalf("an ordinary read was refused: %v", res["content"])
	}
	sc := structured(t, res)
	if p, _ := sc["problems"].(float64); p != 0 {
		t.Errorf("problems = %v, want 0", p)
	}
	ledger, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ledger[path]; !ok {
		t.Error("an ordinary read left no ledger entry")
	}
}

func TestTheCLIReadIsUnaffectedByTheMCPLimit(t *testing.T) {
	// One transport is bounded; the engine is not. This calls read.Run the way
	// cmd/mrw calls it and asserts the whole file still comes back.
	root, path := bigCheckout(t, MaxResultChars/40)
	spec, err := read.ParseSpec(path)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, problems := read.Run(&buf, root, []read.Spec{spec}, read.Options{Numbers: true})
	if problems != 0 {
		t.Fatalf("the CLI path reported %d problems for a file the MCP path refuses", problems)
	}
	if buf.Len() <= MaxResultChars {
		t.Fatalf("the fixture is not over the limit: %d <= %d", buf.Len(), MaxResultChars)
	}
}

func TestTheCappedWriterRetainsNoMoreThanItsLimit(t *testing.T) {
	// A survived mutant on 2026-09-03 showed why this test has to exist:
	// unbounding the writer left every other test green, because the refusal
	// still FIRES — cw.written still exceeds the limit — while the whole point
	// of the capped writer is that refusing is CHEAP. Nothing asserted the
	// cheapness, so nothing noticed. The measured cost of that gap was 2.4 GB.
	cw := &capped{limit: 1000}
	for i := 0; i < 100; i++ {
		chunk := bytes.Repeat([]byte("x"), 10_000)
		n, err := cw.Write(chunk)
		if err != nil || n != len(chunk) {
			t.Fatalf("Write returned %d, %v; a short write would make read.Run error", n, err)
		}
	}
	if cw.buf.Len() > cw.limit {
		t.Errorf("retained %d bytes with a limit of %d — the refusal costs the whole read", cw.buf.Len(), cw.limit)
	}
	if !cw.over {
		t.Error("the writer did not record that it went over")
	}
	if cw.written != 1_000_000 {
		t.Errorf("written = %d, want 1000000 — the refusal reports how far over the caller was", cw.written)
	}
}

func TestTheRefusalDoesNotInventAnInvalidSpec(t *testing.T) {
	// "small.txt:1-2" + ":1-N" is not valid syntax, and with several specs the
	// one that crossed the limit may not be the first. In both cases the
	// message must say what to do without inventing a spec that would fail.
	root, big := bigCheckout(t, MaxResultChars/40)
	if err := os.WriteFile(filepath.Join(root, "small.txt"), []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, specs := range map[string][]any{
		"first spec already ranged": {"small.txt:1-2", big},
		"several specs":             {big, "small.txt"},
	} {
		t.Run(name, func(t *testing.T) {
			res := call(t, root, "mrw_read", map[string]any{"specs": specs})
			txt, _ := res["content"].([]any)[0].(map[string]any)["text"].(string)
			if strings.Contains(txt, ":1-2:1-") {
				t.Errorf("the refusal invented an invalid spec:\n%s", txt)
			}
			if !strings.Contains(txt, "narrower ranges") {
				t.Errorf("the refusal gives no usable advice:\n%s", txt)
			}
			if !strings.Contains(txt, fmt.Sprint(MaxResultChars)) {
				t.Errorf("the refusal does not name the limit:\n%s", txt)
			}
		})
	}
}
