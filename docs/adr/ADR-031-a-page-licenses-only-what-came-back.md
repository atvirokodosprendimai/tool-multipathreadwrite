# ADR-031: A page licenses only the part of it that came back

**Status:** Accepted
**Date:** 2026-09-07
**Owner:** M
**Accepted:** M, 2026-09-07, choosing "page acknowledgement" over a preimage echo and over documenting the gap: *"accepted, finish."*
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-014-a-read-too-large-is-a-first-page-not-a-dead-end.md`, `docs/adr/ADR-023-a-reads-answer-is-the-served-text.md`, `docs/adr/ADR-024-a-page-is-known-by-its-served-text.md`, `docs/adr/ADR-029-one-file-is-one-observation.md`
**Governs:** `internal/mcp/tools.go`, `internal/mcp/ack.go`
**Enforced-by:** `internal/mcp/ack_test.go::TestOnlyAckedSegmentsAreRecorded`
**Invalidates:** none — checked
**Served-path change:** an MCP read that PAGES now carries checkpoint markers in its served text and records nothing until the caller echoes them. A read that fits, a grep index and a refused multi-spec read are unchanged and carry none — the class is narrowed, not closed, and the small-read half is receipted in `docs/adr/BACKLOG.md`. A caller that echoes none is refused on its next write exactly as if it had not read.

## Context

Measured 2026-09-05 on Claude Code 2.1.261 at `d6c62e7`, recorded in
`docs/curve/reading-18-result.md` and in `docs/adr/BACKLOG.md` under ADR-023:

    mrw_read of a 3,619-line file  -> ADR-014's first page, lines 1-2727
    what the model received        -> lines 1-90, "[141140 characters truncated]", lines 2644-2727
    mrw seen                       -> lines 1-2727 (the page, recorded whole)
    @@ f.txt 1500 replace          -> ok, exit 0

Line 1500 is inside the discarded middle. **ADR-002 inverted**: mrw edited a file on the strength of
lines nobody had seen, and said `ok`. The phrase `[141140 characters truncated]` is nowhere in mrw's
source — the host did it, after mrw returned and before the model read anything.

**mrw cannot detect this from inside the server.** A cut result and a delivered one are identical to
it, which is why ADR-024 narrowed the exposure and filed the class rather than claiming to close it.
M chose the shape on 2026-09-07: a page is recorded only when the caller confirms receiving it.

⚠ **The obvious implementation does not work, and the measurement is what says so.** Put one
acknowledgement token at the END of the page and require it echoed: the host's cut took the MIDDLE
and left both ends, so the token survives, the caller echoes it honestly, and the whole page is
licensed including the 2,554 lines nobody saw. A single token proves the caller saw *a* part. Any
design here has to prove receipt *per region*, because that is the shape the damage actually has.

## Existing Primitives Audit

- **`seen.Observation.Spans [][2]int` (`internal/seen/seen.go:53`)** — already a set of spans, and
  `merge` already unions them. **Reused as-is**: recording three disjoint segments of a page needs no
  new shape, only a decision about which segments to record.
- **`seen.Record` (`:274`)** — unchanged. What changes is WHEN the MCP layer calls it.
- **ADR-014's paging and its `-- PARTIAL:` footer (`internal/mcp/tools.go:566`)** — the place the
  caller is already told what it holds and how to continue. **Extended**, not replaced.
- **`internal/state`** — where mrw keeps what belongs outside the tree (ADR-004). **Reused** for the
  pending record; nothing new is invented to hold it.
- **`anchor=`** — named in the BACKLOG as the candidate echo-back. **Rejected here**, see
  Alternatives: it is per-write, not per-read, so it cannot license a read at all.

## Decision

**An MCP read interleaves unguessable CHECKPOINTS through its served text, and records only the
segments whose checkpoints the caller echoes back.**

1. When the MCP layer serves a paged read, it BRACKETS every run of N served content lines:
   `-- ck <16 hex> open lines A-B (N lines follow)` before the run and `-- ck <16 hex> close` after
   it. The ids are random per read, so they cannot be PREDICTED or DERIVED. They can of course be
   RECALLED by whoever received one — that is what acknowledging is, and a pending record outlives
   the session that made it — so promotion CONSUMES the entry and a replayed id licenses nothing.

   ⚠ **The bracket is the whole design, and one marker per span was not enough.** A single marker
   FOLLOWING its lines is the page-level flaw at a smaller scale: a cut beginning inside the span and
   leaving the trailing marker licenses everything the caller did not receive. The first cut of this
   record did exactly that and claimed otherwise; the review of PR #132 found it. With brackets and a
   stated count, a caller that was cut holds one end, or neither, or too few lines — and can tell.
2. The observation goes to a PENDING record under the state directory, keyed by checkpoint. Nothing
   reaches the ledger yet.
3. `mrw_read` and `mrw_write` accept `ack`, a list of checkpoints. A checkpoint that matches a
   pending record promotes exactly its own span into the ledger, via the existing `seen.Record` —
   PROVIDED the file still holds the version that was served. A span acknowledged against a version
   since replaced is dropped: the ledger would refuse it anyway, and dropping it deterministically is
   what keeps a stale acknowledgement from overwriting a current one.
4. A checkpoint nobody echoes licenses nothing. A write addressing those lines is refused with the
   ledger's existing message, naming what WAS acked.

**On the measured failure this licenses NOTHING**, and the first cut of this record got that wrong
too. The host kept lines 1-90 and 2644-2727: the span opening at line 1 lost its close (which sits
after line 200), and the span containing 2644-2727 lost its open (which sits before 2601). Neither
was received intact, so an honest caller acknowledges neither and the probe write to line 1500 is
refused — along with every other line of that page. That is a stricter answer than the "1-90 and
2644-2727" this record first claimed, and a true one.

**What this proves, stated narrowly.** mrw still cannot detect truncation from inside the server, and
this does not change that. What changes is that an honest caller CAN now detect it: before, a cut
page and a whole one were indistinguishable to the caller too, so acknowledging was meaningless. The
count in the open marker is what makes the check possible. A caller that echoes ids without counting
is trusting itself, and no server-side design can stop that — ADR-002 never claimed otherwise.

**What would falsify this:** a host that truncates and then RECONSTRUCTS both markers and the right
number of lines between them. No host does that; a truncation notice is not a forgery.

## Alternatives Considered

- **One token at the end of the page.** Rejected on the measurement above: the observed cut kept both
  ends and removed the middle, so the token survives a truncation that destroyed 2,554 lines. This is
  the design most readers will propose, which is why it is named first.
- **A digest of the whole served page, echoed by the caller.** Strongest in principle — a truncated
  page has a different digest — and rejected because the caller is a language model: it cannot
  compute SHA-256 over 200,000 characters, and asking it to would produce a confident wrong answer
  rather than a refusal.
- **`anchor=` as the echo-back**, as `BACKLOG.md` proposes. Rejected: `anchor=` is a per-hunk guard on
  the WRITE path. It can prove the caller saw the first line of a range it is editing; it cannot
  license a read, cannot speak for lines no hunk touches, and ADR-028 has just finished making it
  strictly weaker than the ledger rather than an alternative to it.
- **Require the caller to echo the line numbers it received.** Rejected: line numbers are derivable
  from the request. A caller that received nothing can produce them.
- **Leave it, and document that the ledger is a served-record.** Rejected by M on 2026-09-07. It is
  the honest version of the current behaviour, but it leaves ADR-002 — the promise the tool is built
  on — false in the one deployment that matters.

## Component / Boundary Impact

`internal/mcp` gains one file, `ack.go`, holding the checkpoint format, the pending store and the
promotion. `internal/read`, `internal/apply`, `internal/plan` and `internal/seen` are unchanged: the
markers are inserted by the MCP layer over text `read.Run` already produced, and promotion calls
`seen.Record` as it stands. The CLI is untouched and takes no `ack` — there is no host between it and
the caller, which is the whole of what this record is about.

## Wiring & Contract Changes

- `mrw_read` and `mrw_write` gain an optional `ack` array of strings. Absent means "I acknowledge
  nothing", which is the safe reading and what an old caller sends.
- The MCP instructions gain the rule and the shape. They are under a hard 4,096-byte bound, so the
  text this adds is paid for by compressing prose elsewhere, not by raising the bound.
- Contract §68 drives it through the built server.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the checkpoint format and the pending store | T1 | T2, T3 | No — new surface |
| `ack` on both tools, and promotion | T2 | T3 | ⚠ Yes for MCP callers: a paged read licenses nothing until acked |

## Implementation

See `docs/adr/ADR-031-a-page-licenses-only-what-came-back/tasks/README.md`.

## Consequences

- **Positive:** the measured defect is closed at its actual shape. A cut anywhere inside a span
  leaves that span unacknowledgeable, which no single-token scheme can express — and, on the cut
  actually observed, licenses nothing at all rather than the two ends.
- **Positive:** the ledger stops recording a server-side belief and starts recording a caller-side
  fact, which is what ADR-002 always claimed it held.
- **Negative, and the reason this needs M's word:** an MCP caller that never sends `ack` can no longer
  write to anything it read through a paged response. That is a breaking change for every existing
  MCP caller, it fails safe, and the page footer says exactly what to send.
- **Negative:** the served text grows by TWO short lines per N lines. At N=200 a 2,727-line page pays
  twenty-eight marker lines, 1.03% of it — the first draft said "under one percent", which is the
  kind of number worth getting right in a record that spends its length on precision.
- **Negative, and the limit the mechanism cannot pass:** bracketing proves receipt per LINE and
  cannot prove it WITHIN one. A single line longer than the whole result cap would produce a one-line
  page that still exceeds it, and a head/tail cut of one numbered line leaves the open marker, the
  `NNN|` prefix and the close marker all standing — the caller satisfies the rule honestly while the
  middle never arrived. Such a read is therefore REFUSED rather than paged, which is the ordinary
  oversized-read path with its limit and line budget. Found by the fifth review of PR #132.
- **Negative, and named because it is the honest limit:** a cut that falls entirely between two spans
  — removing whole spans and nothing else — is indistinguishable to the caller from a page that never
  contained them, unless it notices the gap in the stated line ranges. The ranges are printed for
  that reason, but nothing forces a caller to read them.
- **Neutral:** the CLI is unchanged, so this repository's own `mrw` usage and `scripts/contract.sh`'s
  non-MCP sections behave identically.

## Out of Scope

- The CLI path (permanent: boundary: no host sits between `mrw read` and the caller, so there is nothing to truncate the served text between the two; the defect is a property of the MCP transport)
- Detecting truncation from inside the server (permanent: fact: a cut result and a delivered one are identical to the server, which is why ADR-024 filed the class rather than closing it; citation: file `docs/adr/ADR-024-a-page-is-known-by-its-served-text.md:34`)
- Making `MaxResultChars` caller-set (deferred: `docs/adr/BACKLOG.md` — M chose it on 2026-09-07 and it is its own record)
- Applying checkpoints to small reads that fit whole (deferred: `docs/adr/BACKLOG.md`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The fixture proves a single token would have passed | **High** | High | Pre-registered: the test must include a MIDDLE-CUT case, since the measured truncation kept both ends. A fixture that cuts the tail is green against the design this record rejects |
| A caller echoes every checkpoint reflexively without having received them | Medium | High | Unpreventable server-side and stated as such in the Decision. Checkpoints come from `crypto/rand`, so they cannot be predicted or derived; promotion consumes a pending entry, so a recalled id cannot be replayed. ⚠ The TEST for this rules out a dense counter and nothing more — a statistical test on output cannot establish unpredictability, and the guarantee is the `crypto/rand` dependency rather than the test |
| The pending store grows without bound | Medium | Low | Entries are dropped once promoted, and expire with the state directory they live in; T1 caps the count per root |
| Breaking existing MCP callers silently | **High** | High | It fails SAFE — a write is refused, never wrongly applied — and the refusal names `ack`. T3 asserts the refusal text carries the remedy, which is ADR-015's rule |

## Rollback

Revert the commit. The `ack` field becomes ignored, pending records are orphaned in the state
directory and harmless, and the ledger returns to recording pages when served.

## Follow-ups

- [ ] Re-run reading 18's fixture against a host that truncates, and record whether the middle is refused in practice rather than only in the test.
