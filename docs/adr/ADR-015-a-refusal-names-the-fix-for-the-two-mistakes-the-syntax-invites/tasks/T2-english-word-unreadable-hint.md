# Task ADR-015-T2: English-word UNREADABLE paths name quoting

**Depends-on:** none
**Covers:** F-2, F-10, F-14, UC1-S1, UC1-S2, UC1-S3
**Estimated scope:** S
**Owner:** unassigned
**Produces:** English-word UNREADABLE hint; §106
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `two or more English-word UNREADABLE paths print quoting`, `a single nope.go stays silent`, `a *?[ path keeps the glob hint`

## Goal

One `read` that reports two or more UNREADABLE paths matching `/^[A-Za-z][A-Za-z0-9_-]*$/` with no `*?[` prints one hint naming quoting and `--grep`. A single missing `nope.go` stays silent. A glob path keeps Decision 2's wording.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | Count English-word UNREADABLE paths; print one additive hint when ≥2. |
| `internal/read/read_test.go` | edit | Red tests after the `// */` closer. |
| `scripts/contract.sh` | edit | **§106**. |

## Ordered Steps

1. [S1] Confirm the failing tests for `Covers:` IDs exist. [proof: mutation]
2. [S2] Print the hint when F-10 holds. S1 GREEN. [proof: mutation]
3. [S3] §106 RED then GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 106\. ' scripts/contract.sh \
  && go test ./internal/read/ -count=1 -v \
    -run 'TestSeveralEnglishWordUnreadablePathsNameQuoting|TestAnOrdinaryMissingFileGetsNoEnglishWordHint|TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint' 2>&1 | tee /tmp/adr015-t2.out \
  && grep -q '^--- PASS: TestSeveralEnglishWordUnreadablePathsNameQuoting' /tmp/adr015-t2.out \
  && grep -q '^--- PASS: TestAnOrdinaryMissingFileGetsNoEnglishWordHint' /tmp/adr015-t2.out \
  && grep -q '^--- PASS: TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint' /tmp/adr015-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr015-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/read/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestSeveralEnglishWordUnreadablePathsNameQuoting` | `internal/read/read_test.go` | three English-word UNREADABLE paths name quoting and `--grep` | F-2, UC1-S1 | S1, S2 |
| `TestAnOrdinaryMissingFileGetsNoEnglishWordHint` | `internal/read/read_test.go` | one `nope.go` stays silent | F-10, UC1-S2 | S1, S2 |
| `TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint` | `internal/read/read_test.go` | a `*?[` path keeps the glob wording | F-14, UC1-S3 | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests |
| 2 — something selects it | `read.Run` after the spec loop; deleting the ≥2 print leaves S1 red |
| 3 — the caller can discover it | the hint is the discovery surface; §106 |
| 4 — it is used | field: unquoted regex addresses; ADR-009 refuses telemetry |

## Mutation Log
- 2026-09-15 · 9ff8e35* · mutant killed · exit 1 · `internal/read/read.go` · the quoting hint must fire at two English-word UNREADABLE paths; 99 is a count no fixture reaches · acceptance-sha256:f8e2a04df6dba358fad26c69eb508c7e8e60b2105f9b2bd1f7b2d9ac7fd95f75 · covers:two or more English-word UNREADABLE paths print quoting

## Invariants

- Decision 2's glob hint is unchanged.
- A single missing file is still just missing.
- apply/seen/check/state stay byte-identical vs merge-base.

## Risks

- Existing read tests that compare full output. Additive note; update if an exact-equality test fails.

## Stop Condition

Stop if the hint requires detecting the shell rather than argument shape.

## Out of Scope

- Glob expansion inside mrw (T1).
- `,+N` after a regex (ADR-026).

## Verification Log
- 2026-09-15 · 9ff8e35* · exit 0 · `set -o pipefail …` · acceptance-sha256:f8e2a04df6dba358fad26c69eb508c7e8e60b2105f9b2bd1f7b2d9ac7fd95f75 · ms:1072
- 2026-09-15 · 9ff8e35* · exit 0 · `set -o pipefail …` · acceptance-sha256:f8e2a04df6dba358fad26c69eb508c7e8e60b2105f9b2bd1f7b2d9ac7fd95f75 · ms:1008
- 2026-09-15 · 9ff8e35* · exit 1 · `set -o pipefail …` · acceptance-sha256:f8e2a04df6dba358fad26c69eb508c7e8e60b2105f9b2bd1f7b2d9ac7fd95f75 · ms:953 · test-lock-sha256:bf07dd1baf60fbe94ca3b360aec0cfe9c424bc10ef1382c83212d932603ba945 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBR2xvYlBhdGhLZWVwc1RoZUdsb2JIaW50Tm90VGhlRW5nbGlzaFdvcmRIaW50CTZlOWNlNDI4MDM4NmVhY2Q3ZmU1YzFkZjBmN2EzNjQ0YjA3OTc1NTQ5YWE4NjVjNDZhZWUwODRiMGNlZWI3ZjMKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0QUdsb2JVbnJlYWRhYmxlUGF0aE5hbWVzVGhlR2xvYkVzY2FwZQkyY2MzZmQyNmIwNGYzMjdiMjQ1MWYxMzc1OWM0OTI3Zjk5N2RlZGJkYTM0YTVmMDU0NWJkYjU2NzExNzYwOTRjCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdEFSYW5nZUFnYWluc3RBbkVtcHR5RmlsZUlzUmVwb3J0ZWQJZDg5ZGFhNDJmOTk4OWJkNjViZDEzNWRkZDAwZWI3NDE0NzQyMDg1YzU4NTFiYWFmM2ZkNzE2MTBiNWUyNWRmNgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBUmFuZ2VUaGF0TWF0Y2hlc05vdGhpbmdPYnNlcnZlc05vdGhpbmcJZGZlNjJhODNjNTFiOWQ4MTIyYjdmOWYzNzc3YmViYjljMmU4NzY3ZTJiMmQ5YmZhZDM1NGM0MjVhZmM1YTk5Ygpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBUmV2ZXJzZWRSYW5nZVdyaXR0ZW5XaXRoRG9sbGFySXNOb3RTZXJ2ZWQJODNiODA0NTk3MTU2MjM3MDhhZjVlNmVmYTZhYzA2YzllNWY1ZDgxZDgzODg3YWJmZTI3YjkwNmRjZDJhY2MyMwpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RBbk9yZGluYXJ5TWlzc2luZ0ZpbGVHZXRzTm9FbmdsaXNoV29yZEhpbnQJNjUzYmZmM2E3ZDhkZTQ3MDhhMmYwNzhhM2E4YjY0MTNmMTUzNjMwNzQxYjg1ZDU3NGE1OWRmMjg4M2ZhMGQyNgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RDb250ZXh0QXJvdW5kU2luZ2xlUGF0dGVybglhMDZhZDgwN2IxYTU5YThlOGExZjA5ZjYwMzU3ZGJmZTk0YTQyNWZjZWRhNjczMWM4YjhjZTExZDM5MDNjYzJmCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdERlc2NlbmRpbmdSYW5nZXNBcmVTb3J0ZWRBbmRNZXJnZWQJNzA2NGI0MWE3ODkxM2I3ZmQ3MGE3OWI2MGViNTVkMDEyMmI4ZTMxY2RkYTQ2M2EyYTA4MWUzNjBlZmRjZWY1MApib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3REb2xsYXJBc0FuRW5kU3RpbGxSdW5zVG9FT0YJZjk2MDY1ZjRiOTZjNzU3NzkxZWVhNzkxMzBkY2VmYmY4YjZlYWNjZDMzN2NmZjhjM2QyMzk1OTA3MjNkNjY0NQpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3REb2xsYXJJc1RoZUxhc3RMaW5lTm90VGhlV2hvbGVGaWxlCTRiMzRmMzExYWZiNTYwMjlmN2M1MWIwOWUwODRiZjgyM2RhMjQ4MGEwOGUzMmViNmE5YzA3MTA5YTY3ZjJmN2YKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0TWFueUZpbGVzT25lQ2FsbAk2MzRkN2Y3MzJjYWJmMTkzYTU1NmFhZjg4N2M2ZTE0NjFmNzczNmVjZDY1NmYzY2U3YjgzZGI3Y2E1YzA2NjhhCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdE1hbnlVbnNvcnRlZFJhbmdlc01lcmdlQ29ycmVjdGx5CWRkMzJkNWZkYzA0Njg5N2UyYzM1YTdjYzNjOThhOTYzM2M3OGViNWEzNDExNzMzZDVkODRiOTI3MDE3ODZmMGIKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0TWF4TGluZXNSZXBvcnRzV2hhdEl0V2l0aGhlbGQJMDQ3M2IxNWZkYTkxYTQ2NjIzYzAyMzJkOTc1NGU5MzU5OTQ0Y2VhNjcxMjYwMGFhYjMyNjJlZWNkMTIxMzgzYQpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3ROdW1lcmljUmFuZ2VzQW5kSGVhZGVyCTFhODA4MTliNzQzYTkyY2U0N2NkMGE4OTZkMTllOGNmYzc0NmM2MmE2ODMwMTczNDEyNmNmZDg5ZTdkNTZmN2EKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0T3ZlcmxhcHBpbmdSYW5nZXNBcmVNZXJnZWROb3RSZXBlYXRlZAk3ZTkzZjI0ODQzOWNkNTA5OGZlZWNjYjM2YTlmYWJlYmE0ODljYWI3OTM0ODgzNjZiM2IzYTQ0ZDhjMGU2MGUxCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdFBhcnNlU3BlYwllN2NkNWVmYjAyYTFjN2ZhOTEyZGJkNmMzMjFkYzBjZmNlZGQxMWNmNzUzMWQxZWNiM2JjZTgwNzhiNjVkM2NjCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdFJlYWRIaW50c0FNdWx0aUxpbmVSZXBsYWNlTmVlZHNUaGVOZXh0TGluZQk4MTE5Y2FiMmM4OTA0OTkwODRlNGYxYzg4MDAxMmJiMzBiZjM1NjEzYzcxODk3MjkzYjY3NzI5YjVhNzdkOTZmCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdFJlZ2V4cFJhbmdlCTIzMDM0MWIzZDk5Yzc4Nzc3YTM1YjkyOWZkNjYwNTYxMTYxM2I0NGQ5ZmRkNDA0MWMwNzBhMTdhNjRmMjcyZjUKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0U2V2ZXJhbEVuZ2xpc2hXb3JkVW5yZWFkYWJsZVBhdGhzTmFtZVF1b3RpbmcJYzI2YTQwNTIwNmFhYjJlZTE2YmRjN2ViYzRjOTRkYjJiNjU2YjdiNzc0MDBmNjMyMzJjOTA3ZTZhY2QyYjNiMwpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCVRlc3RTdGF0QXNrc0ZvclRoZUZhY3ROb3RUaGVBcnRpZmFjdAk0MGViZjY1YWMwYmFhYjU4ZDUwNTI3ZjEwZGRhMzc1ZmRjMDQ5YWExMmQxZTc1MmQwODc1Mzk4NzdhMmNmZWViCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28JVGVzdFVubWF0Y2hlZFBhdHRlcm5Jc1JlcG9ydGVkCWU4ZWEzNzFjY2ZiOGM3MzgwYjgxOWRmOWU4Yzg1MDljMTI5ZmVlMTQ2ZTQ3OTc3Y2RhMDhhYjBhZDk3OWMyYzEKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlUZXN0V2hvbGVBbmRQYXJ0aWFsT2JzZXJ2YXRpb25zQXJlVW5jaGFuZ2VkCTZiNmUzNGMyMDRjNmY0MWUwMmEwNWIyZmExZWY0MGE5YjUzNmFhMTNkMzIzYThiMDRkNzBiZGI0YzBmZDc4MzYKYm9keQlpbnRlcm5hbC9yZWFkL3JlYWRfdGVzdC5nbwlyYW5nZWQJYzEwMmY4ODRiZGUwYTg4NWZjN2FlMWMyOWM0NzU0ZjEwNzBjZDY5ODM0NGRjOTI1NDU2YzJjNjZkZTJlYWFjMgpib2R5CWludGVybmFsL3JlYWQvcmVhZF90ZXN0LmdvCXNpbmdsZQlmODQyMzFmNzM0YTNmZjI4NWJiOWE3Y2VhYzY4MjYxZjJlZmUyODlmNDBhZWFmOTk0MGNjN2M2NTA2YjJhYmVmCmJvZHkJaW50ZXJuYWwvcmVhZC9yZWFkX3Rlc3QuZ28Jd2hvbGUJM2QyZTU5ZTkzZDA1MzRlMDJiMjViMDg4YjVmZDRjMjBlZDQyYjRjM2NkM2IyZTJhOWNjYTQzNTJiNjllNWI3ZQ
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
          ==> that  UNREADABLE  open /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestSeveralEnglishWordUnreadablePathsNameQuoting1274782384/001/that: no such file or directory
          ==> will  UNREADABLE  open /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestSeveralEnglishWordUnreadablePathsNameQuoting1274782384/001/will: no such file or directory
  --- FAIL: TestSeveralEnglishWordUnreadablePathsNameQuoting (0.00s)
  === RUN   TestAnOrdinaryMissingFileGetsNoEnglishWordHint
  --- PASS: TestAnOrdinaryMissingFileGetsNoEnglishWordHint (0.00s)
  === RUN   TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint
  --- PASS: TestAGlobPathKeepsTheGlobHintNotTheEnglishWordHint (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.418s
  FAIL
  ```
