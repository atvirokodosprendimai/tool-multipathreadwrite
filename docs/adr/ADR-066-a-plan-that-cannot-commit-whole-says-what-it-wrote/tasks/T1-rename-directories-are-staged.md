# Task ADR-066-T1: rename destinations are checked at validation and their directories made while staging; contract §118

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `abortStage` shared by the staging loops; destination `Lstat` errors refused; rename destination directories staged; §118; campaign probe
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a destination the filesystem rejects is refused before any write`, `a rename directory that cannot be made writes nothing`, `staged rename directories are taken back on abort`, `the binary writes nothing`

## Goal

Two checks move ahead of the first write.

- **Validation.** `planPathOp` (`internal/apply/pathop.go:112`) fails a rename hunk whose destination `Lstat` returns an error other than "does not exist", with the error as the reason.
- **Staging.** After content staging, `apply()` makes each pending rename's destination directories (`missingDirs` then `MkdirAll`) and records them in a dirs-only `staged` entry appended after the content entries. A failure goes through the staging abort: the rename hunk is `failed`, the rest `skipped`, every addressed file is listed unwritten, the directories are removed, and the exit is 2.

The verdict block at `internal/apply/apply.go:575-591` becomes `abortStage(path, err)`, used by both loops.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/pathop.go` | edit | `:112` refuses a non-"does not exist" `Lstat` error |
| `internal/apply/apply.go` | edit | `abortStage`; rename-directory staging loop after `:595` |
| `internal/apply/pathop_commit_test.go` | new | the three tests below |
| `scripts/contract.sh` | edit | §118 |
| `scripts/break-campaign.sh` | edit | probe `[rename-dest-toolong]` |
| `docs/adr/BACKLOG.md` | edit | ADR-066 row; the deferrals in its Out of Scope |

## Ordered Steps

1. [S1] Write the three tests and confirm each RED on `main` on an assertion. [proof: mutation]
   - `TestARenameWhoseDestinationDirectoryCannotBeCreatedWritesNothing`: an edit to `a.txt` and a rename of `b.txt` to `n/<300 bytes>/b.txt`, where `n` does not exist. `Lstat` then answers "does not exist", so validation passes and `MkdirAll` fails at staging. Afterwards `a.txt` is byte-identical, `b.txt` is still there, no `n` exists, the rename is `failed` with ENAMETOOLONG, the edit is `skipped`, and the error is non-nil.
   - `TestARenameWhoseDestinationNameIsRejectedFailsValidation`: a rename of `b.txt` to `ok/<300 bytes>` with `ok/` existing. The hunk is `failed` with the `Lstat` error, the error is nil (a validation failure), and nothing is written.
   - `TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage`: a control plan with only the rename `c.txt → new/deep/c.txt` creates `new/deep`, which proves staging makes it. The test plan renames `c.txt → new/deep/c.txt` and `d.txt → m/<300 bytes>/d.txt`, and afterwards no `new/` exists.
2. [S2] Implement both checks and `abortStage`; confirm GREEN. `TestAFailedStageLeavesTheTreeUntouched`, `TestAnAbortedStageTakesBackOnlyTheDirectoriesItMade` and `TestRenameMovesThePath` stay green. [proof: mutation]
   Mutants:
   - drop the staging `MkdirAll`: kills the first test;
   - don't append the dirs-only entry: kills the third;
   - restore `err == nil` at `:112`: kills the second;
   - mark the staging-failed rename `skipped`: kills the first.
3. [S3] §118 (the next free number after §117). [proof: mutation]
   - Good: a sibling edit and a rename to `d/short/f.txt` exit 0 with `— applied`, and `d/short/f.txt` exists.
   - Must-fail, a 300-byte directory under an absent short parent (`n/<300 bytes>/f.txt`): exit 2; the sibling is unchanged; no `n`; `FAIL` and `skip` rows; no `— applied`; `--json | jq -e '.failed==1 and .applied==false and ([.files[]|select(.written)]|length==0)'`.
   - Must-fail, a 300-byte leaf under an existing directory: exit 1; the sibling is unchanged.
   - Confirm RED against v1.22.3 in a mini-harness (`MRW=$(command -v mrw)`), then GREEN in the full `./scripts/contract.sh`.
4. [S4] Campaign probe; BACKLOG rows; `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 118\. ' scripts/contract.sh \
  && go test ./internal/apply/ -count=1 -v \
    -run 'TestARenameWhoseDestinationDirectoryCannotBeCreatedWritesNothing|TestARenameWhoseDestinationNameIsRejectedFailsValidation|TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage|TestAStagingAbortKeepsAPreExistingDanglingSymlink|TestARenameWhoseLeafIsRejectedUnderANewParentWritesNothing|TestAStagingFailureLeavesNoWriteDetailOnTheFailedHunk|TestAFailedStageLeavesTheTreeUntouched|TestAnAbortedStageTakesBackOnlyTheDirectoriesItMade|TestRenameMovesThePath' 2>&1 | tee /tmp/adr066-t1.out \
  && grep -q '^--- PASS: TestARenameWhoseDestinationDirectoryCannotBeCreatedWritesNothing ' /tmp/adr066-t1.out \
  && grep -q '^--- PASS: TestARenameWhoseDestinationNameIsRejectedFailsValidation ' /tmp/adr066-t1.out \
  && grep -q '^--- PASS: TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage ' /tmp/adr066-t1.out \
  && grep -q '^--- PASS: TestAStagingAbortKeepsAPreExistingDanglingSymlink ' /tmp/adr066-t1.out \
  && grep -q '^--- PASS: TestARenameWhoseLeafIsRejectedUnderANewParentWritesNothing ' /tmp/adr066-t1.out \
  && grep -q '^--- PASS: TestAStagingFailureLeavesNoWriteDetailOnTheFailedHunk ' /tmp/adr066-t1.out \
  && grep -q '^--- PASS: TestAFailedStageLeavesTheTreeUntouched ' /tmp/adr066-t1.out \
  && grep -q '^--- PASS: TestRenameMovesThePath ' /tmp/adr066-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr066-t1.out \
  && ./scripts/contract.sh > /tmp/adr066-t1-contract.out 2>&1 \
  && grep -q '^  PASS  a rename into a new directory applies' /tmp/adr066-t1-contract.out \
  && grep -q '^  PASS  a rename whose directory cannot be made writes nothing' /tmp/adr066-t1-contract.out \
  && grep -q '^  PASS  a rename whose name the filesystem rejects is refused' /tmp/adr066-t1-contract.out \
  && grep -q '^  PASS  a rename whose leaf is rejected under a new parent writes nothing' /tmp/adr066-t1-contract.out \
  && go build -o /tmp/adr066-mrw ./cmd/mrw \
  && MRW=/tmp/adr066-mrw bash scripts/break-campaign.sh > /tmp/adr066-campaign.out 2>/dev/null \
  && grep -q '^\[rename-dest-toolong\] exit=2' /tmp/adr066-campaign.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state \
  && [ -z "$(gofmt -l internal/apply)" ] \
  && go vet ./internal/apply/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARenameWhoseDestinationDirectoryCannotBeCreatedWritesNothing` | `internal/apply/pathop_commit_test.go` | a 300-byte destination directory aborts at staging with nothing written | — | S1, S2 |
| `TestARenameWhoseDestinationNameIsRejectedFailsValidation` | `internal/apply/pathop_commit_test.go` | a 300-byte leaf fails validation instead of commit | — | S1, S2 |
| `TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage` | `internal/apply/pathop_commit_test.go` | an earlier rename's staged directories are removed on abort | — | S1, S2 |
| `TestAStagingAbortKeepsAPreExistingDanglingSymlink` | `internal/apply/pathop_commit_test.go` | an abort does not remove a dangling symlink the run did not make (`missingDirs` uses Lstat) | — | S1, S2 |
| `TestARenameWhoseLeafIsRejectedUnderANewParentWritesNothing` | `internal/apply/pathop_commit_test.go` | a leaf rejected under a new parent aborts at staging, and the parent is taken back | — | S1, S2 |
| `TestAStagingFailureLeavesNoWriteDetailOnTheFailedHunk` | `internal/apply/pathop_commit_test.go` | the staging-failed hunk carries no Echo or Balance | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §118 |
| 2 — something selects it | `apply.Apply` runs both checks on every plan with a rename; `mrw write` and MCP `mrw_write` call it; each mutant fails a test or §118 |
| 3 — the caller can discover it | the receipt names the failed hunk and its error |
| 4 — it is used | found by the 2026-09-24 chaos pass; ADR-009 refuses telemetry |

## Mutation Log
(empty until execute)
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/apply.go` · staging no longer makes the rename directory: TestARenameWhoseDestinationDirectoryCannotBeCreatedWritesNothing and §118 must go red · acceptance-sha256:56aef2057b4fff04354a6ce8416665de5f5f956f01e0c6cb1876fec8a3c5b617 · covers:a rename directory that cannot be made writes nothing
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/apply.go` · the staged rename directories are not recorded for discard: TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage and §118 must go red · acceptance-sha256:56aef2057b4fff04354a6ce8416665de5f5f956f01e0c6cb1876fec8a3c5b617 · covers:staged rename directories are taken back on abort
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/pathop.go` · a destination Lstat error other than not-exist passes validation again: TestARenameWhoseDestinationNameIsRejectedFailsValidation and §118 must go red · acceptance-sha256:56aef2057b4fff04354a6ce8416665de5f5f956f01e0c6cb1876fec8a3c5b617 · covers:a destination the filesystem rejects is refused before any write
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · staging no longer makes the rename directory: TestARenameWhoseDestinationDirectoryCannotBeCreatedWritesNothing and §118 must go red · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · covers:a rename directory that cannot be made writes nothing
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · the staged rename directories are not recorded for discard: TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage must go red · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · covers:staged rename directories are taken back on abort
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/pathop.go` · a destination Lstat error other than not-exist passes validation: TestARenameWhoseDestinationNameIsRejectedFailsValidation and §118 must go red · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · covers:a destination the filesystem rejects is refused before any write
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · missingDirs lists a dangling symlink as missing and discard removes it: TestAStagingAbortKeepsAPreExistingDanglingSymlink must go red · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · covers:staged rename directories are taken back on abort
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · the destination is not asked again after its parents are made: TestARenameWhoseLeafIsRejectedUnderANewParentWritesNothing and §118 must go red · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · covers:a destination the filesystem rejects is refused before any write
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · the staging-failed hunk keeps its Echo: TestAStagingFailureLeavesNoWriteDetailOnTheFailedHunk must go red · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · covers:a rename directory that cannot be made writes nothing

## Invariants

- A content staging failure still aborts before any rename directory is made.
- `staged[i]` pairs with `content[i]` in the commit loop.
- Exit codes unchanged.
- No `§NN` row in the Tests table.

## Risks

- 300 bytes is ENAMETOOLONG for one component on Linux, macOS and Windows. If a runner accepts it, the tests fail loudly rather than pass vacuously.
- The fence runs the full `contract.sh` and the campaign (about 30 s here).

## Stop Condition

- If going green needs a change in `internal/read`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` or `internal/mcp`, stop.
- If a staged directory cannot be taken back without changing `discard`'s refusal of a non-empty directory, stop.

## Out of Scope

- Undoing a failed path-op commit (deferred: ADR-066 T2)
- Commit-failure receipts (deferred: ADR-066 T3)

## Verification Log
(empty until execute)
- 2026-09-24 · c3f5661* · exit 1 · `set -o pipefail …` · acceptance-sha256:56aef2057b4fff04354a6ce8416665de5f5f956f01e0c6cb1876fec8a3c5b617 · ms:1325 · test-lock-sha256:108ddb076893f8740817bc8caa4d3398004e91a0b08ab892a16f47b212436c48 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QVJlbmFtZVdob3NlRGVzdGluYXRpb25EaXJlY3RvcnlDYW5ub3RCZUNyZWF0ZWRXcml0ZXNOb3RoaW5nCWEyNzc3YmEyNWFiMzcwODY0MTEyMzg5ZTQ2MTA2ZjM5YjJhNzc4ZmY5ODA0MWRhZGY4YjBlMmU0ZmZjOGU0ZTgKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFSZW5hbWVXaG9zZURlc3RpbmF0aW9uTmFtZUlzUmVqZWN0ZWRGYWlsc1ZhbGlkYXRpb24JMTIzYWFlMzk0ODE4OGZiOTQzMjkxNmMxZTI4YTc0OWI4MjY4OWI4NTE0MTc1MTgzZmVjYmM5M2JhOTliOTNhMgpib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QVN0YWdlZFJlbmFtZURpcmVjdG9yeUlzVGFrZW5CYWNrV2hlbkFMYXRlclJlbmFtZUNhbm5vdFN0YWdlCTE4MDgwNmQzZDYwNDk2ZjZlYzU3ODJhMWU1YWNiYmJlNGY2NDIyYjI0ZTFjZDExYjk0YzFhMTBmZjcwZDY5ZTY
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
      pathop_commit_test.go:82: a rejected name reached the filesystem: b.txt: rename /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestARenameWhoseDestinationNameIsRejectedFailsValidation3067660688/001/b.txt /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestARenameWhoseDestinationNameIsRejectedFailsValidation3067660688/001/ok/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx: file name too long (ALREADY WRITTEN: a.txt)
  --- FAIL: TestARenameWhoseDestinationNameIsRejectedFailsValidation (0.00s)
  === RUN   TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage
      pathop_commit_test.go:123: the aborted plan left new/ behind: <nil>
  --- FAIL: TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage (0.01s)
  === RUN   TestRenameMovesThePath
  --- PASS: TestRenameMovesThePath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.039s
  FAIL
  ```
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:56aef2057b4fff04354a6ce8416665de5f5f956f01e0c6cb1876fec8a3c5b617 · ms:56483
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:56aef2057b4fff04354a6ce8416665de5f5f956f01e0c6cb1876fec8a3c5b617 · ms:40046
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:56aef2057b4fff04354a6ce8416665de5f5f956f01e0c6cb1876fec8a3c5b617 · ms:35185
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:56aef2057b4fff04354a6ce8416665de5f5f956f01e0c6cb1876fec8a3c5b617 · ms:33006
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · ms:33110
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · ms:31762
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · ms:31861
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · ms:31709
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · ms:31869
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · ms:31531
- 2026-10-06 · d493df7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:c954f084e978f9362b6fe53d824a4a1674b1494a29aed5b4b9662d8f6bc77da3 · ms:0 · test-lock-sha256:e7b18e8e4c5730bbf33efb869bfe6b68d62d4f766c0bded853e24972d366ef90 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcGF0aG9wX2NvbW1pdF90ZXN0LmdvCVRlc3RBRmFpbGVkQ29udGVudENvbW1pdFJlcG9ydHNXcml0dGVuSHVua3NPa0FuZFRoZVJlc3RTa2lwcGVkCTAwYjUwYmYyMTM2YjIyMWE1NWVhZmVkMDUwZjI1ZWQxODU4ZTM3YmYzMGQwNGY5Zjg4ZDdkZDVjZjQzMjUzNmQKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFGYWlsZWRQYXRoT3BDb21taXRSZXBvcnRzTm90aGluZ1dyaXR0ZW5BZnRlclRoZVVuZG8JM2NlYWYzYWQ1NDBiMmU4NzM3OWIzYTU1YjQ1MDQ2MjI0MzMwYjFjZWM0ZjMzNmNlYjViOWU3OTU5ZGQxZThmZApib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QUZhaWxlZFJlbmFtZUFmdGVyQVJlcGxhY2luZ1JlbmFtZUxvc2VzTm9GaWxlCTRlMWVhZGJmNDBlNWIzZjBkMWI2MGNmYjQ5NzYzMzM2MGIyYmI4MDk4ODExMjI2ZTcyZmFmMDhiZTI0OTA3ODQKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFSZW5hbWVXaG9zZURlc3RpbmF0aW9uRGlyZWN0b3J5Q2Fubm90QmVDcmVhdGVkV3JpdGVzTm90aGluZwkwZTY2YmVmNTk2NTEzZDE2ZmVkZWVlY2UxMjllMGRiOTNjYzJiYzRlODJlMzNiZmE1NzkyMmE1MGMzYjc4NzMyCmJvZHkJaW50ZXJuYWwvYXBwbHkvcGF0aG9wX2NvbW1pdF90ZXN0LmdvCVRlc3RBUmVuYW1lV2hvc2VEZXN0aW5hdGlvbk5hbWVJc1JlamVjdGVkRmFpbHNWYWxpZGF0aW9uCTEyM2FhZTM5NDgxODhmYjk0MzI5MTZjMWUyOGE3NDliODI2ODliODUxNDE3NTE4M2ZlY2JjOTNiYTk5YjkzYTIKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFSZW5hbWVXaG9zZUxlYWZJc1JlamVjdGVkVW5kZXJBTmV3UGFyZW50V3JpdGVzTm90aGluZwkwYTE2NjgzODQwYjJiMDQyOWM5ZWMzMjRlMDJkZTI3NThhNDA0YmFhNWJlNTYwNmZjNGNkZGY1Mzk2ZGVmMTExCmJvZHkJaW50ZXJuYWwvYXBwbHkvcGF0aG9wX2NvbW1pdF90ZXN0LmdvCVRlc3RBU3RhZ2VkUmVuYW1lRGlyZWN0b3J5SXNUYWtlbkJhY2tXaGVuQUxhdGVyUmVuYW1lQ2Fubm90U3RhZ2UJMjkyOTRlMWQyNGVmNzc5YWQ3MTQxMjM4MGIxYTA3OGU0ZTVlM2FjNjAzOTBhMTc1Yjg3N2NlYjQzMTdmMmQ3Mwpib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QVN0YWdpbmdBYm9ydEtlZXBzQVByZUV4aXN0aW5nRGFuZ2xpbmdTeW1saW5rCTQ1YWMwNGZlMDI0ODVhNmMyZmJhZTg3Zjc3NzU0Y2EyZWUyMWVmYjg1NDRkMTlmODk1N2NlNzI4MzhiZmNjZTQKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFTdGFnaW5nRmFpbHVyZUxlYXZlc05vV3JpdGVEZXRhaWxPblRoZUZhaWxlZEh1bmsJYmU1YTU2OWE3NWZhM2EzNDc2YzU0NGI4ZTAwZjc1MzgxNGJjYTcwOTQyZTg1YTFjZWIwOTI3MTdiMDk0ZWVjNwpib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QW5VbmRvVGhhdEZhaWxzTG9zZXNOb0ZpbGUJMjc5NThiNWU1OTQwZGE1Y2NhMzg1MDM0MWI3OWM1NjMyMzk0YjRjNDhhNTdhODE4MDVhY2Q2YTU1OTdiZGIwMQ · test-lock-kind:replace
