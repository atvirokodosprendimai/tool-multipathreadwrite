# ADR-144: A damaged read ledger is told once

**Status:** Accepted
**Accepted:** 2026-10-10 on Zy's standing instruction "/loop continue delivering, end to end, no dead code", taking the open BACKLOG entry "A damaged ledger is ignored line by line without a notice" (wcag-web, v1.52.0 and v1.59.0). This is not a per-record yes: the pull request is where Zy can refuse it.
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-068, ADR-108, ADR-136, docs/adr/BACKLOG.md
**Invalidates:** None — no accepted record promised silence about a damaged ledger; it was an unstated gap
**Governs:** `internal/seen/seen.go`, `cmd/mrw/main.go`, `scripts/contract.sh`
**Enforced-by:** `internal/seen/damage144_test.go::TestADamagedLedgerIsCountedAndTold`
**Served-path change:** a CLI command that reads the ledger prints one stderr sentence when lines of the ledger could not be understood and were ignored, or when a line past the record bound made mrw discard the ledger. Exit codes, receipts and the ledger's format are unchanged.

## Context

A ledger line mrw cannot parse is skipped, and a ledger holding a line past the record bound is discarded whole; neither says a word. The Windows round on v1.52.0 and v1.59.0 appended a line of garbage, a 5 MB line and NUL bytes to the ledger: the first and third were skipped silently and the rest trusted, the second emptied the ledger. The direction is safe (a file the lost lines described counts as unread, and the write that needed them is refused "has not been read"), but the caller cannot account for that refusal, which is the failure ADR-136's stale notice exists for: a refusal the caller cannot account for looks like a bug, and "read it again" is the whole remedy.

**Audit of the class** — *a place where the loader drops ledger content without a word*: `mrw read --grep 'parseLine|ErrTooLong|IsStale' internal/seen/seen.go` names three: the header check (told, by `IsStale`), `parseLine` returning false (silent), and the `bufio.ErrTooLong` branch of `Load` (silent). This record tells the second and third. The count was taken on 2026-10-10.

## Existing Primitives Audit

- **`seen.IsStale` and `StaleNotice` (ADR-136)** — the pattern reused: a check beside `Load`, so `Load` keeps returning an empty ledger and no error and the next save heals the file; the CLI says so once from `Before`.
- **`parseLine`** — unchanged; the new check asks it the same question `Load` does, so the two cannot disagree about what is damage.
- **`regular.Open`, `scanLF`, `maxRecordBytes`** — used as `Load` uses them.

## Decision

1. **`seen.DamageNotice(root)` returns the sentence to print, or "".** It opens the ledger as `Load` does and says nothing for a missing, empty, non-regular or stale (header mismatch) ledger, which `IsStale` already handles.
2. **Unparsable lines are counted.** After the header, each line `parseLine` rejects, an empty line included, counts. N > 0 gives: `mrw: N line(s) of the read ledger could not be understood and were ignored; a file they described counts as unread, so read the files you mean to edit again.`
3. **A line past the record bound is told.** `Load` discards such a ledger; the notice is: `mrw: the read ledger holds a line longer than mrw writes, so it has been discarded; read the files you mean to edit again.`
4. **Told once because the call heals it.** The notice is printed from `Before`, ahead of the command; the command's own save rewrites the ledger without the lines, so the next run finds nothing to say. Commands that never read the ledger (`version`, `instructions`, `stats`) print nothing, as for the stale notice.
5. **No exit code, receipt key or format changes.** The notice is stderr text; a ledger line mrw cannot parse stays ignored.

## Alternatives Considered

- **Count inside `Load` and return it** — rejected: `Load` has many callers (the writer, the MCP server, `seen`, the working set) and every one would take a new return value for a notice only the CLI prints. A second pass of the ledger in the CLI costs one scan, as much as one `Load`.
- **Refuse the command when the ledger is damaged** — rejected: the safe direction already holds, and a refusal would stop the call that heals the file.
- **Authenticate the ledger (a checksum per line or per file)** — rejected for now: a format change and a new ledger version for a cache whose worst failure is one re-read; BACKLOG keeps the arm.

## Component / Boundary Impact

`internal/seen` (one function, one message), `cmd/mrw` (one branch in `Before`). The MCP server prints no stderr and is not changed. No other engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `seen.DamageNotice` | new function | T1 | `Before` in `cmd/mrw/main.go` |
| stderr of a CLI command with a damaged ledger | one sentence | T1 | CLI callers |
| `scripts/contract.sh` | §248 | T1 | CI Linux |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a refusal "has not been read" after a damaged ledger has a cause the caller was told.
- **Negative:** a ledger scan per CLI command that reads the ledger, besides the one the command makes.
- **Neutral:** a forged but well-formed line still licenses a write; the ledger is unauthenticated by design.

## Out of Scope

- A ledger cut after a complete line, which loses the lines after the cut (permanent: boundary: nothing in the file says how many lines it should hold; the lost lines count as unread and the write is refused, the safe direction)
- A well-formed line mrw did not write (permanent: boundary: the ledger is an unauthenticated cache of observations, and a caller who can write it can write the tree)
- The MCP server telling the caller (permanent: boundary: it has no stderr; its stale ledger is not told either, and a refusal there names the lines it needs)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a large ledger makes every command scan it twice | Low | milliseconds per MiB | the ledger is bounded by the files read in one checkout, and `Load` already pays the same scan |
| a ledger mrw wrote reports damage | Low | a false notice | the test writes through `Record` and asserts silence |

## Rollback

Revert T1: damaged ledgers are ignored silently as before. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the boundaries above.
