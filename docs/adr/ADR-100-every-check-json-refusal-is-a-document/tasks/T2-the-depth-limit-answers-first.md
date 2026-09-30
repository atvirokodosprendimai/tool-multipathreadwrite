# Task ADR-100-T2: The depth refusal answers first

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the depth refusal as the first check in `checkCmd`'s Action; contract §194
**Consumes:** `refuse` defined before every refusal (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every form at the limit hears the depth refusal`, `below the limit the usage refusal answers`, `a contract row drives the binary`, `no engine package changes`

## Goal

At `MRW_STEP_DEPTH` 8, `mrw check --full x.go`, `mrw check 'x '` and `mrw check` over an unreadable working set
each answer with ADR-095's depth refusal, exit 2, under `--json` as one document. At 7 each answers with its
own refusal, as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `check.DepthRefusal("a check is")` moves from after `iter.Load` to the first line after `refuse` |
| `cmd/mrw/json100_test.go` | edit | the test below |
| `scripts/contract.sh` | edit | §194 |

**What selects it:** `checkCmd`'s Action, which calls `check.DepthRefusal` once.

## Ordered Steps

1. [S1] Write `TestTheDepthLimitAnswersEveryCheckFormFirst`; confirm RED after T1. [proof: mutation]
2. [S2] Move the depth call first. [proof: mutation] Mutant: the depth call moved back after the
   `--full` refusal.
3. [S3] Contract §194, driving `$MRW` at `MRW_STEP_DEPTH=8`: `check --full x.go` exits 2 naming
   `MRW_STEP_DEPTH`; paired, at 7 it names `--full`.
   [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §194's rows printed — the fence greps the section, since the whole contract takes minutes]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestTheDepthLimitAnswersEveryCheckFormFirst|TestACheckIsRefusedAtTheDepthLimit' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheDepthLimitAnswersEveryCheckFormFirst \(' "$out" \
  && grep -q '^# 194\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal | grep '^??')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheDepthLimitAnswersEveryCheckFormFirst` | `cmd/mrw/json100_test.go` | at depth 8, `check --full x.go`, `check 'x '` and `check` over an unreadable working set exit 2 naming `MRW_STEP_DEPTH`, and `check --json --full x.go` is one document naming it; at 7, `check --full x.go` names `--full` | — | S1, S2 |
| `TestACheckIsRefusedAtTheDepthLimit` | `cmd/mrw/depth095_test.go` | (unchanged, ADR-095 T1) every other form still refused at 8 and run at 7 | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the depth call at the top of the Action |
| 2 — something selects it | `checkCmd`'s Action; the test runs `rootCommand`, §194 the built binary |
| 3 — the caller can discover it | the refusal names `MRW_STEP_DEPTH` |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the advisory kept on ADR-099 |

## Mutation Log
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · the depth call moved back behind the edge-whitespace refusal · acceptance-sha256:3c2d55ae4b8bc686122fec8b161e9b8e91cee7a1924c38ac9a2b8ade9f0575f9 · covers:every form at the limit hears the depth refusal
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package changed against the merge-base · acceptance-sha256:3c2d55ae4b8bc686122fec8b161e9b8e91cee7a1924c38ac9a2b8ade9f0575f9 · covers:no engine package changes

## Invariants

- The depth refusal's words and exit code are ADR-095's.
- Below the limit every refusal answers as T1 left it.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- `write`'s depth refusal (permanent: boundary: a write refuses at the limit only when its check is due, ADR-095, so its usage refusals come first by design)

## Verification Log
- 2026-09-30 · 7537718* · exit 1 · `set -o pipefail …` · acceptance-sha256:f677b1f8462bf62abf2d6467579735cf6bba2bdbf873e6c7c9be267d10a63003 · ms:350 · test-lock-sha256:851b244c64550eafbae2342c114a56c65fbe2662c7de3f5f789a6295aca8a2ff · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwktLWpzb24gYW5kIHRoZSB0YWxseQkzOTFkZTUzOGQ3OWE2YmI5Y2RlYThiM2ViOWUxZWFiOTlkOGZiNjlhMDIwMzljZGFjY2UyOWQ3NDg5NmI3YjczCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCVRlc3RBQ2hlY2tJc1JlZnVzZWRBdFRoZURlcHRoTGltaXQJNGU1YWRjMjdkNmNiNDRlOTM0NjdkZDlhNTM3MDNiZGIzZmEwOTUwMGFjMDFmZGFmNzcxZWIxNDg1M2Y3ZTBjMgpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwlUZXN0QUNoZWNrVGhhdFJ1bnNNcndBZ2FpblN0b3BzQXRUaGVEZXB0aExpbWl0CTIyMGNjYmE4MDRiN2RjZjFiMzllYWY3ZDFkYzIzYjg1OWE0NzFjN2U0YjIyMDI4NzFlZWNlZjhlOTAzMmVhMzgKYm9keQljbWQvbXJ3L2RlcHRoMDk1X3Rlc3QuZ28JVGVzdEFXcml0ZVdob3NlQ2hlY2tJc0R1ZUlzUmVmdXNlZEF0VGhlRGVwdGhMaW1pdAk1MjQwYTllNjZmODc0OTg2ZmE1YjBhNTc0NjJhN2FhNGRhMDRmNTk5NjliNzg5NDA5ZTBjZDUxN2IzYjZhOTMxCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCWEgd3JpdGUgdGhhdCBzdGFydHMgbm8gY2hlY2sgbGFuZHMJNDA4NDFhMDg0MjZiZTUzY2U1MjIwMGU4OWE2ZjFhNGIzYzVmZWYyZTQ3MzUxNTNiMWVjNmJmZmJlYzg2NDg0NQpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwlhdCA3IHRoZSB3cml0ZSBsYW5kcyBhbmQgaXRzIGNoZWNrIHJ1bnMgYXQgOAllMTAyZjE1MDI1YTcwNzNhMTRjNmYwNzgzNzlmOTdlZWVlNjRlOWZiYmQ4YTQ2YjQ0MzRmYzcyMTBiYzFiN2VjCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCXRoZSBpbmZlcnJlZCBjaGVjawkzNWYwMDlmN2NlODZlNGY5MGNhNTkwMzlkOTJhZTg1N2IyNDlmMWNlZmY5MDQ4NDcyYTU3M2JiNWU4YmUxZjk5CmJvZHkJY21kL21ydy9qc29uMTAwX3Rlc3QuZ28JVGVzdEV2ZXJ5Q2hlY2tSZWZ1c2FsSXNBSlNPTkRvY3VtZW50VW5kZXJKU09OCTY4YWM1MDI0NWU4ZWMxZTZiYjM4YjdjYTYwZjAyNjYwMWMwOWY2NDVkZmZjODQwMGU4MWZiYTQ0NGI0YjJjM2IKYm9keQljbWQvbXJ3L2pzb24xMDBfdGVzdC5nbwlUZXN0VGhlRGVwdGhMaW1pdEFuc3dlcnNFdmVyeUNoZWNrRm9ybUZpcnN0CTZjMDY4YThjMzIyOWNhMTBlNTFmMDhhMmRiNGY2ZjRkMmJlYWE2NmQ2Y2Y5MzcwZTZkNmM3MTc1NjE3MzA5MDk
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
  --- PASS: TestACheckIsRefusedAtTheDepthLimit (0.02s)
  === RUN   TestTheDepthLimitAnswersEveryCheckFormFirst
      json100_test.go:100: ["-C" "/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestTheDepthLimitAnswersEveryCheckFormFirst2036947854/002" "check" "--full" "x.go"] at 8: err --full runs the whole project; it takes no PATH, want exit 2 naming MRW_STEP_DEPTH
      json100_test.go:100: ["-C" "/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestTheDepthLimitAnswersEveryCheckFormFirst2036947854/002" "check" "x.go "] at 8: err 'x.go ' has edge whitespace the argument parser strips; put -- before the path: mrw check -- 'x.go ', want exit 2 naming MRW_STEP_DEPTH
      json100_test.go:100: ["-C" "/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestTheDepthLimitAnswersEveryCheckFormFirst2036947854/003" "check"] at 8: err read /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestTheDepthLimitAnswersEveryCheckFormFirst2036947854/001/mrw/cf2d1dce46cb29a6/iteration: is a directory, want exit 2 naming MRW_STEP_DEPTH
      json100_test.go:106: check --json --full x.go at 8: err --full runs the whole project; it takes no PATH, stdout not one document naming MRW_STEP_DEPTH:
  --- FAIL: TestTheDepthLimitAnswersEveryCheckFormFirst (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.128s
  FAIL
  ```
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:3c2d55ae4b8bc686122fec8b161e9b8e91cee7a1924c38ac9a2b8ade9f0575f9 · ms:427
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:3c2d55ae4b8bc686122fec8b161e9b8e91cee7a1924c38ac9a2b8ade9f0575f9 · ms:364
- 2026-09-30 · human-observed · S3 observed 2026-09-30: ./scripts/contract.sh run unpiped on this branch (base 7537718), exit 0 'contract holds', with §194 printed: check --full a.go at MRW_STEP_DEPTH=8 names the depth limit, and at 7 names --full
- 2026-09-30 · human-observed · relock 2026-09-30: T2's lock covers cmd/mrw/json100_test.go, where T1's test gained the ran-false receipt assertion after T2's red run (see T1's relock row); TestTheDepthLimitAnswersEveryCheckFormFirst is unchanged since its nil-safe fix before the red run was recorded, and every assertion stands
- 2026-09-30 · 7537718* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3c2d55ae4b8bc686122fec8b161e9b8e91cee7a1924c38ac9a2b8ade9f0575f9 · ms:0 · test-lock-sha256:12ea8de3309981c5c969093ce525fa0b17e68f40faf12f13630f30ece574a4c0 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwktLWpzb24gYW5kIHRoZSB0YWxseQkzOTFkZTUzOGQ3OWE2YmI5Y2RlYThiM2ViOWUxZWFiOTlkOGZiNjlhMDIwMzljZGFjY2UyOWQ3NDg5NmI3YjczCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCVRlc3RBQ2hlY2tJc1JlZnVzZWRBdFRoZURlcHRoTGltaXQJNGU1YWRjMjdkNmNiNDRlOTM0NjdkZDlhNTM3MDNiZGIzZmEwOTUwMGFjMDFmZGFmNzcxZWIxNDg1M2Y3ZTBjMgpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwlUZXN0QUNoZWNrVGhhdFJ1bnNNcndBZ2FpblN0b3BzQXRUaGVEZXB0aExpbWl0CTIyMGNjYmE4MDRiN2RjZjFiMzllYWY3ZDFkYzIzYjg1OWE0NzFjN2U0YjIyMDI4NzFlZWNlZjhlOTAzMmVhMzgKYm9keQljbWQvbXJ3L2RlcHRoMDk1X3Rlc3QuZ28JVGVzdEFXcml0ZVdob3NlQ2hlY2tJc0R1ZUlzUmVmdXNlZEF0VGhlRGVwdGhMaW1pdAk1MjQwYTllNjZmODc0OTg2ZmE1YjBhNTc0NjJhN2FhNGRhMDRmNTk5NjliNzg5NDA5ZTBjZDUxN2IzYjZhOTMxCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCWEgd3JpdGUgdGhhdCBzdGFydHMgbm8gY2hlY2sgbGFuZHMJNDA4NDFhMDg0MjZiZTUzY2U1MjIwMGU4OWE2ZjFhNGIzYzVmZWYyZTQ3MzUxNTNiMWVjNmJmZmJlYzg2NDg0NQpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwlhdCA3IHRoZSB3cml0ZSBsYW5kcyBhbmQgaXRzIGNoZWNrIHJ1bnMgYXQgOAllMTAyZjE1MDI1YTcwNzNhMTRjNmYwNzgzNzlmOTdlZWVlNjRlOWZiYmQ4YTQ2YjQ0MzRmYzcyMTBiYzFiN2VjCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCXRoZSBpbmZlcnJlZCBjaGVjawkzNWYwMDlmN2NlODZlNGY5MGNhNTkwMzlkOTJhZTg1N2IyNDlmMWNlZmY5MDQ4NDcyYTU3M2JiNWU4YmUxZjk5CmJvZHkJY21kL21ydy9qc29uMTAwX3Rlc3QuZ28JVGVzdEV2ZXJ5Q2hlY2tSZWZ1c2FsSXNBSlNPTkRvY3VtZW50VW5kZXJKU09OCTZiMmIxMTIyMmUxNjc4YzY1YzI1YjMyMzU0ODBjMTQ2YzE4YWM4NDkxNWQzNDc2OWEyMDI5YTJkOGZjMjdmMzEKYm9keQljbWQvbXJ3L2pzb24xMDBfdGVzdC5nbwlUZXN0VGhlRGVwdGhMaW1pdEFuc3dlcnNFdmVyeUNoZWNrRm9ybUZpcnN0CTZjMDY4YThjMzIyOWNhMTBlNTFmMDhhMmRiNGY2ZjRkMmJlYWE2NmQ2Y2Y5MzcwZTZkNmM3MTc1NjE3MzA5MDk · test-lock-kind:replace
