// Package mcp serves mrw over the Model Context Protocol on stdio, so an agent
// that speaks MCP reaches the same engine the CLI reaches without shell access.
//
// The transport is deliberately hand-rolled. MCP over stdio is JSON-RPC 2.0
// with a handful of methods, and ADR-010 records the decision not to take an
// SDK dependency for a subset this small — mrw has exactly one dependency and
// keeping it that way is a property worth more than the code saved.
//
// Framing follows the specification's stdio transport: one JSON-RPC message per
// line, delimited by newlines, with no embedded newline inside a message. It is
// NOT the Language Server Protocol's Content-Length header, which is the shape
// this package was first written to by mistake and which no MCP host sends.
package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// legacyVersions are the handshake-era revisions mrw speaks, latest first
// (ADR-067). initialize answers the requested one when it is here, and the
// first otherwise — the lifecycle's "the latest version supported by the
// server". Claude Code 2.1.281 asks for 2025-11-25 (wire capture, 2026-09-24).
var legacyVersions = []string{"2025-11-25", "2025-06-18"}

// negotiate picks the version initialize answers with.
func negotiate(requested string) string {
	for _, v := range legacyVersions {
		if v == requested {
			return v
		}
	}
	return legacyVersions[0]
}

// The 2026-07-28 era (ADR-067 Decision 4). A request carrying
// _meta[metaVersion] is served per request; one without it is served exactly
// as before. mrw keeps no session state, so every version it lists is
// servable on any request — which is what keeps one supported list true on
// -32022, on server/discover and on a retry.
const (
	modernVersion          = "2026-07-28"
	metaVersion            = "io.modelcontextprotocol/protocolVersion"
	metaCapabilities       = "io.modelcontextprotocol/clientCapabilities"
	codeUnsupportedVersion = -32022
	// cacheTTLMs: the tool list is fixed for the binary's life and the same
	// for every caller, so an hour of freshness is honest and cacheScope is
	// "public". A new binary is a new process.
	cacheTTLMs = 3_600_000
)

// supportedVersions is every revision mrw speaks, latest first.
func supportedVersions() []string { return append([]string{modernVersion}, legacyVersions...) }

// era is what one request's _meta asks for. modern means the result is
// decorated with resultType, serverInfo and, on a list, caching hints.
type era struct{ modern bool }

// requestEra reads params._meta once. hasVersion reports whether the request
// named a version at all; a non-nil error refuses the request.
func requestEra(params json.RawMessage) (e era, hasVersion bool, refusal *rpcError) {
	var p struct {
		Meta map[string]json.RawMessage `json:"_meta"`
	}
	if json.Unmarshal(params, &p) != nil {
		return era{}, false, nil
	}
	raw, ok := p.Meta[metaVersion]
	if !ok {
		return era{}, false, nil
	}
	var v string
	if json.Unmarshal(raw, &v) != nil {
		v = string(raw)
	}
	if !slices.Contains(supportedVersions(), v) {
		return era{}, true, &rpcError{Code: codeUnsupportedVersion, Message: "Unsupported protocol version",
			Data: map[string]any{"supported": supportedVersions(), "requested": v}}
	}
	if v != modernVersion {
		return era{}, true, nil
	}
	if caps := bytes.TrimSpace(p.Meta[metaCapabilities]); len(caps) == 0 || caps[0] != '{' {
		return era{}, true, &rpcError{Code: codeInvalidParams,
			Message: "invalid params: a " + modernVersion + " request must carry _meta " + metaCapabilities}
	}
	return era{modern: true}, true, nil
}

// modernResultMeta is the _meta every modern result carries.
func modernResultMeta() map[string]any {
	return map[string]any{"io.modelcontextprotocol/serverInfo": serverInfo()}
}

// decorate adds the modern result fields to m when the request was modern.
func (e era) decorate(m map[string]any) map[string]any {
	if e.modern {
		m["resultType"] = "complete"
		m["_meta"] = modernResultMeta()
	}
	return m
}

// discoverResult answers server/discover. It is modern-shaped whatever
// version the request named, because the method exists only in that era.
func discoverResult() map[string]any {
	return map[string]any{
		"resultType":        "complete",
		"supportedVersions": supportedVersions(),
		"capabilities":      map[string]any{"tools": map[string]any{}},
		"_meta":             modernResultMeta(),
		"instructions":      instructionsText(),
		"ttlMs":             cacheTTLMs,
		"cacheScope":        "public",
	}
}

// JSON-RPC 2.0 error codes, named so a reader does not have to look them up.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternal       = -32603
)

// request is one incoming message. ID is a RawMessage rather than a concrete
// type because its ABSENCE is meaningful: a message without an id is a
// notification, and a notification must never be answered.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// response is one outgoing message. Result and Error are both RawMessage with
// omitempty so exactly one of them is ever encoded — a response carrying both
// is malformed, and a struct that makes that representable invites it.
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError carries a code AND a message. A code alone is not a valid JSON-RPC
// error object, and a host has nothing to show a user without the message.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Data carries the error's structured detail when the specification
	// defines one — UnsupportedProtocolVersion's supported/requested (ADR-067).
	Data any `json:"data,omitempty"`
}

// tool is one entry of tools/list. A host reads this list rather than the
// source, so a tool without an inputSchema is a tool it cannot call.
type tool struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
	// OutputSchema is generated from the type the handler returns, never
	// written beside it — see SchemaOf for the measured reason.
	OutputSchema any `json:"outputSchema,omitempty"`
	// Annotations tell a host what the tool DOES, and a host shows them to a
	// user before asking them to approve a call. An annotation that flatters
	// the tool is a lie the user acts on, so they are asserted by observation
	// in TestTheAnnotationsMatchWhatTheToolDoes rather than by reading them.
	Annotations any `json:"annotations,omitempty"`
	// Meta carries host-specific hints. `anthropic/maxResultSizeChars` tells
	// Claude Code how large a result may get; it is the SAME constant the read
	// path enforces, so the advertised limit and the enforced one cannot drift.
	Meta any `json:"_meta,omitempty"`
}

// Serve reads newline-delimited JSON-RPC from in and writes it to out until in
// is exhausted, answering for the checkout at root.
//
// It returns nil at a clean EOF: a host closing the pipe is how a session ends,
// not a failure. root is accepted here and unused until the tool handlers land,
// because the transport must be startable in a directory with no ledger, no
// plan and no git — a server that cannot say hello without touching the tree
// has coupled the wire to the filesystem.
func Serve(in io.Reader, out io.Writer, root string) error {
	// root binds the tool handlers to this checkout: the ledger they read and
	// write is the one the CLI uses for the same tree, which is what makes a
	// file read over MCP editable from a shell and the reverse.

	// A bufio.Reader rather than a bufio.Scanner: Scanner caps a token at 64 KB
	// by default, and a write plan travelling as a JSON string inside one
	// tools/call is exactly the message that exceeds it. A silent truncation of
	// an edit plan is the worst failure this package could have.
	r := bufio.NewReader(in)
	w := bufio.NewWriter(out)
	defer w.Flush()

	for {
		line, err := r.ReadString('\n')
		if line != "" {
			if resp, answer := handle(line, root); answer {
				if err := write(w, resp); err != nil {
					return err
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("mcp: reading stdin: %w", err)
		}
	}
}

// write encodes one response and terminates it with a newline. Nothing else is
// ever written to out: the specification requires that a server write nothing
// to stdout that is not a valid MCP message, and this binary prints to stdout
// everywhere else, so that rule is a live constraint rather than boilerplate.
// Diagnostics belong on stderr.
func write(w *bufio.Writer, resp response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		// Marshalling our own response failed, so there is nothing valid to
		// say on stdout. Failing the session is better than emitting garbage
		// a host would have to guess about.
		return fmt.Errorf("mcp: encoding response: %w", err)
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("mcp: writing response: %w", err)
	}
	// Flush per message: a host is waiting on this line before it sends the
	// next one, so a buffered response is a deadlock.
	return w.Flush()
}

// handle turns one input line into at most one response. The bool reports
// whether anything should be written at all, which is how notifications stay
// silent — the distinction a plain "return a response" signature cannot make.
func handle(line string, serveRoot string) (response, bool) {
	line = strings.TrimRight(line, "\r\n")
	if strings.TrimSpace(line) == "" {
		// A blank line carries no message. Answering it with an error would
		// put a complaint on the wire about something nobody sent.
		return response{}, false
	}

	var req request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		// The id is unknowable in a message that did not parse, so it is null
		// per JSON-RPC. This is answered rather than dropped: a host that sees
		// a closed pipe cannot tell a crash from a refusal.
		// A JSON array IS valid JSON and is not a message: the 2025-06-18
		// revision dropped batching, so it is refused — but as an invalid
		// REQUEST, because calling it a parse error misdescribes the input.
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			return errorResponse(json.RawMessage("null"), codeInvalidRequest,
				"invalid request: a JSON array is not a message — this protocol revision does not batch"), true
		}
		return errorResponse(json.RawMessage("null"), codeParse, "parse error: the line is not valid JSON"), true
	}

	// No id means a notification. It gets no response even when it is wrong,
	// which includes notifications/initialized — answering that one is a
	// protocol violation some hosts treat as fatal.
	if len(req.ID) == 0 {
		return response{}, false
	}

	// MCP: a request id MUST NOT be null. It is answered as an invalid request,
	// with the null id it arrived with, and never dispatched (ADR-067).
	if string(req.ID) == "null" {
		return errorResponse(req.ID, codeInvalidRequest, "invalid request: an id must not be null"), true
	}

	if req.JSONRPC != "2.0" {
		return errorResponse(req.ID, codeInvalidRequest, `invalid request: jsonrpc must be "2.0"`), true
	}

	// initialize always selects the legacy era, whatever _meta it carries
	// (ADR-067: precedence 2). Every other method takes its era from _meta.
	if req.Method == "initialize" {
		return resultResponse(req.ID, initializeResult(req.Params))
	}
	e, hasVersion, refusal := requestEra(req.Params)
	if refusal != nil {
		return response{JSONRPC: "2.0", ID: req.ID, Error: refusal}, true
	}

	switch req.Method {
	case "server/discover":
		// Without a version it is malformed, which a dual-era client reads
		// as "legacy" and answers with initialize.
		if !hasVersion {
			return errorResponse(req.ID, codeInvalidParams, "invalid params: server/discover must carry _meta "+metaVersion), true
		}
		return resultResponse(req.ID, discoverResult())
	case "tools/call":
		// The handlers are adapters over the same engine functions cmd/mrw
		// calls; the root is what binds them to this checkout. The era goes
		// in, because a modern result is decorated INSIDE the call, where
		// every ceiling measurement can reserve room for it.
		res, rpcErr := callTool(serveRoot, req.Params, e.modern)
		if rpcErr != nil {
			return response{JSONRPC: "2.0", ID: req.ID, Error: rpcErr}, true
		}
		return resultResponse(req.ID, res)
	case "ping":
		// The spec's ping utility: "The receiver MUST respond promptly with an
		// empty response". Hosts send these on a timer to check connection
		// health, and a server that errors every health check is one a host is
		// entitled to drop — so this is a base-protocol obligation, not a
		// capability the ADR chose not to implement. 2026-07-28 removed it; a
		// modern client does not send it, and answering costs nothing.
		// https://modelcontextprotocol.io/specification/2025-06-18/basic/utilities/ping
		return resultResponse(req.ID, e.decorate(map[string]any{}))
	case "tools/list":
		list := map[string]any{"tools": tools()}
		if e.modern {
			list["ttlMs"] = cacheTTLMs
			list["cacheScope"] = "public"
		}
		return resultResponse(req.ID, e.decorate(list))
	default:
		// tools/call lands here until T2 implements the handlers. Declaring the
		// schemas before implementing them keeps tools/list honest from the
		// first commit; answering the call with method-not-found keeps the
		// server honest in the meantime.
		return errorResponse(req.ID, codeMethodNotFound, "method not found: "+req.Method), true
	}
}

// initializeResult is the whole lifecycle response, not just a version. A reply
// carrying only protocolVersion parses as JSON and still fails negotiation,
// because a host that sees no tools capability never calls tools/list.
func initializeResult(params json.RawMessage) map[string]any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	// A missing or malformed params is answered with the latest version, not
	// refused: the handshake is how a host learns what this server speaks.
	_ = json.Unmarshal(params, &p)
	return map[string]any{
		"protocolVersion": negotiate(p.ProtocolVersion),
		"serverInfo":      serverInfo(),
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		// The lifecycle spec's field for "how to drive this server". It is the
		// only documentation an MCP-only caller has: it is in a checkout it did
		// not clone from here, so the format has to travel on the wire.
		"instructions": instructionsText(),
	}
}

// serverInfo says what mrw is: the one Implementation every era reports.
func serverInfo() map[string]any {
	return map[string]any{
		"name":        "mrw",
		"title":       "mrw",
		"version":     Version,
		"description": "Reads many file ranges and applies many edits in one call, with a verdict for every edit.",
	}
}

// Version is what the server reports as its own version during initialize. The
// command sets it from the binary's stamp so the wire and `mrw --version` agree.
var Version = "dev"

// tools declares the surface. The handlers land in T2; the schemas are declared
// here so a host reading tools/list is never told about a tool that does not
// exist, nor left ignorant of one that does.
func tools() []tool {
	return []tool{
		{
			Name:  "mrw_read",
			Title: "Read file ranges",
			// No outputSchema, on purpose: this tool returns no
			// structuredContent (ADR-023, readResult), and the specification
			// makes a declared schema a promise to return one.
			Annotations: map[string]any{
				"title":           "Read file ranges",
				"readOnlyHint":    true,
				"destructiveHint": false,
				"idempotentHint":  true,
				"openWorldHint":   false,
			},
			Meta: map[string]any{"anthropic/maxResultSizeChars": MaxResultChars},
			Description: triggerRule +
				" One mrw_read serves every site, and each served line is recorded so " +
				"mrw_write may later edit it — served lines record nothing until you acknowledge " +
				"them (see ack). Specs use mrw's own syntax: path, path:10-20, path:A,+N for the line A " +
				"plus the N lines after it, path:/regexp/ so the read " +
				"finds its own site, or path:$ for the last line. A read too large for one answer " +
				"comes back as a PAGE, not a failure: the lines that fit, a -- PARTIAL: line, and a " +
				"next_read spec for the rest. Repeat until next_read is absent — its absence is " +
				"how you know you have the whole file, and you may only edit lines a page served. " +
				"The served text is the first text block; the receipt (observed spans, problems, " +
				"next_read) is the second, as JSON. " +
				"Set `grep` to a regexp to FIND files you cannot name: it walks the paths " +
				"(or the whole root) and serves every match, and returns an index of matching " +
				"files when the matches are too large to serve. " +
				"With a shell and mrw on PATH, prefer the CLI `mrw read` — it also has " +
				"--files-from, and `mrw --root DIR read` for any checkout (--root BEFORE the " +
				"subcommand; after `read`, -C is the context flag). Prefer THIS tool with no " +
				"shell.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"specs": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "One or more range specs, e.g. internal/x/y.go:40-60. With `grep` set these are DIRECTORIES OR FILES TO SEARCH instead, and may be omitted to search the whole root.",
						// A worked value, not a sentence about one. The three
						// address forms in one list, because a caller who sees
						// only a line range will make a second call to find the
						// line it should have asked for by regexp.
						"examples": []any{exampleReadSpecs},
					},
					"grep": map[string]any{
						"type":        "string",
						"description": "A regexp. Walks the paths in `specs` (or the whole root when they are omitted) and serves every matching range, so you can find files you cannot name. If the matches are too large to serve you get an INDEX instead — one spec per matching file, no content — which you send back as `specs` to read the ones you want. A path in `specs` may not carry a range when this is set: a range and a grep are two answers to one question.",
						"examples":    []any{"func Serve\\(", "TODO|FIXME"},
					},
					"ast_grep": map[string]any{
						"type":        "string",
						"description": "A structural pattern for the ast-grep CLI on PATH. Walks like grep, maps hits to line ranges, and serves them through the same read. Missing ast-grep names ast-grep. A hit in a file whose lines end in \\r alone is reported, not served; read that file directly. Do not set grep at the same time: they are two sources of specs.",
						"examples":    []any{"fmt.Println($A)"},
					},
					"exclude": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Globs to skip, matched against BOTH the root-relative path and the basename. Only meaningful with `grep` or `ast_grep`. Note that `*` does not cross a separator, which is why the basename is matched too: \"*_test.go\" against the full path alone matches no test file anywhere below the root.",
						"examples":    []any{[]any{"*_test.go", "vendor"}},
					},
					"ack": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "The checkpoint ids from a previous read that served lines. A served answer brackets each run of lines with `-- ck <id> open lines A-B (N lines follow)` and `-- ck <id> close`, and licenses NO write until you send its ids back here. " + AckRule + " Omit an id and its lines stay unwritable, which is the point: a host can cut a result before you see it, and mrw cannot tell.",
						"examples":    []any{[]any{"3f8a1c4d90b27e56"}},
					},
					"after": map[string]any{
						"type":        "string",
						"description": "Resume a paged index: send the SAME `grep` again with this set to the `next_index` the previous call returned, and matching files at or before it are skipped. Files are walked in path order, so this is a position you can read rather than an opaque cursor. Repeat until `next_index` is absent.",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:         "mrw_write",
			Title:        "Apply an edit plan",
			OutputSchema: writeSchema(),
			Annotations: map[string]any{
				"title": "Apply an edit plan",
				// It writes files. A host shows this before asking a user to
				// approve the call, so understating it is a lie they act on.
				"readOnlyHint":    false,
				"destructiveHint": true,
				// A plan addresses the ORIGINAL file (ADR-001), so applying it
				// twice does not apply it twice — the second run refuses.
				"idempotentHint": false,
				"openWorldHint":  false,
			},
			Meta: map[string]any{"anthropic/maxResultSizeChars": MaxResultChars},
			Description: triggerRule +
				" Every edit travels in ONE plan and every hunk gets a verdict, so " +
				"a replacement that matched nothing is reported rather than silently skipped. All " +
				"or nothing: if any hunk fails validation, nothing is written. Every address resolves against " +
				"the ORIGINAL file, so several hunks in one file need no offset arithmetic. mrw " +
				"will not edit a line it has not served you — read it with mrw_read first. If you " +
				"can run shell commands, prefer the CLI `mrw write` — it also has --check, which " +
				"runs the project's tests scoped to what it just wrote. Prefer THIS tool with no " +
				"shell; it needs no --json because its answer is already structured.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ack": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "The checkpoint ids from the read this plan was written against. An unacknowledged serve licenses nothing, so a hunk addressing lines you have not acknowledged is refused. " + AckRule,
						"examples":    []any{[]any{"3f8a1c4d90b27e56"}},
					},
					"dry_run": map[string]any{
						"type": "boolean",
						"description": "Validate and report without writing anything. The receipt is the " +
							"same shape, with dry_run true and no file written.",
					},
					"echo_pad": map[string]any{
						"type":    "integer",
						"default": 0,
						"description": "Print N lines after an applied body so a surviving closer is visible. " +
							"Default 0. Not a checker: a closer in the pad does not fail the hunk. Negative is refused.",
					},
					"strict_balance": map[string]any{
						"type":    "boolean",
						"default": false,
						"description": "Refuse a single-line replace on a non-prose path whose line's {} () [] do not " +
							"balance and whose body does not match them — the wrap-tail shape. A refusal is a failed " +
							"hunk: siblings skip, nothing is written. Off by default; the same hunk then applies with " +
							"a balance row. Braces inside string literals count, so drop it for that plan.",
					},
					"format": map[string]any{
						"type":        "string",
						"enum":        []string{"plan", "apply_patch", "search_replace"},
						"default":     "plan",
						"description": "plan (default) is the native @@ document. apply_patch compiles a Codex *** Begin Patch document. search_replace compiles an Aider SEARCH/REPLACE document. Both use the same compile→Parse→Apply path as write --format. git is refused.",
					},
					"plan": map[string]any{
						"type": "string",
						"description": "The plan document. Each hunk is a header line " +
							"`@@ <path> <address> <op> [guards]` followed by its body lines. " +
							"Ops: replace, insert-after, insert-before, delete, create, unlink, rename. " +
							"unlink takes address `-` and no body. Among line-range ops, ONLY delete may carry no body: an empty file is `@@ new.txt 0 " +
							"create body=0`, and a bare create with nothing under it is " +
							"refused, because a lost body reads exactly like one never " +
							"written. body=@path loads the body from a root-relative file. An address " +
							"is a line number, an N-M range, A,+N (the line A plus the N lines " +
							"after it, refused if it runs past the last line), $ for the last " +
							"line, or a pattern — " +
							"/regexp/ or /from/,/to/. The START pattern must match EXACTLY ONE " +
							"line; none or several fails that hunk, naming the lines it matched. " +
							"The END is a DELIMITER, not a site: the first match at or after the " +
							"start, so it may match many times. Addresses " +
							"resolve against the ORIGINAL file. Guards, checked on " +
							"every op: sha=<hex>, lines=<n>, anchor=\"<text>\". " +
							"sha= and lines= are always optional. " +
							"anchor= is REQUIRED on a replace addressing more than one " +
							"line, and may be omitted elsewhere. An anchor is worth most " +
							"taken from the NNN| content a read printed. " +
							"Pass the ck ids from that read as ack; a serve licenses nothing until you do. " +
							"A body line beginning with @@ needs body=<n> raw=true.",
						// The format is bespoke and no model has it in training
						// data (ADR-009's premise), so one plan that really
						// applies is worth more than the paragraph above it.
						// TestEveryEmbeddedExamplePlanReallyApplies dry-runs it.
						"examples": []string{examplePlan},
					},
				},
				"required": []string{"plan"},
			},
		},
	}
}

// resultResponse marshals a successful result. A marshalling failure is
// reported to the caller as an internal error rather than silently dropped.
func resultResponse(id json.RawMessage, result any) (response, bool) {
	b, err := json.Marshal(result)
	if err != nil {
		return errorResponse(id, codeInternal, "could not encode result: "+err.Error()), true
	}
	return response{JSONRPC: "2.0", ID: id, Result: b}, true
}

// errorResponse builds an error reply carrying both a code and a message.
func errorResponse(id json.RawMessage, code int, msg string) response {
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}
