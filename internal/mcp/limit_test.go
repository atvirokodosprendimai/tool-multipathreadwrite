package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ceilingFixture builds a checkout whose read AND whose write receipt are each
// large enough to cross an ordinary budget: n numbered lines, and a plan that
// replaces every one of them with itself.
//
// The plan is a no-op by content on purpose. What is being measured is the size
// of the RECEIPT — one verdict per hunk — and a plan that changes nothing still
// earns every one of them.
func ceilingFixture(t *testing.T, n int) (root, name, planText string) {
	t.Helper()
	var body, hunks strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&body, "line %05d of the fixture\n", i)
		fmt.Fprintf(&hunks, "@@ f.txt %d replace\nline %05d of the fixture\n", i, i)
	}
	root, name = checkout(t, "f.txt", body.String())
	return root, name, hunks.String()
}

// rawResult drives one tools/call through Serve and returns the EXACT bytes of
// the `result` member the server wrote.
//
// ⚠ It is deliberately not a re-marshal of a decoded map. The cap is a claim
// about what crosses the wire, and ADR-031 shipped a size check on a buffer
// that was not what got sent — twice, in consecutive reviews. A reconstruction
// is how that happens, so the oracle here is the transport's own output.
func rawResult(t *testing.T, root, tool string, args map[string]any) []byte {
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
	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	line := strings.TrimSuffix(out.String(), "\n")
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, line)
	}
	if len(resp.Error) > 0 {
		t.Fatalf("tools/call returned a JSON-RPC error: %s", resp.Error)
	}
	if len(resp.Result) == 0 {
		t.Fatalf("no result member: %s", line)
	}
	return resp.Result
}

// advertisedCeiling reads `anthropic/maxResultSizeChars` per tool from
// tools/list, over the same pipe a host reads it from. Asking tools() directly
// would test the value this package holds rather than the one it publishes.
func advertisedCeiling(t *testing.T, root string) map[string]float64 {
	t.Helper()
	req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	var out bytes.Buffer
	if err := Serve(strings.NewReader(req+"\n"), &out, root); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var resp struct {
		Result struct {
			Tools []struct {
				Name string         `json:"name"`
				Meta map[string]any `json:"_meta"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSuffix(out.String(), "\n")), &resp); err != nil {
		t.Fatalf("tools/list is not JSON: %v", err)
	}
	got := map[string]float64{}
	for _, tl := range resp.Result.Tools {
		n, ok := tl.Meta["anthropic/maxResultSizeChars"].(float64)
		if !ok {
			t.Fatalf("%s advertises no anthropic/maxResultSizeChars: %v", tl.Name, tl.Meta)
		}
		got[tl.Name] = n
	}
	return got
}

// TestTheAdvertisedCeilingBoundsEveryAnswer is ADR-032's enforcement: what a
// tool advertises in _meta is what the server will not exceed, on BOTH tools,
// at the default budget and at one the caller set.
//
// Measured before the fix, at 267c453: a 4,000-hunk dry-run came back at
// 453,632 characters against an advertised 200,000. The read half is here
// because a budget one tool ignores is not a budget.
func TestTheAdvertisedCeilingBoundsEveryAnswer(t *testing.T) {
	for _, budget := range []int{DefaultMaxResultChars, 20_000} {
		t.Run(fmt.Sprintf("budget-%d", budget), func(t *testing.T) {
			restore := MaxResultChars
			MaxResultChars = budget
			t.Cleanup(func() { MaxResultChars = restore })

			root, name, planText := ceilingFixture(t, 4000)

			for tool, adv := range advertisedCeiling(t, root) {
				if int(adv) != budget {
					t.Errorf("%s advertises %d while the server enforces %d; the advertised limit and the enforced one cannot be allowed to drift",
						tool, int(adv), budget)
				}
			}

			// The read comes first because it is what licenses the write. A
			// write whose lines were never served is refused hunk by hunk, and
			// a receipt of 4,000 REFUSALS is one no elision may shorten — so
			// reading first is what makes the write half measure the receipt
			// rather than the ledger.
			read := rawResult(t, root, "mrw_read", map[string]any{"specs": []string{name}})
			if len(read) > budget {
				t.Errorf("mrw_read returned %d bytes against an advertised %d", len(read), budget)
			}

			write := rawResult(t, root, "mrw_write", map[string]any{"plan": planText, "dry_run": true})
			if len(write) > budget {
				t.Errorf("mrw_write returned %d bytes against an advertised %d", len(write), budget)
			}
		})
	}
}

// manyFiles builds a checkout of n one-line files and the specs that name them.
//
// It exists for the band `capped` cannot see: the served TEXT fits the budget
// while the whole ANSWER does not, because the ledger observation for each file
// — a sha and a span list, the same size whatever the range — is the larger
// half for many small files.
func manyFiles(t *testing.T, n int) (root string, specs []string) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("f%04d.txt", i)
		if err := os.WriteFile(filepath.Join(root, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		specs = append(specs, name)
	}
	return root, specs
}

// TestAReadWhoseReceiptOverflowsIsNotServed covers ADR-032's read half: the
// judgement moved off the report text and onto the encoded answer, and it is no
// longer reached only by a `grep`.
//
// Before the change a caller naming its own specs had its report bounded and
// its receipt not, so an answer past the advertised ceiling went out and the
// host truncated it — which ADR-031 exists because mrw cannot see.
func TestAReadWhoseReceiptOverflowsIsNotServed(t *testing.T) {
	const budget = 20_000
	restore := MaxResultChars
	MaxResultChars = budget
	t.Cleanup(func() { MaxResultChars = restore })

	// Sized into the band: enough files that the receipt carries the answer
	// past the budget, few enough that the served text alone stays inside it.
	root, specs := manyFiles(t, 200)

	res := rawResult(t, root, "mrw_read", map[string]any{"specs": specs})
	if len(res) > budget {
		t.Fatalf("mrw_read returned %d bytes against an advertised %d", len(res), budget)
	}
	if !strings.Contains(string(res), "receipt") {
		t.Errorf("the refusal does not say the receipt is what overflowed, so the caller is told to narrow ranges that are already one line: %s", res)
	}
}

// TestAWriteReceiptElidesSuccessesNotFailures is ADR-032 T2: an oversized
// receipt drops the verdicts a caller already knows and keeps the ones they
// must act on.
func TestAWriteReceiptElidesSuccessesNotFailures(t *testing.T) {
	const n = 4000
	root, name, planText := ceilingFixture(t, n)

	// Serve every line, so the plan is refused for the anchors below and not
	// for the ledger.
	rawResult(t, root, "mrw_read", map[string]any{"specs": []string{name}})

	// Three hunks that cannot apply, among 3,997 that can. Under ADR-001 the
	// rest are `skipped` and nothing is written, which is exactly the shape a
	// caller needs the failures out of.
	bad := ""
	for _, line := range []int{7, 500, 3999} {
		bad += fmt.Sprintf("@@ f.txt %d replace anchor=\"no-such-anchor\"\nline %05d of the fixture\n", line, line)
	}
	res := rawResult(t, root, "mrw_write", map[string]any{"plan": planText + bad})

	if len(res) > MaxResultChars {
		t.Fatalf("mrw_write returned %d bytes against an advertised %d", len(res), MaxResultChars)
	}
	var got struct {
		Structured struct {
			Failed int    `json:"failed"`
			Elided string `json:"elided"`
			Hunks  []struct {
				Status string `json:"status"`
				Addr   string `json:"addr"`
			} `json:"hunks"`
		} `json:"structuredContent"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}
	if got.Structured.Failed != 3 {
		t.Errorf("failed count is %d, want 3 — the counts describe the whole plan whatever was elided", got.Structured.Failed)
	}
	if got.Structured.Elided == "" {
		t.Error("the receipt was shortened and says nothing about it; a silently shorter answer is the defect this tool exists to refuse")
	}
	if n := len(got.Structured.Hunks); n != 3 {
		t.Fatalf("the receipt carries %d hunk verdict(s), want the 3 failures and nothing else", n)
	}
	for _, h := range got.Structured.Hunks {
		if h.Status != "failed" {
			t.Errorf("hunk %s survived the elision with status %q; only failures may", h.Addr, h.Status)
		}
	}
	if len(got.Content) == 0 || !strings.Contains(got.Content[0].Text, fmt.Sprintf("%d hunk(s)", n+3)) {
		t.Errorf("the report does not carry the whole plan's counts: %q", got.Content[0].Text)
	}
}
