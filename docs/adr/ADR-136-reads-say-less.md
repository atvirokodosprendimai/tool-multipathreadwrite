# ADR-136: reads say less — the neighbour note once, the stale-ledger notice short, no notice on commands that never use the ledger

**Status:** Accepted
**Accepted:** 2026-10-10 by Zy — "/loop continue delivering, end to end, no dead code", on the survey's top-ranked pain ("reads are quiet", recommended 2026-10-09 and not declined)
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-052, ADR-068, ADR-109, docs/adr/BACKLOG.md
**Invalidates:** None — it changes how often and how long two messages are, not what they promise; ADR-052's refusal is untouched
**Governs:** `internal/read/read.go`, `internal/seen/seen.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `internal/read/quiet136_test.go::TestTheNeighbourNoteIsPrintedOncePerRead`
**Served-path change:** (1) a read that serves several multi-line ranges prints the "a multi-line replace of A-B needs a served line after B" note once, at the first, not at each; (2) the stale-ledger notice is one sentence; (3) `mrw version`, `mrw instructions` and `mrw stats` print no stale-ledger notice, since none uses the ledger. A write is refused, and every exit code and receipt is unchanged.

## Context

A survey of 14 mac and Windows sessions on 2026-10-09 ranked what makes mrw hard to use fluently. Two of its top items are noise, not missing function: the "multi-line replace needs a served line after" note, printed under every ranged read of two or more lines "including read-only ones", was acted on by none of the 7 of 9 mac sessions that named it; and the stale-ledger notice, 90 words listing four mrw versions, appears on `stats`, `version` and the first read after an upgrade, where five sessions found it unhelpful.

**What the note is for, and what it costs.** ADR-052 refuses a multi-line `replace` unless a line after its end was served, and the refusal names the line to read. The note tells the caller beforehand. It is printed for each range of a read: a read of eight ranges prints it eight times, though the rule is the same for each. The refusal, which names the exact line, stays; the caller who needs the note has it once in the call.

**The notice.** `seen.StaleNotice` says the ledger "was written by an older mrw, or its line endings were changed, and has been discarded", then spends 60 words on the history of four fixes (v0.0.11, v1.37.1, v1.41.0). Those fixes are in git and in the ADRs that made them (ADR-068 for the line endings); the reader of the message needs the two causes and the remedy. The root command's `Before` prints it for any subcommand, `version` and `stats` included, though neither reads the ledger.

**Audit of the class** — *a message mrw prints on a successful read-only call that no caller acted on*: `mrw read --grep '-- note:|-- skipped:|-- This serve|StaleNotice' --exclude '*_test.go' internal cmd` names the neighbour note (`internal/read/read.go`), the stale notice (`cmd/mrw/main.go`), the skipped line and the MCP licence line. The skipped line and the licence line carry state a caller needs (what was left out; what to acknowledge) and stay; the two named here are the ones the survey measured as unacted-on.

## Existing Primitives Audit

- **ADR-052's refusal** — names the line to read; the note is a courtesy ahead of it.
- **`seen.IsStale`** — already separate from `Load` (heal on write); the command that calls it is the only choice to change.
- **`cmd.Args().First()`** in the root `Before` — the subcommand name, after the root flags.

## Decision

1. **The neighbour note is printed once per read call**, at the first range that qualifies (two or more lines, not through the last line), in the same words. `Run` keeps the fact in a local across the specs it serves. A single-range read, the common case, is unchanged.
2. **`seen.StaleNotice` is one sentence**: "mrw: the read ledger was written by an older mrw, or its line endings were changed; it has been discarded. Read the files you mean to edit again." It still names both causes and the remedy; the history of the fixes lives in the ADRs.
3. **The root `Before` skips the stale check for `version`, `instructions` and `stats`**, the subcommands that never read the ledger. `read`, `write`, `check`, `iter`, `seen` and `mcp` keep it.

## Alternatives Considered

- **Remove the neighbour note entirely** — rejected: it is correct, and for a caller who has not met the refusal it saves a failed write; once per call keeps that and removes the repetition.
- **Print the note only on the refusal** — rejected: that is the refusal's own text already; the note's value is before the first failure.
- **Tell a newer ledger from an older one** — rejected here: the header is the only version marker and a newer writer's header differs the same way; naming the writer needs a ledger format change (a separate record).

## Component / Boundary Impact

`internal/read`, `internal/seen` (one constant) and `cmd/mrw` (one condition). `apply`, `plan`, `check`, `rooted` and `state` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read` output | the neighbour note once per call | T1 | CLI and MCP readers |
| stderr of every ledger-using command | the shorter notice | T1 | CLI callers |
| stderr of `version`, `instructions`, `stats` | no notice | T1 | CLI callers |
| `scripts/contract.sh` | §237 | T1 | CI Linux |
| `AGENTS.md` | the read paragraph says the note is printed once | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a read of many ranges is shorter by the repeats; the first read after an upgrade shows one line; `version` and `stats` are silent.
- **Negative:** a caller scanning for the note beside a later range no longer finds it there; the refusal still names the line.
- **Neutral:** nothing a caller can act on is removed; no exit code or receipt changes.

## Out of Scope

- The MCP surface's note (permanent: boundary: `mrw_read` shares `read.Run`, so it gets the same once-per-call behaviour; its licence line is a different message and stays)
- Reporting the writer's version of a discarded ledger (deferred: docs/adr/BACKLOG.md — the survey item "a ledger written by a newer mrw should be refused, not discarded" needs a header change)
- A column window around a match, `--count`, `-l`, `--limit`, JSON paths and a quicker throwaway create (deferred: docs/adr/BACKLOG.md — the survey's "look mode" and "cheap create")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a command that does use the ledger is skipped by the new condition | Low | a stale ledger is discarded without a word | the skip list is the three names; `TestTheLedgerCommandsStillAnnounceAStaleLedger` runs `read` and `write` |
| a script greps for the long notice | Low | its match fails | the notice keeps "written by an older mrw" and "line endings", the phrases the contract and tests grep |

## Rollback

Revert T1: the note repeats, the notice is long and printed everywhere again. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the deferred items above.
