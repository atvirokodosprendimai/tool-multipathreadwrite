# ADR-119: a write says when its shape looks wrong

**Status:** Accepted
**Accepted:** 2026-10-02 by Zy — on the gap list of 2026-10-01, item 1 ("structure blindness"): "advisory heuristics only (indent mismatch, duplicate closer), false-positive rate measured on real history before shipping; no parser, no default --strict-balance"; and on 2026-10-02: "Separate keys (Recommended)" and "<5% and 5/5 fixtures (Recommended)". The record's text was drafted after those answers and not shown to Zy before execution: these are the answers, not a review of these words.
**Date:** 2026-10-02
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-048, ADR-052, ADR-054, ADR-055, ADR-056, ADR-111
**Invalidates:** None — additive; the balance row, `advisories` and the pattern keep their meanings (ADR-111)
**Governs:** `internal/apply/apply.go`, `internal/apply/shape.go`, `cmd/mrw/main.go`, `internal/mcp/schema.go`, `docs/receipts.txt`, `AGENTS.md`, `README.md`, `scripts/contract.sh`, `docs/break/shape-hints/stress.py`
**Enforced-by:** `internal/apply/shape119_test.go::TestTheFieldFailuresEachCarryAHint`
**Served-path change:** An applied `replace` hunk may carry `closer` — its body ends in a closer-shaped line (a fence, `}`, `</div>`, `@endif`, …) that one of the next four non-blank lines after it repeats, the closer the file already had. It is advice: the hunk stays ok. The receipt counts such hunks in a top-level `hints`, absent when zero. The indent hint the record first proposed missed the pre-registered bar and does not ship.

## Context

mrw puts lines where it is told and models no syntax. The field reports recorded in BACKLOG ("From the field reports") measured the failures that follow: a body that ends with the closer the file already had leaves a duplicate below it (Blade `@endif`, HTML `</div>`, a markdown fence), and a body at the wrong indent reparents keys with nothing left to find (a YAML block scalar, an Ansible `when:`). In three of four stacks the project's own gates passed the broken file. The balance row (ADR-054) cannot see either: a duplicate closer balanced by its opener nets zero, and indentation is not a delimiter.

Zy chose advice over refusal, measured before shipping: two cheap heuristics, each held to a false-positive bar on real history and a true-positive bar on the field fixtures, both written down (BACKLOG, "Pre-registered for ADR-119") before any measurement.

**Audit of the class** — *a per-hunk receipt row that reports without failing*: `mrw read --grep 'Balance|Echo' --exclude '*_test.go' internal/apply/apply.go` — `HunkResult.Balance` (ADR-054) and `.Echo` (ADR-052), cleared together at every site where an ok hunk stops being ok. The new rows join them at every one of those sites.

## Existing Primitives Audit

- **`balanceDelta`** and the balance row (ADR-054) — the advisory row this sits beside; its count, `advisories`, keeps meaning "balance rows" (ADR-111), so the hints get their own count.
- **`IsProse`** (ADR-054) — the prose list; indent skips prose, closer does not (the markdown fence is prose).
- **`--echo-pad`** (ADR-052) — shows the lines after a body; a human check, not a detector.

## Decision

1. **`closer`**: on an applied `replace`, when the body's last non-blank line, trimmed, is a closer-shaped token (a ` ``` ` or `~~~` fence, a run of `}` `]` `)` with trailing `;` `,` `)`, an end tag `</x>`, `@end…`, `end`, `fi`, `done`, `esac`) and one of the next four non-blank lines after the body in the WRITTEN file equals it, trimmed, the hunk carries `closer` naming that line as it sits in the written file: `line N repeats the body's last line: <text>`. It runs on prose too. This is the definition BACKLOG registered after the first measurement ("Amended after the first measurement"), not the line-right-after comparison first registered: that one caught 2 of its 3 field fixtures, because the markdown fence's survivor was the fourth line after the range.
2. **`indent` does not ship.** The registered indent hint caught both indentation fixtures but fired on 19.35% of `.py`, 14.29% of extensionless, 11.89% of `.js` and 8.99% of `.php` replaces in the first measurement, against a bar of 5%; Zy withdrew it ("K4 closer, defer indent"). It is deferred to BACKLOG "From ADR-119" with its trigger.
3. **Advice only.** The hunk stays ok; nothing is refused, `--strict-balance` included. A skipped or failed hunk carries no `closer` (one `clearWriteRows` clears it with the pad and the balance row at every site). `advisories` and the pattern line keep counting balance rows only (ADR-111); the receipt adds a top-level `hints`, the number of hunks carrying `closer`, absent when zero, and the human summary adds `, M hints` only when M > 0, so every summary a caller already parses stays byte-identical.
4. **Measured before shipping.** `docs/break/shape-hints/stress.py` replays the non-merge history of every repository under `~/GolandProjects` and `~/CursorProjects` as replaces, computes each heuristic per language bucket, and with `--mrw` drives the built binary's `--dry-run --json` to prove the binary reports what was measured. The first run missed the bar on both heuristics; the amended closer was registered before a second run on a fresh sample, which it passed (T2, `docs/break/shape-hints/README.md`).

## Alternatives Considered

- **A syntax parser per language** — rejected by Zy's answer, and by ADR-013: mrw is language-agnostic by construction.
- **`--strict-balance` by default** — rejected by Zy's answer: its pricing bar (ADR-056) was not met.
- **One `advisories` count for both** — rejected by Zy's answer: ADR-111 freezes `advisories` as balance rows.

## Component / Boundary Impact

Owns `internal/apply` (engine). `cmd/mrw`, `internal/mcp` are not engine. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `HunkResult` | `closer` (omitempty) | T1 | CLI, MCP receipts |
| `Result` | `hints` (omitempty) | T1 | CLI, MCP receipts |
| `docs/receipts.txt` | `hints` and `hunks[].closer` for `write` and `mcp_write` | T1 | ADR-111's tests |
| CLI human receipt | a row under the ok line; `, M hints` in the summary | T1 | callers |
| `docs/break/shape-hints/` | the measurement and its result | T2 | the record |
| `scripts/contract.sh` | §219 | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `apply.HunkResult.Closer`, `Result.Hints` | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2.

## Consequences

- **Positive:** the failures the field reports measured are named in the receipt of the write that made them.
- **Negative:** a correct edit sometimes carries a hint; the bar bounds how often.
- **Neutral:** nothing is refused; callers who ignore the keys see no change.

## Out of Scope

- A refusal on either heuristic (permanent: boundary: Zy chose advice; a heuristic that refuses correct edits costs more than it saves)
- A short address that orphans lines ABOVE the body (deferred: docs/adr/BACKLOG.md "From ADR-119")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a hint fires so often it is ignored | Medium | Medium | the pre-registered bar; a heuristic over it does not ship |
| the replayed history is not the edits agents make | Medium | Low | the true-positive bar uses the field failures agents did make |

## Rollback

Revert the tasks. The two receipt keys disappear with them, which ADR-111 calls a removal: revert before a release ships them.

## Follow-ups

- None — the record carries no open follow-up.
