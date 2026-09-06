# ADR-024: A page is known by its served text, not by an error flag

**Status:** Accepted
**Accepted:** 2026-09-06 by M — *"accepted"*, on this record as presented. M directed the investigation that produced it, and the framing is theirs: *"i feel that somehow we limit the mrw full potential. ho come the system limits this?"*, then *"rebuild, remove these self imposed nonsence"*. What that instruction turned out to name is in the Context: the cap it was aimed at is a measured memory guard, and the flag beside it was the actual defect. Quoted rather than inferred, per `.claude/rules/adr.md`.
**Date:** 2026-09-06
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-014-a-read-too-large-is-a-first-page-not-a-dead-end.md`, `docs/adr/ADR-017-the-mcp-surface-can-find-what-it-serves.md`, `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-023-a-reads-answer-is-the-served-text.md`, `docs/adr/ADR-011-the-mcp-server-tells-a-host-what-it-is-and-what-it-will-return.md`, `docs/adr/BACKLOG.md`
**Governs:** `internal/mcp/tools.go`, `internal/mcp/tools_test.go`, `internal/mcp/conformance_test.go`
**Enforced-by:** `internal/mcp/tools_test.go::TestAPageIsKnownByItsServedText`
**Invalidates:** ADR-014 — the clause of its Decision 2 reading "It is still `isError: true` when nothing was asked for narrowly enough"; ADR-017 — the clause of its index decision that keeps the flag, pinned by `TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd`
**Served-path change:** A `mrw_read` whose answer is a page or a grep index no longer sets `isError`, so a host stops treating it as a failed call and stops discarding its middle; the caller sees the whole page it was sent.

## Context

ADR-014 decided that an oversized read returns its first page, and marked that page `isError: true`
so "a caller that ignores the field is not silently handed a third of a file as if it were the
whole." ADR-017 gave the oversized grep index the same flag for the same stated reason.

**That flag is what destroys the page.** Measured 2026-09-06 against Claude Code 2.1.263 with
`claude-haiku-4-5` consumers, on a 3,328-line fixture whose lines are consecutively numbered so a
gap is detectable:

| `isError` on the page | served chars | what the consumer received |
|---|---|---|
| `true` | 152,594 | **GAPPED** — line 78 then line 2309; ~150 of 2,380 lines survived |
| `false` | 152,594 | **CONTINUOUS** — first line 1, last line 2380, no gap |

Nothing else differed between the two runs: same fixture, same page, same host, same consumer class.
The served bytes were read off the wire with a JSON-RPC client driving `mrw mcp` over stdio, not
from a model's account, and in both cases the page carried
`-- PARTIAL: lines 1-2380 of 3328. 948 line(s) remain.` in its text.

Size is not the explanation and was ruled out by the same campaign: an ordinary **196,672-character**
result arrived whole from the same consumers, while the **152,594-character** page — 44,000
characters smaller — was gutted. Hosts truncate error-flagged tool results head-and-tail; that is
what the flag buys.

The consequence reaches past readability into ADR-002. `internal/mcp/tools.go` records the page it
served in the read-before-modify ledger, on the premise stated in its own comment — *"The page WAS
shown, so it is recorded"*. When the host discards the middle, the ledger still licenses it: a plan
replacing line 1500, inside the discarded run, applied with `"status": "ok"` and exit 0 (measured
2026-09-05, recorded in `docs/adr/BACKLOG.md` under ADR-023). mrw edited a line its caller had never
seen. **The flag added to make partiality visible was making the page's middle invisible.**

ADR-023 already moved a read's answer into the served text and removed `structuredContent` from it.
This record finishes that arc: the label ADR-014 wanted survives, in the one channel a model reads
and no host rewrites.

## Existing Primitives Audit

- **The `-- PARTIAL:` notice** already exists in `firstPage` and already carries the line range, the
  file total, the count remaining, and the exact continuation spec. It is reused unchanged; nothing
  new is written to make partiality visible.
- **`next_read` in the receipt at `content[1]`** already names the continuation and is already
  asserted by `TestAPagedReadReassemblesTheWholeFile`. Reused as the second machine-readable signal.
- **`errorResult`** already exists for genuine refusals and keeps its flag. No new result shape is
  introduced; this record only stops two existing shapes from claiming to be failures.

## Decision
**1. An answer that SERVED something does not set `isError`.** `pagedResult`, `indexResult` and the
served-read path that passes `problems > 0` all return `isError` absent. `errorResult` is untouched:
a refusal served nothing, so it is an error, because it is one.

**2. Partiality is carried by the served text, and that is now the promise under test.** A page's
`content[0]` carries `-- PARTIAL: lines N-M of T. K line(s) remain.` and the continuation spec; its
`content[1]` carries `next_read`. Both already exist. The replacement promise is strictly stronger
than the flag it retires, because a host may reinterpret a protocol flag and does not rewrite served
text — which is the same reasoning ADR-023 applied to `structuredContent`.

**3. The class is every answer that carries content, and it has four members, not the one that was
measured.** Enumerated with

```sh
grep -n 'IsError' internal/mcp/tools.go
```

which returns the field declaration at `:48` and four constructions: `errorResult` (`:422` — a
refusal, served nothing, keeps the flag), `pagedResult` (`:637`, ADR-014), `indexResult` (`:814`,
ADR-017), and `readResult`'s `IsError: isErr` (`:90`), whose callers at `:202`, `:319` and `:841`
pass `problems > 0` — so **an ordinary read that served file content and hit one unreadable path
comes back flagged as a failure, carrying the content**.

⚠ **An earlier version of this record enumerated the class with `awk '/IsError: *true/'` and found
three sites.** That command matches only the literal, and `readResult` takes the flag as a
parameter, so the everyday case was missed by the very step meant to prevent missing it. The lesson
is recorded here rather than quietly fixed: a class-enumerating command must be checked against what
it CANNOT match, and the corrected command above is deliberately the broad one.

The `readResult` member is the most exposed of the four. A page needs a file over the cap; this
needs only enough served content to cross the host's threshold plus one bad path, which is an
ordinary multi-file read. The index was not separately measured; neither was this one. Both are
included because they share the flag, the code path and the host behaviour with the page.

**4. The ledger premise is NOT fixed here.** `tools.go` still records what it SENT rather than what
the caller SAW, and this record does not change that. It narrows the exposure to whatever else
truncates a result; it does not remove the class. That work stays in `docs/adr/BACKLOG.md` under
ADR-023, with `anchor=` named as a candidate echo-back.

**5. `MaxResultChars` does not move.** No number changes in this record.

**What would make this decision fail:** a paged read that arrives GAPPED with `isError` absent. The
fixture that could produce that failure exists today — `tmp/probe`-style consecutively numbered
fixtures above the cap — and the contract row in T2 drives exactly that shape through the built
server. The criterion is valid for hosts that truncate error-flagged results; a host that truncates
by size alone would be unaffected either way, and this record makes no claim about one.

## Alternatives Considered

- **Lower `MaxResultChars` so pages stay under the truncation threshold.** Rejected: the threshold is
  the host's, unmeasured, and not a constant we can track — a 196,672-character ordinary result
  survived while a 152,594-character page did not, so there is no size that is safe *as an error*.
- **Keep the flag and require the caller to re-read.** Rejected: the caller cannot see that anything
  was removed. That is the defect, not a workaround for it.
- **Record nothing in the ledger for a paged read.** Rejected by ADR-014 Decision 3 on its own
  merits — a partial read that licensed nothing makes paging useless — and it treats the symptom in
  the write path rather than the cause in the read path.
- **Set `isError` only when the caller asked for a whole file, not a range.** Rejected: it keeps a
  flag whose meaning to a host is "this call failed" on an answer that succeeded, and it makes the
  truncation intermittent rather than absent, which is harder to diagnose.

## Component / Boundary Impact

None — internal to `internal/mcp`. The engine packages (`internal/read`, `internal/apply`,
`internal/plan`, `internal/seen`, `internal/check`, `internal/state`) are untouched, per the standing
engine go/no-go.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_read` MCP result, paged shape | `isError` no longer set; `content[0]` `-- PARTIAL:` notice and `content[1]` `next_read` unchanged | `pagedResult` (`internal/mcp/tools.go`) | any MCP host; `internal/mcp/tools_test.go`, `internal/mcp/conformance_test.go`, `scripts/contract.sh` §62 |
| `mrw_read` MCP result, index shape | `isError` no longer set; `content[0]` report and `content[1]` `index` unchanged | `indexResult` (`internal/mcp/tools.go:814`) | any MCP host; `internal/mcp/tools_test.go`, `internal/mcp/conformance_test.go` |
| `mrw_read` MCP result, served-with-problems shape | `isError` no longer set; the per-path `-- <path>: <reason>` lines in `content[0]` and `problems` in `content[1]` unchanged | `readResult` via `:202`, `:319`, `:841` (`internal/mcp/tools.go`) | any MCP host; `internal/mcp/tools_test.go` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `pagedResult`/`indexResult` return `isError` absent | T1 | T2 | Yes — a host or test asserting `isError: true` on a page sees a changed result; that is the point of the record, and both in-repo assertions are rewritten in T1 |

## Implementation

See `docs/adr/ADR-024-a-page-is-known-by-its-served-text/tasks/README.md`. Two tasks, executed in
order: T1 changes the promise and the code; T2 drives the built binary.

## Consequences

- **Positive:** a paged read arrives whole. Measured: 2,380 of 2,380 lines against ~150 before.
- **Positive:** the ADR-002 inversion recorded under ADR-023 no longer fires on the paging path,
  because the ledger's premise and the caller's experience agree again on that path.
- **Positive:** the oversized grep index stops being exposed to the same silent shortening.
- **Negative:** a host that renders `isError` prominently loses one cue that an answer was partial.
  The `-- PARTIAL:` line is in the text it displays, so the cue moves rather than disappears — but a
  host that only surfaced the flag will look different.
- **Negative:** a caller that branched on `isError` to decide whether to continue paging must branch
  on `next_read` instead. That field already exists and is already the documented exit condition
  ("its absence is how you know you have the whole file"), so no new field is introduced.
- **Neutral:** `errorResult` is unchanged, so genuine refusals still arrive flagged.

## Out of Scope

- The read-before-modify ledger recording what was SENT rather than what was SEEN; `anchor=` as a
  required echo-back on the write path (deferred: `docs/adr/BACKLOG.md` under ADR-023)
- Making `MaxResultChars` a caller-set knob rather than one host's hardcoded ceiling (deferred: `docs/adr/BACKLOG.md` under ADR-023)
- Measuring where an ordinary, non-error result begins to be truncated; everything at or below 196,672 served characters arrived whole (permanent: boundary: this record needs only that the error-flagged shape is what triggers the truncation, which the A/B establishes; the ordinary-result ceiling is a separate question and changes nothing here)
- Any host other than Claude Code 2.1.263 (permanent: fact: the corpus's host measurements are Claude Code only and ADR-023 already carries the open item for other hosts; citation: file `docs/adr/BACKLOG.md:770`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A host uses `isError` as its only partiality cue and a caller stops noticing pages | Low | Med | The `-- PARTIAL:` line is in `content[0]`, which every measured host renders; T1's test asserts it is present and names the remaining count |
| The index change is unmeasured and behaves differently from the page | Low | Low | Same flag, same code path, same host behaviour; T1 covers it with its own assertion, and the record says plainly it was not separately measured |
| Removing the flag masks a genuine read failure | Low | High | `errorResult` is untouched and still flags refusals; T1's test asserts a real refusal is still `isError: true`, so the two shapes cannot collapse into one |
| A future edit re-adds the flag to a partial answer | Med | High | Contract §62 (T2) drives the built server and fails if a page comes back flagged |

## Rollback

Revert the commit. The change is three literal values in `internal/mcp/tools.go` plus the tests that
assert them; no persistent state, no schema, no migration, and no on-disk format is touched. A host
that preferred the old behaviour needs nothing from us — the previous binary serves it.

## Follow-ups

- [ ] Re-run the A/B on one non-Claude-Code host, and record it beside ADR-023's existing "other hosts" item.
