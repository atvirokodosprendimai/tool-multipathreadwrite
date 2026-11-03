# Task ADR-192-T2: bench look and analyze read H against B and A, with the uptake gate

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** comparisons `B:A`, `H:A`, `H:B` under stage7, eligibility per baseline, the UNTESTED verdict, the uptake gate, the stage7 schedule
**Consumes:** configuration H and profile `stage7` (T1), receipt `host_reads` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a comparison names its baseline`, `eligibility reads the comparison's own baseline`, `the gate counts walked final receipts`, `UNTESTED overrides a verdict at its look`, `a configuration runs while an open comparison includes it`, `Stage 5's look reproduces`

## Goal

Under stage7, `bench look` and `bench analyze` give each model three verdicts, each judged and made eligible
against its own baseline, schedule a configuration while any open comparison includes it, and close H's two as
UNTESTED when the uptake gate fails (ADR-192 Decisions 3 and 4).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/look.go` | edit | a comparison is `config:baseline`; the A literals at `:134`, `:143`, `:163`, `:188`, `:229`, `:243`, `:277`, `:352` take the baseline; stage7 lists `B:A`, `H:A`, `H:B`; the schedule; the gate |
| `internal/bench/analyze.go` | edit | call `judgeAgainst` (`:260`) with the baseline; shell reads summed for B and H, not B alone (`:282`) |
| `cmd/bench/main.go` | edit | the analysis table names each row's baseline, not "against A" and "(X−A)/A" (`:447-457`) |
| `internal/bench/look_test.go` | edit | the look tests |
| `cmd/bench/main_test.go` | edit | the table test |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. [proof: acceptance]
2. [S2] A comparison names its baseline, for the verdict and for eligibility; Stage 5's stay `B:A` and `P:A`, so `bench look docs/bench/stage-5` prints what it printed. [proof: mutation]
3. [S3] The stage7 schedule: a configuration runs while any open comparison includes it; closed rows stay frozen. [proof: mutation]
4. [S4] The gate: neither H comparison closes before it is read, at the first look where `H:B` is eligible; there, per model, over block 1's final walked H receipts with exclusions left out, fewer than half with `host_reads >= 1` close `H:A` and `H:B` UNTESTED and H is not scheduled again for that model. [proof: mutation]
5. [S5] The analysis table names the baseline per row. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ ./cmd/bench/ -count=1 -v -run '^(TestLookReadsHAgainstBAndA|TestHAgainstBIsIneligibleWhenBIsNot|TestHClosesUntestedBelowTheUptakeGate|TestAnalyzeNamesEachRowsBaseline)$' 2>&1) \
  && for t in TestLookReadsHAgainstBAndA TestHAgainstBIsIneligibleWhenBIsNot TestHClosesUntestedBelowTheUptakeGate TestAnalyzeNamesEachRowsBaseline; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go test ./internal/bench/ ./cmd/bench/ -count=1 \
  && go run ./cmd/bench look docs/bench/stage-5 > /tmp/adr192-look5.txt \
  && diff /tmp/adr192-look5.txt docs/bench/stage-5/looks/look-5.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestLookReadsHAgainstBAndA` | `internal/bench/look_test.go` | on synthetic stage7 receipts where H is cheaper than B and dearer than A, `H:B` reads H's cost and solved against B's; with `B:A` and `H:A` closed and `H:B` open, A is not scheduled and B and H are; a closed row does not move when later attempts land | none | S2, S3 |
| `TestHAgainstBIsIneligibleWhenBIsNot` | `internal/bench/look_test.go` | 3 excluded B attempts among 40 make `H:B` ineligible while A and H are complete | none | S2 |
| `TestHClosesUntestedBelowTheUptakeGate` | `internal/bench/look_test.go` | 19 of 40 counted H receipts with a Read close both H rows UNTESTED and stop H; 20 of 40 leave them open; an excluded attempt and its replacement count once, as the replacement; a timed-out attempt counts; with `H:B` ineligible at look 1 (3 excluded B attempts), `H:A` does not close at look 1 though its interval says LOSES, and the gate read at look 2 closes both UNTESTED | none | S4 |
| `TestAnalyzeNamesEachRowsBaseline` | `cmd/bench/main_test.go` | `bench analyze` on stage7 receipts prints `H against B` with `(X−B)/B`, and Stage 5's rows as before | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests |
| 2 — something selects it | the stage7 comparison list in `look.go`; deleting `H:B` fails `TestLookReadsHAgainstBAndA` |
| 3 — the caller can discover it | `bench look` and `bench analyze` print the comparison and its baseline |
| 4 — it is used | T4's looks |

## Mutation Log

## Invariants

- `bench look docs/bench/stage-5` is byte-identical to `looks/look-5.txt` (the fence's last clause).

## Risks

- none: no doubles beyond synthetic receipts, which T4's real receipts meet.

## Stop Condition

Reading against B needs a change to the bar itself (ADR-156) rather than to its baseline: stop to Zy.

## Out of Scope

- The mechanism counts (T5's job).

## Verification Log
