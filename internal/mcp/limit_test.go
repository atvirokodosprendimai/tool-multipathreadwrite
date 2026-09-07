package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
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

// rawResponse drives one tools/call and returns the WHOLE response line, so a
// test can see a JSON-RPC error as well as a result. rawResult refuses an
// error; the paths below are about answers that are deliberately not results.
func rawResponse(t *testing.T, root, tool string, args map[string]any) map[string]any {
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
	var resp map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSuffix(out.String(), "\n")), &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	return resp
}

// TestASmallCeilingRefusesTheWriteBeforeApplying is the regression for the
// defect the Codex review of #135 found and this branch reproduced on the built
// binary: with `--max-result-chars 0` a licensed one-hunk write CHANGED the
// file, recorded its ledger entry, and answered "0 of 1 hunk(s) failed and
// nothing was written".
//
// ⚠ THE ASSERTION IS THE FILE, NOT THE MESSAGE. A fix that only corrected the
// wording would leave a write applying under a ceiling that cannot report it,
// and this test would pass. What must hold is that the tree is untouched.
func TestASmallCeilingRefusesTheWriteBeforeApplying(t *testing.T) {
	for _, budget := range []int{0, 1, 64} {
		t.Run(fmt.Sprintf("budget-%d", budget), func(t *testing.T) {
			root, name := checkout(t, "f.txt", "alpha\nbravo\n")
			full := filepath.Join(root, name)

			// Serve line 1 at the default ceiling, so the write below is
			// refused for the BUDGET and not for the ledger.
			restore := MaxResultChars
			MaxResultChars = restore
			rawResult(t, root, "mrw_read", map[string]any{"specs": []string{name + ":1"}})

			MaxResultChars = budget
			t.Cleanup(func() { MaxResultChars = restore })

			resp := rawResponse(t, root, "mrw_write", map[string]any{
				"plan": "@@ " + name + " 1 replace\nMUTATED\n"})

			before, err := os.ReadFile(full)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != "alpha\nbravo\n" {
				t.Errorf("the file was written under a ceiling that cannot report a write: %q", before)
			}
			if _, ok := resp["error"]; !ok {
				t.Errorf("a write that cannot be reported was not refused; got result %v", resp["result"])
			}
			if _, ok := resp["result"]; ok {
				t.Error("the refusal carries a tool result, which is itself subject to the ceiling it could not meet")
			}
		})
	}
}

// TestEveryAnswerFitsIncludingTheRefusals covers what ADR-032's first cut did
// not: `errorResult` carried no size check, so at a small ceiling the REFUSALS
// exceeded the number `_meta` advertises — the promise the record is named for.
//
// ⚠ ITS FIRST VERSION PASSED FOR THE WRONG REASON and the mutation gate caught
// it. Five short refusals were all under 900 bytes on their own, so removing
// the funnel changed nothing and the mutant SURVIVED. A refusal is only a test
// of a size bound if the refusal is genuinely too large — so two of these are
// measured oversized on this fixture (35,765 and 6,109 bytes at an unbounded
// ceiling), and the assertion is the funnel's OWN sentence, which cannot appear
// unless the funnel ran.
func TestEveryAnswerFitsIncludingTheRefusals(t *testing.T) {
	const budget = 900
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })

	root, specs := manyFiles(t, 40)

	// A grep whose paths do not exist reports one walk problem per path, so the
	// refusal grows with what the caller asked for. Codex named this path.
	missing := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		missing = append(missing, fmt.Sprintf("no-such-dir-%03d/nope.txt", i))
	}
	// A bad spec is quoted back, so a long path makes a long refusal.
	longSpec := strings.Repeat("x", 3000) + ":not-a-range"

	MaxResultChars = budget

	for _, c := range []struct {
		name     string
		tool     string
		oversize bool // measured larger than the budget with no ceiling in force
		args     map[string]any
	}{
		{"a grep whose paths all fail to walk", "mrw_read", true, map[string]any{"specs": missing, "grep": "zzz"}},
		{"an unparseable spec quoted back at length", "mrw_read", true, map[string]any{"specs": []string{longSpec}}},
		{"a read too large to serve", "mrw_read", false, map[string]any{"specs": specs}},
		{"exclude without grep", "mrw_read", false, map[string]any{"specs": specs[:1], "exclude": []string{"x"}}},
		{"a plan that does not parse", "mrw_write", false, map[string]any{"plan": "@@ nonsense\n"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			resp := rawResponse(t, root, c.tool, c.args)
			raw, ok := resp["result"]
			if !ok {
				return // a JSON-RPC error carries no result member; no ceiling applies
			}
			b, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(b) > budget {
				t.Errorf("%s returned %d bytes against an advertised %d", c.name, len(b), budget)
			}
			if !c.oversize {
				return
			}
			// This case is larger than the budget on its own, so the funnel is
			// the only thing that can have brought it under. Assert the funnel's
			// own words: without it the answer arrives whole and this fails.
			if !strings.Contains(string(b), "the ceiling in force is") {
				t.Errorf("%s fits, but not because the ceiling funnel trimmed it — its own refusal text is absent: %s",
					c.name, b)
			}
		})
	}
}

// TestTheCeilingNeverShrinksAServedRead pins the ordering withinCeiling's
// correctness rests on: a served read is measured and refused BEFORE its ledger
// record is written, so the final size check never has to rewrite an answer
// that licensed something.
//
// If that order were reversed the check would discard lines the ledger had just
// recorded as seen — ADR-002 inverted by a size guard, which is the shape
// ADR-031 exists about. The test asserts the licence, not the message.
func TestTheCeilingNeverShrinksAServedRead(t *testing.T) {
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })

	root, specs := manyFiles(t, 200)
	MaxResultChars = 20_000 // the report fits; the receipt carries it over

	res := rawResult(t, root, "mrw_read", map[string]any{"specs": specs})
	if len(res) > MaxResultChars {
		t.Fatalf("the read answered with %d bytes against %d", len(res), MaxResultChars)
	}

	// Nothing was served, so nothing may be written. A ledger entry here would
	// mean the size check had thrown away content it had already licensed.
	MaxResultChars = restore
	out := rawResult(t, root, "mrw_write", map[string]any{
		"plan": "@@ " + specs[0] + " 1 replace\nMUTATED\n", "dry_run": true})
	if !strings.Contains(string(out), "has not been read") {
		t.Errorf("a read that was refused for size still licensed a write: %s", out)
	}
}

// TestTheSecondStageElisionDropsFileRecords reaches the branch
// TestAWriteReceiptElidesSuccessesNotFailures could not: with one file there is
// nothing for stage two to drop, so the `Files = nil` path was never executed.
func TestTheSecondStageElisionDropsFileRecords(t *testing.T) {
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })

	root, specs := manyFiles(t, 400)
	rawResult(t, root, "mrw_read", map[string]any{"specs": specs})

	var plan strings.Builder
	for i, s := range specs {
		if i < 2 {
			fmt.Fprintf(&plan, "@@ %s 1 replace anchor=\"no-such-anchor\"\nY\n", s)
			continue
		}
		fmt.Fprintf(&plan, "@@ %s 1 replace\nY\n", s)
	}

	const budget = 3_000
	MaxResultChars = budget
	res := rawResult(t, root, "mrw_write", map[string]any{"plan": plan.String()})
	if len(res) > budget {
		t.Fatalf("the receipt returned %d bytes against an advertised %d", len(res), budget)
	}

	var got struct {
		Structured struct {
			Failed int    `json:"failed"`
			Elided string `json:"elided"`
			Files  []any  `json:"files"`
			Hunks  []struct {
				Status string `json:"status"`
			} `json:"hunks"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}
	if !strings.Contains(got.Structured.Elided, "file record(s)") {
		t.Errorf("stage two was not reached, or did not say it dropped the file records: %q", got.Structured.Elided)
	}
	// ⚠ THIS FIXTURE WRITES NOTHING. Two hunks fail, so under ADR-001 the plan
	// applies nothing and every FileResult has Written false — which is why
	// stage two can drop all 400 records here. It pins the DROP; what stage two
	// KEEPS is TestTheSecondStageNeverElidesAWrittenFile's claim, and this test
	// cannot see it.
	if len(got.Structured.Files) != 0 {
		t.Errorf("stage two kept %d file record(s); with nothing written there is none to keep", len(got.Structured.Files))
	}
	if got.Structured.Failed != 2 || len(got.Structured.Hunks) != 2 {
		t.Errorf("failed=%d with %d hunk(s) kept, want 2 and 2 — a failure is never elided",
			got.Structured.Failed, len(got.Structured.Hunks))
	}
	for _, h := range got.Structured.Hunks {
		if h.Status != "failed" {
			t.Errorf("a %q hunk survived stage two; only failures may", h.Status)
		}
	}
}

// TestTheWriteFloorIsAFloor binds `writeFloor` to the message it is supposed to
// bound. The first version substituted 999999 for each count and called that
// "the widest plausible width"; nothing bounds a plan to a million hunks, so a
// wide enough real count would slip past the guard and then be replaced by the
// funnel's generic refusal, which does not say the write happened.
//
// ⚠ THE FLOOR IS SELF-REFERENTIAL: the message it measures quotes the ceiling
// in force, so a narrower ceiling renders a shorter message and a smaller
// floor. That is consistent — the guard and the real message read the same
// value at the same moment — but it means the boundary cannot be computed at
// one ceiling and tested at another. The contract asserted here is the one that
// actually holds: at any ceiling B, the write is refused exactly when the floor
// computed AT B exceeds B.
func TestTheWriteFloorIsAFloor(t *testing.T) {
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })

	for _, budget := range []int{0, 1, 64, 300, 440, 445, 450, 500, 5_000, restore} {
		t.Run(fmt.Sprintf("budget-%d", budget), func(t *testing.T) {
			root, name := checkout(t, "f.txt", "alpha\nbravo\n")
			MaxResultChars = restore
			rawResult(t, root, "mrw_read", map[string]any{"specs": []string{name + ":1"}})

			MaxResultChars = budget
			floor := writeFloor()

			// The floor must bound the real message for ANY counts at this
			// ceiling, not for the ones a probe happened to pick.
			for _, c := range [][3]int{{0, 0, 0}, {1, 1, 0}, {999999, 999999, 999999}, {math.MaxInt, math.MaxInt, math.MaxInt}} {
				for _, partial := range []bool{false, true} {
					if n := encodedSize(errorResult(appliedButUnreportable(c[0], c[1], c[2], partial))); n > floor {
						t.Errorf("a real message at counts %v (partial=%v) is %d bytes, over the %d-byte floor",
							c, partial, n, floor)
					}
				}
			}

			resp := rawResponse(t, root, "mrw_write", map[string]any{
				"plan": "@@ " + name + " 1 replace\nMUTATED\n"})
			_, refused := resp["error"]
			if want := floor > budget; refused != want {
				t.Errorf("at ceiling %d with floor %d the write refused=%v, want %v",
					budget, floor, refused, want)
			}
		})
	}
}

// TestAPartialApplicationIsNotReportedAsNothingWritten is the regression for the
// second HIGH the Codex review of #135 found: the terminal branch asked
// `res.Applied`, which is FALSE for a partial application.
//
// apply.Apply renames file by file; a rename that fails after earlier ones
// succeeded returns Applied=false with those files already on disk, and the
// engine's own `writtenSoFar` exists to name them. Asking Applied would deny a
// write that happened — the same denial as the first HIGH, one branch over.
//
// ⚠ IT DRIVES boundedReceipt DIRECTLY. Staging a real mid-rename failure needs a
// filesystem the test can break at exactly the right moment, which is a race on
// Unix and a different mechanism on Windows; the defect is in the reporting, so
// the reporting is what is tested.
func TestAPartialApplicationIsNotReportedAsNothingWritten(t *testing.T) {
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })
	MaxResultChars = 600 // large enough for the terminal sentence, far too small for the receipt

	res := apply.Result{
		Root:    "/tmp/whatever",
		Applied: false, // a later rename failed
		Files: []apply.FileResult{
			{Path: "a.go", Written: true},  // already renamed into place
			{Path: "b.go", Written: false}, // the one that failed
		},
	}
	for i := 0; i < 200; i++ {
		res.Hunks = append(res.Hunks, apply.HunkResult{
			Path: "a.go", Addr: fmt.Sprint(i), Op: "replace", Status: apply.StatusOK,
			Reason: strings.Repeat("padding so the full receipt cannot fit ", 4),
		})
	}

	out, rpcErr := boundedReceipt(res, fmt.Errorf("b.go: rename failed"), true)
	if rpcErr != nil {
		t.Fatalf("boundedReceipt: %v", rpcErr)
	}
	if n := encodedSize(out); n > MaxResultChars {
		t.Fatalf("the terminal answer is %d bytes against a %d ceiling", n, MaxResultChars)
	}
	got := out.Content[0].Text
	if strings.Contains(got, "nothing was written") {
		t.Errorf("a partial application was reported as nothing written: %s", got)
	}
	if !strings.Contains(got, "PARTIALLY APPLIED") {
		t.Errorf("the answer does not say the tree changed partially: %s", got)
	}
	if !strings.Contains(got, "1 file(s) changed on disk") {
		t.Errorf("the answer does not count the file that WAS written: %s", got)
	}
}

// TestTheSecondStageNeverElidesAWrittenFile pins the half of stage-two elision
// that a partial application depends on.
//
// ⚠ THE FIXTURE MUST REACH STAGE TWO AND FIT THERE. A budget that also defeats
// stage two falls through to the terminal branch, which
// TestAPartialApplicationIsNotReportedAsNothingWritten already covers and which
// would make this test green for the wrong reason — so it asserts the answer
// came from the loop by requiring the elision note, which only the loop writes.
// Many FILE records and no failed hunk is what keeps stage one over the budget:
// with Failed=0 every hunk is dropped at stage one, so the files are the only
// thing left to make it too large.
func TestTheSecondStageNeverElidesAWrittenFile(t *testing.T) {
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })

	res := apply.Result{Root: "/tmp/whatever", Applied: false} // a later rename failed
	for i := 0; i < 300; i++ {
		res.Files = append(res.Files, apply.FileResult{
			Path:      fmt.Sprintf("pkg/dir%03d/file%03d.go", i, i),
			Written:   i < 3, // the three already renamed into place before the failure
			SHABefore: strings.Repeat("a", 64),
			SHAAfter:  strings.Repeat("b", 64),
		})
		res.Hunks = append(res.Hunks, apply.HunkResult{
			Path: res.Files[i].Path, Addr: "1", Op: "replace", Status: apply.StatusOK,
		})
	}

	const budget = 4_000
	MaxResultChars = budget
	out, rpcErr := boundedReceipt(res, fmt.Errorf("pkg/dir003/file003.go: rename failed"), true)
	if rpcErr != nil {
		t.Fatalf("boundedReceipt: %v", rpcErr)
	}
	if n := encodedSize(out); n > budget {
		t.Fatalf("the receipt returned %d bytes against a %d ceiling", n, budget)
	}

	var got struct {
		Elided string             `json:"elided"`
		Failed int                `json:"failed"`
		Files  []apply.FileResult `json:"files"`
	}
	if out.StructuredContent == nil {
		t.Fatalf("no structured content: the terminal branch answered, not stage two:\n%s", out.Content[0].Text)
	}
	b, err := json.Marshal(out.StructuredContent)
	if err != nil {
		t.Fatalf("structured content is not JSON: %v", err)
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("structured content does not decode: %v", err)
	}
	if !strings.Contains(got.Elided, "NOT written") {
		t.Fatalf("this answer did not come from stage two, so it proves nothing about elision: %q", got.Elided)
	}

	// The claim. With Failed=0 and Applied=false, the file records are the ONLY
	// thing in the receipt that says the tree changed: `applied` says the
	// opposite and `failed` says nothing happened. A host that delivers only
	// the structured value (ADR-023) has nothing else to read.
	if len(got.Files) != 3 {
		t.Errorf("stage two kept %d file record(s), want the 3 that were WRITTEN — dropping them "+
			"reports a partial application as applied:false, failed:0, files:[]", len(got.Files))
	}
	for _, f := range got.Files {
		if !f.Written {
			t.Errorf("stage two kept %q, which was not written; only the written records are evidence", f.Path)
		}
	}
	if got.Failed != 0 {
		t.Errorf("failed=%d, want 0 — the fixture exists because failed=0 is what makes the denial silent", got.Failed)
	}
}
