# ADR-067: The MCP surface speaks the current protocol revisions, reports a caller's argument mistakes to the model, and its index names every problem

**Status:** Accepted
**Accepted:** 2026-09-24 by M — *"accepted"*
**Date:** 2026-09-24
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-010, ADR-011, ADR-017, ADR-023, ADR-024, ADR-032, ADR-044, ADR-051, ADR-065, docs/adr/BACKLOG.md
**Governs:** `internal/mcp/mcp.go`, `internal/mcp/tools.go`, `scripts/contract.sh`, `README.md`
**Enforced-by:** `internal/mcp/era_test.go::TestAModernRequestForAnUnknownVersionIsRefusedWithItsSupportedList`
**Invalidates:** ADR-051 — contract §83's `format=git` row asserts a JSON-RPC `error`. The refusal stays and still names the grammar; its envelope becomes a tool result with `isError: true` (T3).
**Served-path change:**
- An `mrw_read` too large to serve returns an index. Its report now names every path the walk could not use, `-- <path>: <reason>`, as a served answer already does. Before, it carried only a count, so a CR-only file refused by ADR-065 was never named.
- `initialize` answers the version the host asked for when mrw speaks it (`2025-11-25` or `2025-06-18`), and otherwise its latest, `2025-11-25`. It always answered `2025-06-18`.
- A caller's argument mistake is a tool result with `isError: true` that the model reads, no longer a JSON-RPC error. This covers no spec, a bad `echo_pad`, no plan, an unknown or `git` format, and arguments that do not decode.
- A request carrying `_meta["io.modelcontextprotocol/protocolVersion"]` is served per request under the 2026-07-28 rules: `server/discover`, `resultType`, `serverInfo` in the result's `_meta`, caching hints on `tools/list`, and `-32022` for a version mrw does not speak. A request without that key is served exactly as today.

## Context

**What a host sends, measured.** On 2026-09-24, a headless Claude Code 2.1.281 session was pointed at `mrw` v1.23.0 through a logging wrapper, and the wire was captured. The session:
- opened with `initialize`, `protocolVersion: "2025-11-25"`, client capabilities `roots` (listChanged) and `elicitation`, and `clientInfo` naming `claude-code 2.1.281`;
- sent `notifications/initialized`, then `tools/list`, then `tools/call`;
- carried `_meta.progressToken` and `_meta["claudecode/toolUseId"]` on the call;
- never sent `server/discover`, although the binary's strings name it and name `2026-07-28`.

mrw answered `initialize` with `2025-06-18` (`internal/mcp/mcp.go:26`, `:183-184`, `:215-229`), which the legacy lifecycle allows as a downgrade, and the host accepted it.

**What the specification says now.** Read 2026-09-24 at modelcontextprotocol.io:
- **2025-11-25** (the host's version) adds optional icons and tool-name guidance for servers. SEP-1303 says input validation errors "should be returned as Tool Execution Errors rather than Protocol Errors to enable model self-correction".
- **2026-07-28** is the current revision. It removes the `initialize` handshake. Every request carries `io.modelcontextprotocol/protocolVersion` and `clientCapabilities` in `_meta`, and servers MUST implement `server/discover`. Results carry `resultType`. `tools/list` and `server/discover` carry `ttlMs` and `cacheScope`. An unsupported version is `UnsupportedProtocolVersionError`, `-32022`, with `data.supported` and `data.requested`. Roots, Sampling and Logging are deprecated, and `ping` is removed.
- A **dual-era** server MAY serve both eras: a request carrying the per-request `_meta` is served statelessly, and `initialize` selects legacy semantics (`/specification/2026-07-28/basic/versioning`, Backward Compatibility).

**What mrw does, traced.**
- **The index names no problem.** `matchIndex` (`internal/mcp/tools.go:1264`) takes the problem COUNT, and all three call sites (`:348`, `:425`, `:448`) pass one. The served answer prints each walk problem as `-- <path>: <reason>` (`:364-366`); the index prints none. This was the open P2 from the Codex review of #208 (BACKLOG, "An MCP `ast_grep` answer too large to serve…").
- **Argument mistakes are split across two envelopes.** Seven return a JSON-RPC `-32602` (`tools.go:204`, `:218`, `:485`, `:488`, `:496`, `:517`, `:519`). Six mistakes of the same kind already return `isError` results (`:215`, `:221`, `:228`, `:244`, `:262`, `:289`).
- **Two wire defects.** A result that fails to encode is answered as `-32600` Invalid Request (`mcp.go:418`), but it is the server's failure: `-32603`. A request whose `id` is `null` is answered as a request (`mcp.go:174` reads only absence). MCP says an id MUST NOT be null.
- **`serverInfo`** carries `name` and `version` only (`mcp.go:218-221`).
- **Roots, logging and sampling are not used.** The root comes from `CLAUDE_PROJECT_DIR` (`internal/mcp/root.go:17`), which is the migration 2026-07-28 recommends for Roots. Nothing to remove.

**The classes this record governs, enumerated 2026-09-24:**
- every index call: `grep -n 'matchIndex(' internal/mcp/tools.go`: 3 call sites and the definition;
- every JSON-RPC error `tools/call` returns for its arguments: `grep -n 'codeInvalidParams' internal/mcp/tools.go`: 10 sites.
  - 7 move to `isError`: `:204`, `:218`, `:485`, `:488`, `:496`, `:517`, `:519`.
  - 3 stay, as protocol errors by the specification's own list: `:117` (params that do not decode are a malformed `CallToolRequest`), `:133` (an unknown tool), and `:580` (a server ceiling too small to report a write; ADR-032 keeps that a JSON-RPC error because it carries no result to bound).
- every result the server returns: `grep -n 'resultResponse(' internal/mcp/mcp.go`: `initialize`, `tools/call`, `ping`, `tools/list`, plus `server/discover`, which this record adds.

## Existing Primitives Audit

- **`errorResult`** (`tools.go:839`). This is the `isError` result the six consistent sites already use. T3 routes the seven through it, so each one also passes the ceiling funnel `withinCeiling` (`tools.go:145`, ADR-032).
- **The served answer's problem lines** (`tools.go:364-366`). T1 reuses the same format in the index. No new receipt field.
- **`matchIndex`'s measure-and-trim loop** (`tools.go:1291-1362`). Kept. Problem lines are never trimmed; only index entries are.
- **`initializeResult`, `tools()`** (`mcp.go:215-411`) **and `instructionsText()`** (`internal/mcp/instructions.go:65`). T4's `server/discover` answers from the same functions, so the two eras cannot describe different servers.
- **`MaxResultChars` and the measurements that read it** (`tools.go:164-175`, `:303`, `:709`, `:749`, `:833`, `:862`, `:1010`; `grep -n 'MaxResultChars' internal/mcp/tools.go`). T4 routes every in-call comparison through one per-call limit, so a reservation for the modern decoration reaches every measurement at once.
- **`handle`** (`mcp.go:148`). The one dispatch point. The era is decided there, once per request.

## Decision

**1. The index names what the walk could not use.**
- `matchIndex` takes the walk's problems as `[]read.Problem`, plus a count of any other problems.
- Its report prints one `-- <path>: <reason>` line per walk problem, and one sentence for the rest when there are any.
- `problems` in `content[1]` stays a count.
- The trim loop drops index entries only, never problem lines. When the problem lines alone exceed the ceiling, `withinCeiling` refuses legibly, as it does for every answer.

**2. `initialize` negotiates, and the wire says what failed.**
- mrw speaks `2025-11-25` and `2025-06-18`. `initialize` answers the requested version when it is one of those, and `2025-11-25` otherwise, as the legacy lifecycle asks: "the latest version supported by the server".
- `serverInfo` gains `title` and `description`.
- An encode failure is `-32603`.
- A request with `"id": null` is answered `-32600` with a null id and not dispatched.

**3. A caller's argument mistake is a tool execution error.** The seven sites above return `errorResult` with the same sentence.
- The mistakes this covers are the ones inside an arguments OBJECT: a field of the wrong type, a missing spec or plan, a bad `echo_pad`, an unknown or `git` format.
- An `arguments` that is present but not an object (a number, a string, an array, `null`) is a malformed `CallToolRequest`. `callTool` refuses it with `-32602` before dispatch. `arguments: 42` used to reach `:204` or `:485`, and moving those sites wholesale would have made it a tool error. Found by the cold review of this record.
- Params that do not decode, an unknown tool, and a ceiling too small to report a write stay JSON-RPC errors.
- A refusal applies nothing. Acknowledgements sent with it are still promoted, as today: `promote` runs before argument validation (`tools.go:208`, `:492`), and this record does not reorder it.

**4. mrw is dual-era.** `handle` reads `params._meta["io.modelcontextprotocol/protocolVersion"]` once per request.
- **Absent:** the request is served exactly as today, the legacy era. `initialize` and every existing test and host take this path.
- **Present and not a version mrw speaks:** `-32022` with `data: {supported, requested}`.
- **`2026-07-28` without `io.modelcontextprotocol/clientCapabilities`:** `-32602`, as the specification requires for a malformed request.
- **`2026-07-28`, well formed:** the method runs as in the legacy era. Its result gains `resultType: "complete"` and `_meta["io.modelcontextprotocol/serverInfo"]`, and a `tools/list` result gains `ttlMs: 3600000` and `cacheScope: "public"`. The tool list is fixed for the binary's life and the same for every caller.
- **A legacy version named in `_meta`:** served, undecorated. mrw keeps no session state, so a version it speaks is servable on any request. A client that reads `-32022` retries with a version from `supported`; it does not fall back to `initialize` (stdio Backward Compatibility). Whichever listed version it picks is then answered, so the retry cannot loop.
- **`server/discover`:** always answers in the modern shape, whatever version its `_meta` names, because the method exists only in the modern era. It returns `resultType`, `supportedVersions`, `capabilities: {tools: {}}`, `serverInfo` in `_meta`, `instructions` and the caching hints. Without the `_meta` version it is malformed, `-32602`, which a dual-era client reads as "legacy" and answers with `initialize`.
- **The same list everywhere.** `supported` and `supportedVersions` are `["2026-07-28", "2025-11-25", "2025-06-18"]`, from one constant.
- **Precedence, decided in this order:**
  1. A message without an id is a notification and is never answered, before any `_meta` is read. A notification naming an unknown version or missing `clientCapabilities` still gets no reply.
  2. `initialize` always selects the legacy era and ignores any `_meta`, as the specification says `initialize` does.
  3. `server/discover` is always modern-shaped, as above.
  4. Every other method takes its era from the `_meta` version, as above.
- **Inside the ceiling, on every measurement.** A modern call reserves the encoded size of its decoration once, when it starts. Every comparison inside the call reads the ceiling minus that reserve: the write floor checked before applying, receipt elision, page and index sizing, and the final `withinCeiling` funnel. A receipt is never replaced after a write because decoration pushed it over. A refusal the funnel builds is decorated like any other modern result. Found by the cold review of this record: decorating only before `withinCeiling` left the pre-apply floor and receipt sizing undecorated.
- **`ping`** is still answered. 2026-07-28 removed it and a modern client does not send it, so answering costs nothing and breaks no legacy host.

**What would make this decision fail:**
- a legacy host that rejects `2025-11-25` in the `initialize` answer. Only a client asking for a version mrw does not speak is answered `2025-11-25`, and one asking for `2025-06-18` still gets it;
- a modern client that reads `-32022` for a legacy version named in `_meta` as a loop. mrw serves those versions, so it never answers `-32022` with a version it listed.

No host sends the modern era today: the one measured host is legacy. T4's evidence is therefore mrw's own conformance tests plus a replayed modern probe through the binary (§125), and the record says so.

## Alternatives Considered

- **2025-11-25 only; record 2026-07-28 in BACKLOG until a host probes `server/discover`.** Cheaper, and the measured host needs nothing more. Not chosen: M chose one record covering both revisions on 2026-09-24 ("Everything in one ADR"). The modern path adds one method and per-request metadata to a server that already keeps no session state.
- **Adopt an SDK (`github.com/mark3labs/mcp-go`, or the official Go SDK).** ADR-010 Decision 3 names subset growth as the signal to revisit this. The subset grows by one method (`server/discover`) and one metadata rule. Rejected on ADR-010's premise: `go.mod` holds one requirement. Whether either SDK speaks 2026-07-28 was not checked, and swapping transports would put every `internal/mcp` test on a different wire for a gain of one method.
- **Decorate every result with `resultType`, whatever the era.** Simpler. Rejected: a legacy host validating a result against its own revision's schema would meet a field that revision does not define. Keying on the request's era keeps legacy bytes identical, and `TestALegacyResultIsUnchangedByTheModernPath` pins it.
- **List only `2026-07-28` in `supported`.** Rejected: mrw really does serve the legacy versions on any request, and the specification's own dual-era `-32022` example lists both eras. Listing them costs nothing because each one is answered. A client does not use the list to fall back to `initialize`: a recognised modern error means "retry with one of these".
- **Carry named problems in a new `content[1]` field.** Rejected: the served answer names its problems in `content[0]` only, and a second shape for the same fact is two things to keep true.

## Component / Boundary Impact

- `internal/mcp` owns every change: `mcp.go` (dispatch, negotiation, `server/discover`, era decoration) and `tools.go` (index problem lines, argument errors).
- `README.md` documents the protocol revisions.
- Byte-identical against the merge-base: `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines`, `cmd/mrw`.
- `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_read` index, `content[0]` | one `-- <path>: <reason>` line per walk problem | T1 | MCP hosts; models |
| `initialize` result | `protocolVersion` echoes a spoken requested version, else `2025-11-25`; `serverInfo.title`, `serverInfo.description` | T2 | legacy hosts (Claude Code, Desktop) |
| JSON-RPC errors | encode failure `-32603`; `id: null` → `-32600` | T2 | every host |
| `tools/call` argument mistakes | `isError: true` result, not `-32602` | T3 | models (self-correction) |
| `server/discover` | new method | T4 | modern clients |
| per-request `_meta` era | `-32022`, `-32602`; `resultType`, `serverInfo` `_meta`, `ttlMs`, `cacheScope` | T4 | modern clients |
| `README.md` "Use it from an MCP host" | names the revisions spoken and the era rule | T4 | readers |
| contract §122–§125 | one row pair per task | T1–T4 | CI, `adr-verify` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `supportedVersions` constant (legacy and modern lists) | T2 | T4 | No — T4 extends it |
| `serverInfo()` helper (name, title, version, description) | T2 | T4 | No |

## Implementation

See `docs/adr/ADR-067-the-mcp-surface-speaks-the-current-protocol/tasks/README.md`.

## Consequences

- **Positive:**
  - a model is told which file an index could not use;
  - a model sees its own argument mistakes and can correct them;
  - the host gets the revision it asked for;
  - a 2026-07-28 client that probes gets a real answer instead of `-32601`.
- **Negative:** `handle` grows an era branch. Two eras means two shapes to keep true, and the legacy shape is pinned byte for byte so the modern one cannot leak into it.
- **Neutral:** no JSON receipt field is added. The modern decoration exists only on requests that ask for it.

## Out of Scope

- Roots, Sampling and Logging (permanent: fact: 2026-07-28 deprecates all three; citation: url https://modelcontextprotocol.io/specification/2026-07-28/changelog)
- Progress notifications for `_meta.progressToken` (permanent: boundary: every mrw call is one synchronous pass with no intermediate state worth reporting; the host's token is accepted and ignored, which the specification allows)
- Acting on `notifications/cancelled` (permanent: boundary: the server is serial by design, per ADR-010 and the ledger mutex in `tools.go:40`, so a cancellation arrives after the answer; a write must not stop halfway)
- Icons on tools or `serverInfo` (permanent: boundary: mrw ships no image asset, and a data URI in every `tools/list` spends context for a picture)
- The Tasks extension, Multi Round-Trip Requests, elicitation and `subscriptions/listen` (permanent: boundary: two synchronous tools with a fixed list need none of them; ADR-044 keeps the cargo at two tools)
- Streamable HTTP (permanent: boundary: ADR-010 keeps mrw on stdio; a listening socket is a surface this tool has no reason to open)
- Resources and prompts (permanent: boundary: ADR-044 keeps the cargo at two tools)
- The `-32602` code on the too-small-ceiling write refusal at `tools.go:580` (permanent: boundary: ADR-032 decided that refusal is a JSON-RPC error; its code is not an argument mistake the model can fix, and this record does not reopen it)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A legacy host rejects the `2025-11-25` answer | Low | High | Answered only when the request named a version mrw does not speak; `2025-06-18` requests still get `2025-06-18`; T2 tests both, and §123 drives the binary |
| Modern decoration leaks into legacy results | Med | Med | `TestALegacyResultIsUnchangedByTheModernPath` compares legacy bytes before and after T4 for every method |
| Decoration pushes a `tools/call` result past the ceiling, or replaces a write's receipt after the write | Med | High | a per-call reserve every measurement reads; T4 tests a modern write at the floor, a modern partial receipt, a modern served read's pending checkpoints, and a modern refusal's decoration |
| Problem lines alone exceed the ceiling | Low | Low | `withinCeiling` refuses legibly, the existing funnel |
| A host treats the new `isError` argument results as call failures to retry | Low | Low | the text names the mistake; the SEP-1303 guidance is the host side's |
| The modern path is exercised by no real host | High | Low | stated in Decision; §125 replays a modern probe through the binary; revisit when a host probes |

## Rollback

Revert `mcp.go`, `tools.go`, the tests, §122–§125 and the README paragraph. Nothing persistent moves: no ledger, receipt field or state file changes shape.

## Follow-ups

- [ ] Re-capture the Claude Code wire after the next host release that probes `server/discover`, and record whether it chose the modern era.
