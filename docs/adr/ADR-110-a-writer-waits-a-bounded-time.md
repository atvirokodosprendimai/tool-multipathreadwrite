# ADR-110: A writer waits a bounded time, and names who it waited for

**Status:** Accepted
**Accepted:** 2026-10-01 by Zy — "work on the deffered ones", on the ADR-108 deferrals in `docs/adr/BACKLOG.md` "From ADR-108", of which B3 is this record's scope. The record's text, including the 120-second default, was drafted after that instruction and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-10-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-075, ADR-079, ADR-085, ADR-102, ADR-109
**Governs:** `internal/state/lock.go`, `internal/state/lock_unix.go`, `internal/state/lock_windows.go`, `internal/seen/seen.go`, `AGENTS.md`, `README.md`, `scripts/contract.sh`
**Enforced-by:** `internal/state/lock110_test.go::TestAWaitForAHeldLockIsBounded`
**Served-path change:** A write that finds this checkout's write lock held waits up to 120 seconds for it, then is refused, exit 2, saying nothing was applied and naming the holder's pid; `MRW_WRITE_LOCK_TIMEOUT` sets the wait in whole seconds (`0` refuses at once), and a value that is not one is refused the same way. Over MCP the refusal is the write's receipt with `isError`, as every apply error is. Exit codes keep their meanings.

## Context

ADR-075 makes writers take turns: `writer.Apply` holds `seen.write.lock` from validation through the ledger
update (`internal/writer/writer.go:75`), and `state.Hold` waits for it with a blocking `flock`/`LockFileEx`
(`internal/state/lock_unix.go:11`, `lock_windows.go:23`) — without a bound. The kernel releases the lock when its
holder dies, so only a LIVE holder that has stopped blocks the next writer, and it blocks it for ever with nothing
said (B3 in BACKLOG "From ADR-108"; the 2026-10-01 Codex design review). The check runs after `Apply` returns,
outside the lock (`writer.go:73`), so the lock covers validation and commit only: seconds at the scale campaign's
5,000 hunks across 500 files, never minutes.

**Audit of the class** — *a wait on a state lock*: `mrw read --grep 'state\.Hold\(' --exclude '*_test.go'
internal cmd` — every caller. Only `seen.LockWrites` is a writer waiting on another writer. The rest —
`seen.lock`, `iteration.lock`, `pending.json.lock`, the tally's — are taken for a load-change-save of a few
milliseconds, and their readers already read past a lock they cannot take (ADR-079, ADR-085); they keep `Hold`.

## Existing Primitives Audit

- **`state.Hold`** (`internal/state/lock.go:21`) — the one lock primitive; kept, and given a bounded sibling.
- **The apply-error receipt** — both surfaces already answer an error from `writer.Apply` with a receipt: the CLI
  exits 2 with it, MCP returns it with `isError` (`internal/mcp/tools.go:746`, ADR-102). No new path is needed.

## Decision

1. `state.HoldWithin(root, name, wait)` tries the lock without blocking (`LOCK_EX|LOCK_NB`;
   `LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY` on Windows) and retries with a growing pause until `wait`
   has passed; then it returns a `LockTimeoutError` naming the holder. A wait of 0 tries once.
2. Whoever takes a lock through `HoldWithin` writes its pid to `<name>.holder` beside it, so the next waiter can
   name it. A separate file because Windows locks a byte range: a waiter could not read the locked file itself.
3. `seen.LockWrites` waits through `HoldWithin`, for `MRW_WRITE_LOCK_TIMEOUT` seconds, 120 when unset. The default
   is a hang guard, not a performance budget: it only has to outlast the slowest legitimate validate-and-commit.

## Alternatives Considered

- **Bound every `Hold`** — rejected: the other locks guard millisecond rewrites, and their readers deliberately
  read past them; a timeout there adds a failure mode with nothing to guard.
- **Wrap the blocking call in a goroutine and abandon it at the deadline** — rejected: the abandoned call takes
  the lock later and holds it with nobody to release it.
- **Print a notice while waiting and never refuse** — rejected: a caller that is an agent cannot act on a notice it
  is blocked behind; a refusal it can read and report.

## Component / Boundary Impact

Engine packages owned: `internal/state` (T1), `internal/seen` (T2). `internal/read`, `internal/apply`,
`internal/plan`, `internal/check`, `internal/lines`, `internal/rooted` stay byte-identical; `go.mod` keeps one
requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `state.HoldWithin`, `state.LockTimeoutError` | a bounded lock wait that names the holder | T1 | T2 |
| `MRW_WRITE_LOCK_TIMEOUT` | the write-lock wait in seconds, default 120 | T2 | callers |
| a held write lock | refused after the wait, exit 2, nothing applied | T2 | CLI, MCP |
| `AGENTS.md`, `README.md` | say so | T2 | readers |
| `scripts/contract.sh` | §209 (T2) | T2 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `state.HoldWithin`, `state.LockTimeoutError` | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2.

## Consequences

- **Positive:** a stopped writer no longer stops every writer after it without a word; the refusal names it.
- **Negative:** a legitimate writer slower than the wait is refused; the variable lifts the bound.
- **Neutral:** a writer that is not contended takes the lock exactly as before.

## Out of Scope

- The other ADR-108 robustness items, B1, B2, B4 and B5 (deferred: `docs/adr/BACKLOG.md` "From ADR-108")
- A bound on the ledger, working-set, pending and tally locks (permanent: boundary: they guard millisecond rewrites and their readers read past them, ADR-079, ADR-085)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a stale `<name>.holder` names a pid that has exited | Medium | Low | the holder file is rewritten by every taker; the lock itself is the kernel's, so a stale name never refuses on its own |

## Rollback

Revert T1–T2. No receipt or format change; the holder file is ignored by an older binary.

## Follow-ups

- None — the record carries no open follow-up.
