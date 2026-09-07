package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// gate serializes tool calls, and it is worth being exact about what it does
// and does not buy. `Serve` reads one line, handles it fully, and only then
// reads the next — so a SINGLE server never has two tool calls in flight, and
// "calls through the server do not race" rests on that sequential loop, not on
// this mutex. What the mutex covers is several `Serve` instances sharing one
// process, which is what the concurrency test builds.
//
// Keep it, and keep this note: if anyone ever dispatches lines concurrently to
// get parallelism, the loop stops being the guarantee and the mutex becomes
// the only thing standing between two callers and a lost ledger entry.
//
// It is package-level rather than per-Serve because it is the ledger FILE being
// protected, not the session. A CLI process running beside the server is a
// different process and races regardless — still the CLI limitation.
var gate sync.Mutex

// callToolResult is the protocol's envelope. The spec requires a content array;
// a host may reject or hide a tool that answers with a bare result. The verdict
// travels in StructuredContent, encoded from the same value the --json receipt
// carries, so "one answer" is a claim about the verdict and not about the
// envelope around it.
type callToolResult struct {
	Content           []contentBlock `json:"content"`
	StructuredContent any            `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// callParams is the tools/call payload.
type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// text renders one text block.
func text(s string) []contentBlock { return []contentBlock{{Type: "text", Text: s}} }

// result assembles a CallToolResult with the two content blocks the spec asks
// for: the serialized structured content FIRST — "a tool that returns
// structured content SHOULD also return the serialized JSON in a TextContent
// block" — and the human-readable report second.
//
// The JSON is marshalled ONCE here and the same value is handed back as
// structuredContent, so the two halves cannot disagree. Marshalling twice from
// the same value is how they start to.
func result(structured any, report string, isErr bool) (callToolResult, *rpcError) {
	b, err := json.Marshal(structured)
	if err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: "encoding the result: " + err.Error()}
	}
	return callToolResult{
		// THE REPORT FIRST, the JSON second. The spec asks for the serialized
		// structured content in "a TextContent block", not in the first one,
		// and for mrw_read the first block is where the FILE CONTENT lives —
		// which is the entire payload a caller asked for. Putting the receipt
		// there instead would hand a model metadata where it expected the file,
		// and would change what content[0] meant for anyone already reading it.
		Content: []contentBlock{
			{Type: "text", Text: report},
			{Type: "text", Text: string(b)},
		},
		StructuredContent: json.RawMessage(b),
		IsError:           isErr,
	}, nil
}

// readResult is result() without the structuredContent. Every mrw_read answer
// takes this form — served, paged, index, refused — because of a host measured
// on 2026-09-05 (Claude Code 2.1.261, issue #109): a non-error result that
// carries structuredContent reaches the model AS the structuredContent, and the
// content blocks are dropped. For mrw_write that is the verdict; for mrw_read it
// was the receipt without the lines, while the ledger had already recorded
// those lines as seen — ADR-002 inverted by an envelope. So the receipt
// travels in content[1] alone (ADR-023), from the same single marshal.
func readResult(structured any, report string, isErr bool) (callToolResult, *rpcError) {
	res, err := result(structured, report, isErr)
	res.StructuredContent = nil
	return res, err
}

// callTool routes one tools/call. Both tools are adapters: they parse what the
// CLI parses, call the function the CLI calls, and return what it returned. The
// moment one computes a verdict of its own there are two answers to "did this
// apply?", which is the defect class this project exists to refuse.
func callTool(root string, raw json.RawMessage) (callToolResult, *rpcError) {
	var p callParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "params: " + err.Error()}
	}

	// One writer at a time, for the whole call: the read and the ledger record
	// that follows it are one transaction as far as another caller is concerned.
	gate.Lock()
	defer gate.Unlock()

	var res callToolResult
	var rpcErr *rpcError
	switch p.Name {
	case "mrw_read":
		res, rpcErr = readTool(root, p.Arguments)
	case "mrw_write":
		res, rpcErr = writeTool(root, p.Arguments)
	default:
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "unknown tool: " + p.Name}
	}
	if rpcErr != nil {
		return callToolResult{}, rpcErr
	}
	// ⚠ ONE POSTCONDITION, HERE, BECAUSE "EVERY PATH CHECKS" WAS NOT TRUE.
	// ADR-032 bounded the paths that compose a large answer and left the ones
	// that compose a small one — errorResult carried no check at all, so at a
	// small ceiling the REFUSALS exceeded the number _meta advertises, which is
	// the promise the record is named after. Enumerating return sites is how
	// that happened; a funnel cannot be forgotten. Found by the Codex review of
	// #135.
	return withinCeiling(res)
}

// withinCeiling is the last thing every tool result passes through.
//
// ⚠ IT MAY ONLY EVER SHRINK AN ANSWER THAT LICENSED NOTHING. A served read is
// measured and REFUSED before its ledger record is written (readTool), so it
// reaches here already inside the ceiling and this function never rewrites it.
// If that ordering were reversed, this would discard lines the ledger had just
// recorded as seen — ADR-002 inverted by a size check, which is exactly the
// shape ADR-031 was written about. TestTheCeilingNeverShrinksAServedRead pins
// it.
//
// When not even the refusal fits, the answer is a JSON-RPC error: it carries no
// `result` member, so it is outside the ceiling this server advertises, and a
// transport-level failure is the honest reading of "you asked for a budget in
// which I cannot answer at all".
func withinCeiling(res callToolResult) (callToolResult, *rpcError) {
	n := encodedSize(res)
	if n <= MaxResultChars {
		return res, nil
	}
	small := errorResult(fmt.Sprintf("this answer came to %d bytes and the ceiling in force is %d, "+
		"so it is not being sent. Ask for less in one call, or raise the ceiling with "+
		"--max-result-chars.", n, MaxResultChars))
	if encodedSize(small) <= MaxResultChars {
		return small, nil
	}
	return callToolResult{}, &rpcError{Code: codeInternal, Message: fmt.Sprintf(
		"--max-result-chars %d is smaller than any tool result this server can produce; "+
			"nothing was done", MaxResultChars)}
}

// readTool serves ranges and records what it observed, exactly as `mrw read`
// does — including the ledger write, which is how mrw learns what a file holds
// and therefore what a later write is allowed to address.
func readTool(root string, args json.RawMessage) (callToolResult, *rpcError) {
	var a struct {
		Specs []string `json:"specs"`
		// Grep turns the specs from "what to serve" into "where to look":
		// read.Walk finds the files and supplies the specs itself. This is
		// `mrw read --grep` over the wire, calling the same primitive in the
		// same order the CLI calls it (cmd/mrw/main.go:510).
		Grep    string   `json:"grep"`
		Exclude []string `json:"exclude"`
		// After resumes a paged INDEX. It is the missing half of next_index:
		// without an argument that accepts it, the index named a continuation
		// nothing could follow — a field describing a dead end, which is the
		// exact defect this record was written to prevent, one level down.
		// Found by review of #80.
		After string `json:"after"`
		// Ack carries the checkpoints the caller actually received, and it is
		// what turns a served page into a licensed one (ADR-031). Absent means
		// "I acknowledge nothing", which is the safe reading and what a caller
		// written before this field sends.
		Ack []string `json:"ack"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "arguments: " + err.Error()}
	}
	// Promote before serving: the caller is acknowledging the PREVIOUS page,
	// and a write in the same turn must see the licence this call grants.
	if err := promote(root, a.Ack); err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: "ack: " + err.Error()}
	}
	// With grep, no spec is required at all — the walk starts at the root, the
	// way `mrw read --grep P` with no paths does. Without it, a read with
	// nothing to read is the caller's mistake.
	if len(a.Specs) == 0 && a.Grep == "" {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "mrw_read needs at least one spec"}
	}
	if len(a.Exclude) > 0 && a.Grep == "" {
		return errorResult("exclude without grep: there is nothing to exclude from"), nil
	}
	// `after` resumes a grep's index, so without one it means nothing. Silently
	// ignoring it is how a caller believes it is paging while re-reading page
	// one forever — the same silence `exclude` already refuses, and it deserves
	// the same sentence. Found by review of #80.
	if a.After != "" && a.Grep == "" {
		return errorResult("after without grep: it resumes a grep's index, and there is no index without one"), nil
	}

	var specs []read.Spec
	var walked bool
	// walkProblems are paths the WALK could not use — absent, unreadable, or
	// outside the root. The CLI prints every one of them and counts them as
	// failures (cmd/mrw/main.go:553); the first cut of this dropped them on the
	// floor, so a caller naming a directory that does not exist was told "no
	// file matches" — a clean answer about a question nobody asked. With a
	// valid sibling path the bad one vanished entirely. Found by review of #80.
	var walkProblems []read.Problem
	if a.Grep != "" {
		var err error
		specs, walkProblems, err = grepSpecs(root, a.Specs, a.Grep, a.Exclude, a.After)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		walked = true
		if len(specs) == 0 {
			// Not an error: "nothing matched" is a real answer, and the
			// caller asked a question rather than named a file that is
			// missing. But a walk that could not LOOK somewhere is a
			// different answer again, and it is an error — otherwise a
			// typo'd path reads as a searched-and-empty tree.
			report := fmt.Sprintf("no file under the root matches /%s/.", a.Grep)
			for _, p := range walkProblems {
				report += fmt.Sprintf("\n-- %s: %s", p.Path, p.Reason)
			}
			return readResult(map[string]any{
				"observed": map[string]seen.Observation{},
				"problems": len(walkProblems),
				"matches":  0,
			}, report, len(walkProblems) > 0)
		}
	} else {
		specs = make([]read.Spec, 0, len(a.Specs))
		for _, s := range a.Specs {
			sp, err := read.ParseSpec(s)
			if err != nil {
				// A spec mrw cannot parse is the caller's mistake, reported as a
				// tool error rather than served as an empty read: "nothing here"
				// and "I could not understand you" are different answers.
				return errorResult(fmt.Sprintf("%s: %v", s, err)), nil
			}
			specs = append(specs, sp)
		}
	}
	// A CAPPED writer, not a size check afterwards. The first version of this
	// buffered the whole read and then measured it. That refused correctly and
	// left the actual defect in place: 40 x 18 MB still peaked at 2.4 GB, barely
	// under the 2.6 GB it cost with no limit at all. A refusal has to be cheap,
	// or it is only a better error message on the way to the same OOM.
	//
	// capped discards past the limit and remembers that it did, so peak memory
	// is the limit plus one write however large the request was. Measured
	// 2026-09-03: the same 40 x 18 MB request now peaks at 87 MB.
	cw := &capped{limit: MaxResultChars}
	w := bufio.NewWriter(cw)
	observed, problems := read.Run(w, root, specs, read.Options{Numbers: true})
	w.Flush()

	// A result over the declared limit is REFUSED, not truncated.
	//
	// ADR-007's own cap reports itself when it fires, which is right for a
	// person reading a terminal. Over MCP the consumer is a model, and a
	// truncated file that arrives looking like the file is exactly the silent
	// wrong answer this project exists to refuse. An oversized result does not
	// reach the model as the file anyway — the host persists it to disk and
	// replaces it with a file reference — so the choice was never "cap or stay
	// faithful to the CLI"; it was refuse legibly, or pay the memory to build a
	// result the host then takes out of the conversation.
	//
	// The ledger is deliberately NOT written here: a refused read showed the
	// caller nothing, and an entry claiming otherwise would license a later
	// write against a file they never saw. That is ADR-002's guarantee, and it
	// is the one thing a size limit must not quietly spend.
	if cw.over {
		// ADR-014: an oversized read is a FIRST PAGE, not a dead end.
		//
		// The refusal ADR-011 shipped was correct and unactionable past one
		// step: it suggested a single range and left the caller to compute
		// every range after it, so a caller that followed it once had
		// confidently read part of a file.
		//
		// The continuation is always "the rest" — `path:N-` — which is what
		// makes this terminate without touching the normal path: each
		// continuation is itself too large until the remainder fits, at which
		// point it returns as an ordinary successful read carrying no
		// continuation at all. The caller's exit condition is the absence of a
		// field rather than a count it has to keep.
		//
		// It carries NO isError, per ADR-024 — the difference between paging and
		// truncation is the `-- PARTIAL:` line in the served text, not a flag a
		// host reads as "this call failed" and then truncates the answer over.
		// ADR-017: a grep too large to SERVE still answers, with the addresses
		// it found. firstPage cannot help here — it needs one open-ended spec
		// and a walk produces many across many files — so without this branch
		// an oversized grep would fall to the flat refusal below, which is the
		// dead end ADR-014 removed reappearing through a new door, firing on
		// this population's ordinary case rather than an exotic one.
		if walked {
			return matchIndex(specs, len(walkProblems), cw), nil
		}
		if page, ok := firstPage(root, a.Specs, cw); ok {
			return page, nil
		}
		return errorResult(overflowMessage(a.Specs, cw)), nil
	}

	// ⚠ THE WALK'S PROBLEMS TRAVEL WITH THE SERVED ANSWER TOO. They reached the
	// no-match branch and the index and stopped there, so a caller naming a bad
	// path ALONGSIDE a good one got the good one and silence about the other —
	// `problems: 0`, no isError, the bad path unmentioned. The commit that added
	// them claimed this case was fixed; it was not, and the test passed only
	// because it named ONLY the bad path, which takes the no-match branch.
	// Found by review of #80.
	report := cw.buf.String()
	for _, p := range walkProblems {
		report += fmt.Sprintf("\n-- %s: %s", p.Path, p.Reason)
	}
	problems += len(walkProblems)

	// ⚠ AND THE SERVED ANSWER IS BUDGETED, for the same reason the index is.
	// The capped writer bounds the REPORT TEXT and nothing else: `observed`
	// carries a sha and spans per file and travels once more, serialized in
	// content[1] (twice more before ADR-023). A grep resuming onto 2,514
	// small files came back at 794,582 characters — four times the cap this
	// server declares in _meta, and past the ceiling the host truncates at. So
	// a read that will not fit ENCODED degrades to something that does. Found
	// by review of #80.
	//
	// ⚠ THE ANSWER IS COMPOSED ONCE AND THAT SAME OBJECT IS MEASURED. This was
	// a probe assembled beside the answer, which is how a check ends up on a
	// shape that is not what got sent — ADR-031 did exactly that twice in
	// consecutive reviews, and the comment at firstPage's own size check
	// records both. A probe cannot be wrong about the thing it IS.
	//
	// ⚠ AND IT IS NO LONGER ONLY THE WALKED READ (ADR-032). The check was
	// reached only with `grep`, so a caller naming hundreds of specs of its own
	// had its report bounded and its receipt not.
	//
	// The receipt — seen.Observation, no json tags, so its keys are the Go
	// field names — travels in content[1] and NOT in structuredContent
	// (ADR-023; see readResult). readSchema() still describes it for a reader
	// of the code, but tools/list no longer declares it: a schema declared is
	// a structuredContent promised, and none is sent.
	// ⚠ AND AN ANSWER THAT SERVED NOTHING IS AN ERROR (ADR-025). The observation
	// count is the whole test, and it is deliberately not conjoined with
	// `problems > 0`: a spec that served no LINES is still OBSERVED — an empty
	// file addressed by a range, a range that misses, both noted with empty spans
	// and both counting a problem — so the problem count cannot exclude them and
	// the observation count can. (A bare spec on an empty file counts no problem
	// and is excluded by the observation alone.)
	// Neither shape reaches here with an empty map and no problem either: a
	// read naming no spec is refused at :158 when it passes no grep, and a clean
	// grep that matched nothing answers at :202. So the conjunct could never
	// discriminate and no mutation could kill it.
	// ADR-024 removed the flag from answers that DELIVERED something; this restores
	// it for the one case its enumeration missed, so :202 and this return agree
	// rather than disagreeing on whether `grep` was passed.
	served, rpcErr := readResult(map[string]any{
		"observed": observed,
		"problems": problems,
	}, report, len(observed) == 0)
	if rpcErr != nil {
		return callToolResult{}, rpcErr
	}
	if encodedSize(served) > cw.limit {
		// The ledger is deliberately NOT written on any of these paths: the
		// caller is about to be handed something other than these lines, and an
		// entry claiming otherwise licenses a write against a file they never
		// saw. That is ADR-002's guarantee.
		//
		// A walk degrades to its index, which is resumable. A named read
		// degrades to a first page when one spec can carry it, and otherwise
		// says why — with its own sentence, because "your read was too large"
		// is not what happened here.
		if walked {
			return matchIndex(specs, problems, cw), nil
		}
		if page, ok := firstPage(root, a.Specs, cw); ok {
			return page, nil
		}
		return errorResult(receiptOverflowMessage(encodedSize(served), cw.limit)), nil
	}

	// Reading is how mrw learns what a file holds; recording that is what lets
	// a later write know whether its picture is still current.
	if err := seen.Record(root, observed); err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: "recording the ledger: " + err.Error()}
	}

	return served, nil
}

// writeTool applies a plan through apply.Apply and returns the same Result the
// --json receipt carries. Every step here mirrors `mrw write`: parse, resolve
// working-set pointers, load the ledger, apply, record what was written, and
// count the outcome for ADR-009's tally.
func writeTool(root string, args json.RawMessage) (callToolResult, *rpcError) {
	var a struct {
		Plan   string   `json:"plan"`
		DryRun bool     `json:"dry_run"`
		Ack    []string `json:"ack"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "arguments: " + err.Error()}
	}
	// The checkpoints for the page this plan was written against, promoted
	// before the ledger is consulted (ADR-031).
	if err := promote(root, a.Ack); err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: "ack: " + err.Error()}
	}
	if strings.TrimSpace(a.Plan) == "" {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "mrw_write needs a plan"}
	}

	hunks, err := plan.Parse(strings.NewReader(a.Plan))
	if err != nil {
		// A plan that did not PARSE is the outcome ADR-009 counts as the one
		// saying the FORMAT was the problem. Recorded at the site that decided
		// it, on this transport as on the other.
		_ = authoring.Record(root, authoring.RefusedParse)
		return errorResult(err.Error()), nil
	}

	set, err := iter.Load(root)
	if err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: err.Error()}
	}

	in := make([]apply.Input, 0, len(hunks))
	for _, h := range hunks {
		path := h.Path
		if iter.IsPointer(path) {
			got, err := set.Resolve(path)
			if err != nil {
				return errorResult(fmt.Sprintf("line %d: %v", h.SrcLine, err)), nil
			}
			if len(got) != 1 {
				return errorResult(fmt.Sprintf("line %d: %s names %d entries; a hunk needs exactly one",
					h.SrcLine, path, len(got))), nil
			}
			path = iter.Path(got[0])
		}
		in = append(in, apply.Input{
			Path: path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
			StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat, RelEnd: h.Addr.RelEnd, CountedBody: h.CountedBody,
			Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor,
			SrcLine: h.SrcLine, Index: h.Index,
		})
	}

	ledger, err := seen.Load(root)
	if err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: err.Error()}
	}
	// ⚠ REFUSE BEFORE APPLYING WHEN THE VERDICT COULD NOT BE REPORTED.
	//
	// A write is not undoable and a receipt is not optional: once the plan has
	// applied, every answer this server can give must be TRUE about the tree.
	// If the ceiling cannot carry even the smallest honest post-apply sentence,
	// there is no truthful answer left to give afterwards — so the honest
	// moment to refuse is now, with the tree untouched.
	//
	// Measured on the built binary at e8c1a29, before this guard: with
	// `--max-result-chars 0` a licensed one-hunk write changed the file, wrote
	// the ledger, and answered "0 of 1 hunk(s) failed and nothing was written".
	// A false statement about the filesystem, which is the defect this whole
	// tool exists to refuse. Found by the Codex review of #135.
	if !writeFloorFits() {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: fmt.Sprintf(
			"--max-result-chars %d is too small to report what a write did, so nothing was "+
				"applied and the tree is unchanged. Raise the ceiling to at least %d.",
			MaxResultChars, writeFloor())}
	}
	res, applyErr := apply.Apply(root, in, apply.Options{DryRun: a.DryRun, Seen: ledger})
	// ADR-001 rule 3: the receipt is filled even when the filesystem failed, so
	// it is rendered on whichever path we are on rather than discarded.

	// A refusal names the fix (ADR-015). Over MCP the commonest reason a line is
	// unread is now that its page was served but never acknowledged, and the
	// ledger's own message cannot say so — it belongs to the engine, which knows
	// nothing about pages. So the remedy is added HERE, and only when there
	// really is something pending to acknowledge.
	nameTheAck(root, &res)

	if res.Applied && !res.DryRun {
		// A file mrw just wrote is one it knows WHOLLY: it produced every line.
		wrote := map[string]seen.Observation{}
		for _, f := range res.Files {
			if f.Written {
				wrote[f.Path] = seen.Observation{SHA: f.SHAAfter}
			}
		}
		if err := seen.Record(root, wrote); err != nil {
			return callToolResult{}, &rpcError{Code: codeInternal, Message: err.Error()}
		}
	}

	switch {
	case applyErr != nil || res.Failed > 0 || !res.Applied:
		_ = authoring.Record(root, authoring.RefusedApply)
	default:
		// The MCP path never runs --check, so an applied plan is Applied and
		// nothing else. CheckNotRun would claim a check was configured and
		// skipped, which is a different fact.
		_ = authoring.Record(root, authoring.Applied)
	}

	return boundedReceipt(res, applyErr, applyErr != nil || res.Failed > 0)
}

// writeReceipt is what mrw_write returns: the engine's own Result, plus the one
// fact the engine cannot know — that this transport had to shorten it.
//
// The field is embedded and untagged, so encoding/json inlines it and the
// generated schema inlines it too (fieldsOf). `elided` is omitempty, so a
// receipt that dropped nothing is byte-identical to the one this server sent
// before ADR-032.
type writeReceipt struct {
	apply.Result
	// Elided says what this receipt left out to fit the budget. Absent when it
	// left out nothing, which is every ordinary write.
	Elided string `json:"elided,omitempty"`
}

// writeReport renders the per-hunk verdicts, the counts, and any elision.
//
// ⚠ The counts come from the WHOLE result even when `hunks` is a subset of it.
// They are the fact a caller checks first, and a shortened receipt that also
// shortened them would be a lie rather than an omission.
func writeReport(res apply.Result, hunks []apply.HunkResult, applyErr error, elided string) string {
	var b bytes.Buffer
	for _, h := range hunks {
		fmt.Fprintf(&b, "%s %s %s %s\n", h.Status, h.Path, h.Addr, h.Reason)
	}
	fmt.Fprintf(&b, "%d hunk(s), %d file(s), %d failed\n", len(res.Hunks), len(res.Files), res.Failed)
	if applyErr != nil {
		fmt.Fprintf(&b, "error: %v\n", applyErr)
	}
	if elided != "" {
		fmt.Fprintf(&b, "-- %s\n", elided)
	}
	return b.String()
}

// boundedReceipt is the answer mrw_write sends: the whole receipt when it fits
// the budget this server advertises, and otherwise one with its SUCCESSFUL
// detail dropped (ADR-032).
//
// Both tools advertised a ceiling and only the read path kept one. Measured
// 2026-09-07: a 4,000-hunk dry-run returned 453,632 characters against an
// advertised 200,000, and a host that trusts the number truncates — which is
// the answer ADR-031 exists because mrw cannot see.
//
// ⚠ TWO THINGS ARE NEVER ELIDED: a FAILED hunk, and the file record of a file
// that WAS WRITTEN. Under ADR-001 a failure is why nothing was written, so it is
// the one verdict a caller cannot act without; the successes of a plan that
// applied are what `applied` already told them. A written file is the mirror of
// that — it is the evidence that the tree changed, and for a PARTIAL
// application `applied` is false and says the opposite. Files go only after the
// successful hunks, and only if dropping those was not enough.
//
// ⚠ AND THE ELISION IS STATED, in the receipt and in the report both. An answer
// silently shorter than the truth is the defect this tool exists to refuse, and
// ADR-014 makes saying so the rule for any partial answer. It is in the
// STRUCTURED value and not only in the text because a host measured on
// 2026-09-05 delivers mrw_write's answer to the model as the structured value
// alone (ADR-023).
func boundedReceipt(res apply.Result, applyErr error, isErr bool) (callToolResult, *rpcError) {
	full, rpcErr := result(writeReceipt{Result: res}, writeReport(res, res.Hunks, applyErr, ""), isErr)
	if rpcErr != nil || encodedSize(full) <= MaxResultChars {
		return full, rpcErr
	}
	whole := encodedSize(full)

	kept := make([]apply.HunkResult, 0, res.Failed)
	for _, h := range res.Hunks {
		if h.Status == apply.StatusFailed {
			kept = append(kept, h)
		}
	}
	short := res
	short.Hunks = kept

	for _, alsoFiles := range []bool{false, true} {
		note := fmt.Sprintf("elided to fit the %d-byte budget, which the whole receipt exceeded at %d: "+
			"%d successful or skipped hunk verdict(s) are not here",
			MaxResultChars, whole, len(res.Hunks)-len(kept))
		if alsoFiles {
			// ⚠ ONLY THE UNWRITTEN FILES GO. The first cut dropped every file
			// record, and a PARTIAL application — Applied=false, Failed=0,
			// earlier files already renamed — then came back as applied:false,
			// failed:0, files:[], hunks:[], which a host that delivers only the
			// structured value (ADR-023) reads as "nothing happened". That is
			// the same denial the terminal branch below was fixed for, one
			// return earlier: each cut of this record moved it up by one.
			// Keeping the written records cannot hide a write, and when there
			// are too many of them to fit, the terminal branch says so in
			// words. Codex, third review of #135.
			short.Files = writtenFiles(res.Files)
			note += fmt.Sprintf(", nor %d file record(s) for files that were NOT written",
				len(res.Files)-len(short.Files))
		}
		note += ". Every FAILED hunk is here, every file that WAS written is here, " +
			"and the counts are of the whole plan."

		out, rpcErr := result(writeReceipt{Result: short, Elided: note}, writeReport(res, kept, applyErr, note), isErr)
		if rpcErr != nil {
			return out, rpcErr
		}
		if encodedSize(out) <= MaxResultChars {
			return out, nil
		}
	}

	// ⚠ THIS BRANCH IS REACHABLE AFTER A SUCCESSFUL WRITE, and the first cut of
	// it did not know that. It said "nothing was written" unconditionally,
	// reasoning that a receipt this large must be all failures and that ADR-001
	// therefore wrote nothing. A small ceiling breaks that reasoning: with
	// `--max-result-chars 0` a one-hunk write applied, recorded its ledger
	// entry, and was told nothing had happened. The guard before apply.Apply
	// now makes this unreachable for an applied write, and this branch tells
	// the truth anyway — a verdict that depends on a guard elsewhere staying
	// correct is the kind that comes back.
	// ⚠ AND THE TEST IS "DID ANY FILE CHANGE", NOT "DID THE PLAN APPLY". The
	// second cut asked res.Applied, which is FALSE for a partial application:
	// apply.Apply renames file by file, and a rename that fails after earlier
	// ones succeeded returns Applied=false with those files already on disk —
	// the engine has `writtenSoFar` for exactly that case. Asking Applied would
	// deny a write that happened, one review after the same denial for a
	// complete one. Found by the second Codex review of #135.
	written := 0
	for _, f := range res.Files {
		if f.Written {
			written++
		}
	}
	if written > 0 {
		return errorResult(appliedButUnreportable(written, len(res.Hunks), res.Failed, !res.Applied)), nil
	}
	return errorResult(fmt.Sprintf("%d of %d hunk(s) failed and nothing was written. Naming them "+
		"takes more than the %d-byte ceiling this server advertises, so they are not listed here. "+
		"Send fewer hunks in one plan, or use the CLI `mrw write`, which streams and has no such "+
		"limit.", res.Failed, len(res.Hunks), MaxResultChars)), nil
}

// writtenFiles is the subset of file records whose file actually changed on
// disk. It is what stage-two elision keeps: dropping these is what let a receipt
// deny a write that had already happened.
func writtenFiles(files []apply.FileResult) []apply.FileResult {
	out := make([]apply.FileResult, 0, len(files))
	for _, f := range files {
		if f.Written {
			out = append(out, f)
		}
	}
	return out
}

// appliedButUnreportable is what this server says when the tree changed and the
// receipt naming the change will not fit. One function so the message the floor
// is measured against and the message actually sent cannot drift apart.
func appliedButUnreportable(written, hunks, failed int, partial bool) string {
	state := "the plan APPLIED"
	if partial {
		state = "the plan PARTIALLY APPLIED — a later file failed after earlier ones were already written"
	}
	return fmt.Sprintf("%s: %d file(s) changed on disk, %d hunk(s), %d failed. NAMING them takes "+
		"more than the %d-byte ceiling this server advertises, so the per-hunk detail is not here "+
		"— but the write HAPPENED. Read the files, or re-run with a larger --max-result-chars.",
		state, written, hunks, failed, MaxResultChars)
}

// writeFloor is the size of the smallest truthful thing this server can say
// about a write that has already happened, and writeFloorFits asks whether the
// ceiling in force can carry it.
//
// ⚠ IT IS A PROVEN FLOOR, NOT A PLAUSIBLE ONE. The first cut substituted 999999
// for each count and called that "the widest plausible width"; nothing bounds a
// plan to a million hunks, so a wide enough real count would exceed the probe,
// pass the guard, and then be replaced by the funnel's generic refusal — which
// does not say the write happened. math.MaxInt is the widest any int can
// render, and the PARTIAL wording is the longer of the two, so this is an upper
// bound on the real message by construction. Codex, second review of #135.
//
// ⚠ IT IS SELF-REFERENTIAL: the message quotes the ceiling in force, so a
// narrower ceiling renders a shorter message and a smaller floor. That is
// consistent, because the guard and the real message read MaxResultChars at the
// same moment — but it means a floor computed at one ceiling says nothing about
// another, which is what TestTheWriteFloorIsAFloor asserts across ten of them.
func writeFloor() int {
	return encodedSize(errorResult(appliedButUnreportable(math.MaxInt, math.MaxInt, math.MaxInt, true)))
}

func writeFloorFits() bool { return writeFloor() <= MaxResultChars }

// errorResult reports a failure the CALLER caused, inside a normal tool result.
// A tool error is not a protocol error: the request was well-formed and the
// answer is "no", which a host shows to its user rather than treating as a
// transport fault.
func errorResult(msg string) callToolResult {
	return callToolResult{Content: text(msg), IsError: true}
}

// countLines reports how many lines a rendered read produced, so a refusal can
// suggest a range in the units the caller actually addresses — line numbers.
func countLines(buf *bytes.Buffer) int {
	return bytes.Count(buf.Bytes(), []byte{'\n'})
}

// suggestLines picks a line count whose output would fit, from what the full
// read actually measured. It is derived rather than guessed: dividing the real
// size by the real line count gives this file's own average, which beats any
// constant for a file of long lines or short ones.
func suggestLines(chars, lines int) int {
	if lines <= 0 || chars <= 0 {
		return 500
	}
	perLine := chars / lines
	if perLine < 1 {
		perLine = 1
	}
	// Three quarters of the limit, so the suggestion has room to be wrong.
	n := (MaxResultChars * 3 / 4) / perLine
	if n < 1 {
		n = 1
	}
	return n
}

// capped is an io.Writer that keeps at most limit bytes and records that it
// stopped. It exists so a refusal costs the limit rather than the whole read:
// bounding AFTER buffering refuses correctly and still pays the memory, which
// is the failure that made this necessary — measured 2026-09-03 at 2.4 GB for a
// request that was refused.
//
// written counts everything offered, not everything kept, so the refusal can
// tell a caller how far over they were rather than only that they were over.
type capped struct {
	buf     bytes.Buffer
	limit   int
	written int
	over    bool
}

func (c *capped) Write(p []byte) (int, error) {
	c.written += len(p)
	if room := c.limit - c.buf.Len(); room > 0 {
		if room > len(p) {
			room = len(p)
		}
		c.buf.Write(p[:room])
	}
	if c.written > c.limit {
		c.over = true
	}
	// Always report a full write: an io.Writer that short-writes makes its
	// caller error, and read.Run's job is not to know it is being bounded.
	//
	// THE TRADE this makes explicit: the refusal is cheap in MEMORY and not in
	// TIME. read.Run keeps reading and keeps offering bytes we discard, so a
	// 40 x 18 MB request takes seconds to refuse. Short-writing would stop it
	// sooner and would surface inside the engine's own report as a per-file
	// problem, which is a worse answer than a slow correct one — the caller
	// would be told their FILE failed rather than their REQUEST was too large.
	return len(p), nil
}

// firstPage serves as much of a single-file read as fits and names the spec
// that asks for the rest. It reports false when it cannot honestly page, and
// the caller falls back to the flat refusal.
//
// It pages only a SINGLE spec naming a whole file or an open-ended `path:N-`
// range. With several specs the one that crossed the limit may not be the
// first, and a page of the wrong file would be worse than a refusal; with a
// CLOSED range the caller has already said what it wants and mrw narrowing it
// further would be answering a question nobody asked.
func firstPage(root string, specs []string, cw *capped) (callToolResult, bool) {
	if len(specs) != 1 {
		return callToolResult{}, false
	}
	path, start, ok := openEnded(specs[0])
	if !ok {
		return callToolResult{}, false
	}
	total, err := countFileLines(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil || total <= 0 {
		return callToolResult{}, false
	}
	per := suggestLines(cw.buf.Len(), countLines(&cw.buf))
	if per < 1 {
		return callToolResult{}, false
	}
	end := start + per - 1
	if end >= total {
		// The remainder already fits, so there is nothing to page: this can
		// only be reached if the estimate disagrees with the cap, and serving
		// a "page" that is the whole rest while claiming to be partial would
		// be a lie in the safe-looking direction.
		return callToolResult{}, false
	}
	sp, err := read.ParseSpec(fmt.Sprintf("%s:%d-%d", path, start, end))
	if err != nil {
		return callToolResult{}, false
	}
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	observed, problems := read.Run(w, root, []read.Spec{sp}, read.Options{Numbers: true})
	w.Flush()

	// ⚠ AND IF THAT SECOND READ SERVED NOTHING, THIS IS NOT A PAGE (ADR-025).
	// countFileLines succeeded a moment ago, so an empty `observed` here means
	// the file stopped being readable in between — deleted, or its permissions
	// changed. Returning a page then would fabricate a `-- PARTIAL:` notice for
	// content nobody received, and `pagedResult` omits `isError`, so it would be
	// the one served-nothing answer that still claimed success. Declining sends
	// the caller down the ordinary path, which reports the real reason and flags
	// it. Found by the Codex review of #123.
	if len(observed) == 0 {
		return callToolResult{}, false
	}

	// ⚠ AND IF THE PAGE ITSELF DOES NOT FIT, IT IS NOT A PAGE EITHER (ADR-031).
	// The budget above is estimated in LINES, so a single line longer than the
	// whole cap produces a one-line "page" that still exceeds it. A host must
	// then cut it, and a head/tail cut of ONE numbered line leaves the open
	// marker, the line's `NNN|` prefix and the close marker all intact — so the
	// caller can satisfy AckRule honestly while the line's middle never arrived,
	// and acknowledging licenses the whole of it. Bracketing cannot express a
	// partial line: the unit it proves is a line. Declining sends the caller
	// down the ordinary path, which refuses with the limit and a line budget.
	// Found by the fifth review of PR #132.
	// ⚠ THE PAGE WAS SENT, WHICH IS NOT THE SAME AS RECEIVED, AND THIS LINE USED
	// TO CONFUSE THEM. It recorded the served span outright — "the page WAS
	// shown" — and on 2026-09-05 a host cut the middle out of exactly such a
	// page, leaving the model lines 1-90 and 2644-2727 while mrw claimed
	// 1-3619; a write to line 1500 then applied at exit 0. So the span is held
	// PENDING against checkpoints woven through the text, and reaches the
	// ledger only when the caller echoes them back (ADR-031).
	text, spans := interleave(b.String())
	next := fmt.Sprintf("%s:%d-", path, end+1)
	report := fmt.Sprintf("%s\n-- PARTIAL: lines %d-%d of %d. %d line(s) remain.\n"+
		"-- Send specs [%q] to continue, or a narrower range of your own.\n"+
		"-- Stopping here means you have part of this file, not the file.\n"+
		"-- This page licenses NOTHING until you acknowledge it.\n-- "+AckRule+"\n"+
		"-- An id you omit leaves its lines unwritable, which is the point.",
		text, start, end, total, total-end, next)

	// ⚠ THE ENCODED RESULT IS WHAT MUST FIT, measured with encodedSize — the
	// function this file already uses for the grep paths, twenty lines up.
	//
	// This took three attempts and each measured something that was not what a
	// host receives. First the raw buffer, before interleave's markers and this
	// footer. Then len(report), which omits the receipt and the JSON envelope:
	// an ordinary one-line read composed a 199,794-byte report and delivered
	// 200,004 bytes (seventh review of PR #132). The right primitive was in
	// this file the whole time, used correctly a few functions away.
	//
	// It is an assertion about what to SEND, not a guard against doing the
	// work: the composing has happened by now and `capped` is what bounds the
	// cost, stopping the READ at the limit. What it buys is that an over-cap
	// answer is never delivered — and it sits before hold, so nothing is
	// recorded pending for a page that is never sent.
	// ⚠ `observed` DESCRIBES WHAT WAS SERVED, not what is licensed, and passing
	// nil to signal "nothing licensed yet" produced `"observed": null` against
	// a schema that requires an object (readSchema) and a README that says the
	// receipt is unchanged in shape. The conformance assertion checked only key
	// PRESENCE, so null passed it. Licensing is the ledger's business and the
	// checkpoints say what is pending; the receipt goes back to telling the
	// truth about service. Eighth review of PR #132.
	res := pagedResult(report, observed, problems, next)
	if encodedSize(res) > MaxResultChars {
		return callToolResult{}, false
	}
	if err := hold(root, observed, spans); err != nil {
		return callToolResult{}, false
	}
	return res, true
}

// openEnded reports the path and start line of a spec that asks for a whole
// file or for `path:N-`. Anything else is not pageable — see firstPage.
func openEnded(spec string) (path string, start int, ok bool) {
	i := strings.LastIndex(spec, ":")
	if i < 0 {
		return spec, 1, true
	}
	addr := spec[i+1:]
	if !strings.HasSuffix(addr, "-") {
		return "", 0, false
	}
	n, err := strconv.Atoi(strings.TrimSuffix(addr, "-"))
	if err != nil || n < 1 {
		return "", 0, false
	}
	return spec[:i], n, true
}

// countFileLines counts newline-terminated lines without holding the file, so
// the page arithmetic knows where the end is without spending the memory the
// cap exists to save.
func countFileLines(full string) (int, error) {
	f, err := os.Open(full)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}

// overflowMessage explains a refused read and, where it honestly can, names the
// narrower request to make instead.
//
// The example range is only offered when the FIRST spec is a bare path. A spec
// that already carries a range would produce "small.txt:1-2:1-N", which is not
// valid syntax, and with several specs the one that crossed the limit may not be
// the first — so in those cases the message says what to do without inventing a
// spec that might be wrong. A hint that has to be debugged is worse than none.
func overflowMessage(specs []string, cw *capped) string {
	// The per-line average comes from the CAPPED buffer — its own bytes over
	// its own lines. Dividing the FULL byte count by the capped line count
	// mixes two samples and suggested "giant.go:1-2" for a 193 MB file.
	var b strings.Builder
	fmt.Fprintf(&b, "that read would have returned about %d bytes and the limit is %d.\n", cw.written, cw.limit)
	b.WriteString("Nothing was read and nothing was recorded, so no write is licensed by it.\n")
	// ⚠ WHEN ONE LINE CANNOT FIT, A NARROWER RANGE CANNOT HELP. Suggesting
	// `f.txt:1-1` for a file whose first line exceeds the cap sends the caller
	// to retry the identical unservable request. mrw pages by LINE and
	// acknowledges by line, so a line larger than the answer has no smaller
	// unit to fall back to — say so, and name a reader that has one. Found by
	// the sixth review of PR #132.
	// ⚠ THE QUESTION IS WHETHER ANY CONTENT LINE COMPLETED INSIDE THE SAMPLE,
	// not how long the longest run is. The sample is CAPPED at the limit, so a
	// run measured inside it can never reach the limit — a threshold against
	// the cap never fires, and half the cap fires far too often: a file of
	// 110,000-character lines got "no narrower range can be served" while
	// `file:1-1` is an ordinary 110 KB read that works. Both wrong, in
	// opposite directions, in consecutive attempts (seventh review of PR #132).
	//
	// The served text opens with two header lines. If the sample holds no more
	// newlines than that, the first CONTENT line never terminated inside a
	// whole budget — so it cannot come back as a one-line result either, and
	// mrw serves whole lines. That is the only case where no range helps.
	// ⚠ PER FILE, NOT ACROSS THE SAMPLE. Counting newlines over the whole
	// capped buffer means a small file listed FIRST supplies completed lines
	// and hides an unservable long line in the file after it — the caller then
	// gets a per-file line budget that cannot serve the long one (eighth review
	// of PR #132). Each served file's own section is examined: a section whose
	// header is followed by no completed content line is a file no range helps.
	const headerLines = 2
	unterminated := countLines(&cw.buf) <= headerLines
	if !unterminated {
		// ⚠ ONLY LINES THAT ENDED COUNT. Splitting on newline leaves the last
		// element unterminated, and the PREFIX of a giant line arrives — with
		// its "NNNNN| " gutter — so counting every element containing "| "
		// counts the very line that did not fit and hides the case.
		lines := bytes.Split(cw.buf.Bytes(), []byte{'\n'})
		if n := len(lines); n > 0 {
			lines = lines[:n-1] // drop the unterminated tail
		}
		since := 0
		for _, l := range lines {
			if bytes.HasPrefix(l, []byte("==> ")) {
				since = 0
				continue
			}
			if bytes.Contains(l, []byte("| ")) {
				since++
			}
		}
		// The LAST file in the sample is the one the cap cut short; if none of
		// its lines completed, its lines are the unservable ones.
		unterminated = since == 0
	}
	if n := suggestLines(cw.buf.Len(), countLines(&cw.buf)); unterminated || n < 1 {
		fmt.Fprintf(&b, "One line of this file renders to more than the whole %d-character limit, "+
			"and mrw serves whole lines — so no narrower range of it can be served, and retrying "+
			"with one would fail the same way. Read it with the CLI, `mrw read`, which streams and "+
			"has no such limit.", cw.limit)
	} else if len(specs) == 1 && !strings.Contains(specs[0], ":") {
		fmt.Fprintf(&b, "Ask for a range instead — for example %s:1-%d.", specs[0], n)
	} else {
		fmt.Fprintf(&b, "Ask for narrower ranges — around %d lines per file at this file's line length — or name fewer files in one call.",
			suggestLines(cw.buf.Len(), countLines(&cw.buf)))
	}
	return b.String()
}

// receiptOverflowMessage explains the overflow `capped` cannot see: the served
// TEXT fit the budget and the whole ANSWER did not.
//
// It is a different failure from a read that is simply too large, and it takes
// a different remedy, so it gets its own sentence rather than borrowing
// overflowMessage's. The excess is the per-file receipt — one sha and one span
// list per served file, the same size whatever range was asked for — so
// narrowing ranges barely moves it and naming fewer files moves it exactly.
// Telling the caller to retry with a smaller range would send them back for
// the same refusal, which is the failure the sixth review of PR #132 fixed
// once already, one message over.
func receiptOverflowMessage(encoded, limit int) string {
	return fmt.Sprintf("that read rendered inside the %d-byte limit, but its whole answer — "+
		"the lines plus the receipt naming what was served — came to %d.\n"+
		"Nothing was read and nothing was recorded, so no write is licensed by it.\n"+
		"The excess is the per-file receipt, a sha and a span list for every file served and "+
		"the same size whatever range you ask for. Name fewer files in one call rather than "+
		"narrower ranges.", limit, encoded)
}

// pagedResult builds a first-page answer: the page, the receipt, and the spec
// that asks for the rest.
//
// It duplicates little of `result` and deliberately does not reuse it: `result`
// takes isErr as a parameter and threads structuredContent, and the structured
// map here carries a field the normal shape does not. Folding the two would
// mean a boolean and an optional field threaded through the common path for one
// caller's benefit.
func pagedResult(report string, observed map[string]seen.Observation, problems int, next string) callToolResult {
	structured := map[string]any{
		"observed":  observed,
		"problems":  problems,
		"next_read": next,
	}
	b, err := json.Marshal(structured)
	if err != nil {
		// A page whose receipt will not marshal is not a page. Falling back to
		// the flat refusal is honest; returning the prose alone would hand back
		// content with no machine-readable account of what it was.
		return errorResult("could not encode the page receipt: " + err.Error())
	}
	return callToolResult{
		Content: []contentBlock{
			{Type: "text", Text: report},
			{Type: "text", Text: string(b)},
		},
		// No structuredContent: a read's answer is content[0] (ADR-023).
		//
		// AND NO isError, per ADR-024. This was `true`, so that a caller could
		// SEE it received a part — and that is exactly what destroyed the part.
		// A host reads the flag as "this call failed" and truncates such a
		// result head-and-tail: measured 2026-09-06 on Claude Code 2.1.263, the
		// same 152,594-character page arrived gapped with the flag (line 78,
		// then line 2309 of 2,380) and continuous without it. The partiality is
		// carried by the `-- PARTIAL:` notice in content[0] and by next_read in
		// content[1] — the served text, which no host rewrites.
	}
}

// grepSpecs turns a pattern and some paths into the specs read.Run serves, by
// calling read.Walk — the same primitive `mrw read --grep` calls, in the same
// order (cmd/mrw/main.go:510). Nothing here reimplements matching.
//
// The refusals mirror the CLI's, deliberately. A grammar the two surfaces
// disagree on is the class ADR-016 exists to prevent, and the caller who hits
// one should get the same sentence whichever surface it is on.
func grepSpecs(root string, paths []string, pattern string, exclude []string, after string) ([]read.Spec, []read.Problem, error) {
	for _, p := range paths {
		sp, err := read.ParseSpec(p)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %v", p, err)
		}
		if len(sp.Ranges) > 0 {
			// cmd/mrw/main.go:499, word for word: the caller has said both
			// "look here" and "look for this", and mrw will not pick one.
			return nil, nil, fmt.Errorf("%s: a range and grep are two answers to one question", p)
		}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, nil, fmt.Errorf("grep %q: %v", pattern, err)
	}
	specs, problems, err := read.Walk(root, paths, read.WalkOptions{Pattern: re, Exclude: exclude})
	if err != nil {
		return nil, nil, err
	}
	// Resume: drop everything at or before the caller's cursor. read.Walk
	// sorts by path, so "after" is a position in a total order rather than an
	// opaque token — the caller can read it, and two calls with the same
	// cursor return the same page.
	if after != "" {
		i := 0
		for i < len(specs) && specs[i].Path <= after {
			i++
		}
		specs = specs[i:]
	}
	return specs, problems, nil
}

// matchIndex is the answer to a grep whose CONTENT will not fit: the addresses,
// with no content at all.
//
// ENTRIES ARE PATHS, not `path:/pattern/`. The first cut serialized the walk's
// own address into each entry, which reads well and is WRONG for a whole class
// of patterns: `alpha/,/beta` is a valid regexp, and `f.txt:/alpha/,/beta/`
// parses back as a pattern RANGE rather than the single pattern that matched.
// The entry would still look like a spec and would read different lines than
// the ones it claimed to index — a silent wrong answer, which is the one thing
// this tool exists to refuse. A bare path cannot be misparsed, and the caller
// re-sends it WITH the same grep, so the pattern travels in the argument that
// already carries it. Found by review of #80.
//
// NOTHING IS RECORDED. The index served no lines, so it licenses no write; an
// index that licensed edits to files the caller never saw would be ADR-002's
// guarantee spent on a convenience.
func matchIndex(specs []read.Spec, problems int, cw *capped) callToolResult {
	entries := make([]string, 0, len(specs))
	for _, sp := range specs {
		entries = append(entries, sp.Path)
	}

	// The index itself can overflow, and refusing here would be the same dead
	// end one level down — so it pages BY FILE. A list's natural continuation
	// is "resume after this entry", the way a file's is "resume at this line".
	//
	// ⚠ BUDGETED AGAINST THE ENCODED RESULT, NOT THE ENTRY LIST. Every entry is
	// emitted in the JSON text block with JSON quoting and envelope (and was
	// emitted twice, structuredContent included, before ADR-023 — which is why
	// an 8,000-file index now fits and the fixtures grew to 12,000). Counting
	// each entry once selected 7,388 entries for
	// an 8,000-file fixture and produced roughly 650,000 characters against a
	// 200,000 limit, so the index built to fit under the cap blew through it.
	// The measure counts the one copy an index carries now, content[1], with
	// its quoting and envelope (it counted two before ADR-023). Found by review of #80.
	// ⚠ MEASURED, NOT ESTIMATED. Two earlier cuts got this wrong in the same
	// direction. Counting each entry ONCE selected 7,388 of 8,000 files and
	// produced a 650,000-character result against a 200,000 limit; counting it
	// twice still produced 210,289, because the JSON block is escaped AGAIN
	// inside the JSON-RPC envelope. An index built to fit under the cap that
	// blows through it is worse than no index — it is the spill this whole
	// answer exists to avoid. So the result is built, marshalled, and MEASURED,
	// and trimmed until the encoded thing actually fits. Found by review of #80.
	shown := entries
	var b strings.Builder
	var raw []byte
	next := ""
	for {
		next = ""
		if len(shown) < len(entries) {
			// THE LAST ENTRY SHOWN, not the first withheld. The field is
			// `after`, and grepSpecs skips everything at or before it — so
			// naming the first withheld file makes that file skip ITSELF,
			// losing exactly one entry per page boundary. Caught by paging to
			// exhaustion and comparing the union (7,999 of 8,000); a check
			// that the cursor merely names a real path passed it, which is
			// the whole reason ADR-014's Enforced-by reassembles rather than
			// inspects.
			next = shown[len(shown)-1]
		}

		b.Reset()
		fmt.Fprintf(&b, "-- INDEX: %d file(s) match, and their CONTENT would have been about %d bytes against a limit of %d.\n",
			len(entries), cw.written, cw.limit)
		b.WriteString("-- No content was served and nothing was recorded, so no write is licensed by this.\n")
		if next != "" {
			fmt.Fprintf(&b, "-- Showing the first %d of %d. Send the SAME grep again with after=%q for the next page, and repeat until next_index is absent.\n", len(shown), len(entries), next)
		}
		// The paths are NOT repeated in the prose block: they are in the JSON
		// block below, and a second copy is a second share of the cap spent
		// saying the same thing.
		b.WriteString("-- The matching files are listed in this result's index field. Send any of them back as specs WITH the same grep to read its matches, or on its own to read the file.\n")

		structured := map[string]any{
			"matches":    len(entries),
			"index":      shown,
			"next_index": next,
			// Declared by the read schema and always present, so a
			// schema-checking host sees the shape it was promised. An index
			// served nothing, so observed is empty rather than absent —
			// absent would be a different claim.
			"observed": map[string]seen.Observation{},
			"problems": problems,
		}
		var err error
		raw, err = json.Marshal(structured)
		if err != nil {
			return errorResult("could not encode the index: " + err.Error())
		}
		// ⚠ MARSHAL WHAT GOES ON THE WIRE AND MEASURE THAT. The previous cut
		// ESTIMATED — `b.Len() + 2*len(raw)` — which ignores the JSON escaping
		// of the text copy and the envelope around both. The estimate landed
		// on either side of the cap depending on the fixture: 199,998
		// characters for an 8,000-file page and 200,128 for a 12,001-file one,
		// so §51 passed because its fixture happened to come in two under.
		// A record that says it "marshals and MEASURES" has to do it.
		// Found by review of #80.
		total := encodedSize(indexResult(b.String(), raw))
		// ALWAYS KEEP AT LEAST ONE. An entry too large to fit alone would
		// otherwise yield an empty page whose cursor names nothing, so the
		// caller loops forever on no progress — and `shown[len(shown)-1]`
		// would index out of range. Only PATH_MAX keeps that unreachable
		// today, which is not a guarantee this function should rest on.
		if total <= cw.limit || len(shown) <= 1 {
			break
		}
		n := int(float64(len(shown)) * float64(cw.limit) / float64(total) * 0.95)
		if n < 1 {
			n = 1
		}
		if n >= len(shown) {
			n = len(shown) - 1
		}
		shown = shown[:n]
	}
	// The SAME assembler the loop measured, so the thing sent and the thing
	// checked against the cap cannot drift apart.
	return indexResult(b.String(), raw)
}

// indexResult assembles the tool result an index answer sends, so that the
// thing measured and the thing sent are built by one function and cannot drift.
func indexResult(report string, raw []byte) callToolResult {
	return callToolResult{
		Content: []contentBlock{
			{Type: "text", Text: report},
			{Type: "text", Text: string(raw)},
		},
		// No structuredContent: a read's answer is content[0] (ADR-023).
		//
		// AND NO isError, per ADR-024, for the same reason a page carries none:
		// an index that a host truncates is worse than a page, because a
		// shortened list of matching files names no gap for anyone to notice.
		// The report in content[0] says what this is.
	}
}

// encodedSize is the length of the result once marshalled — the quantity the
// cap is actually about, since that is what crosses the wire.
//
// A result that will not marshal is reported as unbounded rather than as zero:
// zero would read as "fits" and ship the thing that could not be encoded.
func encodedSize(res callToolResult) int {
	b, err := json.Marshal(res)
	if err != nil {
		return math.MaxInt
	}
	return len(b)
}

// nameTheAck appends ADR-031's remedy to any hunk refused for lines that were
// served on a page nobody acknowledged.
//
// It is deliberately conditional: with no pending record the advice would be
// wrong, and a refusal that suggests a fix which does not apply is worse than
// one that suggests none. The engine's message is left intact and extended,
// never replaced — it names the file and the served spans, which the caller
// still needs.
func nameTheAck(root string, res *apply.Result) {
	if res == nil || res.Failed == 0 {
		return
	}
	store, err := loadPending(root)
	if err != nil || len(store) == 0 {
		return
	}
	// ⚠ Compared by FILE IDENTITY, not by spelling. internal/apply treats a
	// symlink or a case-only variant as one file (ADR-029), so a page read as
	// real.txt and written as link.txt is one file to the ledger — and this
	// remedy, matching literal strings, used to go missing for exactly the
	// caller who most needs it. Found by the review of PR #132.
	for i := range res.Hunks {
		h := &res.Hunks[i]
		if !strings.Contains(h.Reason, "has not been read") {
			continue
		}
		// ⚠ THE REFUSED ADDRESS MUST INTERSECT A PENDING SPAN. Appending the
		// remedy whenever the FILE has anything pending tells a caller to
		// acknowledge a page that cannot license the line they asked for —
		// lines 1-1000 pending, line 2000 refused, "send ack" (seventh review
		// of PR #132). A refusal naming a fix that cannot work is the failure
		// ADR-015 exists to prevent, wearing the shape of help.
		start, end, known := addrRange(h.Addr)
		// promote() drops a pending span whose version is gone, so recommending
		// ack for one is recommending a step that cannot work (eighth review of
		// PR #132).
		live, _ := currentSHA(root, h.Path)
		covers := false
		for _, p := range store {
			if p.Path != h.Path && !anySameFile(root, h.Path, []string{p.Path}) {
				continue
			}
			if live != "" && p.SHA != live {
				continue
			}
			// An address mrw could not resolve to numbers — a pattern, or $ —
			// is treated as possibly covered. Being wrong here costs a caller
			// one unnecessary sentence; being silent costs the caller who WAS
			// working from that page the only remedy they had.
			if !known || (p.End >= start && p.Start <= end) {
				covers = true
				break
			}
		}
		if !covers {
			continue
		}
		h.Reason += ". A page of this file was served but never acknowledged, and an " +
			"unacknowledged page licenses nothing. " + AckRule
	}
}

// anySameFile reports whether path names the same file on disk as any of the
// candidates, so an alias spelling is recognised the way apply recognises it.
func anySameFile(root, path string, candidates []string) bool {
	want, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return false
	}
	for _, c := range candidates {
		if c == path {
			return true
		}
		got, err := os.Stat(filepath.Join(root, filepath.FromSlash(c)))
		if err == nil && os.SameFile(want, got) {
			return true
		}
	}
	return false
}

// addrRange reads the line span out of an address as the caller wrote it, so a
// refusal's remedy can be matched against what is actually pending. A pattern
// or a $ has no numbers to read and reports known=false.
func addrRange(addr string) (start, end int, known bool) {
	lo, hi, ok := strings.Cut(addr, "-")
	a, err := strconv.Atoi(strings.TrimSpace(lo))
	if err != nil {
		return 0, 0, false
	}
	if !ok {
		return a, a, true
	}
	b, err := strconv.Atoi(strings.TrimSpace(hi))
	if err != nil {
		// `3-$` is a valid open-ended address, and calling it the single line 3
		// makes a pending LATER span look non-intersecting — the caller then
		// loses the remedy that would have worked. Unknown takes the permissive
		// path, which is what the doc comment already promised.
		return 0, 0, false
	}
	return a, b, true
}
