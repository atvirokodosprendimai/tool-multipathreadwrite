# ADR-011: The MCP server tells a host what it is, and what it will return

**Status:** Accepted
**Date:** 2026-09-03
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-010 (the transport this completes), ADR-006 (the root boundary this binds correctly), ADR-002 (the ledger a wrong root would key wrongly), ADR-007 (the read whose output this bounds)
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/conformance_test.go::TestEveryDeclaredOutputSchemaValidatesARealResponse`
**Served-path change:** an MCP host binds `mrw mcp` to the project it launched, sees which tool writes and which only reads, and gets a machine-checkable shape back instead of prose.

## Context

ADR-010 shipped a transport that speaks MCP correctly enough for a host to connect. Three days of
using it, one adversarial review and a stress run say it does not yet tell a host enough to be
driven safely, and one of the gaps is a plain bug.

**The documented config block binds to the wrong directory.** README's block is
`{"command": "mrw", "args": ["mcp"]}`. `--root` defaults to `"."`, so the server binds to whatever
working directory the host happened to launch it with. Measured 2026-09-03: with cwd `/tmp` and
`CLAUDE_PROJECT_DIR` pointing at a repository, `mrw_read` served `/private/tmp`. Claude Code sets
`CLAUDE_PROJECT_DIR` in the server's environment precisely because cwd is not guaranteed
(https://code.claude.com/docs/en/mcp). A tool whose entire safety model is root confinement must not
be confined to the wrong root: every ADR-006 refusal is still correct and every one of them is
answering about a tree nobody asked about, and the ADR-002 ledger is keyed per checkout, so a wrong
root silently starts a second ledger.

**The tool surface is silent about the things a host uses to protect a user.** `tools/list` returns
`name`, `description` and `inputSchema` and nothing else. It does not say that `mrw_read` only reads
and `mrw_write` modifies files, which is what `annotations` exists for; it does not declare the shape
of what comes back, which is what `outputSchema` exists for; and it does not say how large a result
may get, which Claude Code reads from `_meta["anthropic/maxResultSizeChars"]`.

**The result violates a SHOULD, measured.** The 2025-06-18 tools specification says: *"For backwards
compatibility, a tool that returns structured content SHOULD also return the serialized JSON in a
TextContent block."* Ours returns a rendered report — `ok f.txt 1` / `1 hunk(s), 1 file(s), 0 failed`
— so a host reading only `content` gets prose where the spec promises data. This matters more than
tidiness because `structuredContent` is consumed in practice: a peer session measured reading a field
out of `result.structuredContent` from a live client call, so both halves are load-bearing.

**A large read buffers without bound and can take the process down.** Measured against `46d2e2d`:
ten 18 MB reads in one call cost **5.9 MB** peak RSS through the CLI and **1268 MB** over MCP; a
single 193 MB file answers correctly at 1.56 GB peak, emitting one 241 MB JSON line. The CLI streams
through `bufio`; `readTool` renders into a `bytes.Buffer`, takes `String()`, marshals that into the
result and marshals the response again. It is not a leak — 500 sequential calls sit at 13 MB — and
the write path is unaffected. The cost is per-call peak, set by the largest single response, and it
is unbounded, so a large enough read gets the server OOM-killed. That closes the pipe, and ADR-010-T1
argues that a host seeing a closed pipe cannot tell a crash from a refusal.

**The earlier framing of that last one was wrong, and this record corrects it.** `BACKLOG.md` says a
cap would be "a behaviour divergence from the CLI, which ADR-010's whole thesis is careful about",
and left the decision open on that ground. Claude Code caps MCP tool output at **25,000 tokens** by
default (`MAX_MCP_OUTPUT_TOKENS`) and offers a per-tool override up to 500,000 characters via
`_meta["anthropic/maxResultSizeChars"]`. What the host does with an oversized result is not
truncation in place: the documentation says *"results that exceed the default threshold are persisted
to disk and replaced with a file reference in the conversation."* The conclusion is unchanged and
slightly stronger — either way the model never receives the oversized result as the file; it gets a
pointer it must spend another call to read. So the choice was never "cap or stay faithful to the
CLI"; it was "refuse legibly, or have the result taken out of the conversation by someone else after
paying the memory to build it". Reading `cmd/mrw/main.go` over MCP already returns ~13,000
tokens, so two such files exceed the default today.

## Existing Primitives Audit

- **`internal/mcp/mcp.go` and `tools.go` (ADR-010):** the transport and the two adapters. **Extended,
  not replaced.** Every change here is additive to the handshake and the tool descriptors; no tool
  gains a verdict of its own, which is ADR-010-T2's Stop Condition and still binds.
- **`read.Options.MaxLines` (ADR-007):** audited and **NOT taken.** It was the planned mechanism and
  it is the wrong one: `MaxLines` bounds each FILE, so N specs naming one large file are still N
  times the cap — which is the exact shape of the request that motivated this ADR. The cap that
  shipped is a capped `io.Writer` on the transport side. `internal/read` is untouched not because
  the bound lives in an Option it already had, but because the transport is the right place to bound
  a transport's result.
- **`rooted` and ADR-006's boundary:** **reused unchanged.** T1 changes which root is passed in, never
  what confinement means.
- **`apply.Result` / `seen.Observation`:** **reused as the payload.** The schemas T2 declares are
  generated from these types rather than written beside them.
- **An MCP SDK:** still not taken. ADR-010 recorded that decision against `mark3labs/mcp-go`; nothing
  here changes the arithmetic — schemas and annotations are struct literals.

## Decision

**1. The server binds to the checkout the host meant.** When `--root` is not given, `mrw mcp` uses
`CLAUDE_PROJECT_DIR` if it is set and names a directory, and falls back to the working directory
otherwise. An explicit `--root` always wins. The chosen root is reported on stderr at startup, where
a host's log can show it, and never on stdout.

**2. The tool surface declares what a host needs to protect a user.** Each tool carries a `title`, an
`annotations` object stating truthfully whether it reads or destroys, an `outputSchema` **generated
from the Go type the handler returns**, and `_meta["anthropic/maxResultSizeChars"]`.

**3. Every result carries the serialized JSON in a text block**, as the spec's SHOULD asks, with the
human-readable report kept as a second block. The verdict remains one value serialized once; the text
block is that same serialization, not a second rendering of the same facts.

**4. A read that would exceed the declared limit is refused legibly**, naming the limit and what to
ask for instead, rather than being buffered and then truncated by the host.

**Go/no-go, checked during execution and recorded in the task verification logs. If any fails, the
task is withdrawn rather than shipped:**

- **No engine change.** `git status --porcelain --untracked-files=all` and a merge-base `git diff`
  over `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and
  `internal/state` are both empty, for the reasons ADR-010 records: each sees what the other misses.
- **No new dependency.** `go.mod` still declares exactly one requirement.
- **Every declared `outputSchema` validates a REAL response.** Not a schema read alongside the code —
  the test calls the tool and validates what actually came back. A peer's schema gate caught a schema
  attached to the wrong tool and one that validated anything; both read fine.
- **No capability is advertised that no code path can deliver.** `capabilities.tools` stays `{}`
  because nothing sends `notifications/tools/list_changed`. The same peer shipped
  `"listChanged": true` with no sender, leaving clients waiting on a message that never comes.

## Alternatives Considered

- **Leave the root as cwd and document "pass `--root`".** Free, and it is today's README. Rejected:
  the block a reader copies is the one that must be right, an environment variable the host sets for
  this exact purpose is more reliable than an absolute path a user pastes, and a wrong root produces
  correct-looking refusals about the wrong tree — the worst failure shape available here.
- **Read `roots/list` from the client instead.** The protocol's own answer, and Claude Code implements
  it. Rejected for now and recorded in BACKLOG: it requires the server to make a REQUEST of the client
  and correlate the response, which is a direction the transport does not yet go, and
  `CLAUDE_PROJECT_DIR` closes the same gap with an environment lookup. Revisit if a second host needs it.
- **Truncate a large read instead of refusing it.** Rejected on ADR-007's own premise: a cap that
  fires must say so, and a truncation that reaches a model as "the file" is precisely the silent
  wrong answer this project exists to refuse. Refusing names the limit and the caller asks again with
  a range.
- **Stream a large read across several responses.** Not available: MCP is one call, one message.
- **Hand-write the output schemas beside the types.** Rejected on measured evidence: a peer's
  hand-written schema drifted onto the wrong tool and a permissive one validated anything. Generating
  from the type and validating a real response is the only form that cannot silently rot.
- **Raise the cap and keep buffering.** Rejected because the amplification is ~7x and unbounded; a
  bigger constant moves the OOM rather than removing it.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | The wire protocol, the tool descriptors and the result envelope. Still knows JSON-RPC, not the filesystem | Yes — changes when the protocol or the declared surface changes |
| `cmd/mrw` | Flag surface, and now the root resolution the `mcp` subcommand hands in | Yes |
| `internal/read` | Unchanged, and not because the cap lives in it — the bound is a transport-side writer | Untouched |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| `tools/list` entries gain `title`, `annotations`, `outputSchema`, `_meta` | Public contract, additive | Any MCP host |
| `tools/call` result gains a serialized-JSON text block as `content[0]` | Public contract, additive; the report moves to `content[1]` | Any MCP host reading `content` |
| `mrw mcp` root resolution consults `CLAUDE_PROJECT_DIR` | Behaviour | Hosts launching without `--root` |
| A read over the declared limit returns `isError: true` | Behaviour | MCP callers reading large files |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| `mcp.ResolveRoot(explicit string, env func(string) string) string` | T1 | — | No — new |
| the tool descriptor set (title/annotations/outputSchema/_meta) | T2 | T3 | No — additive |
| `maxResultChars`, the declared limit | T2 | T3 | No — new |

## Implementation

Three tasks, executed in order. T2 before T3 because T3 refuses against the limit T2 declares, and a
limit enforced before it is advertised is a refusal a caller cannot have anticipated.

## Consequences

- **Positive:** a host binds to the right tree without the user pasting a path; a host can tell which
  tool writes; a caller gets a shape it can validate; a large read fails legibly instead of taking the
  process down.
- **Positive:** `content[0]` becomes machine-readable, which is what the spec's SHOULD is for.
- **Negative:** the result grows — the same verdict now travels as JSON and as prose. Bounded by the
  receipt's size, which is proportional to hunks and not to file content.
- **Negative:** a read that used to succeed slowly may now be refused. That is the intent, and the
  refusal names the range to ask for instead.
- **Neutral:** `content[1]` is where the report moved. Additive for a host reading `content[0]`,
  a change for one that assumed a single block.

## Out of Scope

- Reading `roots/list` from the client (deferred: docs/adr/BACKLOG.md)
- Concurrent dispatch of tool calls, and the head-of-line blocking a sequential loop causes (deferred: docs/adr/BACKLOG.md)
- `notifications/cancelled` actually cancelling in-flight work (deferred: docs/adr/BACKLOG.md)
- Progress notifications (deferred: docs/adr/BACKLOG.md)
- `_meta["anthropic/requiresUserInteraction"]` on `mrw_write` (permanent: boundary: a host already gates MCP tool calls, and a server demanding a prompt on every write would make the batching mrw exists for unusable)
- `notifications/tools/list_changed` and a `listChanged` capability (permanent: boundary: the tool set is fixed at compile time, so the capability would be a promise with no sender — the exact defect measured in a peer's server on 2026-09-03)
- MCP resources, prompts, sampling (permanent: boundary: restated from ADR-010; the tools ARE the product)
- Changing any CLI behaviour (permanent: boundary: ADR-010's go/no-go, still binding — the engine directories stay byte-identical)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A generated schema is permissive and validates anything | Med | High — it reads as coverage | The Enforced-by test calls the tool and validates a REAL response; a schema of "object, no properties" fails it |
| `CLAUDE_PROJECT_DIR` names a directory outside what the user meant | Low | High | An explicit `--root` always wins, the resolved root is printed to stderr at startup, and ADR-006 confinement still applies to whatever is chosen |
| The cap refuses a read a caller legitimately needs | Med | Low | The refusal names the limit and the range syntax; `--root`-scoped CLI reads are unaffected |
| `content[0]` changing shape breaks a host that assumed prose | Low | Med | Additive: the prose is still there, at `content[1]`, and §40 asserts both blocks |
| Annotations are asserted rather than true | Low | Med | §40 drives `mrw_read` and asserts it wrote nothing; a `readOnlyHint` on a tool that writes is a lie a host acts on |

## Rollback

Revert the commits. The declared fields are additive, so a host that read the old surface still works;
`content[0]` returns to prose; root resolution returns to cwd. No state format moves, no ledger entry
changes meaning, and a tree touched under this ADR is served identically by the previous binary.

## Follow-ups

- [x] `BACKLOG.md`'s read-buffering entry is replaced, in T3 S5, by an entry recording that its stated reason for deferring — that a cap diverges from the CLI — was the thing this record corrects
- [ ] Revisit `roots/list` if a second host needs it, or if `CLAUDE_PROJECT_DIR` proves host-specific in a way that matters
