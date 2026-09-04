package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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

	var specs []read.Spec
	var walked bool
	if a.Grep != "" {
		var err error
		specs, err = grepSpecs(root, a.Specs, a.Grep, a.Exclude)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		walked = true
		if len(specs) == 0 {
			// Not an error: "nothing matched" is a real answer, and the
			// caller asked a question rather than named a file that is
			// missing. Saying so plainly beats an empty successful read.
			return result(map[string]any{
				"observed": map[string]seen.Observation{},
				"problems": 0,
				"matches":  0,
			}, fmt.Sprintf("no file under the root matches /%s/.", a.Grep), false)
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
			return matchIndex(specs, cw), nil
		}
		if page, ok := firstPage(root, a.Specs, cw); ok {
			return page, nil
		}
		return errorResult(overflowMessage(a.Specs, cw)), nil
	}

	// Reading is how mrw learns what a file holds; recording that is what lets
	// a later write know whether its picture is still current.
	if err := seen.Record(root, observed); err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: "recording the ledger: " + err.Error()}
	}

	// The two tools' structuredContent shapes differ and are now DECLARED:
	// mrw_write returns apply.Result, whose json tags make it snake_case; this
	// returns seen.Observation, which carries no tags, so its keys are the Go
	// field names. Both are generated into the outputSchema each tool
	// advertises, so the day seen.Observation gains tags the schema follows it
	// and the conformance test catches any response that does not.
	return result(map[string]any{
		"observed": observed,
		"problems": problems,
	}, cw.buf.String(), problems > 0)
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
		StructuredContent: json.RawMessage(b),
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
func grepSpecs(root string, paths []string, pattern string, exclude []string) ([]read.Spec, error) {
	for _, p := range paths {
		sp, err := read.ParseSpec(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", p, err)
		}
		if len(sp.Ranges) > 0 {
			// cmd/mrw/main.go:499, word for word: the caller has said both
			// "look here" and "look for this", and mrw will not pick one.
			return nil, fmt.Errorf("%s: a range and grep are two answers to one question", p)
		}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("grep %q: %v", pattern, err)
	}
	specs, _, err := read.Walk(root, paths, read.WalkOptions{Pattern: re, Exclude: exclude})
	if err != nil {
		return nil, err
	}
	return specs, nil
}

// matchIndex is the answer to a grep whose CONTENT will not fit: the addresses,
// with no content at all.
//
// One entry per matching file, in the walk's own form — read.Walk returns a
// spec per file addressed by the pattern (internal/read/walk.go:219), so this
// hands back what the walk produced rather than inventing a shape. Each entry
// is a valid spec, so the caller's next call is this call's output.
//
// NOTHING IS RECORDED. The index served no lines, so it licenses no write; an
// index that licensed edits to files the caller never saw would be ADR-002's
// guarantee spent on a convenience.
func matchIndex(specs []read.Spec, cw *capped) callToolResult {
	entries := make([]string, 0, len(specs))
	for _, sp := range specs {
		// Each entry is the file plus the address the walk gave it, which is
		// the pattern. Guarded because a spec with no range would otherwise
		// index the file as a whole read and quietly promise far more than
		// the caller asked for.
		if len(sp.Ranges) == 0 {
			entries = append(entries, sp.Raw)
			continue
		}
		entries = append(entries, sp.Raw+":"+sp.Ranges[0].Text)
	}

	// The index itself can overflow, and refusing here would be the same dead
	// end one level down — so it pages BY FILE. A list's natural continuation
	// is "resume after this entry", the way a file's is "resume at this line".
	//
	// Grown forwards rather than halved: halving discards entries that would
	// have fitted, and an index that is needlessly short costs the caller a
	// round trip for nothing.
	budget := cw.limit - indexOverhead
	shown, used := entries, 0
	for i, e := range entries {
		if used+len(e)+1 > budget {
			shown = entries[:i]
			break
		}
		used += len(e) + 1
	}
	next := ""
	if len(shown) < len(entries) {
		next = specs[len(shown)].Raw
	}

	var b strings.Builder
	fmt.Fprintf(&b, "-- INDEX: %d file(s) match, and their CONTENT would have been about %d bytes against a limit of %d.\n",
		len(entries), cw.written, cw.limit)
	b.WriteString("-- No content was served and nothing was recorded, so no write is licensed by this.\n")
	if next != "" {
		fmt.Fprintf(&b, "-- Showing the first %d. Send the same grep with paths starting at %q to continue.\n", len(shown), next)
	}
	b.WriteString("-- Send any of these back as specs to read it:\n")
	for _, e := range shown {
		b.WriteString(e)
		b.WriteByte('\n')
	}

	structured := map[string]any{
		"matches":    len(entries),
		"index":      shown,
		"next_index": next,
	}
	raw, err := json.Marshal(structured)
	if err != nil {
		return errorResult("could not encode the index: " + err.Error())
	}
	return callToolResult{
		Content: []contentBlock{
			{Type: "text", Text: b.String()},
			{Type: "text", Text: string(raw)},
		},
		StructuredContent: json.RawMessage(raw),
		// An index is not the content that was asked for, so it stays an
		// error for the same reason a page does: the caller must be able to
		// see it did not get what it requested.
		IsError: true,
	}
}

// indexOverhead reserves room for the prose around the entries, so the entry
// budget is not spent on the sentence explaining the entries.
const indexOverhead = 512

func indexLen(entries []string) int {
	n := 0
	for _, e := range entries {
		n += len(e) + 1
	}
	return n
}
