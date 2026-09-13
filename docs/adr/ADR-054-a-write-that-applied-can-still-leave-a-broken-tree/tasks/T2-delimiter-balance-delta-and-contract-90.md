# Task ADR-054-T2: Delimiter-balance delta on non-prose; contract §90

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `HunkResult.Balance` (T2)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `report not refuse`, `net per family`, `quiet hides ok`, `prose omits balance`

## Goal

An applied replace/insert/delete whose path is not prose and whose net `{`/`}` `(`/`)` `[`/`]` differs between the original addressed lines and the body carries `HunkResult.Balance` naming the family and the two nets. The hunk stays `ok`. A prose path (`.md`, `.markdown`, `.txt`, `.rst`, `.adoc`) omits `Balance` even when the nets differ. Quiet hides it. No string lexer. Skip hunks omit it, same as Echo.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | Compute nets; set `Balance` on applied non-prose hunks when they differ. The selector. |
| `internal/apply/apply_test.go` | edit | Red tests: `+1 → 0` stays ok; matching nets omit the field; prose omits even when nets differ. |
| `cmd/mrw/main.go` | edit | `report` prints the balance under the `ok` line, like Echo. |
| `scripts/contract.sh` | edit | **§90** — a replace of a `{`-only line in a `.go` file with a balanced body prints the delta and exit 0; the same replace in a `.md` file does not print it. |

## Ordered Steps

1. [S1] Write `TestADelimiterBalanceDeltaDoesNotFailTheHunk` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestMatchingNetsOmitBalance` and `TestProseOmitsBalance` — RED until the omitempty and prose paths exist. [proof: mutation]
3. [S3] Implement net counts and `HunkResult.Balance`. Un-strike the Tests rows when the funcs exist. Confirm S1–S2 GREEN. Deleting the net compare must fail S1. Deleting the prose skip must fail `TestProseOmitsBalance`. [proof: mutation]
4. [S4] Print the field from `report` on `ok` hunks (not quiet). Write §90 RED then GREEN. [proof: mutation]
5. [S5] Scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 90\. ' scripts/contract.sh \
  && go test ./internal/apply/ -count=1 -v \
    -run 'TestADelimiterBalanceDeltaDoesNotFailTheHunk|TestMatchingNetsOmitBalance|TestProseOmitsBalance' 2>&1 | tee /tmp/adr054-t2.out \
  && grep -q '^--- PASS: TestADelimiterBalanceDeltaDoesNotFailTheHunk' /tmp/adr054-t2.out \
  && grep -q '^--- PASS: TestMatchingNetsOmitBalance' /tmp/adr054-t2.out \
  && grep -q '^--- PASS: TestProseOmitsBalance' /tmp/adr054-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr054-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/apply/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| ~~`TestADelimiterBalanceDeltaDoesNotFailTheHunk`~~ | `internal/apply/apply_test.go` | not yet written — Proposed; T2 S1 writes it | — | S1, S3 |
| ~~`TestMatchingNetsOmitBalance`~~ | `internal/apply/apply_test.go` | not yet written — Proposed; T2 S2 writes it | — | S2, S3 |
| ~~`TestProseOmitsBalance`~~ | `internal/apply/apply_test.go` | not yet written — Proposed; T2 S2 writes it | — | S2, S3 |
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

## Invariants

- ADR-001: the delta never fails a hunk.
- ADR-048: no parser.
- ADR-052: Echo still opt-in; this is a sibling field, not a default echo.
- Skip hunks do not carry Balance.
- The prose list is the Decision's five extensions.
- A balanced insert (original net 0, body net 0) omits Balance even when it leaves the file unclosed. Arm 1 catches that; this arm does not. Do not invent a delta for it.

## Risks

- Braces in strings in code still false-positive. Accepted; report only.
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
