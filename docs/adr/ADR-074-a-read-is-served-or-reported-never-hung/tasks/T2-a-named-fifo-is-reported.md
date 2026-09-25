# Task ADR-074-T2: a named FIFO, socket or device is reported, not opened

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the regular-file check in `read.Run`, on `lines.NotRegular` (ADR-073) as the walk now is
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a FIFO is reported without being opened`, `a directory keeps its own message`, `the file beside it is served`, `a contract row drives the binary`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

`mrw read p` on a FIFO blocked until something wrote to the pipe, and so did `--stat` and a symlink
to one. Report it by name, as the walk has since ADR-007, and serve the rest of the call.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | stat before `os.ReadFile` |
| `internal/read/walk.go` | edit | its reason becomes `lines.NotRegular`, the one sentence |
| `internal/read/fifo_unix_test.go` | new | FIFO, symlink to it, directory, regular file; with and without `--stat` |
| `scripts/contract.sh` | edit | §148 |

## Ordered Steps

1. [S1] Write the test; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: the check dropped; the directory exclusion dropped.
3. [S3] Contract §148: `mrw read p a.go` on a FIFO exits 1 within the alarm, names `not a regular file`, and serves `a.go`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ -count=1 -timeout 180s -run 'TestAFIFOIsReportedByNameNotWaitedOn' -v 2>&1 | tee /tmp/adr074-T2.out \
  && missing=$(for t in TestAFIFOIsReportedByNameNotWaitedOn; do grep -qE "^--- PASS: $t \(" /tmp/adr074-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 148\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/read internal/check internal/subproc ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' ':(exclude)internal/read/astgrep.go' ':(exclude)internal/read/fifo_unix_test.go' ':(exclude)internal/read/msys_hint_test.go' ':(exclude)internal/check/check.go' ':(exclude)internal/subproc/subproc.go' ':(exclude)internal/subproc/interrupt_unix_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/read internal/check internal/subproc ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' ':(exclude)internal/read/astgrep.go' ':(exclude)internal/read/fifo_unix_test.go' ':(exclude)internal/read/msys_hint_test.go' ':(exclude)internal/check/check.go' ':(exclude)internal/subproc/subproc.go' ':(exclude)internal/subproc/interrupt_unix_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFIFOIsReportedByNameNotWaitedOn` | `internal/read/fifo_unix_test.go` | returns within 3 s; FIFO and symlink reported, directory says "is a directory", `a.go` served; 3 problems; with and without `--stat` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the stat before the read |
| 2 — something selects it | every named spec, CLI and MCP |
| 3 — the caller can discover it | the `UNREADABLE` line names the reason |
| 4 — it is used | the round hung `read` on a FIFO |

## Verification Log
(empty until execute)
- 2026-09-26 · da2fd0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:f5a9e5bb8d5e7ea3eff824b9b48c47217ce47c90fb5e1da023f7e7959c879ff4 · ms:3433 · test-lock-sha256:6568ec7c7d561ab67c4352170b6ff10ed8f3cd0231fa3925286f39cdab3e6aaf · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3JlYWQvZmlmb191bml4X3Rlc3QuZ28JVGVzdEFGSUZPSXNSZXBvcnRlZEJ5TmFtZU5vdFdhaXRlZE9uCTJjNjk2YjdiZjNiN2I1MmYwZGFlZDRkNDY3YTgxNWIwNWE1MWZlZDdhYmViZWVlODg2MDY0YTFkZTZlYjBlMTI
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAFIFOIsReportedByNameNotWaitedOn
      fifo_unix_test.go:60: read (stat=false) is still waiting on the FIFO at 3 s
  --- FAIL: TestAFIFOIsReportedByNameNotWaitedOn (3.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	3.166s
  FAIL
  ```
- 2026-09-26 · da2fd0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:f5a9e5bb8d5e7ea3eff824b9b48c47217ce47c90fb5e1da023f7e7959c879ff4 · ms:466
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAFIFOIsReportedByNameNotWaitedOn
  --- PASS: TestAFIFOIsReportedByNameNotWaitedOn (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.197s
  ```
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:f5a9e5bb8d5e7ea3eff824b9b48c47217ce47c90fb5e1da023f7e7959c879ff4 · ms:275
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:f5a9e5bb8d5e7ea3eff824b9b48c47217ce47c90fb5e1da023f7e7959c879ff4 · ms:321
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:f5a9e5bb8d5e7ea3eff824b9b48c47217ce47c90fb5e1da023f7e7959c879ff4 · ms:239

## Mutation Log
(empty until execute)
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/read/read.go` · a FIFO spec is opened and the read blocks · acceptance-sha256:f5a9e5bb8d5e7ea3eff824b9b48c47217ce47c90fb5e1da023f7e7959c879ff4 · covers:a FIFO is reported without being opened
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/read/read.go` · a directory spec is reported as a pipe or device · acceptance-sha256:f5a9e5bb8d5e7ea3eff824b9b48c47217ce47c90fb5e1da023f7e7959c879ff4 · covers:a directory keeps its own message

## Invariants

- Every read that served before this record serves the same bytes.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-074 Out of Scope)

## Stop Condition

Stop if the change needs a package the record does not govern.
