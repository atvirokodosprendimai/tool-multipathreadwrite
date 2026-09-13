# ADR-055: The receipt counts its advisories, notices a pattern, and can refuse the wrap-tail shape

**Status:** Proposed
**Date:** 2026-09-13
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-009, ADR-035, ADR-044, ADR-048, ADR-052, ADR-054, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `internal/apply/apply.go`, `internal/authoring/authoring.go`, `internal/mcp/schema.go`, `scripts/contract.sh`
**Enforced-by:** None — Proposed; T1 writes `cmd/mrw/advisory_test.go::TestTheSummaryLineCountsAdvisories` before the count exists
**Invalidates:** None — extends ADR-054 arm 2; ADR-054's "refuse on a balance delta" rejection stands for the general case and is narrowed here to one signature, opt-in
**Served-path change:** the write summary line gains `, N advisory/advisories`; the receipt gains `advisories: N`; the CLI receipt and `mrw stats` print one line when three of the last ten writes carried an advisory; `--strict-balance` refuses a single-line `replace` whose consumed lines have a non-zero delimiter net that the body does not match (exit 1, nothing written).
**Notes:** M's field run of v1.16.0/v1.16.1 in Zeus, 2026-09-13 (BACKLOG, From ADR-054). M's order by value per work: advisory count, then repeat pattern, then strict balance; the tasks follow it. M, 2026-09-13: *"so plan then backlog"* — design only. Execute on Accept. No default refusal. No harness `covers` glob. 4096 stays. MCP stays two tools.

## Context

**The class this record governs.** Every surface that renders a write's outcome — the human summary line, the JSON receipt, the MCP receipt, `mrw stats` — and the one refusal path ADR-052 left open. Enumerated 2026-09-13 with

```
git ls-files cmd/mrw/main.go internal/apply/apply.go internal/authoring/authoring.go internal/mcp/schema.go scripts/contract.sh
```

Five tracked files. Members left out: `internal/check` (unchanged); `internal/read` (a read has no advisories); `internal/mcp/tools.go` (records the tally today and keeps doing so; gains no flag — the pattern line is CLI and `stats` only, see Out of Scope).

**Why this is a record.** ADR-054 arm 2 put a `balance` row under the `ok` line. In the field run it fired correctly — `balance { +1 → +0` — and was not acted on, twice in one hour, because the line actually read every time is the summary `1 hunk(s), 1 file(s), 0 failed — applied`, which omits it, and the JSON consumer printed only `applied` and `failed` because those are the keys the summary taught it to care about. The tally (ADR-009) is cumulative and per-outcome: it cannot say "this is the fourth replace this session that carried a balance advisory", which is the signal that changes method instead of repeating it. And the shape that has now bitten four times is narrower than "any delta": a `replace` whose addressed lines have a non-zero net and whose body has a different one — the wrap-tail, which ADR-052's neighbour licence refuses only for multi-line addresses. Three of the four Zeus breakages carry the signature; the fourth (a balanced insert) has no delta and stays arm 1's.

## Existing Primitives Audit

- **`HunkResult.Balance` (ADR-054).** Reused as the one advisory that exists. "Advisory" is the class: a row that reports without failing the hunk. `Echo` (ADR-052) is not one — it is opt-in visibility the caller asked for.
- **`report` summary line (`cmd/mrw/main.go:1367`) and `receipt`.** One contract row and zero Go tests grep the summary; the count is added, not substituted.
- **`apply.Result`.** Gains `Advisories int` so CLI JSON and the MCP receipt carry the same number; `internal/mcp/schema.go` describes it (`TestEveryOutputSchemaPropertyIsDescribed` refuses an undescribed property).
- **`internal/authoring` (ADR-009).** Counts only, no paths, no plan text, fails open, never fails a write. Reused for a second file beside the tally: a recent-window ring of write outcomes — op class, advisory count, unix time — nothing ADR-009 refuses. Not a sixth Outcome; the vocabulary stays five.
- **ADR-052's neighbour licence at `apply.go:960`.** Refuses `end > start` without a served `End+1`. `--strict-balance` is the single-line sibling, keyed on nets rather than the ledger, and opt-in.
- **`--echo-pad` (ADR-052).** Precedent for an opt-in flag that ships first and is measured before any default is argued.
- **A parser.** Audited and rejected again (ADR-048): the signature is arithmetic on one hunk.

## Decision

Three arms, ordered by M's value per work. Arm 1 first.

**1. The summary line and the receipt count advisories.**

`report` prints `N hunk(s), M file(s), F failed, A advisor(y|ies) — <state>` — the count always present, including `0 advisories`, for the reason ADR-054 arm 3 gave for `failed_check 0`: a column that appears only when non-zero is a column the reader learns does not exist. `apply.Result.Advisories` is the number of `ok` hunks whose `Balance` is non-empty; JSON and the MCP receipt carry `"advisories": N`. `--quiet` prints the summary and so carries the count. A `skip`ped or `failed` hunk contributes nothing.

**2. A repeat-pattern line.**

Wherever `authoring.Record` runs for a write, the same call site appends one line `<unix-seconds> <op-class> <advisories>` to `recent` beside the tally, keeping the last 10. `op-class` is `write`; no paths, no plan text, no addresses (ADR-009). After a CLI write, when at least 3 of the last 10 recorded writes carried an advisory, `report` prints one line under the summary:

`pattern: 3 of your last 10 writes carried a balance advisory — read past the range before the next one`

`mrw stats` prints the same line when the condition holds and `recent: N write(s) in the window` always. The window is by count, not time: a session is what the caller is doing, not a clock. Thresholds 3 and 10 are constants named in one place. The ring fails open like the tally — unreadable means empty, never an error.

**3. `--strict-balance`, opt-in.**

With the flag, a hunk is refused when all of: `SrcOp == "replace"`; `Start == End`; the path is not prose (ADR-054's list); some family among `{}` `()` `[]` has a non-zero net in the consumed line; and the body's net for that family differs. The refusal reason names the family and both nets. A refused hunk is a failed hunk: siblings `skip`, nothing is written, exit 1 (ADR-001) — no new exit code. Without the flag the same hunk applies with the balance row and counts as an advisory. A multi-line address is ADR-052's territory and is not touched. MCP `mrw_write` gains `strict_balance` the way it gained `echo_pad` — a flag on the existing tool, not a third tool (ADR-044).

Default stays off. A pre-registration goes to BACKLOG in T4: the flag may be argued as default only after a campaign over three real corpora (Zeus, this repository, playtrix) reports the false-positive rate of the signature on plans that did not break the tree.

## Alternatives Considered

- **Print the advisory count only when non-zero.** Rejected: the ADR-054 stats lesson — a name omitted at zero reads as "never happens".
- **Put the pattern line only in `stats`.** Rejected: `stats` is run after the fact; the receipt is read in the turn that caused the breakage. Both.
- **A time window for the pattern.** Rejected: a clock does not know what a session is; the last N writes do. Constants, not flags.
- **Refuse on any balance delta.** Rejected in ADR-054 and still: string literals and generated code. This record refuses one signature, opt-in.
- **Make `--strict-balance` default now.** Rejected: unpriced. `--echo-pad` shipped opt-in and this follows it.
- **Extend the neighbour licence to single-line addresses.** Rejected in ADR-054: the ledger cannot see nets; the licence asks a different question.
- **A sixth authoring Outcome for "applied with advisory".** Rejected: the five names are the five exits; advisories are a count beside them.
- **A harness `covers` glob.** Rejected in ADR-054; M: not proposed again.

## Component / Boundary Impact

`internal/apply` computes `Advisories` beside `Balance` and hosts the strict-balance check next to the ADR-052 licence. `internal/authoring` gains a `recent` file and `Recent(root)` / `Pattern()` beside `Tally`; the vocabulary is unchanged. `cmd/mrw` renders the count, the pattern line and the flag; `stats` renders the window. `internal/mcp/schema.go` describes `advisories` and the `strict_balance` argument. No architecture doc; no new bounded context.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| summary line | `, A advisor(y|ies)` always | `report` | CLI; contract §92 |
| `apply.Result.Advisories` | new int; JSON/MCP `advisories` | `internal/apply` | `receipt`, `mrw_write`; §92 |
| `recent` file | last 10 write outcomes beside the tally | `authoring.RecordRecent` at every write `Record` site | `report` pattern line; `stats`; §93 |
| pattern line | on the CLI receipt and in `stats` when ≥3 of last 10 | `cmd/mrw` | CLI; §93 |
| `--strict-balance` / `strict_balance` | opt-in refusal of the signature | `internal/apply` via `Options.StrictBalance` | CLI, MCP; §94 |
| contract §92–§94 | next free after §91 | `scripts/contract.sh` | `adr-verify`, CI |

Exit codes unchanged. A strict-balance refusal is exit 1 with nothing written.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `apply.Result.Advisories` (T1) | T1 | T2, T4 | No — additive |
| `authoring.Recent` / pattern line (T2) | T2 | T4 | No — additive |
| `Options.StrictBalance` (T3) | T3 | T4 | No — opt-in |

T2 consumes T1's count to know whether a write carried an advisory. T3 is independent of both. T4 teaches all three.

## Implementation

See `docs/adr/ADR-055-the-receipt-counts-its-advisories-and-notices-a-pattern/tasks/README.md`.

## Consequences

- **Positive:** the number that was on the row is on the line every caller reads, and in the key every parser reads. A caller repeating the same mistake is told so in the turn. The wrap-tail signature can be made a refusal by anyone who wants it, today, without waiting for the campaign.
- **Negative:** the summary line grows by one clause and the one contract row that greps it moves. A second small state file beside the tally. `--strict-balance` will false-positive on braces in strings when opted in — that is the flag's stated bargain.
- **Neutral:** MCP writes feed the ring but do not print the pattern line (their receipt is structured; `advisories` is there). The balanced-insert case stays invisible to every arm here; arm 1 of ADR-054 is what catches it.

## Out of Scope

- A default `--strict-balance` (deferred: docs/adr/BACKLOG.md — pre-registered campaign, T4)
- The pattern line on the MCP receipt (deferred: docs/adr/BACKLOG.md — a host renders structure; `advisories` is on the receipt already)
- A parser or refusal on any delta (permanent: boundary: ADR-048; ADR-054 rejection stands for the general case)
- Extending the neighbour licence to single-line addresses (permanent: boundary: ADR-054 rejected it as a fix; the ledger cannot see nets)
- A time-based window (permanent: boundary: a clock does not know what a session is)
- A sixth authoring Outcome (permanent: boundary: five names are the five exits)
- Paths or plan text in `recent` (permanent: fact: ADR-009; citation: file `docs/adr/ADR-009-mrw-counts-what-happens-to-the-plans-it-is-given.md:91`)
- A harness `covers` glob (permanent: boundary: ADR-054 rejected it; M 2026-09-13: not proposed again)
- Raising 4096 / rewriting Shared() (permanent: boundary: teach on `write --help`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Scripts parsing the summary line break on the new clause | Low | Low | One contract row greps it; Go tests none; JSON is the stable surface |
| The pattern line fires on a legitimate run of brace-heavy edits | Med | Low | Advisory, not refusal; thresholds named in one place; `stats` shows the window |
| `--strict-balance` refuses a correct edit (brace in a string) | Med when opted in | Med | Opt-in; the reason names both nets; drop the flag for that plan |
| `recent` and the tally disagree after a crash mid-write | Low | Low | Both fail open; `recent` is advisory input only |
| Callers read the pattern line as a verdict | Low | Med | Wording says what it counted, not what is wrong |

## Rollback

Revert the branch. The summary line loses its clause; `advisories` leaves the receipt; `recent` on disk is ignored by an older binary (ADR-009's reader skips unknown files); `--strict-balance` becomes an unknown flag (usage, exit 2).

## Follow-ups

- Execute on Accept, T1 first. Do not start while Proposed.
- Pre-registered in T4: the default-`--strict-balance` question and its campaign criterion.
