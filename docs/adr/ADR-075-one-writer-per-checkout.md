# ADR-075: One writer per checkout

**Status:** Accepted
**Accepted:** 2026-09-25 by M — answered *"Yes, ADR-075 one writer per root (Recommended)"* when the plan for the v1.25.1 adversarial round asked whether concurrent writers get a record of their own
**Date:** 2026-09-25
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-002, ADR-004, ADR-005, ADR-010, ADR-038, ADR-072, docs/adr/BACKLOG.md, docs/specs/2026-09-16-dangling-high-impact.md
**Governs:** `internal/writer/**`, `internal/seen/seen.go`, `internal/seen/writelock_test.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/instructions.go`, `internal/mcp/mcp.go`, `internal/mcp/mcp_test.go`, `internal/mcp/era_test.go`, `internal/mcp/testdata/legacy_golden.jsonl`, `internal/adversarial/concurrent_write_test.go`, `scripts/contract.sh`, `scripts/chaos.py`, `AGENTS.md`, `docs/adr/BACKLOG.md`, `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/specs/2026-09-16-dangling-high-impact.md`
**Enforced-by:** `internal/writer/writer_test.go::TestNoTwoWritersAreInsideAtOnce`
**Invalidates:** ADR-002 Out of Scope *"Locking, or any protection against two processes writing concurrently"*, for writers that are mrw; the BACKLOG "concurrent writes" entry's *"Locking stays permanently out of scope (ADR-002)"*; the 2026-09-16 spec's UC-3 postcondition *"no lock, no CAS"*; ADR-016 Decision 1b's routing clause *"OR when several callers share one checkout and want their ledger writes serialized"*, since the CLI's writes now take turns too
**Served-path change:** two `mrw write` processes on one checkout, or a write beside an `mrw mcp` server, no longer interleave. Each takes a per-checkout write lock after loading the ledger and holds it through apply and the ledger update; a writer whose file changed while it waited is refused, exit 1, "changed since mrw last saw it", nothing written. No writer exits 0 and loses its edit. The check runs after the lock is released. The MCP handshake and both tool descriptions stop routing callers who share a checkout to `mrw mcp` for serialized writes.

## Context

**What was observed.** The v1.25.1 adversarial round (2026-09-25; BACKLOG, "Two writers off one
read can both exit 0 with one edit lost") measured it on five full-scale corpora on Windows and
reproduced it on macOS: `scripts/chaos.py`'s race suite, eight writers off one read each replacing
a different line of one file, lost 45–53% of the edits with every loser printing `applied`, exit 0.
BACKLOG had recorded the mechanism on 2026-09-02: apply validates the file's sha against the ledger
and renames its result later, with nothing between two processes, so the later rename discards the
earlier edit. The same entry named the remedy — "a lock plus an under-lock sha recheck" — and parked
it under ADR-002's Out of Scope, before the rate was known.

**What ADR-038 already locks, and why it is not this.** ADR-038 serialises the LEDGER's
load-merge-save under `seen.lock`, so concurrent reads keep every entry. It deliberately did not
serialise a write's validate-and-commit, which is the race here.

**What the MCP surface said about it.** ADR-016 routed callers who share a checkout to `mrw mcp`
because one server is one writer while CLI processes race. With a write lock on the checkout, the
CLI's writers take turns too, and that routing would send a caller with a shell to the poorer
surface for nothing. The handshake, both tool descriptions and AGENTS.md say so (`internal/mcp/instructions.go:70-73`,
`internal/mcp/mcp.go:413`, `:481`, `AGENTS.md:121-123`), and `TestTheSurfaceSaysTheCLIIsRicher`
pins the word.

**Where the lock must start.** The ledger snapshot a writer validates against is loaded BEFORE the
lock, on purpose. Loaded inside it, the second writer would always see the first writer's
whole-file licence (ADR-002, ADR-005: a file mrw wrote is wholly known) and its line numbers, taken
from an older read, would apply to the new content with exit 0 — an edit misaddressed instead of
lost. Loaded outside, the snapshot is what this process read, and apply's own sha check, run under
the lock, refuses a file another writer changed meanwhile.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `seen.lock` and `withLock` | `internal/seen/seen.go:348-366` (ADR-038) | The flock/LockFileEx primitive to reuse. Not the same FILE: `Record` and `Drop` take it inside, and one process opening it twice would wait on itself. |
| apply's stale-sha check | `internal/apply/apply.go` (ADR-002) | The recheck. Run under the lock it becomes the guard, unchanged. |
| the ledger update after a write | `cmd/mrw/main.go:1141-1169`, `internal/mcp/tools.go:640-658` | The same loop twice; it moves into the one function both call. |
| the MCP `gate` | `internal/mcp/tools.go:26-40` | Serialises one server's calls in-process only; a CLI process or a second server races it. |

## Decision

1. **`internal/writer.Apply` is the one writer.** It takes `seen.LockWrites(root)`, applies the plan,
   drops and records what landed, and releases. The CLI write and the MCP write tool both call it;
   neither calls `apply.Apply` for a write any more.
2. **The write lock is its own file, `seen.write.lock`,** beside `seen.lock` in mrw's state
   directory (ADR-004). The order is always the write lock, then `seen.lock`; a read takes only
   `seen.lock`.
3. **The caller loads the ledger before the lock** (Context, "Where the lock must start").
4. **The surfaces stop selling serialized writes.** The MCP handshake and both tool descriptions
   route by shell alone, and say that writers take turns on either surface.
5. **The check runs after the lock is released.** A five-minute check must not hold every other
   writer.
6. **What it promises, and what it does not.** No two mrw writers on one checkout validate and
   commit at once, so none exits 0 having lost its edit. It does not lock the target against an
   editor or another tool; the sha guard still catches those after the fact (ADR-002). And a writer
   that STARTS after another has finished still writes on that writer's whole-file licence, because
   the ledger is per checkout, not per caller: M kept that licence on 2026-09-25 in the same plan.
   Its line addresses are then as good as its read; `anchor=` and `sha=` are how a caller pins them.

## Alternatives Considered

- **Load the ledger inside the lock.** Rejected: it turns a lost edit into a misaddressed one
  (Context).
- **Lock inside `seen.Record`'s existing `seen.lock`.** Rejected: `Record` and `Drop` already take it,
  and a nested open waits on itself.
- **Lock the target file.** Rejected: an editor does not take mrw's lock, so it protects nothing
  against the writers that are not mrw, and on Windows it would block their reads.
- **Two lock calls, one in each caller.** Rejected: the MCP `gate` means no in-process test can reach
  the MCP call site, so a dropped lock there would ship; one function both call is tested once.

## Component / Boundary Impact

`internal/seen` gains `LockWrites`; `internal/writer` is new. `internal/apply`, `internal/plan`,
`internal/read`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`,
`internal/rooted` and `internal/subproc` stay byte-identical.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` | validate-and-commit under the checkout's write lock | T1 | CLI |
| `mrw_write` | the same, through the same function | T1 | MCP |
| MCP `instructions`, tool descriptions | route by shell alone; writers take turns on either surface | T2 | hosts |
| contract §150 | eight writers off one read lose nothing | T1 | CI, `adr-verify` |
| `scripts/chaos.py` race suite | a lost edit fails by default | T2 | the chaos pass |

## Inter-task Contracts

T2 states what T1 makes true.

## Implementation

See `docs/adr/ADR-075-one-writer-per-checkout/tasks/README.md`.

## Consequences

- **Positive:** a racing writer is refused or lands; it is never told "applied" for an edit that is
  gone.
- **Negative:** a writer waits for another writer's apply and ledger update, not its check. A process
  that hangs while holding the lock stalls the others until the kernel drops the lock.
- **Neutral:** exit codes keep their meanings; the refusal is the existing stale-sha one.

## Out of Scope

- An editor or another tool writing the file mrw is writing (permanent: boundary: mrw can lock only its own writers; the sha guard reports the rest after the fact, ADR-002)
- A per-caller ledger, so a later writer does not inherit an earlier writer's licence (permanent: boundary: M kept the whole-file licence after a write on 2026-09-25; `anchor=` and `sha=` pin addresses)
- The klientams peer's unlocked `iter.go` and `authoring` writes and `mrw seen`'s unlocked load (deferred: docs/adr/BACKLOG.md "From the v1.25.1 adversarial round")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A filesystem without working flock lets two writers in | Low | Medium | ADR-038 rests on the same primitive; `chaos.py`'s race suite fails on any lost edit |
| A writer killed while holding the lock | Low | Low | the kernel releases a flock with its process |

## Rollback

Revert the two tasks. The lock file is state, not data, and is ignored by an older mrw.

## Follow-ups

- [ ] Release with ADR-071 to ADR-074 as v1.26.0.
