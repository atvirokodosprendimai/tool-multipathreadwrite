# Task ADR-032-T1: The budget is configured, and it bounds the encoded result

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (a constant becomes a field, and the measurement point moves to the answer)
**Owner:** Zy
**Produces:** the configured budget replacing the constant (T2, T3)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the advertised value being the enforced value`, `the bound measured on the encoded result`

## Goal

Let the caller set the ceiling, and measure it where the caller meets it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/schema.go` | edit | `MaxResultChars` stops being the enforced value and becomes the DEFAULT; the server carries a configured budget. |
| `internal/mcp/mcp.go` | edit | `_meta` advertises the configured value on both tools, which is what `mcp.go:81` already promises. |
| `internal/mcp/tools.go` | edit | The bound is checked against the ENCODED result, after the receipt and any structured content are composed. |
| `cmd/mrw/main.go` | edit | `--max-result-chars N` and `MRW_MAX_RESULT_CHARS`; absence means the default, `0` means zero (ADR-033). |
| `internal/mcp/limit_test.go` | create | `TestTheAdvertisedCeilingBoundsEveryAnswer`. |

## Ordered Steps

1. [S1] Write `TestTheAdvertisedCeilingBoundsEveryAnswer` and confirm it is RED for `mrw_write`: a 4,000-hunk dry-run returns 453,632 characters against an advertised 200,000 today. ⚠ Assert against the ACTUAL encoded JSON the server writes, not a reconstruction: ADR-031 shipped a size check on a buffer that was not what got sent, twice. [proof: mutation]
2. [S2] Carry a configured budget through the server, defaulting to `MaxResultChars`. [proof: mutation]
3. [S3] Advertise the configured value in `_meta` on both tools. [proof: mutation]
4. [S4] Move the read path's final judgement onto the encoded result, keeping `capped` as the streaming bound it already is. [proof: mutation]
5. [S5] Wire `--max-result-chars` and `MRW_MAX_RESULT_CHARS`, with `0` meaning zero and absence meaning the default. [proof: acceptance]
6. [S6] Run every gate. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -v \
  -run 'TestTheAdvertisedCeilingBoundsEveryAnswer' 2>&1 | tee /tmp/adr032-t1.out \
  && grep -q '^--- PASS: TestTheAdvertisedCeilingBoundsEveryAnswer' /tmp/adr032-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr032-t1.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheAdvertisedCeilingBoundsEveryAnswer` | `internal/mcp/limit_test.go` | For both tools, the encoded result the server writes is within the value it advertises — measured on the JSON, not on the report text — at the default and at a caller-set budget | — | S1, S2, S3, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestTheAdvertisedCeilingBoundsEveryAnswer` |
| 2 — something selects it | Every tool call composes its result through the same path; the S2 mutation restores the constant and the fence goes red on the advertised value |
| 3 — the caller can discover it | `_meta` on both tools, and the flag's usage string (T2) |
| 4 — it is used | Contract §70 (T3) drives the built server; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · b36aea1* · mutant killed · exit 1 · `internal/mcp/tools.go` · the read path enforces the compiled-in default while _meta advertises the budget the caller set, so the advertised limit and the enforced one drift apart again · acceptance-sha256:a9aaf05f961c3616f845ba4bf135f4c25cac538061b3588b7cf1ac395bd79132 · covers:the advertised value being the enforced value
- 2026-09-07 · b36aea1* · mutant killed · exit 1 · `internal/mcp/tools.go` · the encoded judgement is reached only through grep again, so a caller naming its own specs has its report bounded and its receipt not · acceptance-sha256:a9aaf05f961c3616f845ba4bf135f4c25cac538061b3588b7cf1ac395bd79132 · covers:the bound measured on the encoded result

## Invariants
- The advertised value IS the enforced value, on both tools. That is `mcp.go:81`'s existing promise and today it holds only for reads.
- `0` means zero and absence means the default — ADR-033's decision, not re-argued here.
- ⚠ The bound is measured on the ENCODED result. Measuring a buffer that is not what gets sent is the mistake ADR-031 made twice.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` and `internal/state` stay byte-identical against the merge base.

## Risks

- Measuring the wrong quantity again. The test asserts against the JSON the server actually writes; a reconstruction would repeat ADR-031's error.
- A configured budget that no host sets leaves the enforcement half carrying the record. Acceptable: that half is the measured defect.

## Stop Condition

Stop and ask if bounding the encoded result requires composing it twice for large answers — that is
a cost the record has not weighed, and it would need measuring before it is accepted.

## Out of Scope

- The write receipt's elision — T2
- The contract row and the documentation — T3

## Verification Log
- 2026-09-07 · b36aea1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a9aaf05f961c3616f845ba4bf135f4c25cac538061b3588b7cf1ac395bd79132 · ms:19662
- 2026-09-07 · b36aea1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a9aaf05f961c3616f845ba4bf135f4c25cac538061b3588b7cf1ac395bd79132 · ms:19306
- 2026-09-07 · b36aea1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a9aaf05f961c3616f845ba4bf135f4c25cac538061b3588b7cf1ac395bd79132 · ms:31282
