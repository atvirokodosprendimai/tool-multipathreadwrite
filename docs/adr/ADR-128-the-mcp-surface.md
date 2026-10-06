# ADR-128: the MCP surface: a BOM, the force advice, an unknown ack, a cancel

**Status:** Accepted
**Accepted:** 2026-10-06 by Zy — "MCP cancel stops a check" among the items chosen for this round, and "plan these fixes then", on the Windows peers' MCP findings (BACKLOG "From the Windows peers") and BACKLOG "From ADR-121". The record's text was drafted after that answer.
**Date:** 2026-10-06
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-031, ADR-067, ADR-080, ADR-113, ADR-121
**Invalidates:** None — ADR-121's "a cancel does not stop a running check" is what Decision 4 changes, and ADR-121 names it as BACKLOG's to arm
**Governs:** `internal/mcp/mcp.go`, `internal/mcp/tools.go`, `internal/mcp/ack.go`, `internal/apply/apply.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `internal/mcp/surface128_test.go::TestACancelStopsAWritesRunningCheck`
**Served-path change:** over MCP, a first line that starts with a UTF-8 byte-order mark is read; a refusal no longer advises `--force`, which `mrw_write` does not take; an ack id that matches no checkpoint is named in the answer; and `notifications/cancelled` for an `mrw_write` whose check is running stops the check, which is then reported interrupted.

## Context

The Windows peers (BACKLOG "From the Windows peers") and ADR-121's review left four things on the MCP surface:

- A UTF-8 BOM on the first line dropped `initialize` with -32700 (`internal/mcp/mcp.go:378`): nothing strips U+FEFF before `json.Unmarshal`.
- Six read-before-modify refusals (`internal/apply/apply.go:1183-1223, 1433`) end "or pass --force"; `mrw_write` has no such argument, so the advice can never be followed there.
- An ack id that matches no pending checkpoint is skipped silently (`internal/mcp/ack.go:410`), so a caller who acked a page this session never issued learns nothing until a later write is refused.
- `notifications/cancelled` is dropped with every other notification (`mcp.go:395`), and `writeTool` verifies under `context.Background()` (`tools.go:886, 910`): a host's cancel leaves the check running to its own bound (BACKLOG "From ADR-121").

A replace whose body equals the line reports `written: true` with equal shas: a byte-identical body is still a write, and it stays one (Out of Scope).

**Audit of the class** — *an MCP answer that says less, or other, than the CLI's for the same event*: the four above, found by the peers and ADR-121's review; `mrw read --grep 'pass --force' internal/` (six) and the notification branch at `mcp.go:395`.

## Existing Primitives Audit

- **`check.Interrupted`** (ADR-080) — what a cancelled check already reports; a cancel reaches it through the context.
- **`callModern` / `callRelease`** (ADR-067, ADR-121) — per-call values set under `gate` and restored by `verifyUnlocked`; the call's context joins them.
- **`apply.Options`** — the engine's per-run options; `NoForce` joins `Force`.

## Decision

1. **A leading UTF-8 BOM on a request line is dropped before it is parsed.**
2. **A refusal on a surface with no force says only how to fix it**: `apply.Options.NoForce`, set by `mrw_write`, removes the ", or pass --force" clause from the six refusals; the CLI keeps it.
3. **An ack id that matches no pending checkpoint is named**: the answer's text opens with `-- ack: N id(s) matched no checkpoint and licensed nothing: …` (at most five named). It licenses nothing, as before; nothing is refused for it. The note is advisory and added only when the answer still fits the ceiling with it, so it never moves a write's floor (found by `TestAModernWriteReceiptIsBudgetedWithItsDecoration`: a refused write consumes its acks, and the retry at the ceiling its refusal named must still fit).
4. **`notifications/cancelled` stops a running check**: each `tools/call` runs under a context the server keeps by request id; the notification cancels it. A write's check under it is interrupted (ADR-080: exit-3 meaning), its write has landed, and the receipt says so. A cancel for a finished or unknown request is ignored, as the protocol says.

## Alternatives Considered

- **Strip the BOM only from the first line** — rejected: a BOM later in a stream is as unreadable, and nothing a host sends starts with one on purpose.
- **Refuse a call whose acks match nothing** — rejected: a stale ack from an earlier session would fail the spans acknowledged honestly beside it (ADR-031's reason for ignoring them).

## Component / Boundary Impact

`internal/mcp`, and `internal/apply` for one option and one text transform (engine: the refusal words, not the rules). `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| MCP request lines | a leading BOM is read | T1 | hosts |
| `mrw_write` refusals | no `--force` advice | T1 | MCP callers |
| `mrw_read` / `mrw_write` text | names unknown ack ids | T1 | MCP callers |
| `notifications/cancelled` | stops a running check | T1 | hosts |
| `scripts/contract.sh` | §226 | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a host that writes a BOM, acks a stale page, or cancels a long check gets an answer that matches what happened.
- **Negative:** a cancel now stops a check a host meant to cancel; one that cancelled by accident sees "interrupted" where it saw a verdict.
- **Neutral:** the CLI is unchanged.

## Out of Scope

- `written: true` for a byte-identical body (permanent: boundary: the file was rewritten — a new inode, a new modification time — and the receipt says so; equal shas say the bytes are the same)
- A cancel that stops a write before it lands (permanent: boundary: a write is all-or-nothing and fast; stopping one mid-commit is what ADR-066's PARTIALLY APPLIED exists to report, not a thing to invite)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a host cancels every slow call by reflex | Low | Medium | the check is interrupted, never misreported: the receipt still shows the landed write |

## Rollback

Revert the task: the four behaviours return. No receipt key changes.

## Follow-ups

- None — the record carries no open follow-up.
