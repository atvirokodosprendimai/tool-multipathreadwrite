# ADR-025: A read that served nothing is an error, whichever path produced it

**Status:** Accepted
**Date:** 2026-09-06
**Owner:** Zy
**Accepted:** Zy, 2026-09-06 — *"accepted"*, after the condition was corrected from `problems > 0 && len(observed) == 0` to `len(observed) == 0` in response to the cold review.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-014-a-read-too-large-is-a-first-page-not-a-dead-end.md`, `docs/adr/ADR-017-the-mcp-surface-can-find-what-it-serves.md`, `docs/adr/ADR-023-a-reads-answer-is-the-served-text.md`, `docs/adr/ADR-024-a-page-is-known-by-its-served-text.md`, `docs/adr/BACKLOG.md`, `README.md`
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/tools_test.go::TestAReadThatServedNothingIsAnError`
**Invalidates:** ADR-024 — the clause of its Decision 1 reading "the served-read return at `:319` (with its size probe at `:841`) leave the key absent", in the single case where that return served NOTHING. Its four enumerated members all served something and are untouched; so is the `:202` branch it deliberately left flagging.
**Served-path change:** An `mrw_read` whose every named path was unusable returns `isError: true`, where v1.2.0 returns the key absent. A read that served anything at all is unchanged.

## Context

ADR-024 stopped an answer that SERVED something from claiming to be a failure, because a host read
the flag as a failed call and discarded the middle of a page. Its T1 step S10 changed the served-read
return at `internal/mcp/tools.go:319` from `problems > 0` to an unconditional `false`. That is right
for the shape the record enumerated — a read that served content beside a path it could not use — and
it overshoots by one case, because the same return also handles a read that served nothing at all.

Measured 2026-09-06 at main `d8a1d0f`, driving a server built from that tree over stdio, root holding
only `a.txt`:

| request | served | `observed` | `problems` | `isError` |
|---|---|---|---|---|
| `specs: ["a.txt"]` | a file | 1 | 0 | absent (correct) |
| `specs: ["a.txt:99"]`, a range that misses | no lines | 1 | 1 | absent |
| `specs: ["empty.txt:1"]`, an empty file | no lines | 1 | 1 | absent |
| `specs: ["nope_dir"]` | nothing | 0 | 1 | absent |
| `specs: ["nope_dir"], grep: "X"` | nothing | 0 | 1 | `true` |
| `specs: ["a.txt", "nope_dir"]` | a file | 1 | 1 | absent (ADR-024, intended) |

The two middle rows are the ones that decide the shape of the condition, and they are not what they
look like: a range that misses and an empty file are both OBSERVED — `internal/read` notes the file
with empty spans and counts a problem — so what excludes them from any served-nothing rule is the
observation, never the problem count.

Two calls with the same outcome disagree, and what decides is whether `grep` was passed: the walk's
no-match branch at `:202` flags on `len(walkProblems) > 0`, and the served-read return at `:320`
cannot flag at all.

`README.md:36` states the property the corpus believes it has — *"A refusal that served nothing still
carries the flag."* — and the v1.2.0 binary does not have it. The enforcing test looks like it covers
this and does not: shape 4 of `TestAPageIsKnownByItsServedText` builds its "refusal that served
nothing" from `exclude` without `grep`, which is an `errorResult` refusal and never reaches the
served-read return. Green fixture, unexercised branch.

Raised by M's review of `ae8ce33` on PR #122, posted at 12:07:47Z — four minutes after that PR merged
as `d8a1d0f` at 12:02:55Z and 52 seconds before v1.2.0 was published, so it reads as "fix before
merge" while describing shipped text. M had raised the same gap at all four heads of PR #118, where it
merged with the code and the record disagreeing and was recorded nowhere. Found by that review, not by
this session.

## Existing Primitives Audit

- **`errorResult`** already flags genuine refusals and is reused unchanged. No new result shape.
- **The `:202` no-match branch** already carries exactly this rule for the walk path, with the comment
  that states the principle: "a walk that could not LOOK somewhere is a different answer again, and it
  is an error". Reused as the precedent rather than reshaped.
- **`readResult`'s third parameter** already exists and already takes an expression on the `:202`
  side. This record changes the expression passed at one call site; it adds no parameter and no
  branch of its own.
- **`TestAPageIsKnownByItsServedText`** is extended rather than replaced — its four members stay,
  and ADR-024 keeps its Enforced-by test.

## Decision

The served-read return at `internal/mcp/tools.go:320` passes `len(observed) == 0` instead of `false`.
An answer that delivered none of what was asked for is an error; an answer that delivered anything is
not, exactly as ADR-024 decided. This is ADR-024's own stated principle — *"an answer that delivered
what was asked for does not set `isError`"* — applied to the one member its enumeration missed, and it
makes the `:202` and `:320` paths agree so that passing `grep` no longer decides whether "I could not
look" is an error.

The condition is the observation count alone, and that is a correction to this record's first draft,
which conjoined `problems > 0` and justified it with two examples that do not hold. A read of an
empty file, and a range that matches nothing, both COUNT a problem and are excluded because they are
observed; and on this path `len(observed) == 0` already implies `problems > 0`, because a read naming
no spec at all is refused earlier at `:158` and never reaches here. The conjunct could therefore
never discriminate, and no mutation could kill it.

**What would make this decision fail:** a host that discards the content of a flagged result, on an
answer whose content is worth reading. The served-nothing answer's whole value is its `-- <path>:
<reason>` lines, so if the flag costs a reader those lines the change is a net loss. Data that could
produce that failure exists and points the other way: ADR-024's A/B truncated a 152,594-character
page, and a 196,672-character ordinary result arrived whole, while a served-nothing answer is a few
hundred characters — two orders of magnitude below anything measured to truncate. Valid for Claude
Code 2.1.263, the only host this corpus has measured; ADR-023's other-hosts item already carries the
rest.

## Alternatives Considered

- **Narrow the README sentence to the branch that is true, and file the asymmetry.** Rejected: it
  documents the disagreement rather than removing it, and leaves `grep` deciding whether an
  unreadable path is an error. It is the cheaper fix and it was offered; it buys a true sentence and
  keeps the defect.
- **Flag whenever `problems > 0`, as the code did before ADR-024.** Rejected: that is precisely what
  ADR-024 removed, on a measurement — a page served beside an unusable path is a served answer, and
  flagging it is what got its middle discarded.
- **Stop `:202` flagging instead, so both paths agree the other way.** Rejected: it inverts the
  principle both records rest on, and a walk that could not look where it was told would report a
  searched-and-empty tree, which is the defect that branch's comment was written against.
- **Leave it and record the gap in BACKLOG.md.** Rejected: the README makes it a promise to a reader
  rather than a disagreement between two internal documents, and the fix is one condition.
- **Conjoin `problems > 0` with the observation count, as a guard against a future path that serves nothing without counting a problem.** Rejected on measurement: empty specs are refused at `internal/mcp/tools.go:158` before this return, and every spec that serves nothing counts a problem, so the conjunct cannot discriminate today and nothing could prove it ever does. A conjunct no mutation can kill is the shape this repository refuses in a test, and it is no better in a condition.

## Component / Boundary Impact

None — internal to `internal/mcp`. The engine packages (`internal/read`, `internal/apply`,
`internal/plan`, `internal/seen`, `internal/check`, `internal/state`) are untouched, per the standing
engine go/no-go.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_read` MCP result, served-nothing shape | `isError: true` where the key was absent; the report in `content[0]` and `problems` in `content[1]` are unchanged | the served-read return at `internal/mcp/tools.go:320` | any MCP host; `internal/mcp/tools_test.go`, `scripts/contract.sh` §63 |
| `mrw_read` MCP result, every other shape | unchanged — page, index, served-with-problems stay unflagged; `errorResult` and `:202` stay flagged | `pagedResult`, `indexResult`, `errorResult`, `:202` | any MCP host |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the served-read return flags when `len(observed) == 0` | T1 | T2 | Yes — a host or test asserting the key is absent on a wholly-unusable read sees a changed result; that is the record |

## Implementation

See `docs/adr/ADR-025-a-read-that-served-nothing-is-an-error/tasks/README.md`.

## Consequences

- **Positive:** the README sentence becomes true of the binary, and two calls with the same outcome
  stop disagreeing on the flag.
- **Positive:** a caller that branches on `isError` learns that a read gave it nothing, without
  parsing the report text.
- **Negative:** a host that renders a flagged result less prominently, or collapses it, shows the
  per-path reasons less prominently than it does today.
- **Negative:** a caller that treats any flagged result as fatal now fails on a wholly-mistyped read
  where it previously received an empty answer. That is the intent, and it is a behaviour change on a
  released contract.
- **Neutral:** `problems` and the served report are unchanged, so a caller reading either sees no
  difference.

## Out of Scope

- The `:202` walk branch's own flag, which already behaves this way (permanent: boundary: this record removes the disagreement by changing the side that is wrong, and ADR-024 deliberately left that branch alone)
- A read that serves no LINES but whose file was observed — an empty file, a range that matches nothing (permanent: boundary: `internal/read` notes the file and counts a problem, so the caller learned the file exists and its sha; flagging it would widen this record from "you got nothing" to "you got fewer lines than you asked for", which is a different decision)
- The read-before-modify ledger recording what was SENT rather than what was SEEN (deferred: `docs/adr/BACKLOG.md` under ADR-023)
- Making `MaxResultChars` a caller-set knob rather than one host's hardcoded ceiling (deferred: `docs/adr/BACKLOG.md` under ADR-023)
- Any host other than Claude Code 2.1.263 (permanent: fact: the corpus's host measurements are Claude Code only and ADR-023 already carries the open item for other hosts; citation: file `docs/adr/BACKLOG.md:762`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A host discards the report text of a flagged result, losing the per-path reasons | Low | Med | The answer is a few hundred characters against a measured truncation threshold two orders of magnitude higher; §63 asserts the reasons are still in `content[0]` of the flagged result |
| The condition catches a shape that legitimately serves nothing | Low | Med | Conjunctive on `problems > 0`; T1's test pairs a wholly-unusable read against a served read and against a zero-problem read |
| The change is read as reverting ADR-024 | Med | Low | The Invalidates header names the single clause and the single case; ADR-024's four enumerated members are asserted unchanged in the same test and in §62, which stays |

## Rollback

Restore the unconditional `false` at the served-read return and delete §63; ADR-024's §62 and its
test are untouched by this record and continue to pass either way. No persistent state, no schema and
no ledger format changes, so rollback is the one-line revert plus the row.

## Follow-ups

- [ ] If a host is measured that truncates a flagged result at a few hundred characters, revisit the Decision's falsifiability paragraph rather than relaxing the test.
