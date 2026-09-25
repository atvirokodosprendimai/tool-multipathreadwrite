# Task ADR-072-T2: The receipt is printed before the check runs

**Depends-on:** T1, T3
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `authoring.Reclassify`; the reordered write tail
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the receipt is on stdout before the check starts`, `a landing is counted before the check`, `a reclassify never adds a plan`, `a contract row drives the binary`, `the engine packages are unchanged`

## Goal

A write killed during its check printed nothing, and `mrw stats` never counted it: the receipt and the tally both came after `check.Run`. Print the human receipt and count the landing first; the check's verdict then moves that one count.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | receipt and provisional tally before the check; `Reclassify` after |
| `internal/authoring/authoring.go` | edit | `Reclassify`; `save` shared with `Record` |
| `internal/authoring/reclassify_test.go` | new | one count moves; none is added |
| `cmd/mrw/receipt_before_check_test.go` | new | a built mrw killed by its own check still printed the receipt |
| `scripts/contract.sh` | edit | §143 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN; the existing write and check tests stay green. [proof: mutation]
   Mutants: the receipt moved back after the check; the provisional Applied record removed; Reclassify increments without decrementing.
3. [S3] The contract row drives the built binary with the good case and the one that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ ./internal/authoring/ -count=1 -timeout 180s -run 'TestTheReceiptIsOnStdout|TestAFailedCheckIsCountedOnce|TestReclassify' -v 2>&1 | tee /tmp/adr072-T2.out \
  && missing=$(for t in TestTheReceiptIsOnStdoutBeforeTheCheckStarts TestAFailedCheckIsCountedOnceInStats TestReclassifyMovesOneCountAndNeverAddsOne; do grep -qE "^--- PASS: $t \(" /tmp/adr072-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 143\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheReceiptIsOnStdoutBeforeTheCheckStarts` | `cmd/mrw/receipt_before_check_test.go` | a check that kills mrw (`kill -9 $PPID`) finds the receipt already printed, and `stats` counts the landing | — | S1, S2 |
| `TestAFailedCheckIsCountedOnceInStats` | `cmd/mrw/receipt_before_check_test.go` | a failing check: `failed_check` 1, `applied` 0, landed 1 | — | S1, S2 |
| `TestReclassifyMovesOneCountAndNeverAddsOne` | `internal/authoring/reclassify_test.go` | the move, and its floor at zero | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code and its tests |
| 2 — something selects it | every `mrw write`; `mrw check` for T4 |
| 3 — the caller can discover it | the receipt, the exit code and the refusal text |
| 4 — it is used | the v1.25.1 round reproduced each defect with the shipped binary |

## Verification Log
(empty until execute)
- 2026-09-25 · e3f7978* · exit 1 · `set -o pipefail …` · acceptance-sha256:0f4e6e53a426b156c70e052c3499ebac6efa171d420aaa91ee577c7c85078d7a · ms:683 · test-lock-sha256:945ddc65c80c93fb0d9806411f34962c4700189a4cbca62880a83e5f683f8561 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcmVjZWlwdF9iZWZvcmVfY2hlY2tfdGVzdC5nbwlUZXN0QUZhaWxlZENoZWNrSXNDb3VudGVkT25jZUluU3RhdHMJZTdkYTI0NGIyZTUyMmEwYzBhOTI4MDdmYTk5M2Y3ODhlYTIxMDk5ZTcxN2QwZTRjNjEyZTgyODI1ZWI4YTc4YQpib2R5CWNtZC9tcncvcmVjZWlwdF9iZWZvcmVfY2hlY2tfdGVzdC5nbwlUZXN0VGhlUmVjZWlwdElzT25TdGRvdXRCZWZvcmVUaGVDaGVja1N0YXJ0cwlmMWZjNDVmNGZhY2QzZTQzMmU0NWE3YjkwZDE2MDdmMzJkN2E1ZmVjMDczYWE3YWE5MjgzMjY5ZTVhOGZjNzYxCmJvZHkJaW50ZXJuYWwvYXV0aG9yaW5nL3JlY2xhc3NpZnlfdGVzdC5nbwlUZXN0UmVjbGFzc2lmeU1vdmVzT25lQ291bnRBbmROZXZlckFkZHNPbmUJYTZjZmIxMDMxNTE0YmQ0OTY2OTYwYWUzZmM1YWYxNTUzNDY2YzFlODk5MzgwZDMzMDc2MmM1OGY3OTMzNjg4OA
  ```
  --- last 10 line(s) of stdout (of 15 after folding 15 raw)
          ""
      receipt_before_check_test.go:59: stats does not count the killed landing:
          no plans recorded yet — this says nothing has been MEASURED, not that nothing has failed
  --- FAIL: TestTheReceiptIsOnStdoutBeforeTheCheckStarts (0.33s)
  === RUN   TestAFailedCheckIsCountedOnceInStats
  --- PASS: TestAFailedCheckIsCountedOnceInStats (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.416s
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring [build failed]
  FAIL
  ```
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:0f4e6e53a426b156c70e052c3499ebac6efa171d420aaa91ee577c7c85078d7a · ms:682
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:0f4e6e53a426b156c70e052c3499ebac6efa171d420aaa91ee577c7c85078d7a · ms:2237
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:0f4e6e53a426b156c70e052c3499ebac6efa171d420aaa91ee577c7c85078d7a · ms:1447
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:0f4e6e53a426b156c70e052c3499ebac6efa171d420aaa91ee577c7c85078d7a · ms:2114

## Mutation Log
(empty until execute)
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `cmd/mrw/main.go` · the human receipt is not printed before the check, so a killed write prints nothing · acceptance-sha256:0f4e6e53a426b156c70e052c3499ebac6efa171d420aaa91ee577c7c85078d7a · covers:the receipt is on stdout before the check starts
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `cmd/mrw/main.go` · the landing is not counted before the check, so a killed write is missing from stats · acceptance-sha256:0f4e6e53a426b156c70e052c3499ebac6efa171d420aaa91ee577c7c85078d7a · covers:a landing is counted before the check
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `internal/authoring/authoring.go` · a reclassify adds a plan instead of moving one · acceptance-sha256:0f4e6e53a426b156c70e052c3499ebac6efa171d420aaa91ee577c7c85078d7a · covers:a reclassify never adds a plan

## Invariants

- Exit codes keep their meaning.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-072 Out of Scope)

## Stop Condition

Stop if the change needs an engine package other than `internal/check`.
