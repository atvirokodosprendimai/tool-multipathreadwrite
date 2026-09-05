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

	switch p.Name {
	case "mrw_read":
		return readTool(root, p.Arguments)
	case "mrw_write":
		return writeTool(root, p.Arguments)
	default:
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "unknown tool: " + p.Name}
	}
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
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "arguments: " + err.Error()}
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
		// It stays isError. That is the whole difference between paging and
		// truncation: the caller must be able to see it received a part.
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
	// carries a sha and spans per file and travels twice more, in
	// structuredContent and in its serialized copy. A grep resuming onto 2,514
	// small files came back at 794,582 characters — four times the cap this
	// server declares in _meta, and past the ceiling the host truncates at. So
	// a walked read that will not fit ENCODED degrades to the index, which is
	// the answer that does fit and is resumable. Found by review of #80.
	if walked {
		if res, over := servedOrIndex(specs, problems, cw, observed, report); over {
			return res, nil
		}
	}

	// Reading is how mrw learns what a file holds; recording that is what lets
	// a later write know whether its picture is still current.
	if err := seen.Record(root, observed); err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: "recording the ledger: " + err.Error()}
	}

	// The receipt — seen.Observation, no json tags, so its keys are the Go
	// field names — travels in content[1] and NOT in structuredContent
	// (ADR-023; see readResult). readSchema() still describes it for a reader
	// of the code, but tools/list no longer declares it: a schema declared is
	// a structuredContent promised, and none is sent.
	return readResult(map[string]any{
		"observed": observed,
		"problems": problems,
	}, report, problems > 0)
}

// writeTool applies a plan through apply.Apply and returns the same Result the
// --json receipt carries. Every step here mirrors `mrw write`: parse, resolve
// working-set pointers, load the ledger, apply, record what was written, and
// count the outcome for ADR-009's tally.
func writeTool(root string, args json.RawMessage) (callToolResult, *rpcError) {
	var a struct {
		Plan   string `json:"plan"`
		DryRun bool   `json:"dry_run"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "arguments: " + err.Error()}
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
			StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat,
			Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor,
			SrcLine: h.SrcLine, Index: h.Index,
		})
	}

	ledger, err := seen.Load(root)
	if err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: err.Error()}
	}
	res, applyErr := apply.Apply(root, in, apply.Options{DryRun: a.DryRun, Seen: ledger})
	// ADR-001 rule 3: the receipt is filled even when the filesystem failed, so
	// it is rendered on whichever path we are on rather than discarded.

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

	var report bytes.Buffer
	for _, h := range res.Hunks {
		fmt.Fprintf(&report, "%s %s %s %s\n", h.Status, h.Path, h.Addr, h.Reason)
	}
	fmt.Fprintf(&report, "%d hunk(s), %d file(s), %d failed\n", len(res.Hunks), len(res.Files), res.Failed)
	if applyErr != nil {
		fmt.Fprintf(&report, "error: %v\n", applyErr)
	}

	return result(res, report.String(), applyErr != nil || res.Failed > 0)
}

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

	// The page WAS shown, so it is recorded — and only the span it served, which
	// is what keeps a page from licensing lines the caller never saw. seen.Record
	// merges spans for the same sha, so page two adds to page one rather than
	// replacing it (ADR-002, ADR-014 Decision 3).
	if err := seen.Record(root, observed); err != nil {
		return callToolResult{}, false
	}
	next := fmt.Sprintf("%s:%d-", path, end+1)
	report := fmt.Sprintf("%s\n\n-- PARTIAL: lines %d-%d of %d. %d line(s) remain.\n"+
		"-- Send specs [%q] to continue, or a narrower range of your own.\n"+
		"-- Stopping here means you have part of this file, not the file.",
		b.String(), start, end, total, total-end, next)
	return pagedResult(report, observed, problems, next), true
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
	if len(specs) == 1 && !strings.Contains(specs[0], ":") {
		fmt.Fprintf(&b, "Ask for a range instead — for example %s:1-%d.",
			specs[0], suggestLines(cw.buf.Len(), countLines(&cw.buf)))
	} else {
		fmt.Fprintf(&b, "Ask for narrower ranges — around %d lines per file at this file's line length — or name fewer files in one call.",
			suggestLines(cw.buf.Len(), countLines(&cw.buf)))
	}
	return b.String()
}

// pagedResult builds a first-page answer: the page, the receipt, and the spec
// that asks for the rest.
//
// It duplicates little of `result` and deliberately does not reuse it: `result`
// takes isErr as a parameter and a page is ALWAYS an error, and the structured
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
		// ALWAYS true. A page that reads as a complete answer is truncation,
		// and the caller's ability to see it received a part is the only thing
		// separating this from what ADR-011 refused.
		IsError: true,
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
	// emitted TWICE — once in the JSON text block and once in structuredContent
	// — plus JSON quoting. Counting each entry once selected 7,388 entries for
	// an 8,000-file fixture and produced roughly 650,000 characters against a
	// 200,000 limit, so the index built to fit under the cap blew through it.
	// perEntry counts both copies and the quoting. Found by review of #80.
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
		// block below and in structuredContent, and a third copy is a third of
		// the cap spent saying the same thing.
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
		// An index is not the content that was asked for, so it stays an
		// error for the same reason a page does: the caller must be able to
		// see it did not get what it requested.
		IsError: true,
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

// servedOrIndex decides whether a WALKED read may be served as content.
//
// The capped writer bounds the report text and nothing else; `observed` carries
// a sha and a span list per file and is emitted twice more. For a grep over
// many small documents — this record's ordinary case — that is the difference
// between 178,494 characters of report and a 794,582-character result. When the
// encoded answer will not fit, the index is returned instead: it is the answer
// that does fit, it is resumable, and it licenses nothing, which is the honest
// trade for content that cannot be delivered.
func servedOrIndex(specs []read.Spec, problems int, cw *capped, observed map[string]seen.Observation, report string) (callToolResult, bool) {
	probe, rpcErr := readResult(map[string]any{"observed": observed, "problems": problems}, report, problems > 0)
	if rpcErr != nil {
		// Undecidable, so not degraded: the caller path will report the same
		// encoding failure with its own message.
		return callToolResult{}, false
	}
	if encodedSize(probe) <= cw.limit {
		return callToolResult{}, false
	}
	return matchIndex(specs, problems, cw), true
}
