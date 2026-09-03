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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// protocolVersion is the specification revision this server implements and
// reports during initialize.
const protocolVersion = "2025-06-18"

// JSON-RPC 2.0 error codes, named so a reader does not have to look them up.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
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
}

// tool is one entry of tools/list. A host reads this list rather than the
// source, so a tool without an inputSchema is a tool it cannot call.
type tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
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
	_ = root // T2's handlers are what will need it; see ADR-010.

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
			if resp, answer := handle(line); answer {
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
func handle(line string) (response, bool) {
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
		return errorResponse(json.RawMessage("null"), codeParse, "parse error: the line is not valid JSON"), true
	}

	// No id means a notification. It gets no response even when it is wrong,
	// which includes notifications/initialized — answering that one is a
	// protocol violation some hosts treat as fatal.
	if len(req.ID) == 0 {
		return response{}, false
	}

	if req.JSONRPC != "2.0" {
		return errorResponse(req.ID, codeInvalidRequest, `invalid request: jsonrpc must be "2.0"`), true
	}

	switch req.Method {
	case "initialize":
		return resultResponse(req.ID, initializeResult())
	case "tools/list":
		return resultResponse(req.ID, map[string]any{"tools": tools()})
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
func initializeResult() map[string]any {
	return map[string]any{
		"protocolVersion": protocolVersion,
		"serverInfo": map[string]any{
			"name":    "mrw",
			"version": Version,
		},
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
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
			Name: "mrw_read",
			Description: "Read many ranges across many files in one call, recording each as seen " +
				"so it may later be edited. Specs use mrw's own syntax: path, path:10-20, " +
				"path:/regexp/ or path:$ for the last line.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"specs": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "One or more range specs, e.g. internal/x/y.go:40-60",
					},
				},
				"required": []string{"specs"},
			},
		},
		{
			Name: "mrw_write",
			Description: "Apply an edit plan across one or more files, all or nothing, returning a " +
				"verdict for every hunk. Every address resolves against the ORIGINAL file, so " +
				"several hunks in one file need no offset arithmetic.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"plan": map[string]any{
						"type":        "string",
						"description": "The plan text: @@ <path> <addr> <op> [guards] followed by body lines.",
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
		return errorResponse(id, codeInvalidRequest, "could not encode result: "+err.Error()), true
	}
	return response{JSONRPC: "2.0", ID: id, Result: b}, true
}

// errorResponse builds an error reply carrying both a code and a message.
func errorResponse(id json.RawMessage, code int, msg string) response {
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}
