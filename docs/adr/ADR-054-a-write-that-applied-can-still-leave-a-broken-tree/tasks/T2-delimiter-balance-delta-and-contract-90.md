# Task ADR-054-T2: Delimiter-balance delta on non-prose; contract §90

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `HunkResult.Balance` (T2)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `report not refuse`, `net per family`, `quiet hides ok`, `prose omits balance`, `delete body is a guard`

## Goal

An applied replace/insert/delete whose path is not prose and whose net `{`/`}` `(`/`)` `[`/`]` differs between the original addressed lines and the body carries `HunkResult.Balance` naming the family and the two nets. The hunk stays `ok`. A prose path (`.md`, `.markdown`, `.txt`, `.rst`, `.adoc`) omits `Balance` even when the nets differ. Quiet hides it. No string lexer. Skip hunks omit it, same as Echo.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | Compute nets; set `Balance` on applied non-prose hunks when they differ. The selector. |
| `internal/apply/apply_test.go` | edit | Red tests: `+1 → 0` stays ok; matching nets omit the field; prose omits even when nets differ. |
| `cmd/mrw/main.go` | edit | `report` prints the balance under the `ok` line, like Echo. |
| `internal/mcp/schema.go` | edit | `hunks.balance` description — `TestEveryOutputSchemaPropertyIsDescribed` refuses an undescribed property on the `mrw_write` receipt. |
| `scripts/contract.sh` | edit | **§90** — a replace of a `{`-only line in a `.go` file with a balanced body prints the delta and exit 0; the same replace in a `.md` file does not print it. |

## Ordered Steps

1. [S1] Write `TestADelimiterBalanceDeltaDoesNotFailTheHunk` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestMatchingNetsOmitBalance` and `TestProseOmitsBalance` — RED until the omitempty and prose paths exist. [proof: mutation]
3. [S3] Implement net counts and `HunkResult.Balance`. Confirm S1–S2 GREEN. Deleting the net compare must fail S1. Deleting the prose skip must fail `TestProseOmitsBalance`. [proof: mutation]
4. [S4] Print the field from `report` on `ok` hunks (not quiet). Write §90 RED then GREEN. [proof: mutation]
5. [S5] Scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 90\. ' scripts/contract.sh \
  && go test ./internal/apply/ -count=1 -v \
    -run 'TestADelimiterBalanceDeltaDoesNotFailTheHunk|TestMatchingNetsOmitBalance|TestProseOmitsBalance|TestADeleteWithAnExpectedBodyStillReportsItsNet|TestRandomisedApplyBalanceFollowsTheDecision' 2>&1 | tee /tmp/adr054-t2.out \
  && grep -q '^--- PASS: TestADelimiterBalanceDeltaDoesNotFailTheHunk' /tmp/adr054-t2.out \
  && grep -q '^--- PASS: TestMatchingNetsOmitBalance' /tmp/adr054-t2.out \
  && grep -q '^--- PASS: TestProseOmitsBalance' /tmp/adr054-t2.out \
  && grep -q '^--- PASS: TestADeleteWithAnExpectedBodyStillReportsItsNet' /tmp/adr054-t2.out \
  && grep -q '^--- PASS: TestRandomisedApplyBalanceFollowsTheDecision' /tmp/adr054-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr054-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/apply/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestADelimiterBalanceDeltaDoesNotFailTheHunk` | `internal/apply/apply_test.go` | `{ +1 → 0}` is set on a `.go` hunk; status ok; file written | — | S1, S3 |
| `TestMatchingNetsOmitBalance` | `internal/apply/apply_test.go` | Equal nets → empty Balance, not marshalled | — | S2, S3 |
| `TestProseOmitsBalance` | `internal/apply/apply_test.go` | A `.md` hunk with differing nets has empty Balance and stays ok | — | S2, S3 |
| `TestADeleteWithAnExpectedBodyStillReportsItsNet` | `internal/apply/apply_test.go` | ADR-008's expected body is a guard, not a write: delete's body net stays 0 | — | S3 |
| `TestRandomisedApplyBalanceFollowsTheDecision` | `internal/apply/balance_fuzz_test.go` | 400 random ops/paths per seed against an independent splice and net oracle; found the delete-with-body miss | — | S3 |
| `§90` | `scripts/contract.sh` | Built binary prints the delta on `.go` and exits 0; omits it on `.md` | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §90 |
| 2 — something selects it | apply after a successful hunk; deleting the compare fails S1 |
| 3 — the caller can discover it | T4; the `ok` line grows a balance row |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
(empty until execute)
- 2026-09-13 · f965704* · mutant killed · exit 1 · `internal/apply/apply.go` · the net compare is deleted: no family ever reports, so a `{`-only line replaced by a balanced body carries no Balance and TestADelimiterBalanceDeltaDoesNotFailTheHunk must go red · acceptance-sha256:dfb357dc10e031d0f4b5c4d2a93bb0f73cda049336d5de401eb625a9679660f1
- 2026-09-13 · f965704* · mutant survived · exit 0 · `internal/apply/apply.go` · the prose skip is deleted: a .md hunk with differing nets carries Balance and TestProseOmitsBalance must go red · acceptance-sha256:dfb357dc10e031d0f4b5c4d2a93bb0f73cda049336d5de401eb625a9679660f1
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-13 · f965704* · mutant killed · exit 1 · `internal/apply/apply.go` · the prose skip is deleted: a .md hunk whose consumed line is net +1 against a net-0 body carries Balance, and TestProseOmitsBalance must go red · acceptance-sha256:dfb357dc10e031d0f4b5c4d2a93bb0f73cda049336d5de401eb625a9679660f1
- 2026-09-13 · 38d3831* · mutant inconclusive · exit 1 · `internal/apply/apply.go` · the delete regression: the expected body is fed into the delta and cancels every guarded delete; TestADeleteWithAnExpectedBodyStillReportsItsNet and the randomised test must go red · acceptance-sha256:a52c9e07b166ce9f93bc47a22e50b72ae37ad7fcec561d7365090a247f4eddb4
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-13 · 38d3831* · mutant killed · exit 1 · `internal/apply/apply.go` · the delete regression: the ADR-008 expected body is treated as written and cancels every guarded delete; TestADeleteWithAnExpectedBodyStillReportsItsNet and TestRandomisedApplyBalanceFollowsTheDecision must go red · acceptance-sha256:a52c9e07b166ce9f93bc47a22e50b72ae37ad7fcec561d7365090a247f4eddb4

## Invariants

- ADR-001: the delta never fails a hunk.
- ADR-048: no parser.
- ADR-052: Echo still opt-in; this is a sibling field, not a default echo.
- Skip hunks do not carry Balance.
- The prose list is the Decision's five extensions.
- A balanced insert (original net 0, body net 0) omits Balance even when it leaves the file unclosed. Arm 1 catches that; this arm does not. Do not invent a delta for it.

## Risks

- Braces in strings in code still false-positive. Accepted; report only.
- A delete's ADR-008 expected body is the consumed lines verbatim; fed into the delta it cancels every guarded delete. Shipped that way in 3b9f772, found by the randomised test the same day, fixed to feed the WRITTEN lines (nil for delete).
- Four files in Affected. `report` must print the field or the engine-only test is unreachable from the CLI. Deleting the `report` loop must fail §90.

## Stop Condition

If the only way to go green is to refuse the hunk, stop.
If the only way to go green is to print a delta on a `.md` hunk, stop — that is the High ignore-training finding.
If the only way to go green is to print a delta on a balanced insert (net 0 → 0), stop — that case is invisible to this arm.

## Out of Scope

- Default check (T1)
- stats (T3)
- Teaching (T4)
- YAML indent

## Verification Log
(empty until execute)
- 2026-09-13 · f965704* · exit 1 · `set -o pipefail …` · acceptance-sha256:dfb357dc10e031d0f4b5c4d2a93bb0f73cda049336d5de401eb625a9679660f1 · ms:25 · test-lock-sha256:032bd1e14edc8d215e0e72ce119d7a15b35b8c66e341382c4a26057fd21edb6f · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwp1bnByb3ZlbglpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRGVsaW1pdGVyQmFsYW5jZURlbHRhRG9lc05vdEZhaWxUaGVIdW5rCnVucHJvdmVuCWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdE1hdGNoaW5nTmV0c09taXRCYWxhbmNlCnVucHJvdmVuCWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdFByb3NlT21pdHNCYWxhbmNlCnVucHJvdmVuCXNjcmlwdHMvY29udHJhY3Quc2gJwqc5MA
  ```
  ```
- 2026-09-13 · f965704* · exit 0 · `set -o pipefail …` · acceptance-sha256:dfb357dc10e031d0f4b5c4d2a93bb0f73cda049336d5de401eb625a9679660f1 · ms:501
- 2026-09-13 · f965704* · exit 0 · `set -o pipefail …` · acceptance-sha256:dfb357dc10e031d0f4b5c4d2a93bb0f73cda049336d5de401eb625a9679660f1 · ms:398
- 2026-09-13 · f965704* · exit 0 · `set -o pipefail …` · acceptance-sha256:dfb357dc10e031d0f4b5c4d2a93bb0f73cda049336d5de401eb625a9679660f1 · ms:657
- 2026-09-13 · 38d3831* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c9e07b166ce9f93bc47a22e50b72ae37ad7fcec561d7365090a247f4eddb4 · ms:601
- 2026-09-13 · 38d3831* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c9e07b166ce9f93bc47a22e50b72ae37ad7fcec561d7365090a247f4eddb4 · ms:865
