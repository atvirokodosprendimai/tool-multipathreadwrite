# ADR-127: the drift advisory names another writer's write

**Status:** Accepted
**Accepted:** 2026-10-06 by Zy — "Keep, widen drift (Recommended)", on the Windows peers' finding that a second writer's edit landed inside the first writer's check (BACKLOG "From the Windows peers"), and "plan these fixes then". The record's text was drafted after that answer.
**Date:** 2026-10-06
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-075, ADR-110, ADR-111, ADR-112, ADR-113
**Invalidates:** None — ADR-075 §5 stands: the write lock covers the apply, not the check
**Governs:** `internal/writer/writes.go`, `internal/writer/writer.go`, `internal/writer/flow.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/schema.go`, `docs/receipts.txt`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `internal/writer/writes127_test.go::TestAWriteThatLandsDuringTheCheckIsCounted`
**Served-path change:** a write whose check ran while another write landed in the same checkout carries `drift_writers: N` in its receipt and prints `drift: N other write(s) landed in this checkout while the check ran`; the exit code stays the check's.

## Context

A write's check runs after the write lock is released (ADR-075 §5, `docs/adr/ADR-075-one-writer-per-checkout.md:70`): a five-minute check must not hold every other writer. ADR-112 made the receipt name each file the write touched that changed while its check ran. A Windows peer measured what it does not see (BACKLOG "From the Windows peers"): a second writer's edit to ANOTHER file landed 1.4 s into the first writer's 8 s check, so the first check verified a tree holding an unverified edit, and nothing in its receipt said so.

Zy kept ADR-075 §5 and asked for the advisory to widen.

**Audit of the class** — *a place a write lands under the lock*: `mrw read --grep 'seen.LockWrites\(' internal/ cmd/` — `writer.Apply` alone; every surface lands through it (ADR-113).

## Existing Primitives Audit

- **`writer.Apply`** — holds the write lock for apply and the ledger update; the counter is bumped there, under it.
- **`state.Path` / `state.Write`** — the state directory and its atomic write (ADR-004); the counter is one more file there.
- **ADR-112's `Drift`** and the `drift` receipt key — the advisory this widens, shown on the same line family.

## Decision

1. **Every landed write bumps a counter** in the checkout's state directory (`writes`), under the write lock, after it lands — a partial commit included, a refusal or a dry run not.
2. **A landing remembers the counter it left**, and after its check ran the counter is read again: the difference is the number of other writes that landed in the checkout while the check ran.
3. **The receipt carries it**: `drift_writers` (a number, absent when zero and when no check ran) on `mrw write --json` and on `mrw_write`, appended to `docs/receipts.txt` (ADR-111); the human receipt and the MCP text print `drift: N other write(s) landed in this checkout while the check ran`. Advisory: the exit code stays the check's (ADR-112).
4. **A counter that cannot be read or written says nothing**: the advisory is best effort, as ADR-112's is; the write is not refused over it.

## Alternatives Considered

- **Hold the write lock through the check** — rejected by Zy's answer: a long check would block every other writer, the reason ADR-075 §5 exists.
- **Hash the whole tree before and after the check** — rejected: the cost grows with the checkout, paid on every checked write.

## Component / Boundary Impact

`internal/writer` and the two receipts. No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write --json`, `mrw_write` | `drift_writers` | T1 | callers |
| the human receipt and MCP text | `drift: N other write(s) …` | T1 | callers |
| `docs/receipts.txt` | two keys appended | T1 | ADR-111's tests |
| `scripts/contract.sh` | §225 | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a check that verified a tree another writer changed says so, whichever file the other writer touched.
- **Negative:** one small state-file write per landed write.
- **Neutral:** ADR-075's lock scope and ADR-112's per-file drift are unchanged.

## Out of Scope

- Naming which files the other writer touched (permanent: boundary: the count says the verdict is stale; the files are in that writer's own receipt)
- A write in the checkout that did not go through mrw (permanent: boundary: mrw sees only its own writes; ADR-112's per-file drift still covers the files this write touched)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the counter file is lost or garbled | Low | Low | read as zero; the advisory is silent, never wrong in the other direction by more than one reading |

## Rollback

Revert the task: the counter is no longer written, and the key is no longer emitted (ADR-111 lets a key stay absent).

## Follow-ups

- None — the record carries no open follow-up.
