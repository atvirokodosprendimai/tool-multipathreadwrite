# ADR-137: a read can cut a long line to a window around the match, and a cut line licenses nothing

**Status:** Accepted
**Accepted:** 2026-10-10 by Zy — "/loop continue delivering, end to end, no dead code", on the survey's most-named missing function ("no column window around a match in a very long line", 8 of 14 sessions as reported in the author's 2026-10-09 synthesis)
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-033, ADR-117, docs/adr/BACKLOG.md
**Invalidates:** None — a new flag; no accepted clause promises that a served line is whole
**Governs:** `internal/read/read.go`, `internal/read/maxcols.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/read/maxcols137_test.go::TestAnOverlongLineIsCutToAWindowAndLicensesNothing`
**Served-path change:** `mrw read --max-cols N` (CLI only) serves a line longer than N columns (characters) as a window of N columns around the first match of the read's pattern, or from the start when the range names no pattern, with `…` where text was cut and a `[cols A-B of L]` marker. A line served cut is NOT recorded as read: a plan that replaces it is refused as unread, exactly as for any line mrw did not serve. Without the flag, and for every line within the width, nothing changes. Exit codes do not change: a cut the caller asked for is not a failure.

## Context

Eight of the 14 sessions in the 2026-10-09 survey (as the author's synthesis recorded the replies) named the same gap: a JSONL log line, a 600-character table row or a minified file puts one line of thousands of characters under a pattern read, and the caller pays for all of it to see the 40 characters around the match. They reached for `cut`, `python` and `head -c`, which bypass the ledger.

**The constraint that makes this a record and not a flag.** What mrw records as read is what the caller was SHOWN, line by line (ADR-005); a write to a line mrw has not served is refused. A line printed cut was not shown whole. Recording it would license `@@ f 12 replace` on text the caller never saw: the failure the ledger exists to prevent. ADR-117 settled the same question for `--max-lines`: whatever is withheld is not observed. A cut line is withheld in the same sense, so it is left out of the recorded spans, and a read that cuts a line is a partial observation of its file.

**Audit of the class** — *a read option that serves less than a whole line or span*: `mrw read --grep 'MaxLines|Stat' --exclude '*_test.go' internal/read cmd` names `--max-lines` (withholds whole lines, reports them, exit 1) and `--stat` (serves none). `--max-cols` is the third, and the first to serve part of a line. The MCP `mrw_read` shares `read.Run` but takes neither the flag nor a checkpoint for a cut line (Out of Scope).

## Existing Primitives Audit

- **`read.Options.MaxLines`** (ADR-033, ADR-117) — the pattern for "withheld is not observed" and for a flag whose zero value must be safe; here the zero value is "no cut".
- **`served` spans in `Run`** — what `note` records; the cut line is simply not in them.
- **The span's own patterns** — `Range.Re` of the spec, the same expression that selected the line, finds the column.

## Decision

1. **`read.Options.MaxCols` (0 = no cut) and `mrw read --max-cols N`.** A negative N is a usage error, exit 2. The width is counted in characters (runes), not bytes.
2. **A line longer than N is printed as a window of N characters**: centred on the first match of any of the spec's patterns when the range has one (clamped to the line), else from its start. `…` marks each side where text was cut, and the line ends `  [cols A-B of L]` (1-based, inclusive).
3. **A cut line is not recorded as read.** The recorded spans are the printed lines minus the cut ones, so a span with a cut line in it is recorded as the lines on either side. The file is then not "wholly read": a later write to a cut line is refused "has not been read". A line within the width is unaffected.
4. **Nothing else changes**: the header, the `@@ A-B` address, the other lines, the exit code (0) and the receipt. A cut does not count as a problem, since the caller asked for it.

## Alternatives Considered

- **Record the cut line as read** — rejected: it licenses an edit of text the caller did not see (ADR-002, ADR-005).
- **Refuse to cut when the line could be edited** — rejected: the caller reading for orientation is the case the flag is for, and "could be edited" is every line.
- **Cut by bytes** — rejected: it splits a UTF-8 sequence; the survey's logs and tables are not ASCII.
- **Also add it to `mrw_read`** — deferred: a checkpoint (`-- ck … open lines A-B (N lines follow)`) promises N whole numbered lines the caller counts (ADR-031); a cut line inside a checkpoint changes that contract and needs its own record.

## Component / Boundary Impact

`internal/read` (one option, one small function, the print loop) and `cmd/mrw` (one flag). `apply`, `plan`, `seen`, `rooted`, `state` and `mcp` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --max-cols N` | new flag | T1 | CLI callers |
| `read.Options.MaxCols` | new field | T1 | `cmd/mrw` |
| `scripts/contract.sh` | §238 | T1 | CI Linux |
| `AGENTS.md`, `README.md` | the read section names the flag and the licence | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a pattern read over a log or a minified file costs the window, not the line, without leaving mrw's ledger.
- **Negative:** a caller who then wants to edit a cut line must read it again without the flag; the refusal names the line.
- **Neutral:** no change without the flag; no receipt key is added.

## Out of Scope

- `--max-cols` on `mrw_read` (deferred: docs/adr/BACKLOG.md — a checkpoint counts whole numbered lines, ADR-031)
- A files-only or count mode: `--stat` already lists the matching files with their length and sha, and `-N` drops the numbers (permanent: boundary: both exist; the survey sessions did not know them, which is a guidance question, BACKLOG)
- A JSON path address, `--limit` that keeps the exit code, a quicker create (deferred: docs/adr/BACKLOG.md — the survey's remaining look-mode items)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a cut line is recorded as read | Low | a write to unseen text is licensed | `TestAnOverlongLineIsCutToAWindowAndLicensesNothing` asserts the recorded spans exclude it and a plan to it is refused |
| the window misses the match | Low | the caller sees no match text | the first match of the spec's own pattern centres it; `TestTheWindowCentresOnTheMatch` |
| a multi-byte character is split | Low | invalid UTF-8 in the answer | the window is cut on runes |

## Rollback

Revert T1: the flag is gone and lines are served whole. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the deferred items above.
