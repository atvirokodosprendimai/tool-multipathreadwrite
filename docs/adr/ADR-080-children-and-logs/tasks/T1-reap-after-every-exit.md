# Task ADR-080-T1: nothing mrw starts outlives the call

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `subproc.Run`, `subproc.Output`, `reap`
**Consumes:** `subproc.Command`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a clean exit's grandchild is reaped`, `the check reaps`, `ast-grep reaps`, `a contract row drives the binary`, `the packages vet for Windows`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

A check that passed and an ast-grep that exited 0 left a background grandchild running after mrw returned.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/subproc/subproc.go`, `subproc_unix.go`, `subproc_other.go` | edit | `Run`, `Output`, `reap` |
| `internal/check/check.go`, `internal/read/astgrep.go` | edit | both run through them |
| `internal/subproc/reap080_unix_test.go`, `internal/check/reap080_unix_test.go`, `cmd/mrw/astgrep_reap080_unix_test.go` | new | the tests below |
| `scripts/contract.sh` | edit | §162 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: `reap` kills nothing; the check back on `c.Run`; ast-grep back on `cmd.Output`.
3. [S3] Contract §162. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/subproc/ ./internal/check/ ./cmd/mrw/ -count=1 -timeout 240s -run 'TestAGrandchildOfACleanExitIsReaped|TestACheckThatPassesLeavesNoProcessBehind|TestAnAstGrepThatExitsCleanlyLeavesNoGrandchild|TestATimedOutCheckLeavesNoGrandchild' -v 2>&1 | tee /tmp/adr080-T1.out \
  && missing=$(for t in TestAGrandchildOfACleanExitIsReaped TestACheckThatPassesLeavesNoProcessBehind TestAnAstGrepThatExitsCleanlyLeavesNoGrandchild TestATimedOutCheckLeavesNoGrandchild; do grep -qE "^--- PASS: $t \(" /tmp/adr080-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 162\. ' scripts/contract.sh \
  && GOOS=windows go vet ./internal/subproc/ ./internal/check/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/lines internal/iter internal/seen internal/state internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/lines internal/iter internal/seen internal/state internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAGrandchildOfACleanExitIsReaped` | `internal/subproc/reap080_unix_test.go` | a redirected background grandchild of a clean exit is gone, through Run and Output | — | S1, S2 |
| `TestACheckThatPassesLeavesNoProcessBehind` | `internal/check/reap080_unix_test.go` | a passing check's background process is gone | — | S1, S2 |
| `TestAnAstGrepThatExitsCleanlyLeavesNoGrandchild` | `cmd/mrw/astgrep_reap080_unix_test.go` | the hit is served and the grandchild is gone | — | S1, S2 |
| `TestATimedOutCheckLeavesNoGrandchild` | `internal/check/group_unix_test.go` | ADR-072's pair: a timed-out check still reaps | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the functions under Produces |
| 2 — something selects it | every check run and every `--ast-grep` read |
| 3 — the caller can discover it | the receipt says what it removed and names what it kept |
| 4 — it is used | the review of #232 and the 3,103 logs on one machine |

## Verification Log
(empty until execute)
- 2026-09-26 · 6835512* · exit 1 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:4861 · test-lock-sha256:3862d83c9f6068b8f3a4dd9f2898bf9dc3a7112a88a810f810364e75e1cc9700 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcF9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwVGhhdEV4aXRzQ2xlYW5seUxlYXZlc05vR3JhbmRjaGlsZAk0MzU5YjRhODE1NDQwYTM0MzViZmU5YTA4YjFkNWRhMDExYmI0YTk2YzU0ZjI1NjE4YjE0ZDk3OTQ4Y2E1NGVjCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBVGltZWRPdXRDaGVja0xlYXZlc05vR3JhbmRjaGlsZAkzMzRiYjUwMzlhNjYxZTE0YzdiZDE0Njk5OWM2ZmIwMzFkNWU3M2M3NjYwZDhmMzIyZjM4ZWM2NWRmMjBlZmMwCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBbkludGVycnVwdGVkQ2hlY2tTYXlzU28JNGYwOWU2YjY5ZGI2YTY3NWM2YWFmNjVlZGRiYmMzYjNiYTI3ZTk2M2M5MzA2NWE5NDcxNjQxZjQ1YTdlZDQwYQpib2R5CWludGVybmFsL2NoZWNrL2dyb3VwX3VuaXhfdGVzdC5nbwlUZXN0VGhlQ2hlY2tTdG9wc09uSGFuZ3VwVW5sZXNzSGFuZ3VwSXNJZ25vcmVkCWE0NjkyYTI0MWZmZGEyZmNmOTNkNmMwNGVjMDFjYjAwZWQ3MDlhYWEwZWVkZTVjZTMxNWQzOTBkZWZjNTA0MTUKYm9keQlpbnRlcm5hbC9jaGVjay9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QUNoZWNrVGhhdFBhc3Nlc0xlYXZlc05vUHJvY2Vzc0JlaGluZAk5NjRiOGNhZTY2ZTk3YjQ0OGViODIzNTIyYzE2OGVhNjA0M2VhZTgxMzYzZDUxODIwZTUzNDZkMjAwOWJlZTZiCmJvZHkJaW50ZXJuYWwvc3VicHJvYy9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QUdyYW5kY2hpbGRPZkFDbGVhbkV4aXRJc1JlYXBlZAkwOTZiZTRlZTI3NTUxNzg0MTgxODg1NTdiMTMzMmY2ODIyZTRlZTlkZTFjZTNkMjNiM2ZiODU5OGUwMjMwNmJi
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  internal/check/logs080_test.go:51:9: res.Pruned undefined (type Result has no field or method Pruned)
  internal/check/logs080_test.go:52:37: res.Pruned undefined (type Result has no field or method Pruned)
  internal/check/logs080_test.go:72:31: undefined: Interrupted
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check [build failed]
  === RUN   TestAnAstGrepThatExitsCleanlyLeavesNoGrandchild
      astgrep_reap080_unix_test.go:42: ast-grep's grandchild outlived a clean exit
  --- FAIL: TestAnAstGrepThatExitsCleanlyLeavesNoGrandchild (3.22s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	3.418s
  FAIL
  ```
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:1780
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:1420
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:1549
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:1634
- 2026-09-26 · 6835512* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:0 · test-lock-sha256:3862d83c9f6068b8f3a4dd9f2898bf9dc3a7112a88a810f810364e75e1cc9700 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcF9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwVGhhdEV4aXRzQ2xlYW5seUxlYXZlc05vR3JhbmRjaGlsZAk0MzU5YjRhODE1NDQwYTM0MzViZmU5YTA4YjFkNWRhMDExYmI0YTk2YzU0ZjI1NjE4YjE0ZDk3OTQ4Y2E1NGVjCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBVGltZWRPdXRDaGVja0xlYXZlc05vR3JhbmRjaGlsZAkzMzRiYjUwMzlhNjYxZTE0YzdiZDE0Njk5OWM2ZmIwMzFkNWU3M2M3NjYwZDhmMzIyZjM4ZWM2NWRmMjBlZmMwCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBbkludGVycnVwdGVkQ2hlY2tTYXlzU28JNGYwOWU2YjY5ZGI2YTY3NWM2YWFmNjVlZGRiYmMzYjNiYTI3ZTk2M2M5MzA2NWE5NDcxNjQxZjQ1YTdlZDQwYQpib2R5CWludGVybmFsL2NoZWNrL2dyb3VwX3VuaXhfdGVzdC5nbwlUZXN0VGhlQ2hlY2tTdG9wc09uSGFuZ3VwVW5sZXNzSGFuZ3VwSXNJZ25vcmVkCWE0NjkyYTI0MWZmZGEyZmNmOTNkNmMwNGVjMDFjYjAwZWQ3MDlhYWEwZWVkZTVjZTMxNWQzOTBkZWZjNTA0MTUKYm9keQlpbnRlcm5hbC9jaGVjay9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QUNoZWNrVGhhdFBhc3Nlc0xlYXZlc05vUHJvY2Vzc0JlaGluZAk5NjRiOGNhZTY2ZTk3YjQ0OGViODIzNTIyYzE2OGVhNjA0M2VhZTgxMzYzZDUxODIwZTUzNDZkMjAwOWJlZTZiCmJvZHkJaW50ZXJuYWwvc3VicHJvYy9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QUdyYW5kY2hpbGRPZkFDbGVhbkV4aXRJc1JlYXBlZAkwOTZiZTRlZTI3NTUxNzg0MTgxODg1NTdiMTMzMmY2ODIyZTRlZTlkZTFjZTNkMjNiM2ZiODU5OGUwMjMwNmJi · test-lock-kind:replace
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:2212
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:1745
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:1611
- 2026-09-26 · 6835512* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:1573
- 2026-09-26 · 52debba* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:9267
- 2026-09-26 · 31fd88c* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:0 · test-lock-sha256:3862d83c9f6068b8f3a4dd9f2898bf9dc3a7112a88a810f810364e75e1cc9700 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcF9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwVGhhdEV4aXRzQ2xlYW5seUxlYXZlc05vR3JhbmRjaGlsZAk0MzU5YjRhODE1NDQwYTM0MzViZmU5YTA4YjFkNWRhMDExYmI0YTk2YzU0ZjI1NjE4YjE0ZDk3OTQ4Y2E1NGVjCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBVGltZWRPdXRDaGVja0xlYXZlc05vR3JhbmRjaGlsZAkzMzRiYjUwMzlhNjYxZTE0YzdiZDE0Njk5OWM2ZmIwMzFkNWU3M2M3NjYwZDhmMzIyZjM4ZWM2NWRmMjBlZmMwCmJvZHkJaW50ZXJuYWwvY2hlY2svZ3JvdXBfdW5peF90ZXN0LmdvCVRlc3RBbkludGVycnVwdGVkQ2hlY2tTYXlzU28JNGYwOWU2YjY5ZGI2YTY3NWM2YWFmNjVlZGRiYmMzYjNiYTI3ZTk2M2M5MzA2NWE5NDcxNjQxZjQ1YTdlZDQwYQpib2R5CWludGVybmFsL2NoZWNrL2dyb3VwX3VuaXhfdGVzdC5nbwlUZXN0VGhlQ2hlY2tTdG9wc09uSGFuZ3VwVW5sZXNzSGFuZ3VwSXNJZ25vcmVkCWE0NjkyYTI0MWZmZGEyZmNmOTNkNmMwNGVjMDFjYjAwZWQ3MDlhYWEwZWVkZTVjZTMxNWQzOTBkZWZjNTA0MTUKYm9keQlpbnRlcm5hbC9jaGVjay9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QUNoZWNrVGhhdFBhc3Nlc0xlYXZlc05vUHJvY2Vzc0JlaGluZAk5NjRiOGNhZTY2ZTk3YjQ0OGViODIzNTIyYzE2OGVhNjA0M2VhZTgxMzYzZDUxODIwZTUzNDZkMjAwOWJlZTZiCmJvZHkJaW50ZXJuYWwvc3VicHJvYy9yZWFwMDgwX3VuaXhfdGVzdC5nbwlUZXN0QUdyYW5kY2hpbGRPZkFDbGVhbkV4aXRJc1JlYXBlZAkwOTZiZTRlZTI3NTUxNzg0MTgxODg1NTdiMTMzMmY2ODIyZTRlZTlkZTFjZTNkMjNiM2ZiODU5OGUwMjMwNmJi · test-lock-kind:replace
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:1962
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:2239
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:2037
- 2026-09-26 · 31fd88c* · exit 0 · `set -o pipefail …` · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · ms:2482

## Mutation Log
(empty until execute)
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · reap kills nothing · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:a clean exit's grandchild is reaped
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `internal/check/check.go` · the check back on c.Run · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:the check reaps
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `internal/read/astgrep.go` · ast-grep back on cmd.Output · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:ast-grep reaps
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · reap kills nothing · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:a clean exit's grandchild is reaped
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `internal/check/check.go` · the check back on c.Run · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:the check reaps
- 2026-09-26 · 6835512* · mutant killed · exit 1 · `internal/read/astgrep.go` · ast-grep back on cmd.Output · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:ast-grep reaps
- 2026-09-26 · 31fd88c* · mutant killed · exit 1 · `internal/subproc/subproc_unix.go` · reap kills nothing · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:a clean exit's grandchild is reaped
- 2026-09-26 · 31fd88c* · mutant killed · exit 1 · `internal/check/check.go` · the check back on c.Run · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:the check reaps
- 2026-09-26 · 31fd88c* · mutant killed · exit 1 · `internal/read/astgrep.go` · ast-grep back on cmd.Output · acceptance-sha256:84b8d7ce6a8fec49815103fd5d800587e5d7342cdfe634e41bd0de3057dc7962 · covers:ast-grep reaps

## Invariants

- No process in a group mrw started outlives the call that started it.

## Risks

- A check that wants a process to outlive it must detach it with setsid.

## Out of Scope

- A setsid grandchild (permanent: fact: its group is not mrw's)

## Stop Condition

The fence exits 0 and the latest Mutation Log row for each mutant reads `killed`.
