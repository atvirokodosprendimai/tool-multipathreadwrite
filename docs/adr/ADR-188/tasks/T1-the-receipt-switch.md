# Task ADR-188-T1: `MRW_RECEIPT` selects the write receipt

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the variable's reading in `reportAs`; `TestTheReceiptSwitchSetsTheWriteReceiptUnlessAFlagDoes`; contract §323
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the variable selects the receipt`, `a flag wins over it`

## Goal

Decisions 1-4 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/receipt188_test.go` | new | the red test |
| `cmd/mrw/main.go` | edit | `reportAs` reads `MRW_RECEIPT`; the `--quiet` and `--terse` usage strings and the write help name it |
| `internal/guide/guide.go` | edit | the full text names the variable beside `--terse` |
| `scripts/contract.sh` | edit | §323 |
| `README.md`, `AGENTS.md` | edit | the variable, beside `--terse` |

## Ordered Steps

1. [S1] Write `TestTheReceiptSwitchSetsTheWriteReceiptUnlessAFlagDoes`, a two-file write each time: unset and `full` print an `ok` row per hunk and a `wrote` line per file; `quiet` prints the summary line and no `ok` row; `terse` prints one `ok PATH N hunk(s)` line per file; `--terse` with `MRW_RECEIPT=quiet` prints terse and `--quiet` with `MRW_RECEIPT=terse` prints quiet; `MRW_RECEIPT=loud` prints the full receipt and names `MRW_RECEIPT=loud` on standard error; under `quiet` a failing plan still prints its `FAIL` row and exits 1. See it RED.
2. [S2] `reportAs` reads the variable when no flag is given. Mutant: ignore the variable — the quiet and terse cases go red. [proof: mutation]
3. [S3] A flag wins. Mutant: read the variable before the flags — the flag-wins cases go red. [proof: mutation]
4. [S4] Help, guide, README and AGENTS.md name the variable. [proof: acceptance]
5. [S5] Contract §323: with `MRW_RECEIPT=quiet` a two-hunk write of the built binary prints no `ok` row and its summary; the pair, unset, prints both `ok` rows. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestTheReceiptSwitchSetsTheWriteReceiptUnlessAFlagDoes|TestInstructionsCoreFlagPrintsTheCore|TestTheCoreNamesOnlyFlagsTheBinaryHas)$' 2>&1) \
  && grep -qE '^--- PASS: TestTheReceiptSwitchSetsTheWriteReceiptUnlessAFlagDoes \(' <<<"$out" \
  && grep -qE '^--- PASS: TestInstructionsCoreFlagPrintsTheCore \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreNamesOnlyFlagsTheBinaryHas \(' <<<"$out" \
  && go test ./internal/guide/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 323\. ' scripts/contract.sh \
  && grep -qF 'PASS  MRW_RECEIPT=quiet prints the summary and no ok row' <<<"$c" \
  && grep -qF 'PASS  the pair: unset, the same write prints an ok row per hunk' <<<"$c" \
  && grep -qF 'MRW_RECEIPT' README.md \
  && grep -qF 'MRW_RECEIPT' AGENTS.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/read internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheReceiptSwitchSetsTheWriteReceiptUnlessAFlagDoes` | `cmd/mrw/receipt188_test.go` | each value's receipt, a flag winning, a bad value ignored with a note, a failure printed under quiet | none | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every `mrw write` whose receipt `reportAs` prints |
| 3 — the caller can discover it | `mrw write --help`, `mrw instructions`, README, AGENTS.md |
| 4 — it is used | no telemetry (ADR-009); the evidence is runs 6-8's receipt sizes |

## Mutation Log
- 2026-11-01 · f2c04f18* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the variable is ignored · acceptance-sha256:8f1623ac52641db33b60576fa0acf44bfa7171246ec32bc7abb321501f2ddc6f · covers:the variable selects the receipt
- 2026-11-01 · f2c04f18* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: the variable is read over the flags · acceptance-sha256:8f1623ac52641db33b60576fa0acf44bfa7171246ec32bc7abb321501f2ddc6f · covers:a flag wins over it

## Invariants

- Unset, every receipt is byte-identical to before.
- `--json`, `mrw_write` and every refusal are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0.

## Out of Scope

- `mrw_write` (permanent: boundary: ADR-188 Out of Scope)

## Verification Log
- 2026-11-01 · f2c04f18* · exit 1 · `set -o pipefail …` · acceptance-sha256:8f1623ac52641db33b60576fa0acf44bfa7171246ec32bc7abb321501f2ddc6f · ms:468 · test-lock-sha256:3c1ed415c88a7775c9b72d66a5fc4098ea1992529f8421c8034cc6be089c66bc · test-lock-file:docs/adr/locks/3c1ed415c88a7775c9b72d66a5fc4098ea1992529f8421c8034cc6be089c66bc.lock
  ```
  ```
- 2026-11-01 · f2c04f18* · exit 0 · `set -o pipefail …` · acceptance-sha256:8f1623ac52641db33b60576fa0acf44bfa7171246ec32bc7abb321501f2ddc6f · ms:79528
- 2026-11-01 · f2c04f18* · exit 0 · `set -o pipefail …` · acceptance-sha256:8f1623ac52641db33b60576fa0acf44bfa7171246ec32bc7abb321501f2ddc6f · ms:80034
- 2026-11-01 · f2c04f18* · exit 0 · `set -o pipefail …` · acceptance-sha256:8f1623ac52641db33b60576fa0acf44bfa7171246ec32bc7abb321501f2ddc6f · ms:79739
