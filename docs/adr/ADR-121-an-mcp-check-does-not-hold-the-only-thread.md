# ADR-121: an MCP check does not hold the only thread

**Status:** Accepted
**Accepted:** 2026-10-02 by Zy — the amendment to the plan of that day: "ADR-121 an MCP check does not hold the only thread — progress notifications while a check runs, and other calls answered during it; arms BACKLOG 'From ADR-113'". The record's text was drafted after that answer and not shown to Zy before execution: this is the answer, not a review of these words.
**Date:** 2026-10-03
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-024, ADR-067, ADR-075, ADR-113, ADR-115
**Invalidates:** The `gate` note in `internal/mcp/tools.go` that "a single server never has two tool calls in flight"
**Governs:** `internal/mcp/mcp.go`, `internal/mcp/tools.go`, `AGENTS.md`, `README.md`, `scripts/contract.sh`
**Enforced-by:** `internal/mcp/thread121_test.go::TestAPingIsAnsweredWhileAWritesCheckRuns`
**Served-path change:** While an `mrw_write`'s check runs, `mrw mcp` answers the requests that arrive after it — a ping, a read, another write — instead of holding them until the check ends, and when the write's request carries a progress token it sends `notifications/progress` while the call is in flight. Every other request is still answered in the order it arrived.

## Context

`Serve` reads one line, answers it, and reads the next; `callTool` holds the package mutex `gate` for the whole call. Since ADR-113 a code write runs the project's check inside that call — five minutes by default, longer when a project says so — and for that whole time the server reads nothing: a host's `ping`, which the spec says MUST be answered promptly, waits; so does a read the caller could usefully make. BACKLOG "From ADR-113" also records that Claude Code's idle window for a stdio server resets on a progress notification, and the server sends none.

**Audit of the class** — *a request handler that can run long*: `mrw read --grep 'land\.Verify\(|check\.Run\(|subproc\.Run\(' --exclude '*_test.go' internal/mcp/` — one site, `writeTool`'s `land.Verify` (the check and its steps). A read is bounded by its ceiling and ast-grep by its 2 s timeout; neither is in scope.

## Existing Primitives Audit

- **`gate`** — the ledger mutex every call takes. It is kept for everything that reads or writes the ledger, and released only across `Verify`, which touches neither the ledger nor the pending acks.
- **`callModern` / `callReserve`** — per-call values read only under `gate`; `writeTool` saves them before releasing it and restores them after, as callTool set them.
- **The write lock** (ADR-075, ADR-110) — already released before the check (ADR-075 §5), so a write arriving during a check lands as it would from a second CLI process; ADR-112's drift advisory names a touched file that changes.

## Decision

1. **The loop is released when a write's check starts.** `Serve` still waits for each request before reading the next, so every quick answer keeps its order; a write that reaches its check releases the loop, as it releases `gate`, and its answer is written when the check ends. Requests read meanwhile are answered as they finish.
2. **Answers are written whole, one at a time.** One mutex serializes every line `Serve` writes; at end of input it waits for every call in flight before it returns.
3. **Progress while a call runs.** A `tools/call` whose `params._meta.progressToken` is set gets `notifications/progress` with that token every 15 seconds while it is in flight — `progress` counting the seconds, `message` naming the tool — and none after its answer.
4. **No cancellation.** `notifications/cancelled` stays ignored; stopping a running check from the host is out of scope.

## Alternatives Considered

- **Every request on its own goroutine** — rejected: quick answers would race each other and arrive out of order, which transcript tests and a careless host would both notice, for no gain outside a check.
- **Progress only, the loop still held** — rejected by Zy's answer: a ping unanswered for five minutes is the protocol failure, not only the idle window.

## Component / Boundary Impact

`internal/mcp` only; it is not an engine package. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `Serve` | releases the loop when a write's check starts; one writer for every line | T1 | MCP hosts |
| `notifications/progress` | sent while a call with a progress token runs | T1 | MCP hosts |
| `AGENTS.md`, `README.md` | say so | T1 | callers |
| `scripts/contract.sh` | §220 | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a host's ping and a caller's read are answered during a long check; a host with an idle window hears progress.
- **Negative:** answers can now arrive out of request order — only across a check, which JSON-RPC allows.
- **Neutral:** a write during a check lands as a second CLI writer's would.

## Out of Scope

- `notifications/cancelled` stopping a running check (deferred: docs/adr/BACKLOG.md "From ADR-121")
- Concurrent reads outside a check (permanent: boundary: a read is bounded and ordered answers cost nothing)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a call reads `callModern`/`callReserve` set by another | Low | High | both are read only under `gate`, saved and restored around `Verify` |
| an answer and a progress line interleave | Low | High | one mutex for every line; the ticker stops before the answer is written |
| a session ends with a check in flight | Low | Medium | `Serve` waits for every call before it returns |

## Rollback

Revert the task: the loop holds a check again and no progress is sent. No receipt key, exit code or stored state changes.

## Follow-ups

- None — the record carries no open follow-up.

## Amendment 2026-10-06 — the Codex review of v1.42.0..v1.47.0

A write that asked for a step released Serve's loop even when no step would run (a dry run, a plan refused at validation), so a later quick call could be answered before it. The steps half of the release condition now also needs a write that applied with no failed hunk (`tools.go`); `internal/mcp/retro147_test.go::TestARequestedStepReleasesTheLoopOnlyWhenTheWriteLanded` pins it. No contract row: the ordering is not observable through the binary without a race.
