# Task ADR-060-T7: `replace` / `insert-*` parse with `body=@`

**Depends-on:** T4, T6
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** parse accepts empty Body when `BodyFile` is set; §105
**Consumes:** T4 `LoadBodyFiles`; T6 curve load
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `replace body=@ parses`, `insert body=@ parses`, `empty replace without @ still refused`, `CLI replace writes loaded bytes`

## Goal

`body=@path` on `replace`, `insert-after` and `insert-before` parses. Validate currently refuses those ops with an empty Body, and `body=@` arrives as empty Body plus `BodyFile`, so Load never runs. Skip the empty-body refuse when `BodyFile` is set. After load, an empty file is still `body=0`: Apply refuses empty replace/insert. Do not load inside Parse.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | Empty-body refuse for replace/insert waits when `BodyFile` is set. |
| `internal/plan/bodyfile_replace_test.go` | create | Red: those ops parse with `body=@`; inline empty replace still refuses; load fills Body. Dedicated file so T4's lock on `bodyfile_test.go` does not move. |
| `cmd/mrw/bodyfile_replace_write_test.go` | create | CLI replace/insert from a body file writes those bytes. |
| `internal/curve/score_bodyfile_replace_test.go` | create | Scorer Hit via replace `body=@payload.txt` containing the planted line. |
| `scripts/contract.sh` | edit | **§105**. |
| `docs/adr/ADR-060-plan-ux-from-the-v1-19-field-report.md` | edit | Decision 4 names replace/insert; Governs §105. |

## Ordered Steps

1. [S1] Write the failing tests. [proof: mutation]
2. [S2] Defer empty-body validate when `BodyFile` is set. S1 GREEN. [proof: mutation]
3. [S3] §105 RED then GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 105\. ' scripts/contract.sh \
  && go test ./internal/plan/ ./cmd/mrw/ ./internal/curve/ -count=1 -v \
    -run 'TestBodyAtPathReplaceParsesAndLoads|TestWriteBodyAtPathEditsFromFile|TestScoreTrialHitsAReplaceFromBodyAtPath' 2>&1 | tee /tmp/adr060-t7.out \
  && grep -q '^--- PASS: TestBodyAtPathReplaceParsesAndLoads' /tmp/adr060-t7.out \
  && grep -q '^--- PASS: TestWriteBodyAtPathEditsFromFile' /tmp/adr060-t7.out \
  && grep -q '^--- PASS: TestScoreTrialHitsAReplaceFromBodyAtPath' /tmp/adr060-t7.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr060-t7.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/plan/ ./cmd/mrw/ ./internal/curve/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/seen internal/check internal/state internal/read \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestBodyAtPathReplaceParsesAndLoads` | `internal/plan/bodyfile_replace_test.go` | `replace`/`insert-after` `body=@src.txt` parse; load fills those lines; `replace` with no body and `replace body=0` still refuse | — | S1, S2 |
| `TestWriteBodyAtPathEditsFromFile` | `cmd/mrw/bodyfile_replace_write_test.go` | CLI replace and insert-after `body=@` write the file's bytes | — | S1, S2 |
| `TestScoreTrialHitsAReplaceFromBodyAtPath` | `internal/curve/score_bodyfile_replace_test.go` | replace `body=@payload.txt` with the planted line is Hit; skipping parse leaves RefusedParse | — | S1, S2 |

Subtests of those names so a mutant of any arm is inside the fence.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `validate` skips empty-body refuse when `BodyFile != ""`. Dropping the skip leaves S1 red |
| 3 — the caller can discover it | §105; T5 already taught `body=@path` (op-agnostic) |
| 4 — it is used | field-report 600-line bodies were create; M 2026-09-14: replace/insert are equally the path. No telemetry (ADR-009) |

## Mutation Log
_(tool-written)_
- 2026-09-14 · fef9763* · mutant killed · exit 1 · `internal/plan/plan.go` · BodyFile exception dropped so replace/insert body=@ must fail parse again · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · covers:replace body=@ parses
- 2026-09-14 · fef9763* · mutant killed · exit 1 · `internal/plan/plan.go` · insert empty-body refuse ignores BodyFile so insert body=@ must fail parse again · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · covers:insert body=@ parses
- 2026-09-14 · fef9763* · mutant killed · exit 1 · `internal/plan/plan.go` · empty-body refuse never fires so replace with no body and replace body=0 must parse · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · covers:empty replace without @ still refused
- 2026-09-14 · fef9763* · mutant killed · exit 1 · `cmd/mrw/main.go` · CLI skips LoadBodyFiles so replace body=@ applies empty and the CLI test must go red · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · covers:CLI replace writes loaded bytes

## Invariants

- Integer `body=N` unchanged. `replace body=0` still refuses (ADR-006 / ADR-027).
- Empty body file ≡ `body=0`. Apply still refuses empty replace/insert after load.
- Create `body=@` from T4 unchanged.
- `rooted.IsRooted`, never `filepath.IsAbs`.
- Load stays after Parse, before Apply. Parse still takes only `io.Reader`.
- Engine go/no-go: `internal/apply`, `internal/seen`, `internal/check`, `internal/state`, `internal/read` identical against the merge-base. This task owns `internal/plan`.

## Risks

- Skipping the empty-body check with `CountedBody` instead of `BodyFile` would let `replace body=0` parse (ADR-006).

## Stop Condition

Stop if `body=@` has to load inside `Parse`.

## Out of Scope

- Loading `body=@` inside `Parse` (permanent: fact: Parse takes only `io.Reader`; citation: file `internal/plan/plan.go:156`).
- Teaching AGENTS.md (permanent: boundary: T5 already taught `body=@path` with no op restriction).
- `rename` / `unlink` `body=@` (permanent: boundary: rename dest is one inline line; unlink takes no body).

## Verification Log
_(tool-written)_
- 2026-09-14 · fef9763* · exit 1 · `set -o pipefail …` · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · ms:627 · test-lock-sha256:4b79e0f176f4eb984e9ddd6feb75123888bffe3f1e2ee24df99fa1d8d53721bd · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYm9keWZpbGVfcmVwbGFjZV93cml0ZV90ZXN0LmdvCVRlc3RXcml0ZUJvZHlBdFBhdGhFZGl0c0Zyb21GaWxlCWIzNDI5NmQxNjhmMTAwZjJhODBiZWI5ZTY5MThiNTc1OTAxYTBjMGRiYjY5NGVmZWU4NmVkZjJkYjhjOWMyNmYKYm9keQljbWQvbXJ3L2JvZHlmaWxlX3JlcGxhY2Vfd3JpdGVfdGVzdC5nbwlpbnNlcnQtYWZ0ZXIJZTAyNDBiM2Y4ZmIxYjRmZjVhZjgyMTkzNzIwZTUyNWFhNTRhODU5MTQ1YmY5MjI0ZjJhZjU0OWI2NDhjZDM4YQpib2R5CWNtZC9tcncvYm9keWZpbGVfcmVwbGFjZV93cml0ZV90ZXN0LmdvCXJlcGxhY2UJNTZmY2Y5NTAyOGZjN2U0MDVlZjc4NWE2NmE5NGFlMjg5OWVkNjY3MjZiNjRkOTIzZjY4MDYxZjNhMzhiZmI0Zgpib2R5CWludGVybmFsL2N1cnZlL3Njb3JlX2JvZHlmaWxlX3JlcGxhY2VfdGVzdC5nbwlUZXN0U2NvcmVUcmlhbEhpdHNBUmVwbGFjZUZyb21Cb2R5QXRQYXRoCTM4YjJiMzc3MmE0MWRiOGUwZGNhMjA0OWI5NzNiYTZhOGFlMTBkZTkxZWIzNWRjZmUzZGQyODc0YjY5ZmE4MWYKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3JlcGxhY2VfdGVzdC5nbwlUZXN0Qm9keUF0UGF0aFJlcGxhY2VQYXJzZXNBbmRMb2Fkcwk1MzM3MDM1NmFhNjgzNzcxY2FlYWM1NDNmN2JkYWM0NGQ2ZTRkOGQ3MzUyNjcyNDYwMzc3ZjA2OTkwZGYzMTI0CmJvZHkJaW50ZXJuYWwvcGxhbi9ib2R5ZmlsZV9yZXBsYWNlX3Rlc3QuZ28JYm9keV96ZXJvCTFmYmNkMTQyNDlmOTdhNTUyOWIzOTczMTlmMmYyNzM0OWQzY2FkZTg2ODMwMjkyZjU0M2ZkYTk3OWM3YTlkNTEKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3JlcGxhY2VfdGVzdC5nbwllbXB0eV9pbmxpbmUJMGM2NDk1NTA4Zjc4ZTFjNTZjOTU3ODVlNDM2YzgwNDE0NmQyNjgwMmZiMDkyNGQzYWZjMDU5ZmQxY2QzMzhiMgpib2R5CWludGVybmFsL3BsYW4vYm9keWZpbGVfcmVwbGFjZV90ZXN0LmdvCWluc2VydC1hZnRlcglmZWY3MDFlMGE2ZmEzNDYzNjIzNzIwN2Y0YTUyMzYwODNhNzFjNjE2NzczM2VlNTBjNThlYzExMmNjZGQ1Y2M0CmJvZHkJaW50ZXJuYWwvcGxhbi9ib2R5ZmlsZV9yZXBsYWNlX3Rlc3QuZ28JaW5zZXJ0LWJlZm9yZQk2ODAxNTJmZDI2OTdiMWI2MjNmNWY0MTA4MjdmM2YzN2JiYjUwNGE3OWMxNmUyNjg0ZWRmNmRhMTgyOWFmOGRhCmJvZHkJaW50ZXJuYWwvcGxhbi9ib2R5ZmlsZV9yZXBsYWNlX3Rlc3QuZ28JcmVwbGFjZQk2ZjRkZDNlZThjOGNmNTY3ZjNlNDllYjIxOGVmNjcyMjg4NGY2NTQ2OGQwNjVhOWJiMjYyMDg4ZTg2ODYyNzRk
  ```
  --- last 10 line(s) of stdout (of 37 after folding 37 raw)
      --- FAIL: TestWriteBodyAtPathEditsFromFile/insert-after (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.270s
  === RUN   TestScoreTrialHitsAReplaceFromBodyAtPath
      score_bodyfile_replace_test.go:26: replace body=@payload.txt scored refused_parse (plan has 1 error(s):
            line 1: replace with an empty body would delete 45 — say delete if that is what you mean, and check the body did not go missing if it is not), want hit — parse must accept empty Body while BodyFile is set
  --- FAIL: TestScoreTrialHitsAReplaceFromBodyAtPath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/curve	0.402s
  FAIL
  ```
- 2026-09-14 · fef9763* · exit 0 · `set -o pipefail …` · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · ms:1763
- 2026-09-14 · fef9763* · exit 0 · `set -o pipefail …` · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · ms:913
- 2026-09-14 · fef9763* · exit 0 · `set -o pipefail …` · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · ms:773
- 2026-09-14 · fef9763* · exit 0 · `set -o pipefail …` · acceptance-sha256:401e8d9962b073d4fdaa8079b490f8e7eed361e9a21c38dbd1b8050116a1b799 · ms:742
