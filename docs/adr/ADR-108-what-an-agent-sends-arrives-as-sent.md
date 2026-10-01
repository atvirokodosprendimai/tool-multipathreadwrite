# ADR-108: What an agent sends arrives as sent

**Status:** Accepted
**Accepted:** 2026-10-01 by Zy — "go ahead fixing these codex findings", on the 2026-10-01 Codex solution design review of `98feab5` (v1.37.1), whose seven defects A1–A7 are this record's scope; A8 and A9, from the Codex review of this record's PR (#304), joined it under the same instruction. The record's text was drafted after that instruction and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-10-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-031, ADR-039, ADR-065, ADR-071, ADR-073, ADR-104, ADR-107
**Governs:** `internal/mcp/ack.go`, `internal/mcp/tools.go`, `internal/links/links.go`, `internal/plan/plan.go`, `internal/ingest/applypatch.go`, `internal/seen/seen.go`, `internal/iter/iter.go`, `scripts/contract.sh`
**Enforced-by:** `internal/mcp/gap108_test.go::TestACheckpointCoversOnlyConsecutiveServedLines`
**Invalidates:** ADR-104 Out of Scope *"`apply`'s load and the state files (permanent: boundary: … state files are written by mrw)"*, for the ledger: a record mrw writes can exceed what its own loader reads, so the ledger gets a record bound on both sides (T5). ADR-107's audit, which searched `apply`, `mcp`, `writer` and `cmd/mrw`, is extended to `ingest` and `plan` (T4).
**Served-path change:** (1) An MCP read that serves separated lines brackets each run of consecutive lines with its own checkpoint, so acknowledging it licenses exactly the lines served; a sparse read used to license the gaps. (2) On Windows, a filename holding a character whose low byte is `/` or `\` (`Я`, `Ŝ`) is no longer split into directories. (3) A `body=@file` body is split like every other text in mrw, so a CRLF body into a CRLF file no longer writes `\r\r\n`. (4) A foreign-format document or a body file over 1 GiB is refused, naming its size and the limit. (5) A ledger record mrw cannot read back is not written; the file it describes needs reading again. (6) MCP acknowledgement of a file that is no longer a regular file returns at once and licenses nothing. (7) An MCP read whose served lines hold bytes that are not valid UTF-8 serves them without checkpoints and says so, since JSON replaces those bytes; the check is per served range, so a valid range of such a file is checkpointed as before. (8) An MCP read too large for its ceiling no longer blocks on a FIFO, or reads an oversized file, while counting lines for a first page. (9) `mrw iter note` or `iter add` of a line over 64 KiB is refused, exit 2, naming its size; a working set holding such a line, from an older binary, loads without it. Exit codes keep their meanings.

## Context

The Codex solution design review of 2026-10-01 (`98feab5`, `codex-design3`) found seven defects, A1–A7, each
source-traced and two of them checked against the source by this session before acting:

1. **A1** — `interleave` (`internal/mcp/ack.go:138-161`) groups served lines into checkpoints by count with no break
   at a gap, and records each group as `start..end`; `markServed` holds those spans and acknowledgement promotes
   them, so a read serving lines 1 and 100 licensed 1–100.
2. **A2** — `components` (`internal/links/links.go:104`) narrows each rune with `uint8(r)` before
   `os.IsPathSeparator`; `Я` (U+042F) narrows to `/`, so the Windows link walker splits `aЯb.txt`.
3. **A3** — `LoadBodyFiles` (`internal/plan/plan.go:897`) splits a body file on `\n` alone, keeping each `\r`.
4. **A4** — `targetBytes` (`internal/ingest/applypatch.go:307`) and `LoadBodyFiles` (`plan.go:889`) read whole
   files with `os.ReadFile`, before apply's capped loader.
5. **A5** — the ledger loader's `bufio.Scanner` keeps its default ~64 KiB token limit (`internal/seen/seen.go:134`)
   while the writer puts every span of a path on one line (`seen.go:277`, `:435`): a record past 64 KiB fails every
   later `Load`, and `Record` loads first, so no read can repair it.
6. **A6** — `currentSHA` (`ack.go:458`) opens and streams the path without checking it is still a regular file: a
   FIFO swapped in blocks the open, under the pending-store lock.
7. **A7** — an MCP read prints raw bytes; JSON replaces invalid UTF-8, while the checkpoint licenses the original.
8. **A8** — found by the Codex review of #304: `countFileLines` (`internal/mcp/tools.go:1240`), which `firstPage` runs
   before paging a read that overflows the ceiling, used `os.ReadFile` on the joined path. A refusal that itself
   overflows a small ceiling reaches it, so a FIFO blocked the server and a file of any size was read.
9. **A9** — same review: `iter.Save` (`internal/iter/iter.go:118`) wrote a line of any length while `load` kept
   `bufio.Scanner`'s ~64 KiB default (`iter.go:93`): a long note saved, then every load — every write, and
   `mrw iter clear` — failed with `token too long`.

**Audit of the classes.** For A2, *a rune narrowed to a byte*: `mrw read --grep 'uint8\(r\)|byte\(r\)' --exclude
'*_test.go' internal/ cmd/` — **1** site. For A4, *a whole-file read before the capped loader*: `mrw read --grep
'os\.ReadFile' --exclude '*_test.go' internal/ingest/ internal/plan/` — **2** sites (`targetBytes`, shared by both
compilers, and `LoadBodyFiles`). A1 and A7 share the one place checkpoints are made, `interleave`, which both the
fitting serve (`markServed`) and the first page (`tools.go:1170`) call.

The review of #304 widened two classes. *An open of a caller-named path outside the engine packages*: `mrw read
--grep 'os\.(Open|OpenFile|ReadFile)\(' --exclude '*_test.go' internal/mcp internal/ingest internal/plan
internal/authoring internal/writer internal/guide cmd/mrw` — **10** sites: `countFileLines` (A8); `currentSHA` (A6);
`targetBytes` and `readBounded` (A4, each refusing a non-regular file by `Stat` before it opens); `loadPending` and
the three `authoring` loaders, which read mrw's own state; and the two `cmd/mrw` opens of a plan or a
`--files-from` list, where a pipe is a legitimate input. *A `bufio.Scanner` with no `Buffer` call*: `mrw read --grep
'bufio\.NewScanner|\.Buffer\(' --exclude '*_test.go' internal cmd` — **5** unbounded: `iter.load` (A9); the three
`authoring` loaders, whose records are fixed-shape lines mrw writes and which fail open; and `seen.IsStale`, which
reads only the header line mrw writes.

## Existing Primitives Audit

- **`lines.Split`** (`internal/lines/lines.go:14`) — the one line model (ADR-065), used for body files now.
- **`read.readCapped`** (`internal/read/read.go:62`) — the size-then-bounded-read pattern copied into `ingest` and `plan`.
- **The stale-ledger rule** (`seen.go:152-160`) — a ledger mrw cannot honour is discarded, never parsed loosely; T5
  applies it to an over-long record.

## Decision

1. **A1** `interleave` ends a checkpoint when the next served line is not the next number.
2. **A2** `components` treats a rune as a separator only when it is ASCII and the OS calls it one.
3. **A3** `LoadBodyFiles` splits with `lines.Split`.
4. **A4** `targetBytes` and `LoadBodyFiles` refuse a file over 1 GiB by size and read through a bound of the limit
   plus one byte.
5. **A5** The ledger has a record bound, `maxRecordBytes` (16 MiB): `save` leaves out a record longer than it (that
   file needs reading again; coverage is never widened), and `Load` reads lines up to it; a longer line, which only
   an older binary can have written, discards the ledger as a stale header does.
6. **A6** `currentSHA` opens without blocking and refuses a descriptor that is not a regular file.
7. **A7** `interleave` issues no checkpoint for a served slice that is not valid UTF-8 and prints why; a valid slice
   of the same file is checkpointed, since JSON carries its bytes unchanged.
8. **A8** `countFileLines` resolves the path inside the root, opens without blocking, refuses a descriptor that is
   not a regular file, refuses past 1 GiB by size, and reads through a bound. `firstPage` already declines when its
   own read serves nothing (ADR-025), so a refused target is never paged; no second guard is added at its call site.
9. **A9** The working set has a line bound, `maxEntryBytes` (64 KiB): `Save` refuses a note or entry past it, naming
   the size (the caller typed it, so it is not dropped as a ledger record is), and `load` reads lines through a
   reader that skips a longer one and keeps the rest.

## Alternatives Considered

- **Transcode non-UTF-8 content for MCP** — rejected: it changes bytes nobody named; the CLI serves them as they are.
- **Merge sparse spans when a ledger record is long** — rejected: merging licenses the gaps, the defect A1 fixes.
- **Raise the Scanner limit alone** — rejected: the writer would still be unbounded; both sides need the same number.

## Component / Boundary Impact

Engine packages owned: `internal/plan` (T3, T4), `internal/seen` (T5); `internal/links` (T2), `internal/ingest` (T4),
`internal/mcp` (T1, T6, T7, T8) and `internal/iter` (T9) are not on the engine go/no-go list. `internal/read`,
`internal/apply`, `internal/state`, `internal/lines`, `internal/rooted`, `internal/check` stay byte-identical;
`go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_read` checkpoints | one per run of consecutive lines | T1 | MCP hosts |
| Windows paths | a non-ASCII rune is never a separator | T2 | every Windows caller |
| `body=@file` | split like every other text | T3 | plan authors |
| foreign formats, `body=@file` | over 1 GiB refused by size | T4 | callers |
| ledger | record bound on save and load | T5 | every run |
| `mrw_read` acknowledgement | a non-regular file returns at once | T6 | MCP hosts |
| `mrw_read` | a served range not valid UTF-8 gets no checkpoint | T7 | MCP hosts |
| `mrw_read` first page | its line count opens only what read would serve | T8 | MCP hosts |
| working set | a line bound on save and load | T9 | `mrw iter`, every write |
| `scripts/contract.sh` | §202 (T1), §203 (T3), §204 (T8), §205 (T9) | T1, T3, T8, T9 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| none | — | — | the nine tasks are independent |

## Implementation

See `tasks/README.md`: T1–T9, one per defect, one wave. T8 and T9 came from the Codex review of #304.

## Consequences

- **Positive:** a licence over MCP covers exactly what was served; bytes a plan sends land as sent; no entry point
  reads a file without a bound; the ledger always reads back what it saved.
- **Negative:** a non-UTF-8 file cannot be edited over MCP (the CLI edits it); a file over 1 GiB cannot be patched;
  a working-set note or entry over 64 KiB is refused.
- **Neutral:** everything else is unchanged.

## Out of Scope

- The review's robustness improvements B1–B5 (deferred: `docs/adr/BACKLOG.md` "From ADR-108")
- A file swapped for a FIFO between a loader's path check and its open, in `ingest`, `plan`, `read`, `apply` and ast-grep's probes (deferred: `docs/adr/BACKLOG.md` "From ADR-108")
- Contract rows for A2, A4–A7 (permanent: boundary: A2 runs only on Windows, which contract.sh does not; A4 needs a file over 1 GiB; A5 needs a 16 MiB ledger record; A6 a file swapped for a FIFO between a read and its acknowledgement; A7 is asserted at the JSON wire by its unit test; each task's test drives it)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| more checkpoints per sparse read | High | Low | the markers are bounded per run; a host acknowledges each it holds |
| a legacy over-long ledger is discarded | Low | Low | one re-read; the stale-ledger rule already accepts this cost |
| a long note an agent writes is refused | Low | Low | the refusal names the size and the 64 KiB limit |

## Rollback

Revert T1–T9. No receipt change; the ledger and working-set formats are unchanged.

## Follow-ups

- None — the record carries no open follow-up.
