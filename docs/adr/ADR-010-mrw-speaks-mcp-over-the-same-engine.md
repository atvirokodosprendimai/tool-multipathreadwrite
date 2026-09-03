# ADR-010: mrw speaks MCP over the same engine, and stays a binary

**Status:** Accepted
**Date:** 2026-09-03
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (whose rejected alternative this revisits), ADR-002 (the ledger this must not weaken), ADR-004 (the state directory a session must not replace), ADR-003 (the check surface, unchanged)
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/tools_test.go::TestTheWriteToolReturnsTheSameResultAsTheCLI`
**Invalidates:** ADR-001 — the clause of its Alternatives Considered reading "An MCP server instead of a binary: rejected because a server binds at session start and is unrecoverable mid-session". That rejection is superseded, NOT deleted: this record adds a transport and removes nothing, so every other clause of ADR-001 stands unchanged.
**Served-path change:** `mrw mcp` serves the read and write engine over MCP on stdio, so an agent that speaks MCP reaches mrw without shell access and without being told to.

## Context

**ADR-001 rejected an MCP server in one sentence, and that sentence has aged.**
It reads: *"a server binds at session start and is unrecoverable mid-session; a binary is
re-invoked per call and cannot enter that state."* That was a fair description of early stdio MCP.
Hosts commonly restart a dead stdio server now — host behaviour, not a spec guarantee, so it is
recorded here as context rather than load-bearing; the Recovery go/no-go below holds either way. The
reasoning was never re-examined, and it was written when the alternative was "server INSTEAD OF
binary", which is not the choice on the table.

**mrw is not alone, and the closest competitor is an MCP server.** Surveyed 2026-09-03:

| | |
|---|---|
| [`tumf/mcp-text-editor`](https://github.com/tumf/mcp-text-editor) (199★, Python) | line ranges, multi-file, SHA-256 conflict detection, read-before-write by design |
| [Multi Edit MCP](https://glama.ai/mcp/servers/eaisdevelopment/mcp-multi-edit) | atomic batch find-replace, "all edits succeed or none" |
| [AI FileSystem MCP](https://mcpservers.org/servers/proofmath-owner/ai-filesystem-mcp) | transactional multi-file with rollback |
| [`mcp-readedit`](https://github.com/abnersajr/mcp-readedit) | collapses read+edit into one call — mrw's own pitch |

mrw is ahead on substance — addresses resolve against the ORIGINAL file where `mcp-text-editor`
applies bottom-to-top, atomicity is a pinned contract rather than undocumented, and nobody else
publishes a per-hunk verdict or 291 adversarial assertions. It is behind on REACH: every one of
those plugs into an agent directly, while mrw needs shell access AND an agent that knows to reach
for it. `AGENTS.md` is a documentation fix for what is a distribution problem.

**The two gaps are one gap.** The README says: *"Run mrw one call at a time against a checkout…
parallel invocations overwrite one another's entries — 40 racing reads kept 5."* That is not a
ledger defect. It is a consequence of being a per-invocation binary: read-before-write must persist
to disk because nothing else survives between calls, so every read rewrites the whole file and
parallel PROCESSES race. A server is one process and can serialize its own calls in-process — **the
concurrency limitation dissolves between calls made through the server**, without changing the
ledger format at all. It does not dissolve between a CLI invocation and a server running beside it:
those are still two processes rewriting one ledger file, and that remains the CLI limitation.

**The plan text arrives JSON-escaped on this path, and that is not a reversal.** ADR-001 rejected
JSON as the PLAN FORMAT because it would make an author escape every newline and quote in a code
body. Here the plan is authored in the same line-oriented format and carried as one string inside
the tool call, so the escaping is done by the transport's encoder and undone by its decoder, and no
human or model ever writes an escape. The format ADR-001 chose is unchanged.

## Existing Primitives Audit

- **`internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`:** the
  engine, and it is already transport-free — `cmd/mrw` holds flag domains, precedence and rendering
  in 17 functions, and imports the engine rather than containing it. **Reused unchanged.** This ADR
  adds a second caller, not a second engine, and the moment it needs its own semantics it is wrong
  (see the Stop Condition in T2).
- **`cmd/mrw`'s subcommand shape** (`read`, `write`, `check`, `iter`, `seen`, `stats`): **reused as
  precedent** — `mcp` is one more, registered the same way.
- **`internal/state` + the ledger:** **reused unchanged.** The server shares the same per-checkout
  ledger, which is what keeps a CLI write after an MCP read licensed. A session-only in-memory
  ledger was considered and rejected below.
- **An MCP SDK as a dependency — specifically `github.com/mark3labs/mcp-go`, which this team's
  centralised `effective-go` skill names as the default for MCP work:** audited and NOT taken. mrw
  has exactly one dependency today
  (`urfave/cli/v3`). MCP over stdio is JSON-RPC 2.0 with a handful of methods; the subset needed —
  `initialize`, `tools/list`, `tools/call` — is small enough to own. ADR-004 refused a git
  dependency on the same premise, and a protocol SDK is a larger surface than the protocol subset.

## Decision

**Add `mrw mcp`: a stdio MCP server exposing the existing read and write engine as tools. Keep the
binary exactly as it is.**

This is additive on every axis. No existing subcommand, exit status, plan format, ledger entry or
flag changes. A caller who never runs `mrw mcp` cannot tell this shipped.

**Three properties that are the decision, not the implementation:**

1. **One engine, one answer.** The MCP tools call `read.Run` and `apply.Apply`, the same functions
   `cmd/mrw` calls. A per-hunk verdict returned over MCP is the same verdict the CLI prints, from
   the same code. If the server ever needs to compute its own, this ADR is wrong — two answers to
   "did this apply?" is the defect class this project exists to refuse.
2. **The ledger is shared, and the server serializes access to it.** One process, one writer, so
   the "one call at a time" limitation does not apply to MCP callers. A CLI write after an MCP read
   is still licensed, because both wrote the same file.
3. **The protocol is hand-rolled and small.** No new module in `go.mod`. If the subset needed grows
   past what is comfortable to own, that is the signal to reconsider — recorded as a go/no-go below.

**Go/no-go, checked during execution and recorded in the task's verification log. If any fails,
`mrw mcp` is withdrawn and the binary is the whole answer:**

- **No new dependency.** `go.mod` still declares exactly one requirement and it is `urfave/cli/v3`,
  checked by counting requirement lines rather than by diffing against a remote ref — a fence that
  needs `origin/main` fetched is a fence that fails in a shallow CI checkout for a reason unrelated
  to what it is asking.
- **No engine change.** `git status --porcelain --untracked-files=all` over `internal/read`,
  `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` is
  empty for this ADR's tasks. Untracked is the operative word: a `git diff` form passes while a NEW
  engine file sits beside the old ones, which is exactly how an engine grows. This is a
  working-tree gate and it proves nothing about a change already committed — that is what the diff
  in review is for, and the task's verification log records the SHA it ran against.
- **The transport is asserted through a real pipe**, not in-process — a server tested by calling its
  handler is not a server, for the same reason `contract.sh` exists rather than more Go tests.
- **Recovery.** ADR-001's original concern was a server that is unrecoverable mid-session. Killing
  the server mid-session must lose nothing a CLI invocation would not: the ledger is on disk, the
  host restarts the process, and the next call proceeds. Asserted, not assumed.
- **The tool result is the protocol's envelope with mrw's verdict inside it.** `tools/call` returns
  a `CallToolResult`, not a bare `apply.Result`: the spec requires a `content` array, and a host is
  entitled to reject or hide a tool that answers with something else. The verdict travels in
  `structuredContent`, serialized by the same code that writes the `--json` receipt, and the
  identity test compares the DECODED `structuredContent` against that receipt. "One answer" is a
  claim about the verdict, never about the envelope carrying it.

## Alternatives Considered

- **Stay binary-only and document the recipe.** Free, and it is today. Rejected because the gap is
  distribution rather than documentation: `AGENTS.md` reaches a repository, not an agent, and four
  competitors are one config line away from any MCP host.
- **Replace the binary with a server.** What ADR-001 actually rejected, and still right to reject.
  The binary works with no server, no session and no host, in any repository — that is the
  differentiator, and a server that replaced it would trade the thing that makes mrw usable from a
  shell script or a CI job.
- **Make the ledger stateless — pass the SHA back in the request, as `mcp-text-editor` does.** This
  fixes concurrency without any server, and it is the tempting one. Rejected because it moves the
  read-before-write guarantee INTO the caller: a caller that echoes back a SHA it did not read has
  licensed itself, and ADR-002 exists precisely so the tool holds that fact rather than trusting the
  caller to. Recorded in BACKLOG.md as the fallback if the server path fails.
- **A session-only in-memory ledger on the MCP path.** Faster and cleaner in isolation. Rejected
  because it splits the guarantee: a file read over MCP would not license a CLI write, and the two
  transports would disagree about what has been seen. One ledger or two products.
- **Shell out to `mrw` from a generic filesystem MCP server.** Zero code here. Rejected because the
  receipt would then be parsed from text by something that did not run the write, which is the
  arrangement `--json` exists to replace.
- **Adopt `github.com/mark3labs/mcp-go`.** Less code to own, and it is the standing team default —
  the centralised `effective-go` skill says "for mcp use github.com/mark3labs/mcp-go". Declining a
  convention is a choice that has to be made knowingly rather than by omission, so it is named here
  rather than left as "an SDK". Rejected on ADR-004's premise: mrw has one dependency, the needed
  subset is three methods, and the SDK's surface is larger than that subset. If the subset grows —
  resources, sampling, progress — this is the first thing to revisit, and the Risks table says so.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` (new) | The wire protocol and the tool schemas. Knows JSON-RPC, not the filesystem | Yes — changes when the protocol or the tool surface changes |
| `cmd/mrw` | Flag surface and now one more subcommand that starts the server | Yes |
| `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` | Unchanged — they gain a caller, not a responsibility | Yes |

`internal/mcp` does not import `cmd/mrw`, and `cmd/mrw` wires it — the same split ADR-001 drew
between `plan` and `apply`, for the same reason: the transport must be testable without the CLI.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw mcp` | new public subcommand | `cmd/mrw` | MCP hosts |
| MCP tool `mrw_read` | new public contract | `internal/mcp` | agents |
| MCP tool `mrw_write` | new public contract | `internal/mcp` | agents |
| `mcp.Serve(in io.Reader, out io.Writer, root string) error` | new internal API | `internal/mcp` | `cmd/mrw` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `mcp.Serve`, the JSON-RPC frame (T1) | T1 | T2, T3 | No — new package |
| MCP tools `mrw_read` / `mrw_write` (T2) | T2 | T3 | No — new surface |

## Implementation

See `ADR-010-mrw-speaks-mcp-over-the-same-engine/tasks/README.md`. Three tasks: the transport, the
two tools over the unchanged engine, and the contract rows that drive the server through a real
pipe.

## Consequences

- **Positive:** mrw reaches an agent without shell access and without `AGENTS.md` — the distribution
  gap every competitor already closed.
- **Positive:** the "one call at a time" limitation does not apply to MCP callers, for free, because
  one server is one writer. The README's serial note becomes a statement about the CLI path only.
- **Positive:** the per-hunk verdict — the thing no competitor publishes — becomes reachable by the
  audience most exposed to a write that silently changed nothing.
- **Negative:** a second transport is a second place a bug can live, and the engine's guarantees are
  only as good as the thinner surface around them. The go/no-go "no engine change" condition is what
  keeps it thin.
- **Negative:** hand-rolled JSON-RPC is maintenance mrw did not have. The alternative was a
  dependency, and the subset is three methods.
- **Neutral:** `mrw` grows a long-running mode, which it has never had. Nothing else about the binary
  changes, and killing the server loses nothing the ledger did not already hold.

## Out of Scope

- Removing or deprecating any CLI subcommand (permanent: boundary: the binary working with no server and no host is the differentiator, and it is what ADR-001 was right to protect)
- A network-listening or hosted server — stdio only (permanent: boundary: a listening socket is a security surface this tool has no reason to open, and stdio is the transport every host this decision is aimed at already speaks; the specification defines a second standard transport and says a client SHOULD support stdio, so this is a scoping choice rather than a claim about every host)
- MCP resources, prompts, sampling, or any capability beyond `tools` (permanent: boundary: the tools ARE the product; a resource surface would be a second way to read, and two ways to read is two answers to what the caller saw)
- Adopting an MCP SDK, including the team default `github.com/mark3labs/mcp-go` (permanent: fact: mrw has exactly one module dependency and the needed protocol subset is three methods; citation: file `go.mod:5`)
- A stateless hash-in-request mode, as `mcp-text-editor` uses (deferred: docs/adr/BACKLOG.md)
- Exposing `check`, `iter`, `seen` or `stats` as MCP tools (deferred: docs/adr/BACKLOG.md)
- Fixing the CLI path's parallel-read limitation (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The server grows semantics the CLI does not have | Med | High — it is the whole thesis | T2's Stop Condition: the tools call the same functions or the ADR is wrong. The go/no-go "no engine change" is checked as a diff |
| ADR-001's original concern is real — a wedged server | Low | Med | A go/no-go condition: killing it mid-session must lose nothing, asserted through a real pipe rather than assumed |
| Hand-rolled JSON-RPC has a framing bug a host trips over | Med | Med | The first draft of T1 specified `Content-Length` framing, which is LSP and not MCP; a real-pipe test written to the same mistake passes, so §38 is NOT the mitigation on its own. T1 S2 cites the spec's stdio section for the frame, the stdout rule and the stderr rule, and the alternative (`mark3labs/mcp-go`) is recorded and can be adopted if the subset grows |
| Two transports drift on the receipt shape | Med | Med | The MCP tool returns the same `apply.Result` the `--json` receipt carries, serialized once |
| It becomes a second product with its own roadmap | Med | High | Out of Scope refuses resources, prompts and the other subcommands; each is a separate decision |

## Rollback

Delete `internal/mcp`, the `mcp` subcommand and its contract section. Nothing else references them,
`go.mod` is unchanged by construction, no state format moves, and a tree touched by the server is
served identically by the previous binary. An MCP host loses a tool; every CLI caller is unaffected.

## Follow-ups

- [ ] Re-read ADR-001's Alternatives clause once this ships and confirm it points here, so a future reader does not re-derive the rejection
- [ ] `adr-lint` passes with five advice lines, named here so nobody re-derives them: `Enforced-by` points at a test that does not exist yet and `Governs:` matches no tracked file — both resolve when T1 and T2 land, and both would be wrong to silence before then. The other three count Acceptance-fence segments against `Rests-on:` names; the fences rest on what is declared, and the extra segments are the `go test ./...` and `contract.sh` runs every task in this repository ends with
