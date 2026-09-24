# Task ADR-066-T2: a failed path-op commit undoes every unlink and rename

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `commitRenameFn` seam and path-op undo
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a replacing rename is undone before its unlink is restored`, `an undo that fails overwrites nothing`

## Goal

On any failure inside `commitPathOps` (`internal/apply/pathop.go:128-216`):
- every rename that completed is reversed, newest first (dest back to source);
- then each aside is restored to its original path, **unless that path is still occupied by a rename whose undo failed**;
- an aside that is not restored stays where it is, as a recovery file, and the error names it;
- each undo step's error is checked;
- a step that could not be undone keeps its file record as written, and the returned error names it.

No undo step may overwrite a file. Records for undone operations leave `res.Files`. A `commitRenameFn = os.Rename` seam (the pattern of `stageFileFn`, `internal/apply/apply.go:1465`) is used for every commit and undo rename (`apply.go:597`, `pathop.go:132,149,167`), so tests can fail one chosen rename.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `commitRenameFn` at `:597` |
| `internal/apply/pathop.go` | edit | seam at `:149,:167`; completed renames tracked; `restore()` becomes an ordered, checked undo that updates `res.Files` |
| `internal/apply/pathop_commit_test.go` | edit | two tests below |

## Ordered Steps

1. [S1] Write the two tests and confirm each RED on `main` on an assertion. [proof: mutation]
   - `TestAFailedRenameAfterAReplacingRenameLosesNoFile`: `@@ c.txt - unlink`, `@@ b.txt - rename` → `c.txt`, and `@@ d.txt - rename` → `e.txt`, with the seam failing the rename onto `e.txt`. Afterwards `c.txt` holds `OLD-C`, `b.txt` holds `B-CONTENT`, `d.txt` holds `D`, and no `.mrw-aside-*` exists. This is the data loss reproduced on v1.22.3.
   - `TestAnUndoThatFailsLosesNoFile`: the same plan, with the seam also failing the undo of `b.txt → c.txt`. Afterwards `c.txt` still holds `B-CONTENT`; `OLD-C` survives in the `.mrw-aside-*` file, which the error names; `d.txt` holds `D`. `res.Files` still reports the rename as written. Both original contents exist on disk.
2. [S2] Implement the seam and the ordered, checked undo; confirm GREEN. `TestRenameMovesThePath` and T1's tests stay green. [proof: mutation]
   Mutants:
   - reverse renames AFTER restoring asides (the old order): kills the first test;
   - `restore()` a no-op: kills the first test (ADR-057's surviving mutant);
   - restore an aside onto a path whose rename undo failed: kills the second test;
   - ignore an undo error: kills the second test.
3. [S3] Scoped tests, `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -count=1 -v \
    -run 'TestAFailedRenameAfterAReplacingRenameLosesNoFile|TestAnUndoThatFailsLosesNoFile|TestRenameMovesThePath' 2>&1 | tee /tmp/adr066-t2.out \
  && grep -q '^--- PASS: TestAFailedRenameAfterAReplacingRenameLosesNoFile ' /tmp/adr066-t2.out \
  && grep -q '^--- PASS: TestAnUndoThatFailsLosesNoFile ' /tmp/adr066-t2.out \
  && grep -q '^--- PASS: TestRenameMovesThePath ' /tmp/adr066-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr066-t2.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state \
  && [ -z "$(gofmt -l internal/apply)" ] \
  && go vet ./internal/apply/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFailedRenameAfterAReplacingRenameLosesNoFile` | `internal/apply/pathop_commit_test.go` | a later rename failure undoes the replacing rename before restoring the unlink | — | S1, S2 |
| `TestAnUndoThatFailsLosesNoFile` | `internal/apply/pathop_commit_test.go` | a failed undo overwrites nothing: both original contents survive, the recovery aside is named | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | every plan with an unlink or rename commits through `commitPathOps`; ADR-066 T3's §119 drives the undo through the binary with a read-only directory |
| 3 — the caller can discover it | the tree is left as it was; the error names any step that could not be undone |
| 4 — it is used | reproduced on v1.22.3 on 2026-09-24; ADR-009 refuses telemetry |

## Mutation Log
(empty until execute)
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/pathop.go` · completed renames are not undone before the asides: TestAFailedRenameAfterAReplacingRenameLosesNoFile must go red · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · covers:a replacing rename is undone before its unlink is restored
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/pathop.go` · the asides are never restored (ADR-057 surviving mutant): TestAFailedRenameAfterAReplacingRenameLosesNoFile must go red · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · covers:a replacing rename is undone before its unlink is restored
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/pathop.go` · an aside is restored onto a path whose rename undo failed: TestAnUndoThatFailsLosesNoFile must go red · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · covers:an undo that fails overwrites nothing
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/pathop.go` · a failed rename undo is not recorded as holding its path: TestAnUndoThatFailsLosesNoFile must go red · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · covers:an undo that fails overwrites nothing
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/pathop.go` · completed renames are not undone before the asides: TestAFailedRenameAfterAReplacingRenameLosesNoFile must go red · acceptance-sha256:55d640207594245a472cca1c179024ed95d933a2b130396fb96b42bc9b8bc838 · covers:a replacing rename is undone before its unlink is restored
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/pathop.go` · the asides are never restored (ADR-057 surviving mutant): TestAFailedRenameAfterAReplacingRenameLosesNoFile must go red · acceptance-sha256:55d640207594245a472cca1c179024ed95d933a2b130396fb96b42bc9b8bc838 · covers:a replacing rename is undone before its unlink is restored
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/pathop.go` · an aside is restored onto a path whose rename undo failed: TestAnUndoThatFailsLosesNoFile must go red · acceptance-sha256:55d640207594245a472cca1c179024ed95d933a2b130396fb96b42bc9b8bc838 · covers:an undo that fails overwrites nothing
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/pathop.go` · a failed rename undo is not recorded as holding its path: TestAnUndoThatFailsLosesNoFile must go red · acceptance-sha256:55d640207594245a472cca1c179024ed95d933a2b130396fb96b42bc9b8bc838 · covers:an undo that fails overwrites nothing

## Invariants

- A plan whose path ops all succeed commits exactly as before.
- Content renames already committed are not undone (ADR-001).
- No `§NN` row in the Tests table.

## Risks

- The seam is package state; tests restore it with `t.Cleanup` and do not run in parallel.

## Stop Condition

- If the undo needs a backup copy of any file, stop and return to the record.

## Out of Scope

- Receipts after the undo (deferred: ADR-066 T3)
- Staging the unlink placeholder (deferred: docs/adr/BACKLOG.md)

## Verification Log
(empty until execute)
- 2026-09-24 · c3f5661* · exit 1 · `set -o pipefail …` · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · ms:449 · test-lock-sha256:4e832c08206a2bd395992f2ece70f059441616f7c74948982aab30ff49bd1fa0 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QUZhaWxlZFJlbmFtZUFmdGVyQVJlcGxhY2luZ1JlbmFtZUxvc2VzTm9GaWxlCTRlMWVhZGJmNDBlNWIzZjBkMWI2MGNmYjQ5NzYzMzM2MGIyYmI4MDk4ODExMjI2ZTcyZmFmMDhiZTI0OTA3ODQKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFSZW5hbWVXaG9zZURlc3RpbmF0aW9uRGlyZWN0b3J5Q2Fubm90QmVDcmVhdGVkV3JpdGVzTm90aGluZwlhMjc3N2JhMjVhYjM3MDg2NDExMjM4OWU0NjEwNmYzOWIyYTc3OGZmOTgwNDFkYWRmOGIwZTJlNGZmYzhlNGU4CmJvZHkJaW50ZXJuYWwvYXBwbHkvcGF0aG9wX2NvbW1pdF90ZXN0LmdvCVRlc3RBUmVuYW1lV2hvc2VEZXN0aW5hdGlvbk5hbWVJc1JlamVjdGVkRmFpbHNWYWxpZGF0aW9uCTEyM2FhZTM5NDgxODhmYjk0MzI5MTZjMWUyOGE3NDliODI2ODliODUxNDE3NTE4M2ZlY2JjOTNiYTk5YjkzYTIKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFTdGFnZWRSZW5hbWVEaXJlY3RvcnlJc1Rha2VuQmFja1doZW5BTGF0ZXJSZW5hbWVDYW5ub3RTdGFnZQkxODA4MDZkM2Q2MDQ5NmY2ZWM1NzgyYTFlNWFjYmJiZTRmNjQyMmIyNGUxY2QxMWI5NGMxYTEwZmY3MGQ2OWU2CmJvZHkJaW50ZXJuYWwvYXBwbHkvcGF0aG9wX2NvbW1pdF90ZXN0LmdvCVRlc3RBblVuZG9UaGF0RmFpbHNMb3Nlc05vRmlsZQkyNzk1OGI1ZTU5NDBkYTVjY2EzODUwMzQxYjc5YzU2MzIzOTRiNGM0OGE1N2E4MTgwNWFjZDZhNTU5N2JkYjAx
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
      pathop_commit_test.go:193: a failed rename was not reported
  --- FAIL: TestAFailedRenameAfterAReplacingRenameLosesNoFile (0.00s)
  === RUN   TestAnUndoThatFailsLosesNoFile
      pathop_commit_test.go:218: a failed rename was not reported
  --- FAIL: TestAnUndoThatFailsLosesNoFile (0.00s)
  === RUN   TestRenameMovesThePath
  --- PASS: TestRenameMovesThePath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.019s
  FAIL
  ```
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · ms:1090
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · ms:626
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · ms:687
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:a628f24f8e2a099195c1ca25ade7bf865a74efa73fb0f4a1a6830815b338a6bd · ms:723
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:55d640207594245a472cca1c179024ed95d933a2b130396fb96b42bc9b8bc838 · ms:274
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:55d640207594245a472cca1c179024ed95d933a2b130396fb96b42bc9b8bc838 · ms:268
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:55d640207594245a472cca1c179024ed95d933a2b130396fb96b42bc9b8bc838 · ms:254
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:55d640207594245a472cca1c179024ed95d933a2b130396fb96b42bc9b8bc838 · ms:254
