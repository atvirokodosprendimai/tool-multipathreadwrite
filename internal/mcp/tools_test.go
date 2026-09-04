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
	// ADR-007's cap reports itself, which is right for a human reading a
	// terminal; over MCP the consumer is a model, and a truncated file that
	// arrives looking like the file is the silent wrong answer.
	//
	// ⚠ RETARGETED BY ADR-014. A SINGLE oversized spec now returns a first page
	// and a continuation, and records the span it served — so "carries no file
	// content" and "records no ledger entry" are deliberately false there, and
	// TestAPagedReadReassemblesTheWholeFile and TestAPageLicensesOnlyWhatItServed
	// carry that case. What ADR-014 does NOT change is the multi-spec request:
	// the spec that crossed the limit may not be the first, so paging one of
	// them could serve the wrong file, and the flat refusal still applies. This
	// test now pins that branch, which is where its assertions are still true.
	root, path := bigCheckout(t, MaxResultChars/40)
	other := path + ".2"
	if err := os.WriteFile(filepath.Join(root, other), []byte("// small\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path, other}})

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
	//
	// ⚠ RETARGETED BY ADR-014, and the retargeting is the interesting part. For
	// a single file the "range to retry" is no longer prose to be parsed: the
	// caller is handed the page AND `next_read`, which is strictly more
	// actionable, and the paging tests assert it. What remains prose — and so
	// remains here — is the multi-spec case, where mrw cannot know which file
	// to narrow and says so with a line budget instead of inventing a spec.
	root, path := bigCheckout(t, MaxResultChars/40)
	other := path + ".2"
	if err := os.WriteFile(filepath.Join(root, other), []byte("// small\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path, other}})
	content, _ := res["content"].([]any)
	txt, _ := content[0].(map[string]any)["text"].(string)

	if !strings.Contains(txt, fmt.Sprint(MaxResultChars)) {
		t.Errorf("the refusal does not name the limit %d:\n%s", MaxResultChars, txt)
	}
	// A line budget the caller can act on. The NUMBER matters, not just the
	// shape: an early version derived the per-line average from two different
	// samples and suggested a two-line range for a 193 MB file, which passed a
	// prefix check and wasted the retry it exists to save.
	m := regexp.MustCompile(`around (\d+) lines per file`).FindStringSubmatch(txt)
	if m == nil {
		t.Fatalf("the refusal gives no line budget to retry with:\n%s", txt)
	}
	n, _ := strconv.Atoi(m[1])
	if n < 100 {
		t.Errorf("the suggested budget is %d lines, too small to be a useful retry", n)
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

// nextOf pulls the continuation spec out of a read result, or "" when the
// result names none — which is how a caller knows it has reached the end.
func nextOf(t *testing.T, res map[string]any) string {
	t.Helper()
	sc, ok := res["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("no structuredContent in %v", res)
	}
	n, _ := sc["next_read"].(string)
	return n
}

// TestAPagedReadReassemblesTheWholeFile is ADR-014's Enforced-by.
//
// It FOLLOWS the continuation to exhaustion and compares the reassembly to the
// file, rather than checking that a continuation exists. A spec that points at
// the wrong lines — off by one, or at the same page forever — passes every
// presence check and loses or repeats content, which is the silent wrong answer
// the whole project refuses. It also bounds its own iterations, so a
// continuation that never clears fails the test instead of hanging it.
func TestAPagedReadReassemblesTheWholeFile(t *testing.T) {
	// Large enough that the first read is over the limit several times over.
	const lines = 12000
	root, path := bigCheckout(t, lines)

	var got []string
	spec := path
	for page := 0; ; page++ {
		if page > 20 {
			t.Fatalf("still paging after %d pages — the continuation is not advancing", page)
		}
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{spec}})
		text, _ := res["content"].([]any)
		if len(text) == 0 {
			t.Fatalf("page %d carried no content", page)
		}
		body, _ := text[0].(map[string]any)["text"].(string)
		got = append(got, numberedLines(t, body)...)
		next := nextOf(t, res)
		if next == "" {
			break
		}
		if next == spec {
			t.Fatalf("page %d hands back the spec it was given (%q) — that never terminates", page, next)
		}
		spec = next
	}
	if len(got) != lines {
		t.Fatalf("reassembled %d lines, want %d — paging lost or repeated content", len(got), lines)
	}
	for i, l := range got {
		if l != "// padding padding padding padding padding padding padding" {
			t.Fatalf("line %d of the reassembly is wrong: %q", i+1, l)
		}
	}
}

// numberedLines strips mrw's rendered "   12| " prefixes back to the file's own
// lines, so a reassembly can be compared with what is on disk.
func numberedLines(t *testing.T, rendered string) []string {
	t.Helper()
	var out []string
	for _, l := range strings.Split(rendered, "\n") {
		i := strings.Index(l, "|")
		if i < 0 {
			continue // a header or a blank separator, not a served line
		}
		if _, err := strconv.Atoi(strings.TrimSpace(l[:i])); err != nil {
			continue
		}
		out = append(out, strings.TrimPrefix(l[i+1:], " "))
	}
	return out
}

// TestAPageLicensesOnlyWhatItServed is the ledger property, and the one that
// would turn paging into a read-before-write bypass if it were wrong.
//
// A page shows the caller part of a file. It must license editing THAT part and
// refuse the rest — licensing everything would be the bypass; licensing nothing
// would make paging useless.
func TestAPageLicensesOnlyWhatItServed(t *testing.T) {
	const lines = 12000
	root, path := bigCheckout(t, lines)

	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}})
	if nextOf(t, res) == "" {
		t.Fatal("a file this size must page; the fixture is not exercising the path")
	}

	// A write inside page one is licensed.
	ok := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": "@@ " + path + " 1 replace\n// page one\n", "dry_run": true}))
	if n, _ := ok["failed"].(float64); n != 0 {
		t.Errorf("a write to a line page one served was refused: %v", ok["hunks"])
	}

	// A write far past it is not.
	no := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": fmt.Sprintf("@@ %s %d replace\n// last page\n", path, lines), "dry_run": true}))
	if n, _ := no["failed"].(float64); n != 1 {
		t.Fatalf("failed = %v, want 1 — a page must not license lines it never served", no["failed"])
	}
}

// TestAnOversizedReadStillReadsAsIncomplete keeps the property ADR-011 bought.
//
// Paging differs from truncation only because the caller can see it received a
// part and must ask for the rest. If the result stopped saying so, this would
// be truncation with extra steps.
func TestAnOversizedReadStillReadsAsIncomplete(t *testing.T) {
	root, path := bigCheckout(t, 12000)
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}})

	if e, _ := res["isError"].(bool); !e {
		t.Error("a paged read is not marked isError, so a caller that stops here believes it has the file")
	}
	sc, _ := res["structuredContent"].(map[string]any)
	if _, ok := sc["next_read"]; !ok {
		t.Error("structuredContent does not name the continuation")
	}
	blocks, _ := res["content"].([]any)
	var all string
	for _, b := range blocks {
		s, _ := b.(map[string]any)["text"].(string)
		all += s
	}
	if !strings.Contains(all, "next_read") && !strings.Contains(all, "continue") {
		t.Error("neither content block tells a human reader that more remains")
	}
}

// grepTree plants n files under the root, each carrying matchLines lines that
// contain the needle, plus one filler line so a file is never made ENTIRELY of
// matches by construction.
//
// ⚠ matchLines, not padding. A grep serves only the lines that MATCH, so a file
// of 900 filler lines and one needle costs 19 bytes to serve, not 45,000. The
// first version of this fixture padded and could not overflow the cap at any
// size — the test failed for a reason that had nothing to do with the code, and
// that is worth a comment because the same instinct will return.
func grepTree(t *testing.T, n, matchLines int) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for i := 0; i < n; i++ {
		var b strings.Builder
		b.WriteString("a line that matches nothing\n")
		for j := 0; j < matchLines; j++ {
			b.WriteString("the NEEDLE is here\n")
		}
		name := filepath.Join(root, fmt.Sprintf("document%05d.csv", i))
		if err := os.WriteFile(name, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestGrepServesWhatItFindsAndRecordsIt(t *testing.T) {
	root := grepTree(t, 3, 2)

	res := call(t, root, "mrw_read", map[string]any{"grep": "NEEDLE"})
	if res["isError"] == true {
		t.Fatalf("a grep that fits should not be an error: %v", res)
	}
	all := fmt.Sprint(res["content"])
	for i := 0; i < 3; i++ {
		if want := fmt.Sprintf("document%05d.csv", i); !strings.Contains(all, want) {
			t.Errorf("the grep did not serve %s:\n%s", want, all)
		}
	}
	if !strings.Contains(all, "the NEEDLE is here") {
		t.Errorf("the grep found files but served no matching content:\n%s", all)
	}

	// What was SERVED is what is licensed. The ledger must know these files,
	// or a grep would be a read that teaches mrw nothing.
	obs := structured(t, res)["observed"]
	if obs == nil || len(obs.(map[string]any)) != 3 {
		t.Errorf("a served grep recorded %v, want 3 files observed", obs)
	}
}

func TestGrepRefusesARangedSpec(t *testing.T) {
	root := grepTree(t, 1, 1)

	res := call(t, root, "mrw_read", map[string]any{
		"specs": []any{"document00000.csv:1-2"},
		"grep":  "NEEDLE",
	})
	if res["isError"] != true {
		t.Fatalf("a range plus a grep should be refused, got %v", res)
	}
	// The CLI's own sentence, so a caller hitting this on either surface reads
	// the same explanation (cmd/mrw/main.go:499).
	if all := fmt.Sprint(res["content"]); !strings.Contains(all, "two answers to one question") {
		t.Errorf("the refusal does not give the CLI's reasoning:\n%s", all)
	}
}

// TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd is ADR-017's Enforced-by.
//
// It drives a REAL overflow rather than constructing a result, and it asserts
// the index is USABLE — every entry parses as a spec, and sending one back
// actually reads that file. An index of the wrong files, or of strings that are
// not specs, would pass any check that only asked whether an index key exists.
func TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd(t *testing.T) {
	// Enough content that serving every match blows the cap, with few enough
	// files that the INDEX itself still fits — that is the case under test.
	root := grepTree(t, 60, 400)

	res := call(t, root, "mrw_read", map[string]any{"grep": "NEEDLE"})
	if res["isError"] != true {
		t.Fatal("an oversized grep must still read as an error — a partial answer that looks whole is what this project refuses")
	}
	st := structured(t, res)
	if got := st["matches"]; got == nil || int(got.(float64)) != 60 {
		t.Errorf("the index reports %v matches, want 60", got)
	}
	idx, ok := st["index"].([]any)
	if !ok || len(idx) == 0 {
		t.Fatalf("an oversized grep returned no index: %v", st)
	}

	// NO CONTENT, THEREFORE NO LICENCE — asserted against the LEDGER, not
	// against the response.
	//
	// ⚠ The first version compared the length of the printed `observed` field
	// and would have passed with the ledger written: adding seen.Record before
	// matchIndex, or dropping it from a fitting grep, changed nothing it
	// looked at. The Risks table rates this Critical, so the check has to be
	// the thing the risk is about. A write to a matched file must be REFUSED,
	// which is the only statement that means "no licence". Found by review
	// of #80.
	if obs, ok := st["observed"].(map[string]any); !ok || len(obs) != 0 {
		t.Errorf("the index reported observed=%v, want an empty map", st["observed"])
	}
	first := fmt.Sprint(idx[0])
	w := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": "@@ " + first + " 2 replace\nCHANGED\n",
	}))
	if applied, _ := w["applied"].(bool); applied {
		t.Fatalf("a write to %q was APPLIED after only an index — the index licensed a file the caller never saw", first)
	}
	if failed, _ := w["failed"].(float64); failed == 0 {
		t.Errorf("the write after an index was not refused: %v", w)
	}

	// THE ENTRIES ARE USABLE. An entry is a PATH, so sending it back with the
	// same grep must read that file's matches — the round trip the caller is
	// told to make.
	back := call(t, root, "mrw_read", map[string]any{"specs": []any{first}, "grep": "NEEDLE"})
	if back["isError"] == true {
		t.Fatalf("index entry %q is not a spec this tool accepts: %v", first, back)
	}
	if all := fmt.Sprint(back["content"]); !strings.Contains(all, "NEEDLE") {
		t.Errorf("reading index entry %q served no match:\n%s", first, all)
	}
}

func TestAnIndexTooLargeToServePagesByFile(t *testing.T) {
	// Each entry is roughly 30 bytes, so this needs many files rather than
	// large ones — which is exactly the Desktop folder this record is for.
	const files = 8000
	root := grepTree(t, files, 1)

	res := call(t, root, "mrw_read", map[string]any{"grep": "NEEDLE"})
	st := structured(t, res)
	idx, ok := st["index"].([]any)
	if !ok {
		t.Fatalf("no index came back: %v", st)
	}
	// ⚠ AGAINST THE FIXTURE'S OWN COUNT. This was written as a literal 9000
	// while the fixture was resized to 8000, which made it an assertion that
	// could never fire — found because a mutant that disabled paging entirely
	// sailed past it and failed the NEXT check instead. A threshold that does
	// not track the fixture is a test with a hole in the middle.
	if len(idx) >= files {
		t.Fatalf("the index served all %d entries; this fixture exists to overflow it", len(idx))
	}
	next, _ := st["next_index"].(string)
	if next == "" {
		t.Fatal("the index was cut short and did not say where to resume — that is ADR-014's dead end, one level down")
	}
	// The continuation names a real file, not an opaque token.
	if _, err := os.Stat(filepath.Join(root, next)); err != nil {
		t.Errorf("next_index %q is not a path under the root: %v", next, err)
	}

	// ⚠ FOLLOW IT TO EXHAUSTION, and compare the union to the whole match set.
	//
	// The first version of this test stopped at the two checks above: that a
	// continuation is PRESENT and names a real path. It passed while nothing
	// in the tool accepted that value — there was no `after` argument, so the
	// index advertised a continuation no caller could follow. A presence check
	// cannot tell a working continuation from a decorative one, which is the
	// same lesson ADR-014's Enforced-by already carried and this test did not
	// inherit. Found by review of #80.
	seenPaths := map[string]bool{}
	countIndex := func(st map[string]any) string {
		// A resumed page comes back in ONE OF TWO SHAPES, and both are right:
		// another index while the remainder is still too large, or an ordinary
		// SERVED read once it fits — the same way ADR-014's last page is a
		// normal read carrying no continuation. A test that demanded an index
		// every time would be asserting a worse tool than the one that exists.
		if idx, ok := st["index"].([]any); ok {
			for _, e := range idx {
				s := fmt.Sprint(e)
				if i := strings.LastIndex(s, ":"); i > 0 {
					s = s[:i]
				}
				if seenPaths[s] {
					t.Errorf("entry %q appears on two pages — a cursor that overlaps loses the caller's place", s)
				}
				seenPaths[s] = true
			}
			next, _ := st["next_index"].(string)
			return next
		}
		for p := range st["observed"].(map[string]any) {
			if seenPaths[p] {
				t.Errorf("file %q appears on two pages", p)
			}
			seenPaths[p] = true
		}
		return ""
	}

	next = countIndex(st)
	for pages := 1; next != ""; pages++ {
		if pages > 20 {
			t.Fatalf("following next_index did not terminate after %d pages", pages)
		}
		res = call(t, root, "mrw_read", map[string]any{"grep": "NEEDLE", "after": next})
		next = countIndex(structured(t, res))
	}
	if len(seenPaths) != files {
		t.Errorf("paging to exhaustion yielded %d distinct files, want %d — entries were lost between pages", len(seenPaths), files)
	}
}

// TestTheIndexSurvivesAPatternThatLooksLikeARange pins the reason index entries
// are bare PATHS.
//
// The first cut emitted `path:/<pattern>/`, which reads well and is wrong for a
// whole class of patterns: `alpha/,/beta` is a valid regexp, and the entry
// `f.txt:/alpha/,/beta/` parses back as a pattern RANGE — start /alpha/, end
// /beta/ — rather than the single pattern that matched. The entry still LOOKS
// like a spec and still reads successfully; it just reads different lines than
// the ones it claims to index. That is the silent wrong answer this tool exists
// to refuse, and a simple needle could never expose it. Found by review of #80.
func TestTheIndexSurvivesAPatternThatLooksLikeARange(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	body := "alpha/,/beta here\n" + strings.Repeat("noise\n", 5) + "tail\n"
	for i := 0; i < 40; i++ {
		name := filepath.Join(root, fmt.Sprintf("doc%03d.txt", i))
		if err := os.WriteFile(name, []byte(strings.Repeat(body, 400)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	res := call(t, root, "mrw_read", map[string]any{"grep": `alpha/,/beta`})
	st := structured(t, res)
	idx, ok := st["index"].([]any)
	if !ok || len(idx) == 0 {
		t.Fatalf("expected an index for an oversized grep: %v", st)
	}
	for _, e := range idx {
		s := fmt.Sprint(e)
		if strings.Contains(s, ":") {
			t.Fatalf("index entry %q carries an address; a pattern with a slash makes that ambiguous, so entries must be bare paths", s)
		}
	}
	// And the round trip serves the SAME matches the grep found.
	back := call(t, root, "mrw_read", map[string]any{"specs": []any{fmt.Sprint(idx[0])}, "grep": `alpha/,/beta`})
	if back["isError"] == true {
		t.Fatalf("the round trip failed: %v", back)
	}
	all := fmt.Sprint(back["content"])
	if !strings.Contains(all, "alpha/,/beta here") {
		t.Errorf("the round trip did not serve the matching line:\n%s", all[:min(400, len(all))])
	}
	if strings.Contains(all, "noise") {
		t.Errorf("the round trip served lines BETWEEN two patterns — the range reading, not the single-pattern one:\n%s", all[:min(400, len(all))])
	}
}

// TestAWalkProblemIsReportedAndNotSwallowed keeps a failed lookup from reading
// as an empty result.
//
// The CLI prints every path the walk could not use and counts it (main.go:553).
// The first cut of the MCP path discarded read.Walk's []Problem entirely, so a
// caller naming a directory that does not exist was told "no file under the
// root matches" — a clean answer to a question nobody asked. Found by review
// of #80.
func TestAWalkProblemIsReportedAndNotSwallowed(t *testing.T) {
	root := grepTree(t, 2, 1)

	res := call(t, root, "mrw_read", map[string]any{
		"specs": []any{"no-such-directory"},
		"grep":  "NEEDLE",
	})
	all := fmt.Sprint(res["content"])
	if !strings.Contains(all, "no-such-directory") {
		t.Errorf("a path the walk could not use vanished from the answer:\n%s", all)
	}
	if res["isError"] != true {
		t.Errorf("a walk that could not look where it was told reported success: %v", res)
	}
}
