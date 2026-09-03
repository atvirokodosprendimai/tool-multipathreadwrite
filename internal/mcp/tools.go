package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// gate serializes tool calls. The ledger is a whole-file rewrite, so parallel
// writers lose entries — the README's "one call at a time" limitation. One
// server is one writer, and this is where that becomes true: calls made THROUGH
// the server no longer race. A CLI process running beside it still does, which
// is still the CLI limitation and not something a mutex here can fix.
//
// It is package-level rather than per-Serve because two Serve calls in one
// process share the same ledger file, and it is the FILE that is being
// protected, not the session.
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

func text(s string) []contentBlock { return []contentBlock{{Type: "text", Text: s}} }

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
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "arguments: " + err.Error()}
	}
	if len(a.Specs) == 0 {
		return callToolResult{}, &rpcError{Code: codeInvalidParams, Message: "mrw_read needs at least one spec"}
	}

	specs := make([]read.Spec, 0, len(a.Specs))
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

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	observed, problems := read.Run(w, root, specs, read.Options{Numbers: true})
	w.Flush()

	// Reading is how mrw learns what a file holds; recording that is what lets
	// a later write know whether its picture is still current.
	if err := seen.Record(root, observed); err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: "recording the ledger: " + err.Error()}
	}

	return callToolResult{
		Content: text(buf.String()),
		StructuredContent: map[string]any{
			"observed": observed,
			"problems": problems,
		},
		IsError: problems > 0,
	}, nil
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

	return callToolResult{
		Content:           text(report.String()),
		StructuredContent: res,
		IsError:           applyErr != nil || res.Failed > 0,
	}, nil
}

// errorResult reports a failure the CALLER caused, inside a normal tool result.
// A tool error is not a protocol error: the request was well-formed and the
// answer is "no", which a host shows to its user rather than treating as a
// transport fault.
func errorResult(msg string) callToolResult {
	return callToolResult{Content: text(msg), IsError: true}
}
