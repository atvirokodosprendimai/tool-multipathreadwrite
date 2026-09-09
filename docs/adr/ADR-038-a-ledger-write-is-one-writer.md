# ADR-038: A ledger write is one writer, even across processes

**Status:** Accepted
**Accepted:** 2026-09-09 by M — *"accpeted the gate 2"*
**Date:** 2026-09-09
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-002 (the ledger this must not weaken), ADR-004 (the 2026-09-02 amendment this takes), ADR-010 (MCP serializes in-process; the CLI race was deferred), ADR-037 (deferred the same race), docs/adr/BACKLOG.md (From ADR-010, From ADR-037)
**Governs:** `internal/seen/**`
**Enforced-by:** `internal/seen/seen_test.go::TestConcurrentRecordsKeepEveryPath`
**Invalidates:** ADR-004 — the 2026-09-02 amendment clause reading *"It is not a defect to fix"*. ADR-002's refusal of locking the *target* file stands.
**Served-path change:** N concurrent `mrw read` processes against one checkout keep N ledger entries. The README and the MCP handshake stop teaching that parallel CLI processes overwrite one another.

## Context

**The class this record governs.** Every writer of the per-checkout ledger file. Enumerated
2026-09-09 with

```
rg -n 'seen\.Record' --glob '*.go'
```

Five live call sites, one function: `seen.Record` is the only writer (`save` is private). Callers
are `cmd/mrw` read, `cmd/mrw` write, MCP `readTool`, MCP write, MCP ack. Members left out: `Load`
(read-only) and `apply.Apply` (consumes a ledger, does not persist one).

M asked for this as priority #5 after ADR-037. The BACKLOG entry ADR-010 left, and ADR-037
re-receipted, is the CLI path's whole-file rewrite: 40 racing `mrw read` processes kept 5. Nothing
is corrupted and nothing is wrongly written — §24 pins that. The cost is a re-read. MCP does not
pay it, because one server is one process.

**What ADR-002 actually parked, and what later records read into it.** ADR-002's Out of Scope is
*"Locking, or any protection against two processes writing concurrently"* — a one-shot command, the
ledger detects a change to the *target* after the fact. ADR-004's 2026-09-02 amendment, and
contract §24, read that sentence as covering the *ledger file* too, and called the lost entries
accepted. Those are two files. This record locks the ledger. It does not lock the file a plan
edits.

**Why this is a record.** Taking it changes a public teaching (README, MCP `initialize`) and a
contract row that currently asserts loss. It does not change the ledger format. `go.mod` stays one
requirement.

## Existing Primitives Audit

- **`seen.Record` (load, merge, save):** **reused, wrapped.** The merge rules do not change. The
  second lookup is not reintroduced — ADR-029's one observation still holds; this is who may write
  the file those observations live in.
- **`state.Path(root, name)`:** **reused** for a sibling lock file `seen.lock` in the same
  directory. No new state root.
- **`syscall.Flock` / `LockFileEx`:** **taken**, behind build tags, so `go.mod` gains no module.
  `golang.org/x/sys` was audited and **not** taken — ADR-034 already refused a second requirement
  for a Windows ACL.
- **An append-only ledger, or a v3 header:** audited and **not** taken. The lost-entry failure is
  last-writer-wins on a whole-file rewrite. An exclusive lock around the existing rewrite closes it
  without discarding every on-disk ledger.
- **A `mkdir` lock:** audited and **not** taken. A crash leaves the directory behind; flock is
  released when the process dies.
- **Locking the target file a plan writes:** audited and **not** taken. That is ADR-002's permanent
  boundary and it stays one.

## Decision

**`seen.Record` holds an exclusive lock on `seen.lock` for the whole load-merge-save.** Every
caller already goes through `Record`. CLI processes, an MCP server beside a CLI, and two MCP
servers pointed at one checkout, serialize. The ledger format stays `#mrw-seen v2`.

1. **The lock is the ledger's, not the target's.** Two `mrw write` processes editing one source
   file remain last-writer-wins on that file, as ADR-002 accepted. The sha guard still fires
   sometimes and is still silent when it loses.
2. **`Load` stays unlocked.** A reader sees either the previous complete file or the next one.
   `os.WriteFile` under the lock is the existing save. A torn read during the write is unmeasured
   and is not this record.
3. **§24's safety property stays.** Writability still follows the ledger. What changes is the
   loss: 40 concurrent reads must keep 40. The skip-when-no-race branch becomes the pass, and a
   new §76 drives that through `$MRW`.
4. **Teaching changes in the same commit.** README's "40 racing reads kept 5" and the handshake
   clause *"parallel CLI processes race"* become false the moment the lock ships. Leaving them is
   how a caller is told to serialize calls that no longer need it.

**Go/no-go, checked during execution. If any fails, the task is withdrawn rather than shipped:**

- **`go.mod` still declares exactly one requirement** and it is `urfave/cli/v3`.
- **The ledger header is still `#mrw-seen v2`.** A bump that discards every existing ledger is the
  rejected alternative.
- **`internal/read`, `internal/apply`, `internal/plan`, `internal/check`, `internal/state` stay
  byte-identical** against the merge-base. This record owns `internal/seen` only.
- **`maxInstructionsChars` is still 4096.** Shorten MCP-only prose if the replacement clause does
  not fit; do not raise the bound.

**What would falsify this:** a filesystem where two processes holding `LOCK_EX` on the same
`seen.lock` both complete a `Record` that drops the other's paths. Then identity is not flock and
the format change comes back.

## Alternatives Considered

- **Leave it.** Today's state. Rejected because M named it next, the teaching is now a product
  claim, and MCP's "we serialize, they race" contrast becomes a lie the moment anyone runs two CLI
  processes on a machine that happens not to lose entries.
- **Append-only log, compact on load.** Rejected: a format bump (v2 was for poisoned whole-file
  entries) to fix a writer race the lock closes. Revisit if flock is unimplementable on a platform
  this repository ships.
- **Refuse any second process** (`seen.lock` held for the session). Rejected: mrw is one-shot; a
  lock that outlives the process is the `mkdir` hazard. The lock lives inside `Record`.
- **Hash-in-request, no ledger.** Rejected again, for ADR-010's reason: it moves the
  read-before-write guarantee into the caller.
- **Lock the target file.** Rejected: that is ADR-002's Out of Scope, and it is a different race
  (silent last-writer-wins on the source). This record does not reopen it.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/seen` | The ledger, and now the lock around writing it | Yes — `Record` is the one writer |
| `cmd/mrw`, `internal/mcp` | Unchanged callers of `Record` | Teaching text only (T3) |
| `internal/apply` | Unchanged | Untouched — it consumes a ledger |

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| ledger file | writers serialize; format unchanged | `seen.Record` | every `mrw read` / `write`, MCP read/write/ack |
| `scripts/contract.sh` | §76: 40 concurrent `$MRW read` keep 40; §24 keeps the safety assertion | T2 | CI |
| README + MCP `initialize` | stop teaching the race | T3 | callers choosing a surface |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `Record` holds the lock (T1) | T1 | T2, T3 | No — sequential callers unchanged |
| §76 (T2) | T2 | T3 | No — T3 is teaching |

## Implementation

See `docs/adr/ADR-038-a-ledger-write-is-one-writer/tasks/README.md`.

## Consequences

- **Positive:** parallel CLI reads of different files keep what they served. A CLI process beside
  an MCP server no longer clobbers it. §24's spurious fail (measure `kept`, then lose an entry
  before the writes) closes for the same reason.
- **Negative:** `Record` now waits. A hung process holding the lock stalls every other writer on
  that checkout until the kernel drops the flock.
- **Neutral:** one-file `mrw read` of many paths was already unaffected. The lock is paid by the
  parallel-process shape this record exists for.

## Out of Scope

- Locking the target file a plan writes (permanent: boundary: ADR-002 parked that; a silent last-writer-wins on the source is a different race)
- An append-only ledger or a v3 header (permanent: boundary: the lock closes this race without discarding on-disk ledgers)
- A shared lock on `Load`, or an atomic save via rename (deferred: docs/adr/BACKLOG.md)
- MCP tools for `check`, `iter`, `seen`, `stats` (deferred: docs/adr/BACKLOG.md)
- ADR-019 reach / multi-root (permanent: boundary: that number is reserved and this is 038)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The unit test is green without the lock (scheduler never interleaves) | **High** | High — a hollow fence | T1 forces the gap with an unexported hook between load and save, so the unlocked mutant drops entries. §76 is 40 processes, which is the measured race |
| Windows `LockFileEx` is wrong and CI skips rather than fails | Med | High | T1's Windows file is compiled, not skipped; §76 is Linux-only and says so |
| Handshake replacement overflows 4096 | Low | High | Go/no-go: shorten MCP-only prose; Stop Condition if the bound is the proposed fix |
| A one-sided contract row (kept == 40) passes a build that refuses every write | Low | High | §24's safety assertion stays: after the 40 reads, writability still follows the ledger |

## Rollback

Revert the commit. Delete `seen.lock` leftovers; they are empty and unused. No ledger format
change, so existing `seen` files stay valid. Additive lock; a caller who never raced cannot tell.

## Follow-ups

- [ ] If a torn `Load` during `WriteFile` is measured, take the deferred atomic save rather than
      adding a shared lock first.
