# Task ADR-052-T1: Neighbour license on multi-line replace; contract §87

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** neighbour license in `Apply` (T1)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the neighbour refuse`, `EOF skip`, `single-line unchanged`

## Goal

A multi-line `replace` whose ledger observation does not cover End+1 fails, siblings skip, nothing is written. Single-line replace is unchanged. End = last line skips the license. Insert falls out.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | After ADR-035's anchor check: when a ledger observation exists, `replace` with `end > start` and `end < total` requires `Covers(end+1, end+1)`. The selector. |
| `internal/apply/apply_test.go` | edit | Red tests for the refuse, the sibling skip, the single-line case, EOF skip, insert out. |
| `scripts/contract.sh` | edit | **§87** — reserve after #166's §86. Pair unread-neighbour (exit 1, FAIL+skip, unchanged) with served End+1 (exit 0). |

## Ordered Steps

1. [S1] Write `TestAMultiLineReplaceWithoutAServedLineAfterEndWritesNothing` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestASingleLineReplaceDoesNotNeedANeighbour` and `TestAnEOFMultiLineReplaceDoesNotNeedALineAfterEnd` and confirm they fail for the wrong reason or the implementation is absent — then they stay the green pair once S1's check exists. [proof: mutation]
3. [S3] Implement the neighbour check in `planFile`. `--force` does not waive a partial observation. Nil `Seen` skips. Confirm S1–S2 GREEN. Deleting the End+1 `Covers` call must fail S1. [proof: mutation]
4. [S4] Write §87 against a binary that does not yet refuse the neighbour and confirm it is RED, then rebuild and confirm GREEN. [proof: mutation]
5. [S5] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 87\. ' scripts/contract.sh \
  && go test ./internal/apply/ -count=1 -v \
    -run 'TestAMultiLineReplaceWithoutAServedLineAfterEndWritesNothing|TestAMultiLineReplaceWithAServedLineAfterEndApplies|TestASingleLineReplaceDoesNotNeedANeighbour|TestAnEOFMultiLineReplaceDoesNotNeedALineAfterEnd' 2>&1 | tee /tmp/adr052-t1.out \
  && grep -q '^--- PASS: TestAMultiLineReplaceWithoutAServedLineAfterEndWritesNothing' /tmp/adr052-t1.out \
  && grep -q '^--- PASS: TestAMultiLineReplaceWithAServedLineAfterEndApplies' /tmp/adr052-t1.out \
  && grep -q '^--- PASS: TestASingleLineReplaceDoesNotNeedANeighbour' /tmp/adr052-t1.out \
  && grep -q '^--- PASS: TestAnEOFMultiLineReplaceDoesNotNeedALineAfterEnd' /tmp/adr052-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr052-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/apply/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAMultiLineReplaceWithoutAServedLineAfterEndWritesNothing` | `internal/apply/apply_test.go` | Ranged read of Start-End refuses a multi-line replace; sibling skip; file unchanged | — | S1, S3 |
| `TestASingleLineReplaceDoesNotNeedANeighbour` | `internal/apply/apply_test.go` | Single-line replace with only that line served still applies | — | S2, S3 |
| `TestAnEOFMultiLineReplaceDoesNotNeedALineAfterEnd` | `internal/apply/apply_test.go` | End = last line skips the licence | — | S2, S3 |
| `TestAMultiLineReplaceWithAServedLineAfterEndApplies` | `internal/apply/apply_test.go` | Served End+1 applies the multi-line replace | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §87 |
| 2 — something selects it | `planFile` after resolution; deleting the End+1 check fails S1 and §87 |
| 3 — the caller can discover it | T3 teaches; the refusal names the missing line |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-13 · 2f49847* · mutant killed · exit 1 · `internal/apply/apply.go` · deleting the End+1 Covers call must fail S1 · acceptance-sha256:3f2dfff3775e397bf7db5878d397353a5461b4d7f02fd8f7f3ed776cf3f3e6fd
- 2026-09-13 · 4158e26 · mutant killed · exit 1 · `internal/apply/apply.go` · the neighbour licence never fires: a multi-line replace with no served line after End applies, and TestAMultiLineReplaceWithoutAServedLineAfterEndWritesNothing must go red · acceptance-sha256:c39971e1bdc5a742370eed349681d647e0835987a3f680cd631d11f0ece6deca

## Invariants

- ADR-001: any failing hunk writes nothing; siblings `skip`.
- Single-line replace unchanged.
- Nil `Seen` still disables ledger checks.
- `--force` does not waive a partial observation's neighbour.
- Engine packages other than `internal/apply` stay byte-identical unless a later task owns them.

## Risks

- A whole-file observation already `Covers` End+1 — do not special-case Whole(); `Covers` is the primitive.
- Quoting End+1's text in the refusal would read back an unserved line (ADR-028). Name the line number, not the closer.

## Stop Condition

A mutant that ignores the neighbour still exits 0 on S1. If the only way to go green is a parser, stop.

## Out of Scope

- Echo (T2)
- Teaching (T3)
- Insert neighbour

## Verification Log
- 2026-09-13 · 2f49847* · exit 0 · `set -o pipefail …` · acceptance-sha256:3f2dfff3775e397bf7db5878d397353a5461b4d7f02fd8f7f3ed776cf3f3e6fd · ms:813
- 2026-09-13 · 4158e26 · exit 0 · `set -o pipefail …` · acceptance-sha256:c39971e1bdc5a742370eed349681d647e0835987a3f680cd631d11f0ece6deca · ms:1146
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:c39971e1bdc5a742370eed349681d647e0835987a3f680cd631d11f0ece6deca · ms:271
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:c39971e1bdc5a742370eed349681d647e0835987a3f680cd631d11f0ece6deca · ms:577
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:c39971e1bdc5a742370eed349681d647e0835987a3f680cd631d11f0ece6deca · ms:274
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:c39971e1bdc5a742370eed349681d647e0835987a3f680cd631d11f0ece6deca · ms:267
