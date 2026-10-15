# Task ADR-159-T2: a descriptive row shows each stage without the tasks superseded since it ran

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `Analyze(rs, superseded...)`'s row `withoutSupersededSince`; `bench analyze --tasks-dir`
**Consumes:** `superseded_by` on `cac-rename-post-send` (T1)
**Data dependency:** `docs/bench/stage-1-sonnet/` and `docs/bench/stage-1b-sonnet/`, in the tree
**Proof map:** v1
**Rests-on:** `the row appears only where a superseded task ran`, `the verdict row still counts every task`, `the published verdicts are unchanged`

## Goal

Decision 2 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/analyze.go` | edit | `Analyze` takes the superseded ids; the row and its label |
| `cmd/bench/main.go` | edit | `analyze` loads the corpus (`--tasks-dir`) and passes them |
| `internal/bench/analyze_test.go` | edit | the tests below |
| `docs/bench/README.md` | edit | the row |

## Ordered Steps

1. [S1] Write the failing tests and see them red:
   - `TestARowWithoutTheTasksSupersededSinceIsDescriptive`: receipts of two tasks, one superseded, give the verdict over both and a row over the other, labelled as registered after the stage ran; with no superseded id, or none that ran, no such row.
   - `TestThePublishedStagesWithoutTheTasksSupersededSince`: `Analyze` over `docs/bench/stage-1-sonnet` and `docs/bench/stage-1b-sonnet` with the corpus's superseded ids keeps each verdict row as published and adds the row (Stage 1: 27 tasks; Stage 1b: 39 tasks).
2. [S2] The row in `Analyze`. [proof: mutation]
3. [S3] `bench analyze` passes the corpus's superseded ids. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/bench/ ./cmd/bench/ -count=1 -timeout 600s -run 'TestARowWithoutTheTasksSupersededSinceIsDescriptive|TestThePublishedStagesWithoutTheTasksSupersededSince|TestStageOnesPublishedVerdictIsUnchanged' -v > "$out" 2>&1 \
  && grep -qE '^--- PASS: TestARowWithoutTheTasksSupersededSinceIsDescriptive \(' "$out" \
  && grep -qE '^--- PASS: TestThePublishedStagesWithoutTheTasksSupersededSince \(' "$out" \
  && grep -qE '^--- PASS: TestStageOnesPublishedVerdictIsUnchanged \(' "$out" \
  && go run ./cmd/bench analyze docs/bench/stage-1b-sonnet > "$out" 2>&1 \
  && grep -q 'without the tasks superseded after the stage ran' "$out"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARowWithoutTheTasksSupersededSinceIsDescriptive` | `internal/bench/analyze_test.go` | the row where a superseded task ran and nowhere else; the verdict unchanged | none | S1, S2 |
| `TestThePublishedStagesWithoutTheTasksSupersededSince` | `internal/bench/analyze_test.go` | both published verdicts unchanged; the row's task counts | none | S1, S2 |
| `TestStageOnesPublishedVerdictIsUnchanged` | `internal/bench/analyze_test.go` | Stage 1's numbers without the new argument | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the row |
| 2 — something selects it | `bench analyze`, from the corpus |
| 3 — the caller can discover it | `docs/bench/README.md`; the label |
| 4 — it is used | Stage 1 and 1b's `bench analyze` output; ADR-156 T6 |

## Mutation Log
(empty until execute)
- 2026-10-15 · ffbcfd04* · mutant killed · exit 1 · `internal/bench/analyze.go` · S2: no row without the tasks superseded since · acceptance-sha256:3041c204b085c9a9e66557aa28ff96bad84f6a1d243ce32440c908a2f5f96428
- 2026-10-15 · ffbcfd04* · mutant killed · exit 1 · `cmd/bench/main.go` · S3: bench analyze passes no superseded task, so the row never prints · acceptance-sha256:3041c204b085c9a9e66557aa28ff96bad84f6a1d243ce32440c908a2f5f96428

## Verification Log
(empty until execute)
- 2026-10-15 · ffbcfd04* · exit 1 · `set -o pipefail …` · acceptance-sha256:3041c204b085c9a9e66557aa28ff96bad84f6a1d243ce32440c908a2f5f96428 · ms:137 · test-lock-sha256:1d9fe098eada913af0a1db99a33104f24270345476df2165e83b79cff7917537 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUGlsb3RJc05ldmVyQ291bnRlZAkxNzExNDE3YjczMzQ5NzMxMzgwMzNjODE4ZWM1NGFjM2Y2M2Q2NWUyYWUyYjFhNzhkOThiYTU1NjFlYjIwYTVhCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUm93V2l0aG91dFRoZVRhc2tzU3VwZXJzZWRlZFNpbmNlSXNEZXNjcmlwdGl2ZQkwYTc3OGYzZDBjZjE3NWRiMTkyNzEwOWI4YWMwNDRjM2JjOGJhMGJkNDM0OTE4YWY0YjA1ZWUyMTU1MjhlNzY0CmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RPbmx5VGhlUmVnaXN0ZXJlZEV4Y2x1c2lvbnNBcmVBZG1pdHRlZAlkYjg0NzQyNGM3Njc0ZTcwYWU1NDc3N2U5OTM0NDYxYWU4ZDU5MDJiZmQ2ZTJkYzkwYjRiNDE2YmFjOGJiZjVmCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RTdGFnZU9uZXNQdWJsaXNoZWRWZXJkaWN0SXNVbmNoYW5nZWQJOTYwOWJiZWUzNjkwYmExZDFiMWNlMjI1Y2JmZjk0MzUxZjZmMGNkNDczZDk5Yzk3YTU1YjYxZDJlNjUyY2E0NQpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VGhlQW5hbHlzaXNQYWlyc0J5VGFza0FuZEtlZXBzRXZlcnlBdHRlbXB0CWIwOTAzZmRkZmY1ZjMyZjlkYzhhNGFlNTYyOTNmZTFlMDQyNDAyMGY1MTU1OGVmOTAyNmZjNDc3MTBjMjcwMWIKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFRoZVB1Ymxpc2hlZFN0YWdlc1dpdGhvdXRUaGVUYXNrc1N1cGVyc2VkZWRTaW5jZQk4MGFhOWU5NmYyMjBjYjAyYzM1MDExYjhlM2RjMmE5ZjNhMjAzZGFhN2I5OTk3NmZmZDRmYjZmNTdlODA5NmZiCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RUaGVWYXJpZWRNdWx0aVNpdGVSb3dJc0Rlc2NyaXB0aXZlCTZlZWY0MzFhYTZlN2M5YmRiZmI5OGZiZTc4YzdhMzE2YjdiYzBlNGZjY2FhYjFjZTYwNWVhMWU2YjdlMjk1NDEKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFRoZVZlcmRpY3RGb2xsb3dzVGhlUmVnaXN0ZXJlZEJhcgk1MTY1MTYzNDJlMDZhN2I2Mzk1ZGYwODQyYTc0OTdmYzE3YjRkNTA3MGQwMWQyMzNmNTVmZDMyMGI4NTMzOWUzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RUd29TdGFnZXNBcmVOZXZlclBvb2xlZAk1ODEwZjk2ZmE2MjZjZjM4Yjg4NjU5ZjI5Y2Q1YzUwOWQ4OTZkNjFjZDZkZjg2ZWJlMGM4ZmZkM2VmYmYxZjA2
  ```
  ```
- 2026-10-15 · ffbcfd04* · exit 0 · `set -o pipefail …` · acceptance-sha256:3041c204b085c9a9e66557aa28ff96bad84f6a1d243ce32440c908a2f5f96428 · ms:1180
- 2026-10-15 · ffbcfd04* · exit 0 · `set -o pipefail …` · acceptance-sha256:3041c204b085c9a9e66557aa28ff96bad84f6a1d243ce32440c908a2f5f96428 · ms:1080

## Invariants

- Every verdict row is the registered bar's, over every task of its stage.

## Risks

- The row is read as a verdict: its label says it was registered after the stage ran, and it sits in parentheses.

## Out of Scope

- A row for any other post-hoc grouping (permanent: boundary: one row, for one known defect class, chosen by Zy)

## Stop Condition

The fence exits 0.
