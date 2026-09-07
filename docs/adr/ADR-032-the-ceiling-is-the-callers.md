# ADR-032: The ceiling is the caller's, and it bounds the whole answer

**Status:** Accepted
**Date:** 2026-09-07
**Owner:** M
**Accepted:** M, 2026-09-07, choosing the shape from three questions put to them: the budget is *"Caller-set, bound whole result"*, `--max-lines 0` and by inheritance every numeric limit here means *"0 means zero"*, and the standing direction for this session is *"keep closing the open issues, highest rank. the product must not deteriorate and make the premise of his - false."*
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-011-the-mcp-server-tells-a-host-what-it-is-and-what-it-will-return.md`, `docs/adr/ADR-014-a-read-too-large-is-a-first-page-not-a-dead-end.md`, `docs/adr/ADR-024-a-page-is-known-by-its-served-text.md`, `docs/adr/ADR-031-a-page-licenses-only-what-came-back.md`, `docs/adr/ADR-033-a-cap-of-zero-is-a-cap.md`
**Governs:** `internal/mcp/schema.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `cmd/mrw/main.go`
**Enforced-by:** `internal/mcp/limit_test.go::TestTheAdvertisedCeilingBoundsEveryAnswer`
**Invalidates:** none — checked
**Served-path change:** `mrw_write` now returns a bounded receipt instead of one that can exceed the cap it advertises, and both tools' ceiling can be set by the caller.

## Context

Two defects, one cause: the ceiling is a constant one host chose, and only one path enforces it.

**1. `mrw_write` advertises a cap it does not keep.** Both tools carry
`_meta["anthropic/maxResultSizeChars"] = MaxResultChars` (`internal/mcp/mcp.go:254` and `:320`), and
the read path bounds its report with `capped` (`tools.go:226`). The write path bounds nothing.
Measured 2026-09-07 against `267c453`: a 4,000-hunk dry-run plan returned **453,632 characters**
against an advertised 200,000. A host that trusts the number truncates, and ADR-031 exists because a
truncated answer is one mrw cannot see.

**2. The number is Claude Code's, hardcoded.** `schema.go:174` says so itself — "The value is Claude
Code's per-tool ceiling" — while mrw runs under any MCP host. A host with a larger budget cannot use
it; a host with a smaller one is not protected.

**And what is bounded is the report text, not the answer.** `capped` limits the text mrw composes;
the RESULT also carries the receipt in `content[1]`, and for a WRITE in `structuredContent` as well.
⚠ A READ carries none — ADR-023 removed it after a host delivered the receipt to the model instead of
the lines — and an earlier draft of this paragraph said it did. ADR-024's own note records the gap it
leaves regardless: 178,494 characters of report inside a 794,582-character result. A caller that
budgets against the advertised number is budgeting against the wrong quantity.

M chose the shape on 2026-09-07: caller-set, bounding the whole encoded result, with `mrw_write`'s
cap enforced, and explicitly NOT `0` = unlimited.

## Existing Primitives Audit

- **`capped` (`internal/mcp/tools.go:466`)** — an `io.Writer` that keeps at most `limit` bytes and
  records what it dropped. **Reused as-is for the READ path only.** ⚠ An earlier draft of this line
  said the write path would get the same type; it does not, and could not usefully — `capped` bounds
  a STREAM as it is produced, and a write receipt is a value that exists in full before it is
  rendered. The write path measures the composed result with `encodedSize` and elides, which is a
  second mechanism for a second shape rather than reuse avoided.
- **`servedOrIndex` (`tools.go:874`)** — already decided "the encoded answer will not fit, so return
  the thing that does". **ABSORBED, not reused, and the function is gone.** It composed a PROBE beside
  the answer and measured that; the same judgement now measures the answer itself, composed once and
  returned. Measuring a shape that is not what gets sent is the mistake ADR-031 made twice in
  consecutive reviews, and a probe cannot be wrong about the thing it IS. Its walked-only condition
  went with it: that narrowness was the read half of this defect.
- **`_meta["anthropic/maxResultSizeChars"]` (`mcp.go:254`, `:320`)** — the advertisement. **Reused**,
  now carrying the configured value, which is the point: `mcp.go:81` already promises the advertised
  limit and the enforced one cannot drift, and today that promise holds only for reads.
- **ADR-033's `--max-lines` decision** — settles the shape of a numeric limit in this codebase: zero
  means zero, and absence means "no opinion". **Inherited, not re-argued.**

## Decision

**The ceiling is a caller-set budget, it bounds the ENCODED RESULT, and every tool obeys it.**

1. `mrw mcp --max-result-chars N` and `MRW_MAX_RESULT_CHARS`, defaulting to today's 200,000 when
   neither is given. `0` means zero — a server that may return nothing — per ADR-033; absence is how
   a caller asks for the default.
2. The bound is measured on the ENCODED result, after the receipt and any structured content are
   composed, not on the report text alone. ADR-031 already had to move its page check for exactly
   this reason: the composed answer is what a host receives.
3. `mrw_write`'s receipt is bounded like a read's. Over budget, the per-hunk detail is elided and the
   verdict counts are kept — a receipt that says "3 of 4,000 hunks failed, detail elided" is useful,
   and one that is truncated by the host is not.
4. `_meta` advertises the configured value on both tools.

**What would falsify this:** a host that cannot express its own budget, so that the knob is only ever
the default. Then the change costs a flag nobody sets and the enforcement half still stands on its
own.

## Alternatives Considered

- **Enforce `mrw_write`'s cap and leave the number hardcoded.** Rejected as half the defect: the
  advertisement would be honest and still wrong for every host but one.
- **Make the number caller-set and leave the write path unbounded.** Rejected the same way, and worse
  — it makes the advertised number configurable while one tool still ignores it.
- **`0` = unlimited.** Rejected by M on 2026-09-07, consistent with ADR-033 and with `body=0`.
- **Bound the report text only, as today.** Rejected: it is the quantity the caller does not receive.
  ADR-024 measured 178,494 against 794,582.
- **Drop the cap and let hosts truncate.** Rejected on measurement: `tools.go` records an uncapped
  40 × 18 MB read peaking at 2.6 GB against 87 MB capped. The guard is not ceremony.

## Component / Boundary Impact

No new component. `internal/mcp` gains a configured field where it had a constant; `cmd/mrw` gains
one flag and reads one environment variable. `internal/read`, `internal/apply`, `internal/plan`,
`internal/seen` and `internal/state` are untouched.

## Wiring & Contract Changes

- New flag `mrw mcp --max-result-chars N` and env `MRW_MAX_RESULT_CHARS`.
- `_meta["anthropic/maxResultSizeChars"]` becomes the configured value on both tools.
- `mrw_write` gains a bounded receipt and an elision notice.
- Contract §70 drives all of it through the built server.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the configured budget replacing the constant | T1 | T2, T3 | No — the default is today's number |
| the bounded write receipt | T2 | T3 | ⚠ Yes for a caller parsing every hunk out of a very large receipt |

## Implementation

See `docs/adr/ADR-032-the-ceiling-is-the-callers/tasks/README.md`.

## Consequences

- **Positive:** the advertised number and the enforced number are the same on both tools, which is
  what `mcp.go:81` already claims and what only reads deliver today.
- **Positive:** a host with a different budget can say so, and mrw stops encoding one host's number.
- **Negative:** a write of thousands of hunks returns an elided receipt. That is a real loss for a
  caller that reads every verdict, and it is the trade against a host truncating the whole answer —
  which loses the same detail plus the counts, silently.
- **Neutral:** the default is unchanged, so a caller that sets nothing sees today's behaviour.

## Out of Scope

- The CLI's own output, which streams and has no such ceiling (permanent: boundary: no host sits between `mrw read` and the caller; ADR-031 draws the same line for acknowledgement)
- Reads that FIT recording on serve rather than on acknowledgement (deferred: `docs/adr/BACKLOG.md`)
- Changing what `capped` does to a read's report (permanent: fact: it is reused unchanged and this record adds a second measurement point rather than altering the first; citation: file `internal/mcp/tools.go:466`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Measuring the "encoded result" measures the wrong thing again | **High** | High | The bound is asserted against the ACTUAL JSON the server writes, not against a reconstruction of it — ADR-031 shipped a check on a buffer that was not what got sent, twice |
| The write elision hides the verdict a caller needed | Medium | Medium | The counts and every FAILED hunk are kept; only successful hunks are elided, since a failure is what a caller must act on |
| A caller sets a tiny budget and every answer becomes a refusal | Low | Low | That is the caller's choice made visible, and `0` meaning zero is the extreme of it (ADR-033) |

## Rollback

Revert the commit; `MaxResultChars` returns to a constant and the write path to an unbounded
receipt. No persistent state is involved.

## Follow-ups

- [ ] If a second host is measured with a different budget, record what it advertises and whether the default should follow it.
