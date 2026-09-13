# ADR-056: The receipt carries the pattern on both transports, and the tally prices `--strict-balance`

**Status:** Accepted
**Accepted:** 2026-09-13 by M — *"address these tow, and then commit, push, merge, release"*
**Date:** 2026-09-13
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-023, ADR-044, ADR-054, ADR-055, docs/adr/BACKLOG.md ("Pre-registration: a default `--strict-balance`")
**Governs:** `cmd/mrw/main.go`, `internal/apply/apply.go`, `internal/authoring/authoring.go`, `internal/mcp/tools.go`, `internal/mcp/schema.go`, `scripts/contract.sh` (§95–§96)
**Enforced-by:** `internal/mcp/pattern_test.go::TestTheMCPReceiptCarriesThePattern`
**Served-path change:** `mrw write --json` and `mrw_write` receipts gain a `pattern` object on every write; `mrw stats` gains a `strict-balance pricing` block and `--json` a `pricing` key. No verdict, exit code or refusal changes.
**Invalidates:** None — closes the two items ADR-055 left open: its Out of Scope deferred the MCP `pattern`, and its T4 pre-registered a campaign whose data source did not exist.

**Notes:** ADR-055 shipped as v1.17.0 with two threads open, both named on the release: the MCP receipt had no `pattern` (the CLI printed a line; the structured receipt did not carry it), and the pre-registered campaign for a default `--strict-balance` had nowhere to read from — mrw's state holds no plan text (ADR-009), so "replay every landed single-line replace" cannot be done from any corpus's state after the fact. M: *"address these tow"*. The first is a field; the second is instrumentation that makes the campaign runnable from the day it ships, counts only.

## Context

ADR-055 T2 put the repeat-pattern line on the CLI receipt and in `stats`, and
wrote the MCP receipt into BACKLOG because that receipt is structured and the
line is prose. But the reason the line exists is that the balance ROW was not
read; an MCP caller reads a JSON object, and a fact absent from that object is a
fact the caller does not have. The CLI `--json` receipt had the same gap: the
line printed on the human receipt only.

ADR-055 T4 pre-registered the criterion under which `--strict-balance` could
become a default: three corpora, replay every landed single-line replace, false
positives under 5% of refusals, at least fifty refusals. The criterion is sound
and the replay is impossible: ADR-009 refuses plan text, paths and addresses in
the tally, so no corpus holds what would be replayed. The campaign as written
would have to be reconstructed from agent transcripts — the wrong source, and
one that does not know whether the tree broke afterwards. mrw does know, at the
moment of the write: the CLI has the check verdict in hand (ADR-054), and the
engine has the hunk.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `authoring.Pattern(entries)` | `internal/authoring/authoring.go` | Returns `(advisory, total, fires)`. Both receipts can carry it; nothing does. |
| `writeReceipt` | `internal/mcp/tools.go` | `apply.Result` + `Elided`. Gains `Pattern`. |
| `receipt` | `cmd/mrw/main.go` | `apply.Result` + `Check`. Gains `Pattern`. |
| contract row "the MCP receipt equals the CLI receipt for the same plan" | `scripts/contract.sh` ~1730 | Pops `root` and `check` and compares. A field on ONE transport breaks it — so the field goes on both, and the row stays as it is. |
| `wrapTailDelta(consumed, body)` | `internal/apply/apply.go` | Computed only under `StrictBalance`. It is exactly the predicate "would the flag have refused this hunk"; computing it with the flag off costs the same scan `balanceDelta` already does. |
| `authoring.Record` / `Tally` | `internal/authoring/authoring.go` | Counters only, one file. A second counters-only file beside it is the same shape and the same ADR-009 posture. |
| `check.Result.OK()` / `Ran` | `cmd/mrw/main.go` after a write | The CLI knows, per write, whether the tree broke. MCP never runs a check (ADR-044) and can only say `unchecked`. |

## Decision

**1. Both JSON receipts carry `pattern`, always.** `pattern` is an object
`{advisory_writes: K, window: N, fires: bool}` read from the recent-window ring
after this write joined it (ADR-055 T2: only a landed write joins). It is present
on every receipt, fires or not, on `mrw write --json` and on `mrw_write`, so a
caller that reads keys sees the key on a quiet day and does not learn it does
not exist. The human line on the CLI receipt is unchanged. The MCP output schema
describes the object and its three members.

**2. The tally prices the flag as writes land.** A second counters-only file,
`pricing`, beside the tally. Per landed write with `StrictBalance` OFF:

- `strict_candidates` — landed writes with at least one ok single-line
  `replace` on a non-prose path (the population the flag looks at).
- `strict_would_refuse` — of those, writes where at least one such hunk has a
  non-empty `wrapTailDelta` (the flag would have refused the plan).
- `strict_would_refuse_broke` / `_held` / `_unchecked` — how the would-refuse
  writes ended: the check ran and failed (exit 3); the check ran and passed; no
  check ran (MCP, `--no-check`, prose-mixed plans without a harness, or no
  command). Exactly one of the three increments per would-refuse write.

`held` is the false-positive count the pre-registration asks for; `broke` is the
true positive; `unchecked` is neither and is reported so a corpus that never
checks cannot flatter the flag. A write with the flag ON is not priced: the
question is what the flag WOULD do. The engine exposes `Result.StrictWouldRefuse`
(the count of ok hunks matching the signature) with `json:"-"` — it is an input
to the tally, not a receipt field; the receipt's `balance` rows already show the
hunks. `mrw stats` prints one block:

```
strict-balance pricing: C landed writes with a single-line code replace; the flag would have refused R — broke B, held H, unchecked U
  false positives H of B+H checked (p%) — pre-registered bar: under 5% in every corpus, 50 refusals total (BACKLOG)
```

and `--json` carries `pricing` with the five counters. When `B+H` is zero the
percentage line says `no checked refusals yet` rather than dividing.

**3. The pre-registration's data source is this block.** The criterion in
BACKLOG is unchanged; its "replay" sentence is corrected to "read
`mrw stats --json` `.pricing` in each corpus". Nothing is armed by this record.

## Alternatives Considered

- **`pattern` on MCP only** — breaks the transport-equality contract row and
  leaves `mrw write --json` with the gap the human line was written to close.
- **Reconstruct the campaign from agent transcripts** — the wrong source (a
  transcript does not know the tree's later state) and not reproducible by the
  next person; the pre-registration's own rule is that the number must be
  cheap and honest to produce.
- **Log the refused/would-refuse hunks with their addresses** — ADR-009 refuses
  it, and the count is what the criterion needs.
- **Price with the flag ON too** — a refused plan did not land; there is no
  outcome to attribute. Priced writes are flag-off by definition.
- **Fold the counters into the existing tally file** — the tally's vocabulary
  is the five outcome names ADR-054 fixed and `stats` prints them all; adding
  five foreign names there would print them in that list. A second file keeps
  the two vocabularies apart.

## Component / Boundary Impact

`internal/apply` computes one more count per run. `internal/authoring` gains a
second counters-only file, same posture as the first. `cmd/mrw` and
`internal/mcp` gain a receipt field and one record call each. No new tool, no
new flag.

## Wiring & Contract Changes

- `apply.Result.StrictWouldRefuse int` (`json:"-"`).
- `receipt.Pattern` / `writeReceipt.Pattern` (`json:"pattern"`), type
  `authoring.PatternInfo{AdvisoryWrites, Window int; Fires bool}`.
- `authoring.RecordPricing(root, candidate, wouldRefuse bool, outcome PricingOutcome)`,
  `authoring.LoadPricing(root) Pricing`, `Pricing.FalsePositiveRate()`.
- MCP output schema: `pattern`, `pattern.advisory_writes`, `pattern.window`,
  `pattern.fires`.
- `mrw stats`: the pricing block; `--json` `pricing`.
- Contract **§95** (the receipt carries `pattern` on both transports; fires on
  the third advisory) and **§96** (pricing counts a would-refuse write and its
  outcome; `--strict-balance` writes are not priced).

## Inter-task Contracts

| Produces | Consumes |
|----------|----------|
| T1: `authoring.PatternInfo`, `pattern` on both receipts | — |
| T2: `Result.StrictWouldRefuse`, `pricing` file, stats block | T1 nothing |
| T3: teach + BACKLOG correction | T1, T2 |

## Implementation

Tasks in `docs/adr/ADR-056-the-receipt-carries-the-pattern-and-the-tally-prices-the-flag/tasks/`. Each: red test → implement → mutants through `adr-verify --mutant` → contract row → exit-0 under the current digest. Assertions written whole BEFORE the red (the ADR-055 T2 lesson: a strengthened body after the red costs the status word).

## Consequences

- An MCP caller and a `--json` caller now hold the pattern as data.
- Every landed write costs one more small file write. Same cost class as the
  tally and the ring.
- The pre-registered campaign becomes `mrw stats --json` in three checkouts
  after enough writes have landed there. Nothing about the default changes here.
- `unchecked` will dominate in MCP-heavy corpora; the block says so, and that
  is a finding about the corpus, not a reason to count it either way.

## Out of Scope

- Any change to the default of `--strict-balance`. (deferred: docs/adr/BACKLOG.md "Pre-registration: a default `--strict-balance`")
- A per-hunk `would_refuse` field on the receipt: the `balance` row already
  shows the hunk, and a second row about the same hunk is the noise ADR-055 was
  about. (permanent: boundary: the receipt names each hunk once)
- Pricing the multi-line replace. (permanent: boundary: the flag's predicate is single-line by ADR-055; pricing measures the flag)

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `pattern` on both receipts still diverges (ring state differs per root) | Low | Both read the same ring after the same record call; §95 asserts the object on both transports, and the existing equality row runs unchanged. |
| The would-refuse predicate drifts from the flag's refusal predicate | Med | One function, `wrapTailDelta`, used by both; a test asserts a hunk the flag refuses is a hunk the pricing counts. |
| Pricing silently stops recording | Low | §96 reads the counter back through `stats --json` after a real write. |

## Rollback

Delete the `pattern` field from both receipt structs and the four schema rows; delete `RecordPricing`/`LoadPricing`, the stats block and `Result.StrictWouldRefuse`; remove §95–§96. The `pricing` file in existing state directories is ignored by an older binary.

## Follow-ups

- When three corpora hold ≥ 50 refusals between them, run the campaign as the
  pre-registration says and record the result there — arm with *"price strict
  balance"*.
