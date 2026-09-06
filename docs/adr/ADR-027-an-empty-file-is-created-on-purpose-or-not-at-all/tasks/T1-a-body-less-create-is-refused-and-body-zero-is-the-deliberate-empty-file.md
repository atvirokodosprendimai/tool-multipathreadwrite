# Task ADR-027-T1: A body-less create is refused, and body=0 is the deliberate empty file

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one validation branch, plus its test)
**Owner:** Zy
**Produces:** the body-less `create` refusal and its wording (T2)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the refusal of a create carrying no body`, `body=0 still creating an empty file`, `the neighbouring ops keeping their own refusals`

## Goal

Stop `create` reporting `ok` for a hunk that carried nothing, while keeping the deliberate empty file
available under a spelling that says it was deliberate.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `validate`'s `OpCreate` branch refuses a hunk with no body lines unless it declared `body=0`. The `Hunk` needs to carry whether a count was DECLARED, which the parser already tracks as `fixed` while checking overcounts but does not keep — a hunk with no body and no `body=` is indistinguishable from `body=0` after parsing today. |
| `internal/plan/plan_test.go` | edit | `TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero` is what SELECTS the new branch: nothing else parses a `create` with an empty body. |
| `internal/plan/plan_test.go` (existing row) | edit | `TestDeleteIsTheOnlyRangeConsumingOpThatNeedsNoBody` carried `create` as `emptyBodyOK: true` — it encoded the behaviour this record changes. Its own claim is the CONJUNCTION "range-consuming AND needs no body", which `create` fails on the range half either way, so the claim is untouched and only the row moves. Named here rather than quietly edited, because a test changed by a record is a decision, not a fixup. |

## Ordered Steps

1. [S1] Write `TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero` and confirm it is RED. Four shapes so no parser can satisfy it by refusing everything: a body-less `create` is refused and the message names `body=0`; `create body=0` applies and yields an empty body; `create body=0 raw=true` likewise, since `raw=` and `body=` already interact; and an ordinary `create` with a body is untouched.
2. [S2] Carry the "a count was declared" fact from the parser onto the hunk, rather than inferring it from a zero-length body — the two are the same thing after parsing, which is exactly why this defect existed. [proof: mutation]
3. [S3] Refuse the body-less `create` in `validate`, naming both readings: `body=0` for the caller who meant an empty file, and a lost body for the caller who did not. ADR-015's rule, and the same shape as the empty-`replace` refusal one branch above. [proof: mutation]
4. [S4] Confirm the ops that already refuse an empty body are untouched: `replace` still says it would delete the addressed lines, and `insert-after`/`insert-before` still say they would change nothing. [proof: acceptance]
5. [S5] Run the package, `gofmt` and `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ -count=1 -v \
  -run 'TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero' 2>&1 | tee /tmp/adr027-t1.out \
  && grep -q '^--- PASS: TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero' /tmp/adr027-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr027-t1.out \
  && go test ./internal/plan/ ./internal/adversarial/ -count=1 -v -run 'TestDeleteIsTheOnlyRangeConsumingOpThatNeedsNoBody|TestRawWithoutBodyIsRefused|TestAReplaceWithNoBodyIsRejected' 2>&1 | tee /tmp/adr027-t1n.out \
  && grep -q '^--- PASS: TestAReplaceWithNoBodyIsRejected' /tmp/adr027-t1n.out \
  && grep -q '^--- PASS: TestRawWithoutBodyIsRefused' /tmp/adr027-t1n.out \
  && ! grep -qE 'no tests to run|^FAIL|^--- FAIL' /tmp/adr027-t1n.out \
  && go test ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero` | `internal/plan/plan_test.go` | A body-less `create` is refused naming `body=0`; `body=0` and `body=0 raw=true` both apply with an empty body; a `create` with a body is unchanged | — | S1, S2, S3 |
| `TestAReplaceWithNoBodyIsRejected` | `internal/adversarial/planformat_test.go` | Unchanged: ADR-006's empty-`replace` refusal is not disturbed | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero` |
| 2 — something selects it | `validate` is on the only path a parsed hunk takes, for the CLI and for `mrw_write` alike; the S3 mutation removes the branch so a body-less `create` reports `ok` again and the fence goes red |
| 3 — the caller can discover it | The refusal names `body=0`, and T2 writes the form into `README.md` and `AGENTS.md` |
| 4 — it is used | Contract §65 (T2) drives both shapes through the built binary; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · dd0cc3a* · mutant killed · exit 1 · `internal/plan/plan.go` · the create branch stops refusing a body-less hunk, so a plan whose last create lost its body creates an empty file and reports ok — the behaviour this record removes · acceptance-sha256:8d332ebfef1a0bc0a7dc362ec806a0fb9e850d1676aab971d72cbd59597b3356 · covers:the refusal of a create carrying no body
- 2026-09-06 · dd0cc3a* · mutant killed · exit 1 · `internal/plan/plan.go` · the declaration is dropped on the way out of the parser, so `create body=0` becomes indistinguishable from a lost body again and the deliberate empty file is refused too · acceptance-sha256:8d332ebfef1a0bc0a7dc362ec806a0fb9e850d1676aab971d72cbd59597b3356 · covers:body=0 still creating an empty file
- 2026-09-06 · a5e4347* · mutant killed · exit 1 · `internal/plan/plan.go` · the create branch stops refusing a body-less hunk, so a plan whose last create lost its body creates an empty file and reports ok · acceptance-sha256:fbd6f89cf93719c55c333eddab55f6822493eba44da2a804f64124aac5827cd7 · covers:the refusal of a create carrying no body
- 2026-09-06 · a5e4347* · mutant killed · exit 1 · `internal/plan/plan.go` · the declaration is dropped leaving the parser, so `create body=0` is refused too and the change becomes a ban rather than a narrowing · acceptance-sha256:fbd6f89cf93719c55c333eddab55f6822493eba44da2a804f64124aac5827cd7 · covers:body=0 still creating an empty file
- 2026-09-06 · a5e4347* · mutant killed · exit 1 · `internal/plan/plan.go` · the insertion ops stop refusing an empty body, which the new create rule must not have traded away — the neighbour segment of this fence ran ZERO tests before this round and exited 0 · acceptance-sha256:fbd6f89cf93719c55c333eddab55f6822493eba44da2a804f64124aac5827cd7 · covers:the neighbouring ops keeping their own refusals

## Invariants

- `create body=0` still produces an empty file: this record changes the spelling required, never the capability.
- The empty-body refusals for `replace`, `insert-after` and `insert-before` are untouched — `create` was the last op where a lost body was reported as success, and closing it must not restate the other three.
- `body=N` counting, overcount detection and the `raw=` interaction are unchanged.
- `internal/read`, `internal/apply`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base. `go.mod` declares exactly one requirement.

## Risks

- Carrying "a count was declared" onto the hunk widens a struct the whole plan path reads. Mitigated by it being one bool set where the parser already computes it, and by the engine packages staying byte-identical.
- A caller creating empty files in bulk without `body=0` breaks. Measured 2026-09-06: every one of the 18 `create` hunks in this repository carries a body, so nothing in-tree relies on it; the cost falls on external callers and the fix is one token.

## Stop Condition

Stop and ask if refusing a body-less `create` requires changing what `body=0` means for any other
op. `body=0` is a count and this record reuses it as a declaration; if the two readings turn out to
conflict, that is a decision about the guard rather than about `create`.

## Out of Scope

- The contract row and the documented grammar — that is T2's job
- Requiring `body=N` on every op (permanent: boundary: the count is opt-in and stays opt-in; this record uses it as the one available way to SAY "deliberately nothing", not as a new obligation)

## Verification Log
- 2026-09-06 · dd0cc3a* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d332ebfef1a0bc0a7dc362ec806a0fb9e850d1676aab971d72cbd59597b3356 · ms:5245
- 2026-09-06 · dd0cc3a* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d332ebfef1a0bc0a7dc362ec806a0fb9e850d1676aab971d72cbd59597b3356 · ms:5304
- 2026-09-06 · dd0cc3a* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d332ebfef1a0bc0a7dc362ec806a0fb9e850d1676aab971d72cbd59597b3356 · ms:6295
- 2026-09-06 · a5e4347* · exit 0 · `set -o pipefail …` · acceptance-sha256:fbd6f89cf93719c55c333eddab55f6822493eba44da2a804f64124aac5827cd7 · ms:5288
- 2026-09-06 · a5e4347* · exit 0 · `set -o pipefail …` · acceptance-sha256:fbd6f89cf93719c55c333eddab55f6822493eba44da2a804f64124aac5827cd7 · ms:5257
- 2026-09-06 · a5e4347* · exit 0 · `set -o pipefail …` · acceptance-sha256:fbd6f89cf93719c55c333eddab55f6822493eba44da2a804f64124aac5827cd7 · ms:5212
- 2026-09-06 · a5e4347* · exit 0 · `set -o pipefail …` · acceptance-sha256:fbd6f89cf93719c55c333eddab55f6822493eba44da2a804f64124aac5827cd7 · ms:5401
