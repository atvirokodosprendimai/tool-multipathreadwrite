# Task ADR-192-T2: bench look reads H against B and A, with the uptake gate

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** comparisons `B:A`, `H:A`, `H:B` under stage7; the UNTESTED verdict; the uptake gate
**Consumes:** configuration H and profile `stage7` (T1), receipt `host_reads` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a comparison names its baseline`, `the gate reads block 1 only`, `UNTESTED is neither PAYS nor LOSES`, `the bench suite still passes`, `Stage 5's look reproduces`

## Goal

Under stage7, `bench look` and `bench analyze` give each model three verdicts, H against B among them, and close
H's two as UNTESTED when fewer than half of its block-1 attempts called the host Read (ADR-192 Decisions 3 and 4).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/look.go` | edit | a comparison is `config:baseline`; stage7 lists `B:A`, `H:A`, `H:B`; the gate; a configuration runs on while any comparison it is in is open |
| `internal/bench/analyze.go` | edit | `judgeWith` takes the baseline |
| `internal/bench/look_test.go` | edit | the two tests |

## Ordered Steps

1. [S1] Write the two failing tests and see them red. [proof: acceptance]
2. [S2] A comparison names its baseline; Stage 5's stay `B:A` and `P:A`, so `bench look docs/bench/stage-5` prints what it printed. [proof: mutation]
3. [S3] The gate: at look 1, per model, H's attempts with `host_reads > 0` under half of H's block-1 attempts close `H:A` and `H:B` UNTESTED, and H is not scheduled again for that model. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestLookReadsHAgainstBAndA|TestHClosesUntestedBelowTheUptakeGate)$' 2>&1) \
  && for t in TestLookReadsHAgainstBAndA TestHClosesUntestedBelowTheUptakeGate; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go test ./internal/bench/ -count=1 \
  && go run ./cmd/bench look docs/bench/stage-5 > /tmp/adr192-look5.txt \
  && diff /tmp/adr192-look5.txt docs/bench/stage-5/looks/look-5.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestLookReadsHAgainstBAndA` | `internal/bench/look_test.go` | on synthetic stage7 receipts where H is cheaper than B, the `H:B` row reads H's cost against B's, not A's; a configuration whose comparisons are all closed is not scheduled | none | S2 |
| `TestHClosesUntestedBelowTheUptakeGate` | `internal/bench/look_test.go` | 19 of 40 H attempts with a host Read close both H rows UNTESTED and stop H; 20 of 40 leave them open | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | the stage7 comparison list in `look.go`; deleting `H:B` fails `TestLookReadsHAgainstBAndA` |
| 3 — the caller can discover it | `bench look`'s rows print the comparison |
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
