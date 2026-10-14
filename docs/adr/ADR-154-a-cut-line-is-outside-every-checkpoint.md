# ADR-154: mrw_read takes max_cols, and a line it served cut is outside every checkpoint

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — approved the plan `glimmering-frolicking-harp` ("ADR-154 — `max_cols` on `mrw_read`") and set the goal "implement this plan end to end".
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-031, ADR-039, ADR-108, ADR-117, ADR-137
**Governs:** `internal/mcp/ack.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/mcp/maxcols154_test.go::TestACutLineIsOutsideEveryCheckpoint`
**Invalidates:** none — checked: ADR-137's CLI rule is unchanged; ADR-031's checkpoints still bracket whole numbered lines, and now only lines the read recorded
**Served-path change:** `mrw_read` accepts `max_cols` (an integer, 0 for no cut, a negative refused) and serves a line longer than that as `mrw read --max-cols` does; a line served cut is in no checkpoint, so acknowledging every id licenses the lines served whole and not the cut one, and a `-- not licensed:` line names it

## Context

ADR-137 gave `mrw read` a column window (`--max-cols N`) and kept it off `mrw_read`, because a checkpoint (`-- ck <id> open lines A-B (N lines follow)`, ADR-031) promises N whole numbered lines, and a line cut to a window is not a whole line. The 14-session survey (2026-10-09, BACKLOG "Look mode, what is left") asked for it on the MCP surface: an MCP-only host has no other way to look at a minified file or a log with 10,000-character lines without receiving them whole.

The CLI already records a cut line as not read (`internal/read/read.go`: the line ends one recorded span and the next starts after it). `read.Run` returns that record as the observation. The MCP read does not use it to place checkpoints: `interleave` (`internal/mcp/ack.go`) groups every numbered line of the served text, ending a group only at a gap in the numbers (ADR-108) or every `ckEvery` lines, and `promote` records an acknowledged checkpoint's span as it stands. Passing `max_cols` through alone would put a cut line inside a checkpoint, and an ack would license it.

**Audit of the class** — *code that turns served text into a licence on the MCP surface*: `mrw read --grep 'interleave\(|markServed\(' --exclude '*_test.go' internal/mcp/` names two producers of checkpoints, `markServed` (a fitting read) and `firstPage` (a page of an oversized one), both through `interleave`; `promote` is the one consumer. Both producers change; `promote` does not.

## Existing Primitives Audit

- `read.Options.MaxCols` and `window` (ADR-137) cut the line; `read.Run`'s observation leaves it out of the recorded spans.
- `seen.Observation.Covers` is the predicate the write's own licence check uses.
- `interleave` already ends a checkpoint at a gap (ADR-108); a line the read did not record is treated the same way.
- ADR-108's `-- not licensed:` line is how a served slice already says it licenses less than it shows.

## Decision

1. **`mrw_read` takes `max_cols`**, an integer: 0 cuts nothing, as the CLI's 0 does; a negative value is refused, as `max_lines` is (ADR-117), and so is a width beside `count`, which serves no line to cut (as ADR-153 refuses `max_lines` there). It reaches `read.Run` on the fitting read and on a first page, so a page of an oversized read is cut the same way; a continuation (`next_read`) is a new call and carries `max_cols` only if the caller sends it again.
2. **A checkpoint brackets only lines the read recorded.** For each served slice whose version is the observation's, a numbered line the observation does not cover ends the current checkpoint, as a gap does, and starts none: it is in no checkpoint's span. Acknowledging every id therefore licenses the lines served whole and never the cut one.
3. **The cut is known from the observation, not from the text.** The `[cols A-B of L]` marker is printed after the line's content, and a file can hold that text itself; a line whose content says `[cols 1-5 of 9]` is served whole, recorded, and bracketed like any other.
4. **The answer names what it left out**: after such a slice, one line — `-- not licensed: line(s) 2, 9 were served cut by max_cols and are in no checkpoint; read them without max_cols to edit them` — at most ten numbers, then `and N more`.

It fails if any checkpoint's span contains a cut line, if a write to a cut line applies after every id was acknowledged, or if a line served whole that happens to contain the marker is left out.

## Alternatives Considered

- **Find cut lines by their `[cols A-B of L]` marker in the served text.** Rejected: the marker is text a file can contain; the observation is the record the CLI already licenses from.
- **Give a cut line a checkpoint of its own marked partial.** Rejected: a checkpoint is a promise of whole lines (ADR-031); a second kind is a protocol change for no gain, since a cut line is not editable anyway.
- **Refuse `max_cols` together with a read that would page.** Rejected: an oversized file is the case `max_cols` is most wanted for.

## Component / Boundary Impact

`internal/mcp` only: the read tool's arguments and schema, and `interleave`'s grouping. `internal/read` is unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_read` arguments and input schema | `max_cols` | T1 | MCP hosts |
| `mrw_read` served text | cut lines in no checkpoint; one `-- not licensed:` line | T1 | the model reading it |
| AGENTS.md, README.md | `--max-cols` is no longer CLI only | T1 | agents, people |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/`.

## Consequences

- **Positive:** an MCP-only host can look at long lines without receiving them whole, under the same licence rule as the CLI.
- **Negative:** a read with many cut lines carries more, shorter checkpoints.
- **Neutral:** without `max_cols`, the observation covers every served line and the grouping is what it was.

## Out of Scope

- A partial-line licence (permanent: boundary: a checkpoint promises whole lines, ADR-031)
- `--max-cols` defaults or a host-chosen width (permanent: boundary: the caller asks for a cut, as on the CLI)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A cut line inside a checkpoint is licensed by an ack | Low | High | the test acknowledges every id and requires the write to the cut line refused, on a fitting read and on a page |
| A whole line containing the marker text loses its licence | Low | Low | the test requires that line's write to apply |

## Rollback

Revert T1: the argument, the schema entry, the predicate passed to `interleave`, and the doc sentences.

## Follow-ups

None.
