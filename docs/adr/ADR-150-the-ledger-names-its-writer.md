# ADR-150: The read ledger names the mrw that wrote it, and a newer one is left alone

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — approved the plan `glimmering-frolicking-harp` ("ADR-150 — the ledger names its writer; a newer one is not overwritten") and set the goal "implement this plan end to end".
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-108, ADR-118, ADR-136, ADR-144, ADR-149
**Governs:** `internal/seen/seen.go`, `cmd/mrw/main.go`, `internal/writer/flow.go`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/seen/writer150_test.go::TestANewerLedgerIsNeitherUsedNorOverwritten`
**Invalidates:** none — checked: ADR-136's stale notice keeps its words for an older ledger; ADR-144's damage notice keeps its rule for a current one; a v4 ledger stays trusted (ADR-118 defined v4, and v5 changes only the header line)
**Served-path change:** the ledger's first line becomes `#mrw-seen v5 <writer>`; a v4 ledger is still read; a ledger whose format is newer than this mrw's is not used and not overwritten, and every command that reads the ledger says so once, naming its writer

## Context

The 14-session survey (2026-10-09, `docs/adr/BACKLOG.md`, "A ledger written by a NEWER mrw is discarded as older"; five sessions) found several mrw versions on one machine: a PATH binary, a build in a checkout, Claude Desktop's own copy. Each compares the ledger's first line to its own header byte for byte (`internal/seen/seen.go`, `Load`, `IsStale` and the damage count). Any other first line — an older format, a newer one, CRLF — is discarded, and the CLI prints "written by an older mrw, or its line endings were changed". So when two versions share a checkout, each throws away what the other recorded, and the newer one is told the older one is "older" when it was the other way round. Nothing in the file says who wrote it.

ADR-136 deferred this to "the next ledger format change". ADR-149 now gives a caller a way to learn which mrw is current (`mrw version --check-latest`), which is the remedy this notice can point at.

**Audit of the class** — *code that judges the ledger by its first line*: `mrw read --grep 'header' internal/seen/seen.go` names four sites: `Load`, `IsStale`, the damage scan behind `DamageNotice`, and `save`. All four change together. `seen.Record` is called from `cmd/mrw/main.go` (read), `internal/mcp/ack.go` (ack) and `internal/writer/writer.go` (after a write); none needs its own rule, because `Record` itself decides.

## Existing Primitives Audit

- `Load`, `IsStale`, `save` and the damage scan in `internal/seen/seen.go`; `StaleNotice` and `DamageNotice` (ADR-136, ADR-144) are how the CLI says a ledger was set aside, once, from the root command's `Before`.
- `versionString()` in `cmd/mrw/main.go` is what `mrw version` prints; `internal/seen` cannot import `main`, so the writer's name is handed in once at start-up.

## Decision

1. **The header names its writer.** `save` writes `#mrw-seen v5 <writer>`, where `<writer>` is `versionString()` (`v1.66.0 (abc1234)`, `dev (abc1234, dirty)`). A binary that was not told its version writes `unknown`.
2. **The format is read as a number.** A first line `#mrw-seen v<N>`, optionally followed by a space and a writer, is format N:
   - N = 4 or 5 is **current**: parsed. v4 entries mean what v5 entries mean (ADR-118 defined them; only the header changed), so an upgrade costs nobody a re-read.
   - N < 4, or a first line of any other shape (CRLF included), is **older**: discarded, and the CLI's notice is ADR-136's, word for word.
   - N > 5 is **newer**: this mrw cannot know what its entries mean, so it licenses nothing from it, and it does **not overwrite it** — `Record` leaves the file as it is, so a read by this mrw records nothing. The CLI prints one line instead of the stale notice:
     `mrw: the read ledger was written by a newer mrw (<writer>); this one is <ours> and leaves it untouched, so nothing it reads licenses a write. Use the newer mrw; mrw version --check-latest names the latest.`
3. **A write refused for an unread line while the ledger is newer says why.** The write path (CLI and `mrw_write`, `internal/writer/flow.go`) appends the same sentence to its refusal, since `mrw_write` has no stderr and a "has not been read" refusal after a read would otherwise look like a bug.
4. **What cannot change:** a binary already shipped (v1.65.0 and earlier) compares the line byte for byte, so it still discards a v5 ledger as "older" and overwrites it with v4 — which this mrw then reads. The record cannot reach those binaries; README says so.

It fails if a v4 or v5 ledger is not trusted, if an older one is, if a newer one licenses anything, or if a newer one's bytes change after a read or a write by this mrw.

## Alternatives Considered

- **Bump to v5 and discard v4.** Rejected: v4 entries are as trustworthy as they were, and the header comment's own rule bumps the version only when old contents can no longer be trusted.
- **Put the writer on a second comment line and keep v4.** Rejected: binaries since ADR-144 count an unparsed line as damage and would print the damage notice on every call.
- **Let a newer ledger be overwritten, with a better notice.** Rejected: the five sessions' cost was the lost observations; telling them more accurately while still destroying the newer binary's ledger keeps the cost.
- **Refuse every command while the ledger is newer.** Rejected: reads and `create` plans need no licence, and refusing them would stop a caller who only has the older binary at hand.

## Component / Boundary Impact

`internal/seen` owns the format (an engine package; this record owns the change). `cmd/mrw` hands in the writer name and prints the notice. `internal/writer/flow.go` adds the sentence to a refusal.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| ledger file, first line | `#mrw-seen v5 <writer>`; v4 still read | T1 | every mrw that shares the state directory |
| CLI stderr, once per command | the newer-ledger line | T1 | the caller |
| write refusal (CLI and `mrw_write`) | the same sentence when the ledger is newer | T1 | the caller, the MCP host |
| AGENTS.md, README.md | one sentence each | T1 | agents, people |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/`.

## Consequences

- **Positive:** a newer mrw's observations survive an older mrw on the same machine; every notice names the binary that wrote the ledger; upgrading costs no re-read.
- **Negative:** an older binary run after a newer one records nothing until the caller switches binaries.
- **Neutral:** the MCP read path prints no notice of its own (it has no stderr); the write refusal carries it.

## Out of Scope

- Binaries already shipped still call a v5 ledger "older" (permanent: fact: they compare the first line byte for byte; citation: file `internal/seen/seen.go:200`)
- Merging two versions' observations (permanent: boundary: an older binary cannot know what a newer format's entries mean)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A newer ledger is read as current and licenses a line it should not | Low | High | the test writes a v6 ledger whose entries would license the write, and requires the write refused |
| `Record` overwrites a newer ledger after all | Low | Medium | the test compares the file's bytes before and after a read and a write |

## Rollback

Revert T1. A v5 ledger written meanwhile is then "older" to the reverted binary and discarded once, which costs one re-read.

## Follow-ups

None.
