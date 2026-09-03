# Task ADR-010-T1: A stdio JSON-RPC transport, hand-rolled, reachable

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `mcp.Serve(in io.Reader, out io.Writer, root string) error`, the JSON-RPC frame, `initialize` and `tools/list`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the newline-delimited framing`, `the absence of a new go.mod requirement`, `the subcommand registration`

## Goal

`mrw mcp` starts a stdio MCP server that completes a host's `initialize` handshake and lists its
tools, with no new module dependency.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | add | the frame, the JSON-RPC envelope, `initialize`, `tools/list` |
| `internal/mcp/mcp_test.go` | add | its tests |
| `cmd/mrw/main.go` | edit | **THE CALL SITE.** `mcpCmd()` and its entry in the `Commands` slice — the line that makes the package reachable. `rooted.Descendable` was built, tested and deleted unused on 2026-09-03 for want of exactly this |
| `go.mod` | **must not change** | the go/no-go condition, checked by the fence rather than by intention |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a framed `initialize` request gets a framed
   response carrying a protocol version; `tools/list` names both tools; a malformed frame is
   answered with a JSON-RPC error rather than a panic or a silent close.
2. [S2] Implement the framing the MCP stdio transport actually specifies: **one JSON-RPC message per
   line, delimited by newlines, and no embedded newline inside a message.** There is no
   `Content-Length` header in MCP stdio — that is Language Server Protocol, and a server that speaks
   it fails every host's `initialize` while passing any test written to the same mistake. Source:
   https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#stdio — *"Messages are
   delimited by newlines, and MUST NOT contain embedded newlines."* The same page carries two rules
   that bind mrw specifically, because this binary prints to stdout everywhere else: the server
   *"MUST NOT write anything to its `stdout` that is not a valid MCP message"*, and it *"MAY write
   UTF-8 strings to its standard error"* for logging. A stray `fmt.Println` on the server path is a
   protocol violation, not a cosmetic one. [proof: acceptance]
3. [S3] Implement the JSON-RPC envelope: `id`, `method`, `params`, and an error object with a code.
   A request with an unknown method is an error RESPONSE, never a dropped connection — a host that
   sees a closed pipe cannot tell a crash from a refusal.
4. [S4] Answer `initialize` and `tools/list`. The tool SCHEMAS are declared here and their handlers
   land in T2, so `tools/list` is honest from the first commit rather than growing later.
5. [S5] Add `mcpCmd()` and **register it** in the `Commands` slice, so `mrw mcp` resolves and
   `mrw --help` lists it. [proof: mutation]
6. [S6] Assert no dependency was added — the go/no-go condition. The fence diffs `go.mod` and
   `go.sum` against this branch's merge-base with `main`, so it is red when and only when the branch
   changed what mrw requires. A dependency added by accident is exactly what this project's
   one-dependency posture loses quietly. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr010-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr010-t1.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum \
  && go build -o /tmp/mrw-t1 ./cmd/mrw \
  && /tmp/mrw-t1 --help 2>&1 | grep -q 'mcp' \
  && go test ./...
```

Every clause names something this task creates, and the fence was RUN before any code existed and
observed to exit non-zero — `internal/mcp` does not compile, so it fails at the first clause. That
check is not optional: three fences in the 2026-09-03 session were green on an untouched tree
because nobody ran them, including one that named a `contract.sh` section which already existed.

The `go.mod` clause is the go/no-go condition made mechanical. It was written first as
`[ "$(grep -c '^\t' go.mod)" = "1" ]`, which counts tab-indented lines rather than requirements and
returns **0** on this repository — a fence red on an untouched tree, which is the same defect as a
fence green on one. The diff form is red when and only when the branch changed what mrw requires.
The `--help` clause is rung 3 — a subcommand a caller cannot discover is not shipped.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInitializeCompletesTheHandshake` | `internal/mcp/mcp_test.go` | S4 — a host can connect | — | S1, S4 |
| `TestOneMessagePerLineRoundTrips` | `internal/mcp/mcp_test.go` | S2 — newline-delimited framing, and that no message carries an embedded newline | — | S1, S2 |
| `TestAMalformedFrameIsAnErrorNotAClose` | `internal/mcp/mcp_test.go` | S3 — a host can tell a refusal from a crash | — | S1, S3 |
| `TestToolsListNamesBothTools` | `internal/mcp/mcp_test.go` | S4 — the surface is declared before it is implemented | — | S1, S4 |
| `TestAnUnknownMethodIsAnErrorResponse` | `internal/mcp/mcp_test.go` | S3 | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests above |
| 2 — something selects it | the `Commands` slice entry (S5); removing it makes the fence's `--help \| grep mcp` clause red, and that clause is inside the Acceptance command rather than beside it |
| 3 — the caller can discover it | `mrw --help` lists `mcp`, asserted by the fence; the README is T3's |
| 4 — it is used | nothing measures this yet — `mrw stats` counts plans, not transports, and counting the transport is not this ADR's question |

## Mutation Log

## Invariants

- `go.mod` and `go.sum` are unchanged against the branch's merge-base with `main`.
- Everything the server writes to stdout is a valid MCP message; anything else goes to stderr.
- No message written to stdout contains an embedded newline.
- No file under `internal/read`, `internal/apply`, `internal/plan` or `internal/seen` is touched.
- A malformed or unknown request produces a JSON-RPC error response; the connection stays open.

## Risks
- Framing is where hosts differ in practice, and the first draft of this task specified LSP framing
  outright. Mitigated by T2's §38 driving a real pipe; note that a real-pipe test written against
  the wrong frame passes too, so the spec citation in S2 is the actual mitigation.
  test would pass on a frame no host sends.
- A long-running mode is new to this binary. Nothing else about it changes, and T2 owns the
  behaviour that touches files.

## Stop Condition

Stop if the transport needs anything from the engine to answer `initialize` or `tools/list`. It must
be startable in a directory with no ledger, no plan and no git — a server that cannot say hello
without touching the tree has coupled the wire to the filesystem, and T2's tools become the only
place that coupling belongs.

## Out of Scope

- The tool handlers — T2's job.
- README documentation — T3's job.

## Verification Log
