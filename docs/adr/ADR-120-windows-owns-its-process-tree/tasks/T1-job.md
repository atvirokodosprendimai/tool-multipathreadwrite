# Task ADR-120-T1: the job sequence, kernel32 behind it, and the Windows grandchild tests

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `subproc.runInJob`, `subproc.jobAPI`, `subproc.killOnCloseLimits`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the job sequence, kernel32 behind it, and the Windows grandchild tests`

## Goal

On Windows a child started through `subproc` runs suspended until it is inside a job that kills every member when closed; a cancel terminates the job, and the job is terminated and closed when the child exits.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/subproc/job.go` | add | `jobAPI`, `jobTree`, `runInJob`, `killOnCloseLimits`, the limit structs; built on Windows and under the `mrwjobs` tag, so the sequence is tested anywhere and no unix build carries it as dead code |
| `internal/subproc/subproc_windows.go` | add | `group`, `run` and the kernel32 `jobAPI` |
| `internal/subproc/subproc_other.go` | edit | `!unix && !windows`; `run` |
| `internal/subproc/subproc_unix.go` | edit | `run` |
| `internal/subproc/subproc.go` | edit | `Run` calls `run`; the package comment |
| `internal/subproc/job120_test.go` | add | the fake-driven sequence tests, on every platform |
| `internal/subproc/job120_windows_test.go` | add | the real grandchild tests, on Windows |
| `cmd/mrw/checklog080_test.go`, `internal/check/logs080_test.go` | edit | the Windows skip becomes `needShell` |

## Ordered Steps

1. [S1] Write `TestTheJobSequenceAssignsBeforeResuming`, `TestAFailedAssignKillsTheChildAndClosesTheJob`, `TestACancelTerminatesTheJobOnceAssigned`, `TestTheJobLimitsKillOnCloseAndNothingElse`, `TestTheJobLimitStructHasTheWin32Size` against a fake `jobAPI`, and the Windows tests `TestACancelledCommandStopsItsGrandchildOnWindows` and `TestACleanExitReapsTheGrandchildOnWindows`. Confirm RED. [proof: mutation]
2. [S2] `job.go` and the three platform files. Mutants: resume before assign; no kill when assign fails; a cancel that kills only the child after assignment; `BREAKAWAY_OK` in the limits. [proof: mutation]
3. [S3] Un-skip the two timed-out-check tests on Windows, and record the windows CI shards' run for the head. [proof: human: the windows-shard run URL for the PR head, where the grandchild and timeout tests ran]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test -tags mrwjobs ./internal/subproc/ -count=1 -timeout 300s -run 'TestTheJobSequenceAssignsBeforeResuming|TestAFailedAssignKillsTheChildAndClosesTheJob|TestACancelTerminatesTheJobOnceAssigned|TestTheJobLimitsKillOnCloseAndNothingElse|TestTheJobLimitStructHasTheWin32Size' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheJobSequenceAssignsBeforeResuming \(' "$out" \
  && grep -qE '^--- PASS: TestAFailedAssignKillsTheChildAndClosesTheJob \(' "$out" \
  && grep -qE '^--- PASS: TestACancelTerminatesTheJobOnceAssigned \(' "$out" \
  && grep -qE '^--- PASS: TestTheJobLimitsKillOnCloseAndNothingElse \(' "$out" \
  && go test ./internal/subproc/ ./internal/check/ ./internal/read/ ./cmd/mrw/ -count=1 -timeout 900s \
  && GOOS=windows go vet ./internal/subproc/ ./internal/check/ ./internal/read/ ./cmd/mrw/ \
  && GOOS=windows go test -c -o /dev/null ./internal/subproc/ \
  && grep -q '^func TestACancelledCommandStopsItsGrandchildOnWindows(' internal/subproc/job120_windows_test.go \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/rooted \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/check ':!internal/check/*_test.go' \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheJobSequenceAssignsBeforeResuming` | `internal/subproc/job120_test.go` | create, assign, resume, then terminate and close after the exit, in that order | none | S1, S2 |
| `TestAFailedAssignKillsTheChildAndClosesTheJob` | `internal/subproc/job120_test.go` | a failed assign kills the child, closes the job, returns the error | none | S1, S2 |
| `TestACancelTerminatesTheJobOnceAssigned` | `internal/subproc/job120_test.go` | before assignment a cancel kills the child; after it, terminates the job | none | S1, S2 |
| `TestTheJobLimitsKillOnCloseAndNothingElse` | `internal/subproc/job120_test.go` | the limit flags are KILL_ON_JOB_CLOSE alone, no BREAKAWAY_OK | none | S1, S2 |
| `TestTheJobLimitStructHasTheWin32Size` | `internal/subproc/job120_test.go` | the extended limit struct is 144 bytes on 64-bit | none | S1, S2 |
| `TestACancelledCommandStopsItsGrandchildOnWindows` | `internal/subproc/job120_windows_test.go` | a cancelled child's grandchild is gone | none | S1, S3 |
| `TestACleanExitReapsTheGrandchildOnWindows` | `internal/subproc/job120_windows_test.go` | a child that exits 0 leaves no grandchild | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `runInJob`, the kernel32 `jobAPI` |
| 2 — something selects it | `subproc.Run` calls `run`, which on Windows is `runInJob` |
| 3 — the caller can discover it | the check and ast-grep go through `Run`; the docs in T2 |
| 4 — it is used | the Windows peer's measurement of 2026-10-02 |

## Mutation Log
- 2026-10-02 · fd17179 · mutant killed · exit 1 · `internal/subproc/job.go` · S2: the child is assigned before it is resumed, once · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e
- 2026-10-02 · fd17179* · mutant killed · exit 1 · `internal/subproc/job.go` · S2: a child that cannot be contained is killed, not left running · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e
- 2026-10-02 · fd17179* · mutant killed · exit 1 · `internal/subproc/job.go` · S2: once assigned, a cancel terminates the job, not the child alone · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e
- 2026-10-02 · fd17179* · mutant killed · exit 1 · `internal/subproc/job.go` · S2: no BREAKAWAY_OK, which Git for Windows sh would use to leave the job · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e

## Invariants

- No caller of `subproc` changes; unix keeps its process group.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a Windows CI shard shows a child left suspended, or the grandchild tests cannot be made to pass without `BREAKAWAY_OK`.

## Out of Scope

- A contract row (permanent: boundary: contract.sh drives a POSIX shell on Linux only)

## Verification Log
- 2026-10-02 · e6be523* · exit 1 · `set -o pipefail …` · acceptance-sha256:849df87149f191c3064f97a07570455855df17b5edb8abc9cec1fcc23ac85f8b · ms:210 · test-lock-sha256:5bf2360adad2dd2d0b0087d24e1f948c86b0c4a0a7c28ee8adf21a627bc781c8 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvc3VicHJvYy9qb2IxMjBfdGVzdC5nbwlUZXN0QUNhbmNlbFRlcm1pbmF0ZXNUaGVKb2JPbmNlQXNzaWduZWQJZDJkOTc3ZDk3ZWY0MjI3NWY2OWZmNTg5MGY1Y2JhODFjNDgzOWFhZjc1NTMzNzFmNTA4ZTYwMGIwOTBlMTVkOQpib2R5CWludGVybmFsL3N1YnByb2Mvam9iMTIwX3Rlc3QuZ28JVGVzdEFGYWlsZWRBc3NpZ25LaWxsc1RoZUNoaWxkQW5kQ2xvc2VzVGhlSm9iCWZlZWU1NmZmMTc5NWRhYzAwOTIzYWQ3Zjk1OTE4NzI2NjcxNTY4OTFjM2Y5MDNlNzExMjRmYjZlNmNmYjFjNzIKYm9keQlpbnRlcm5hbC9zdWJwcm9jL2pvYjEyMF90ZXN0LmdvCVRlc3RKb2JIZWxwZXIJODUzZWJjOWFiZGJkYWU1OTRjNmE3MDU0NTI5MzY5MjFhNjMxZmZiYjMxZjNhNTU5MGNiMGNjMzYyZGU3MDNkMQpib2R5CWludGVybmFsL3N1YnByb2Mvam9iMTIwX3Rlc3QuZ28JVGVzdFRoZUpvYkxpbWl0U3RydWN0SGFzVGhlV2luMzJTaXplCTZjY2Y3NmEwMjY3ZjM0MzQxOTE5ZGEwYzVhNWEzNTc2YjVjYzQ4ZGQyMDE4MzllNmZlOWE4OWMyMmFmYjZkNzEKYm9keQlpbnRlcm5hbC9zdWJwcm9jL2pvYjEyMF90ZXN0LmdvCVRlc3RUaGVKb2JMaW1pdHNLaWxsT25DbG9zZUFuZE5vdGhpbmdFbHNlCTZkMjNkYjk5NmFkNDNjN2Y0OTIzMmZhMDBmMjhkZTJlYjNjMWE5ZGJhMzM2ZmFlZDQ4NjI0NzYzMDY2MGNhZmYKYm9keQlpbnRlcm5hbC9zdWJwcm9jL2pvYjEyMF90ZXN0LmdvCVRlc3RUaGVKb2JTZXF1ZW5jZUFzc2lnbnNCZWZvcmVSZXN1bWluZwk1NmNmOWU2NDRiOWNkMDNiY2FkZmEyMDdjMTU5YjVkMjdkYWIyMWZlMDIyM2JmMzk1ZGQ5Nzk3MDJjOTE2MGQ3CmJvZHkJaW50ZXJuYWwvc3VicHJvYy9qb2IxMjBfd2luZG93c190ZXN0LmdvCVRlc3RBQ2FuY2VsbGVkQ29tbWFuZFN0b3BzSXRzR3JhbmRjaGlsZE9uV2luZG93cwkwN2QzYWRjMTFiMmQ5ZGMyNTgyMzYzYmE5YjdlMzIzYmI0MTEyYmVhMjQ4ZjM2YmNhMjJhZGM0MjllMDQxZGMwCmJvZHkJaW50ZXJuYWwvc3VicHJvYy9qb2IxMjBfd2luZG93c190ZXN0LmdvCVRlc3RBQ2xlYW5FeGl0UmVhcHNUaGVHcmFuZGNoaWxkT25XaW5kb3dzCTkzZWYxOTkzN2QzODM1YzI2NTIxN2Q0YTk5YjBhMzMwNjM0Y2Q5N2FlZTFiNDdiMDQwYzNiMTQyYzk0ZGU5MWY
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
  internal/subproc/job120_test.go:93:25: undefined: jobTree
  internal/subproc/job120_test.go:110:9: undefined: runInJob
  internal/subproc/job120_test.go:110:22: undefined: jobTree
  internal/subproc/job120_test.go:129:11: undefined: jobTree
  internal/subproc/job120_test.go:146:12: undefined: killOnCloseLimits
  internal/subproc/job120_test.go:146:73: undefined: jobObjectLimitKillOnJobClose
  internal/subproc/job120_test.go:147:71: undefined: jobObjectLimitKillOnJobClose
  internal/subproc/job120_test.go:157:26: undefined: extendedLimitInformation
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc [build failed]
  FAIL
  ```
- 2026-10-02 · fd17179 · exit 0 · `set -o pipefail …` · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e · ms:38000
- 2026-10-02 · fd17179* · exit 0 · `set -o pipefail …` · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e · ms:37817
- 2026-10-02 · fd17179* · exit 0 · `set -o pipefail …` · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e · ms:38004
- 2026-10-02 · fd17179* · exit 0 · `set -o pipefail …` · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e · ms:37669
- 2026-10-02 · fd17179* · exit 0 · `set -o pipefail …` · acceptance-sha256:5262e1b839f96d9ec5d40e01564d70054138801522978138b885ee4d896d508e · ms:37358
