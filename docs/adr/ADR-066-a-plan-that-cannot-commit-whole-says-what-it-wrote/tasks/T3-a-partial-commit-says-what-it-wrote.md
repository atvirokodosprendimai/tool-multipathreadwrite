# Task ADR-066-T3: a commit failure reports ok / failed / skipped by what reached disk; contract §119

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** commit-failure verdicts; consistent Echo, Balance and Advisories; `PARTIALLY APPLIED`; §119; ADR-001 rule 3 amendment
**Consumes:** `abortStage` shared by the staging loops (T1); `commitRenameFn` seam and path-op undo (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `written hunks stay ok and the rest are skipped`, `a hunk that is not ok carries no write detail`, `unused rename directories are taken back after a commit failure`, `the summary does not say applied`, `the binary loses nothing and says what it wrote`

## Goal

Both commit-failure returns (`internal/apply/apply.go:599`, and `commitPathOps`) go through one function that leaves a receipt saying what reached disk:
- `ok` on hunks whose file is written, `failed` on the hunk whose commit failed (with its error), `skipped` on the rest;
- `Echo` and `Balance` empty on every non-`ok` hunk; `Advisories` recounted;
- `Failed ≥ 1`, `Applied` false;
- `files[]` holds the written records plus every other addressed file with `written: false`;
- `writtenSoFar` lists written files only;
- rename directories staged by T1 that no completed rename used are removed.

`report()` (`cmd/mrw/main.go:1539`) prints `PARTIALLY APPLIED` when a hunk failed and a file was written, tested before `NOTHING WRITTEN`. The `Status` comment (`apply.go:33-34`) and ADR-001 rule 3 say `skipped` means "this hunk's file was not written".

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the verdict function; `writtenSoFar` (written records only); `Status` comment |
| `internal/apply/pathop.go` | edit | returns the failing path to the verdict function |
| `internal/apply/pathop_commit_test.go` | edit | two tests below |
| `internal/apply/apply_test.go` | edit | `TestAPartialWriteNamesWhatWasAlreadyWritten` fixtures become `Written: true`, plus an unwritten sibling that must be omitted |
| `cmd/mrw/main.go` | edit | `report()` summary word |
| `cmd/mrw/report_partial_test.go` | new | `TestAPartialCommitIsNotSummarisedAsApplied` |
| `internal/mcp/partial_receipt_test.go` | new | `TestTheMCPReceiptOfAPartialCommitNamesEachHunk` |
| `internal/mcp/schema.go`, `internal/mcp/mcp.go` | edit | `failed`, `files.written`, `hunks.status` and `mrw_write`'s all-or-nothing sentence describe a partial commit truthfully (Codex review of #207) |
| `internal/mcp/partial_description_test.go` | new | `TestTheReceiptDescriptionsAllowAPartialCommit` |
| `internal/guide/guide.go`, `internal/guide/guide_test.go` | edit | the Shared sentence no longer promises that any failed hunk means nothing was written; `TestSharedSaysAFailedCommitIsReportedPartial` |
| `scripts/contract.sh` | edit | the four Shared-sentence literals (§75 and its siblings) follow the new wording |
| `scripts/contract.sh` | edit | §119 |
| `docs/adr/ADR-001-a-plan-addresses-the-original-file-and-applies-whole-or-not-at-all.md` | edit | amendment: `skipped` after ADR-066 |

## Ordered Steps

1. [S1] Write the tests and confirm each RED on T1+T2's tree on an assertion. [proof: mutation]
   - `TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped`: brace-changing single-line edits to `a.go`, `b.go` and `c.go` (non-prose, so `Balance` is computed) with `EchoPad` 1, plus a rename to `new/r.txt`. The seam fails the rename onto `b.go`. A control run of the same plan with no seam applies with a non-empty `Balance` on all three edits, which proves the fixture produces it. Then:
     - `a` is `ok` and written, with its `Balance`; `b` is `failed`; `c` and the rename are `skipped`; neither `b` nor `c` carries `Echo` or `Balance`;
     - `Failed==1`, `Applied==false`, `Advisories==1`;
     - no `new/` directory and no `.mrw-*` file is left.
   - `TestAFailedPathOpCommitReportsNothingWrittenAfterTheUndo`: T2's replacing-rename plan, with the seam failing the last rename. Every path-op hunk is `failed` or `skipped`, no file is recorded written, and `writtenSoFar` says "nothing was written".
   - `TestAPartialCommitIsNotSummarisedAsApplied`: `report()` on a Result with `Failed:1` and one written file prints `PARTIALLY APPLIED` and not `— applied`.
   - The updated `TestAPartialWriteNamesWhatWasAlreadyWritten`: an unwritten record is not named.
   A guard that already holds, proved by mutation rather than a red run: `TestTheMCPReceiptOfAPartialCommitNamesEachHunk`. `writeReport` on the same Result prints the `ok`, `failed` and `skipped` rows, and a mutant that prints `ok` for every hunk must kill it.
2. [S2] Implement; confirm GREEN, and that T1's and T2's tests stay green. [proof: mutation]
   Mutants:
   - written-file hunks marked `skipped`: kills the content test;
   - `Echo` kept on a skipped hunk: kills the content test;
   - the unused rename directory left: kills the content test;
   - `writtenSoFar` lists unwritten records: kills the path-op test and the updated `TestAPartialWriteNamesWhatWasAlreadyWritten`;
   - `Advisories` not recounted: kills the content test;
   - the partial case placed after `NOTHING WRITTEN`: kills the report test.
3. [S3] §119, driving the binary with a destination directory made `chmod 0555`. As in §21, the row writes a probe into the directory first and SKIPs visibly if the write succeeds (root). [proof: mutation]
   - Good: the replacing-rename plan into a writable directory exits 0.
   - Must-fail, the replacing-rename plan (unlink `c`, rename `b → c`, rename `d → ro/d.txt`): exit 2; `c.txt` holds `OLD-C`, `b.txt` holds `B-CONTENT`, `d.txt` is present; no `— applied`.
   - Must-fail, a mixed plan (an edit to `a.txt`, rename `d → ro/d.txt`): exit 2; `a.txt` changed; the output has `ok`, `FAIL` and `PARTIALLY APPLIED`; `--json | jq -e '.failed==1 and .applied==false and ([.files[]|select(.written)]|length==1)'`.
   - Confirm RED against v1.22.3 in a mini-harness, then GREEN in the full `./scripts/contract.sh`.
4. [S4] Amend ADR-001 rule 3; `Status` comment; BACKLOG; `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 119\. ' scripts/contract.sh \
  && grep -q 'Amended 2026-09-24 by ADR-066' docs/adr/ADR-001-a-plan-addresses-the-original-file-and-applies-whole-or-not-at-all.md \
  && go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ -count=1 -v \
    -run 'TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped|TestAFailedPathOpCommitReportsNothingWrittenAfterTheUndo|TestAPartialCommitIsNotSummarisedAsApplied|TestTheMCPReceiptOfAPartialCommitNamesEachHunk|TestTheReceiptDescriptionsAllowAPartialCommit|TestAFailedRenameAfterAReplacingRenameLosesNoFile|TestARenameWhoseDestinationDirectoryCannotBeCreatedWritesNothing|TestAPartialWriteNamesWhatWasAlreadyWritten' 2>&1 | tee /tmp/adr066-t3.out \
  && grep -q '^--- PASS: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped ' /tmp/adr066-t3.out \
  && grep -q '^--- PASS: TestAFailedPathOpCommitReportsNothingWrittenAfterTheUndo ' /tmp/adr066-t3.out \
  && grep -q '^--- PASS: TestAPartialCommitIsNotSummarisedAsApplied ' /tmp/adr066-t3.out \
  && grep -q '^--- PASS: TestTheMCPReceiptOfAPartialCommitNamesEachHunk ' /tmp/adr066-t3.out \
  && grep -q '^--- PASS: TestTheReceiptDescriptionsAllowAPartialCommit ' /tmp/adr066-t3.out \
  && grep -q '^--- PASS: TestAFailedRenameAfterAReplacingRenameLosesNoFile ' /tmp/adr066-t3.out \
  && go test ./internal/guide/ -count=1 -v -run 'TestSharedSaysAFailedCommitIsReportedPartial' 2>&1 | tee /tmp/adr066-t3g.out \
  && grep -q '^--- PASS: TestSharedSaysAFailedCommitIsReportedPartial ' /tmp/adr066-t3g.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr066-t3.out \
  && ./scripts/contract.sh > /tmp/adr066-t3-contract.out 2>&1 \
  && grep -qE '^  (PASS  a failed rename after a replacing rename loses no file|SKIP  a read-only directory is writable here)' /tmp/adr066-t3-contract.out \
  && grep -qE '^  (PASS  a partial commit is reported as partially applied|SKIP  a read-only directory is writable here)' /tmp/adr066-t3-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state \
  && [ -z "$(gofmt -l internal/apply cmd/mrw internal/mcp internal/guide)" ] \
  && go vet ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ ./internal/guide/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped` | `internal/apply/pathop_commit_test.go` | verdicts by what reached disk; no write detail on non-ok; unused rename directory removed | — | S1, S2 |
| `TestAFailedPathOpCommitReportsNothingWrittenAfterTheUndo` | `internal/apply/pathop_commit_test.go` | after T2's undo, nothing is reported written | — | S1, S2 |
| `TestAPartialCommitIsNotSummarisedAsApplied` | `cmd/mrw/report_partial_test.go` | the CLI summary says `PARTIALLY APPLIED` | — | S1, S2 |
| `TestTheMCPReceiptOfAPartialCommitNamesEachHunk` | `internal/mcp/partial_receipt_test.go` | guard: the MCP report prints each hunk's verdict | — | S1 |
| `TestAPartialWriteNamesWhatWasAlreadyWritten` | `internal/apply/apply_test.go` | only written records are named | — | S1, S2 |
| `TestTheReceiptDescriptionsAllowAPartialCommit` | `internal/mcp/partial_description_test.go` | the MCP receipt descriptions no longer say a non-zero `failed` means nothing was written | — | S1, S2 |
| `TestSharedSaysAFailedCommitIsReportedPartial` | `internal/guide/guide_test.go` | the Shared sentence says a failed commit is reported PARTIALLY APPLIED | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §119 |
| 2 — something selects it | both commit-failure returns call the verdict function; `mrw write` renders it; MCP `writeReport` prints the statuses; §119 drives it through the binary |
| 3 — the caller can discover it | per-hunk statuses, `PARTIALLY APPLIED`, and `ALREADY WRITTEN` on stderr |
| 4 — it is used | found by the chaos pass and its review; ADR-009 refuses telemetry |

## Mutation Log
(empty until execute)
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/apply.go` · hunks on written files are marked skipped: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped and §119 must go red · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · covers:written hunks stay ok and the rest are skipped
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/apply.go` · a skipped hunk keeps its Echo: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped must go red · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · covers:a hunk that is not ok carries no write detail
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/apply.go` · a content commit failure leaves the staged rename directory: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped must go red · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · covers:unused rename directories are taken back after a commit failure
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/apply/apply.go` · writtenSoFar names unwritten records: TestAPartialWriteNamesWhatWasAlreadyWritten and TestAFailedPathOpCommitReportsNothingWrittenAfterTheUndo must go red · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · covers:written hunks stay ok and the rest are skipped
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `cmd/mrw/main.go` · the partial case is tested after NOTHING WRITTEN: TestAPartialCommitIsNotSummarisedAsApplied and §119 must go red · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · covers:the summary does not say applied
- 2026-09-24 · c3f5661* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP report prints every hunk as ok: TestTheMCPReceiptOfAPartialCommitNamesEachHunk must go red · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · covers:the binary loses nothing and says what it wrote
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · hunks on written files are marked skipped: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped and §119 must go red · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · covers:written hunks stay ok and the rest are skipped
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · a skipped hunk keeps its Echo: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped must go red · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · covers:a hunk that is not ok carries no write detail
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · a content commit failure leaves the staged rename directory: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped must go red · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · covers:unused rename directories are taken back after a commit failure
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/apply/apply.go` · writtenSoFar names unwritten records: TestAPartialWriteNamesWhatWasAlreadyWritten must go red · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · covers:written hunks stay ok and the rest are skipped
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `cmd/mrw/main.go` · the partial case is tested after NOTHING WRITTEN: TestAPartialCommitIsNotSummarisedAsApplied and §119 must go red · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · covers:the summary does not say applied
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP report prints every hunk as ok: TestTheMCPReceiptOfAPartialCommitNamesEachHunk must go red · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · covers:the binary loses nothing and says what it wrote
- 2026-09-24 · b7df6e1* · mutant killed · exit 1 · `internal/mcp/schema.go` · the failed description again says any failure wrote nothing: TestTheReceiptDescriptionsAllowAPartialCommit must go red · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · covers:the summary does not say applied
- 2026-09-24 · a87ba4d* · mutant killed · exit 1 · `internal/apply/apply.go` · hunks on written files are marked skipped: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped and §119 must go red · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · covers:written hunks stay ok and the rest are skipped
- 2026-09-24 · a87ba4d* · mutant killed · exit 1 · `internal/apply/apply.go` · a skipped hunk keeps its Echo: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped must go red · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · covers:a hunk that is not ok carries no write detail
- 2026-09-24 · a87ba4d* · mutant killed · exit 1 · `internal/apply/apply.go` · a content commit failure leaves the staged rename directory: TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped must go red · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · covers:unused rename directories are taken back after a commit failure
- 2026-09-24 · a87ba4d* · mutant killed · exit 1 · `internal/apply/apply.go` · writtenSoFar names unwritten records: TestAPartialWriteNamesWhatWasAlreadyWritten must go red · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · covers:written hunks stay ok and the rest are skipped
- 2026-09-24 · a87ba4d* · mutant killed · exit 1 · `cmd/mrw/main.go` · the partial case is tested after NOTHING WRITTEN: TestAPartialCommitIsNotSummarisedAsApplied and §119 must go red · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · covers:the summary does not say applied
- 2026-09-24 · a87ba4d* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP report prints every hunk as ok: TestTheMCPReceiptOfAPartialCommitNamesEachHunk must go red · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · covers:the binary loses nothing and says what it wrote
- 2026-09-24 · a87ba4d* · mutant killed · exit 1 · `internal/mcp/schema.go` · the failed description again says any failure wrote nothing: TestTheReceiptDescriptionsAllowAPartialCommit must go red · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · covers:the summary does not say applied
- 2026-09-24 · a87ba4d* · mutant killed · exit 1 · `internal/guide/guide.go` · the Shared sentence again promises nothing is written on any failure: TestSharedSaysAFailedCommitIsReportedPartial and the contract Shared rows must go red · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · covers:the summary does not say applied

## Invariants

- The `hunks.status` enum stays `ok|failed|skipped`; no JSON field is added.
- Exit code 2 on a commit failure, unchanged.
- `internal/mcp` code changes only in the receipt and tool descriptions; its rendering is unchanged.
- No `§NN` row in the Tests table.

## Risks

- §119 SKIPs as root; the Go tests carry the proof there.
- The seam is package state; tests restore it with `t.Cleanup`.

## Stop Condition

- If an honest verdict needs a new status value or JSON field, stop and return to the record.

## Out of Scope

- Recording the ledger after a partial commit (deferred: docs/adr/BACKLOG.md)
- Undoing content renames already committed (permanent: boundary: ADR-001 accepts a partial tree on a failing rename and names what was written)

## Verification Log
(empty until execute)
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:33738
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:48568
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:52957
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:44800
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:56008
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:64383
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:66951
- 2026-09-24 · c3f5661* · exit 1 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:871 · test-lock-sha256:715dcc5c48a14f85b0559222ee2b8e7f3d859800e08e7ddbe7b50471e6a936f6 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcmVwb3J0X3BhcnRpYWxfdGVzdC5nbwlUZXN0QVBhcnRpYWxDb21taXRJc05vdFN1bW1hcmlzZWRBc0FwcGxpZWQJMWY0YjRlOGI3ZjllNmM2YjdlODBlZjg4NmJkZmU1OTgzMjRmM2U3MGRiNzBkMTdjYmRkY2Y0MjNlMGE5ODAwMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFCb2R5bGVzc0RlbGV0ZUlzVW5jaGFuZ2VkCWMzNzIyMjcyMTVjODkyMzcxYTRmYWJlZTE3N2U4NmU2YzdkYjcxZWRiMjdlYzJlYzZlMWUwODVjZDRiNTgxNjEKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBQ3JlYXRlQ2Fycmllc05vQmFsYW5jZQk4Yjk4ZDIwODEwOTI3M2RmYzFkODkwMjU3YjM0MWJhYmI2MTVhNTliYTQ0M2IxMmVjZmYxNTJjMmMzMDVhZmVkCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QURlbGV0ZU9mQmxhbmtMaW5lc1N0aWxsUmVjb3Jkc0JvdW5kcwk4ZGUzYTQwN2I4ODg2MDg0ZjI2MDQ1ZjhjODA2NmY1NzI1ZDQ0NzUzZjY4YzBmNjRiZjBhMjk2MmZjYjQ1MGFkCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QURlbGV0ZVdob3NlRXhwZWN0ZWRSZW1vdmFsRGlmZmVyc05hbWVzVGhlTGluZQllZjdmYTg4ZTFmYzRjYTJhMDUxYmE3YjYxYTE4MTJkZjc0NWI4NzMxYzY3Mzc5ZDZkYjliZTEzOTk4YmViMWFiCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QURlbGV0ZVdob3NlRXhwZWN0ZWRSZW1vdmFsTWF0Y2hlc0FwcGxpZXMJZGEzMWQ1NjhjZjI0NmZkZTIwNjFkZWIwOGEyNGNkYTJkZTcxMjNhNjQ3ZTFhNDNlY2UxMzhmMzU5NDU5YjMxOApib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFEZWxldGVXaXRoQW5FeHBlY3RlZEJvZHlTdGlsbFJlcG9ydHNJdHNOZXQJNWNhOTIxZGY2NGY4ZmIyYTU0ZGRkOTNmYWY1YjgyMzgyMDZkNTkxYmJkYmQyZjVjNmE2NmQwNDNhYTNkNjI3MApib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFEZWxpbWl0ZXJCYWxhbmNlRGVsdGFEb2VzTm90RmFpbFRoZUh1bmsJMzljYjVmYWYwNTQ2OTljZDRlZTcxYzRhMzVhNzRjOWQ1NTBmNmYyMTZmYzYyOWMyNTNjNGMwNTQ3ZjMxZWE5ZQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFGYWlsZWRGaWxlSXNSZXBvcnRlZE9uY2VOb3RQZXJIdW5rCTIwODQ0YTQ2YTg5ODg5YTk3ZmM3NDY5NTk2MGU1NDYzODQzZDFmM2VjOTA2ZmU4OTdjZjYxNzY0ZTE5Yzg1Y2UKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRmFpbGVkT3JTa2lwcGVkRGVsZXRlUmVjb3Jkc05vQm91bmRzCWJjYzBmYzEyYzY2NjFmNDMyZDA1N2QwZmQxNGM2MDk0MmVhMTI0MGFmMDc5MDliNDNmMDMwNTRiNGZlZTUzM2MKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRmFpbGVkU3RhZ2VMZWF2ZXNUaGVUcmVlVW50b3VjaGVkCWVhZTI2Njc3YzRlYmVlNGNhY2NiZTgwMzgzYTczZDE5NmNmNmNiNTI5NDIxYWQ0MDQ0Y2Q2ZmQyNzAwZDNhYjAKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRmlsZUNoYW5nZWRCZWhpbmRNcndzQmFja0Nhbm5vdEJlRWRpdGVkCTgwYjY2N2NiMDA1Mjk4ZjIwZGRlOTVmNjkzNzNiNmRmMDYxNWM2ZWVkY2U2NjYxNWVmZWYzMjgwMDY5NmQzYmIKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBRmlsZU5ldmVyU2VlbkNhbm5vdEJlRWRpdGVkCTkzOGNlY2UwNzIzOWRiNDFmOWJjODQ3ZDFiMTZkNTM5NjI5YzE0M2EyZDIxYmEwZGRiODU1YjFlMTRjYzg3NjcKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBTXVsdGlMaW5lUmVwbGFjZVdpdGhBU2VydmVkTGluZUFmdGVyRW5kQXBwbGllcwk3ZmI2ODE0ZGVjNzVlNzE1NDg3YzA5MjA1NzUwOGFjMjU2NDM0OGJiNjk4ZmJmYzZiOTEzODE4MWUyYzgyYjc0CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QU11bHRpTGluZVJlcGxhY2VXaXRoQW5BbmNob3JTdGlsbEFwcGxpZXMJYTRiNGI1OGJlNWVlODE2ZWM5NDc5ZmNmZWQ3MDdiMmM5MjJiMzkwYWM0OWExYTQzODFiNTFmN2ZlNmRmZDFkMQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFNdWx0aUxpbmVSZXBsYWNlV2l0aG91dEFTZXJ2ZWRMaW5lQWZ0ZXJFbmRXcml0ZXNOb3RoaW5nCTRlYjJmY2JjNzFiNmQ5MjhjODUyZjFiNzEyZDhmNzBlYWUwM2QwY2MzNGU3YmUyNTg2YzRjZTFhNzg1Mzk3MzIKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBTXVsdGlMaW5lUmVwbGFjZVdpdGhvdXRBbkFuY2hvcklzUmVmdXNlZAk5MGM1ODkxNzkxZDMxZTdkYTY0NzRmN2U1NTVlYTAwMTNiNDdmMmY4MzE4MjY5NDdhNzE1NGRhYmFjZjU0Nzk5CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QU9uZUxpbmVEZWxldGVSZWNvcmRzVGhlU2FtZUxpbmVUd2ljZQk1NDFiY2Y5YWExZmNjZGFhNTE4OTNhMmRkYTQzNTdlNjA2NWJhNzJjYzYxMjUwNDE1MWYzNzE0NjhlNjhhNWZkCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVBhZGRlZFdyaXRlRWNob1Nob3dzVGhlTGluZUFmdGVyVGhlQm9keQllNWQzYjY4MWMwYTFhNjZlMjExZGNiNmRlZjVmNjZmNjBlYTk1NzRhOTQ4YzQwYTg0ZjhlMjJkMmMwNGVhYTI2CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVBhZGRlZFdyaXRlRWNob1Nob3dzVGhlV3JpdHRlbkZpbGVOb3RUaGVPcmlnaW5hbFRhaWwJZGExNTcwOTgxZWM3OTY5Y2MzODkyYTg5NjAyMDU2MDlhOTEzOTc0ZTBmZjg2NGI0N2QwOGQzMjNiMDQwYjg2Zgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFQYXJ0aWFsV3JpdGVOYW1lc1doYXRXYXNBbHJlYWR5V3JpdHRlbgliMmFhZDA0NWRiNjhmZjY4MjQ3MzA4MTIxN2M3MGM0Zjc1NmQ2NTZkOWM4NWJlYzA3OWExNTNlZDc2NGU4YzllCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVBhdHRlcm5NYXRjaGluZ05vdGhpbmdJc1JlZnVzZWQJY2E4MTgxYjkxZDJhOGMzNWFkMmE4MGZhZjZiNzhmMzVmZmNiMmQ3YWRjYjE3YjkzNDAxZmRmMmE3NDZhZDM3OQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFQYXR0ZXJuUmFuZ2VTcGFubmluZ01hbnlMaW5lc05lZWRzQW5BbmNob3IJNDMyYzYzMzdiMjY1ODFmNDQzZTg1ZDg5ZWM4ZjNiYmExZWVlYzE1YjQyOGQ5MGRlZWNiOGEzZWRkOWNkNjIxMApib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFQbGFuVGhhdE5hbWVzT25lRmlsZVR3aWNlSXNSZWZ1c2VkV2hpY2hldmVyVGhlU3BlbGxpbmcJZTlhNGE0ZWExZDBmMjUxM2QzNmMxMjZmZDVjMzIwOGIxMTliMjYxMzEzNjM2YzA0NDE5YTRmMmQ4NGY5OTVlOQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFSZWFkVW5kZXJPbmVTcGVsbGluZ0xpY2Vuc2VzVGhlU2FtZUZpbGVVbmRlckFub3RoZXIJZjYyMmE0NjRmY2MzNjIzNGQ5NTUxMzEyZjkzOWVjZDdlYzU0YjQyNzE4ZjQ3Mzk1ZTU1NDc1YzkyNTRkOGU4MQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFSZWNlaXB0RWNob2VzVGhlUmVsYXRpdmVFbmRUaGVDYWxsZXJXcm90ZQk1NjBhYmFjYTFhOTEyMjZlOGRkYWQ4YWMxNGNlZDgyYTI5NTBkOGE4NzAzNTQ5MmViNDNmMzdjYWFkYzFkOTJhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVJlZ2V4QWRkcmVzc0lzU3RpbGxTdWJqZWN0VG9UaGVMZWRnZXIJZjc0Y2RkZmM1NzcyMzlhODYwMzlkMDUyZDdmMGRjODNjYzE5MDYyMDNkODFhOWM5OGIxZmNiMjJiODBmZmY0Ywpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFSZWdleEFkZHJlc3NSZXNvbHZlc0FnYWluc3RUaGVPcmlnaW5hbEZpbGUJMzQwY2QxYmIzYzgyOWNmYjViMjU1MjE0NmM0ZjFiYzUwNzI4MjYxMWM5N2E4NWZhZDhjZDAxMzRmMTc2YjU1Mgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFSZWxhdGl2ZUVuZEFkZHJlc3Nlc1RoZUxpbmVzSXRSZXBsYWNlcwljNjg3ZGJhN2UyNTcwNzcyODQ0ZjAzNmRkYTE0OTRlM2I3Y2ZhNTAxOGVjYjRhMzM5ZWYwZGNiZGJhYzhlZWJhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kQXRUaGVJbnRlZ2VyQm91bmRhcnlEb2VzTm90V3JhcAk5NjkwMmRiMzdkZWNkOWEwNWZiM2Y4MGRhMTlhMTkzMWYyYzI4NjNmOTYzMmFhMTY1MTIyZWNjOTE4OTE1NGVkCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVJlbGF0aXZlRW5kUGFzdFRoZUxhc3RMaW5lSXNSZWZ1c2VkT25UaGVQbGFuUGF0aAkzMWE0NTkyOWUzYTdjMTE0N2FjZWI1MjdmMjQwZjU0MjA4YTc5NzgyMWNhMDg5NDkyNjRjNzUxNmZlYjU2YTFhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVJldmVyc2VkUmFuZ2VGcm9tRU9GU2F5c1NvCTE1YjMyM2M0NTM4ZjMzOGRmNzhhMjI0NTQ5MzE0NzI0YTY4YmMyZjE0NDE4YWJmNDY2ZWM4NDdmMTRkZDQ3NTIKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBU2VlbkZpbGVJc0VkaXRhYmxlCTViY2JkMDgyNGU5NjMxYTMyZWJjODMwYjY1OTMzMzIzMjk5NTcwNjZlNmNjYTQ5YWRiZTNjMjkzODczOGYzNGMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBU2luZ2xlTGluZVJlcGxhY2VEb2VzTm90TmVlZEFOZWlnaGJvdXIJNTk1NGJlOWJjNWM4MjczZmU4YzBhZmJmNjRmNjcyZTliMTU1Y2ZmNjc3YmY3NGM0MmNmOWMyMTMzZWM2YjlmMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFTaW5nbGVMaW5lUmVwbGFjZU5lZWRzTm9BbmNob3IJMTZhOGM0NzJmODJjOWZhNGQzZjQ5ZGUyNGNmNGNiYWNjMWFlZWI4ZDdjZDdkNzEyOTIyNzU4YjFjNTU4ZGM0Ywpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFTa2lwcGVkSHVua09taXRzRWNobwkzNmViZjA1N2I2MzI0NzRjMGQ1MTE0MzZmNzkwMGM3NWRiYTRkZGNlMWVlMTNiYjUzNWE3MzdiM2NlYTU5ZDU1CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0QVN0YWdlVGhhdEZhaWxzQWZ0ZXJNYWtpbmdEaXJlY3Rvcmllc0dpdmVzVGhlbUJhY2sJMmFmNzBmMTBjM2Q4ZDFjZWNjYzc2NDY5MDRjMTBjZTVlNGQxZWU3NjUwNTMwZjlkYWMzN2RiZmQxZDkzNzJjOApib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFkZHJlc3Nlc1Jlc29sdmVBZ2FpbnN0VGhlT3JpZ2luYWxGaWxlCWZiN2I3MDMxMzdlODFhNWZkYmQzZDU0MjEyZmUzOGNiZWFhMzA0ZGU1YjRhNWI2NGE0MWQwYzFlZGRhMzU4YmQKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBZHZpc29yaWVzQ291bnRzT25seU9rSHVua3NXaXRoQUJhbGFuY2UJNzgzYWQ3ZWQ0OThiMzQyYjY1ZGNjM2RjOTdhNjA1MzAzN2UwMDQxYjhiNGRiOTAzYTQxNWEwMjYxMDE3YjU0MQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFuQWJvcnRlZFN0YWdlVGFrZXNCYWNrT25seVRoZURpcmVjdG9yaWVzSXRNYWRlCWUyMmIzYzkwY2FiNDcyNTc3YjhlYzgxM2ZlNzdlNTc0NTg5MzM1ZjZkMzRlM2JhZThhOGNjYjM2Zjk0MzMzYTMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBbkFic29sdXRlUGF0aFNheXNJdFdhc1Jlc29sdmVkQWdhaW5zdFRoZVJvb3QJM2JhYzIyN2UzNGExNDllNGY5MTAxOTczNWI3ZDAwMTYwZTRkYTczN2U0MTcxNmIxOTY1NGMxOTg3N2Y0OWVmNwpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFuQW1iaWd1b3VzUmVnZXhBZGRyZXNzSXNSZWZ1c2VkCTlmZDczODAxMWUzOTFiMzhiYmRjMWJmNDQ0ZmEwZTA1ZjEzMmYwOTA5MDllOTI1ZDFlY2FjMTZiNGE2Y2Y4YjQKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBbkVPRk11bHRpTGluZVJlcGxhY2VEb2VzTm90TmVlZEFMaW5lQWZ0ZXJFbmQJNzE2NmU2YjMzMTM0MTZjMjFhMWE5NjRkN2NiZWI0ZDcwMTNlZGE0ZWRlYWI4M2YyMTkyNjRmYmFjOTc4ZDYxMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFuRW5kUGF0dGVybk9ubHlBYm92ZVRoZVN0YXJ0SXNSZWZ1c2VkCTZjYmJhZmJlNmQ1Y2MzNzgxYjk3ZmQ2MDEwMjEzN2M5OGQyMTQ4MjdlNGJlYmYzMDMyMzdhZmQyOGIyY2Y2ZWYKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBbkV4cGVjdGVkUmVtb3ZhbERpZmZlcnNPbmx5SW5XaGl0ZXNwYWNlCWM3OTVmMTY0ZWEyMTc0ZTU3NzA0Yjg5YTdhZWYzNjdlNjRmZGM1YzVkMDFkZGNiZWM4OTg1MGU4OTJkNzgyMWUKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RBbkV4cGVjdGVkUmVtb3ZhbElzTm90Q2hlY2tlZEFnYWluc3RBblVuc2VlbkZpbGUJZjlhM2I4YmY5MDU0ZTc0MzM3NGYwMmZjZDcyZjBmZWIwMGFjMTA0ZTJjZjhiNWU3NzUxN2Q3NDcxZWE0Mjc5YQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEFuRXhwZWN0ZWRSZW1vdmFsT2ZUaGVXcm9uZ0xlbmd0aElzUmVqZWN0ZWQJNGNhNzA4Y2FmZDM4N2U2NDY3ZmQ4YjQ1MTVmZmVmNDBkZjI3YTNlMzNmOTA3OGI3N2Y4NTcyMjI2YzMwODFiYQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdENyZWF0ZU5lZWRzTm9Qcmlvck9ic2VydmF0aW9uCWVhODEyYzZlNTljY2JmZWY1NDRjMjgxOWZhOGFhNTJlZDJmZTI0Y2E0Y2EwODZhNDE3NjdlNTliYjkyY2U5YjAKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3REZWxldGVCb3VuZHNBcmVUcmltbWVkCTM5YTYwYTdmYjIxMTM3ZWU3ZjJiMjUyMjc2NTZhMGNhYmM2MDJjYjVlMGExN2JkM2U3YmM3YjRiMzYzNzdlM2QKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3REZWxldGVCb3VuZHNBcmVUcmltbWVkT25BUnVuZUJvdW5kYXJ5CTUzNmFmMjg1NDQ5ZmZlMWFjYjk1YWZmOThiZTEyMjVlYzE1ZjM2NjEzN2JhZWJkOTkyYTM1MGU5YTBjODQxY2MKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3REZWxldGVSZWNvcmRzSXRzQm91bmRzCTA1M2UwMzcxZDY0NmVhMWE2YjJlYjVlOWJjZTE2MzdlMTI4YTQ5NmY5MjVjMzAwMTg4ODI5MGNjMzhiMjZiMWUKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3REcnlSdW5Xcml0ZXNOb3RoaW5nQnV0UmVwb3J0c1RoZU91dGNvbWUJMDEyZjc4OTdkZjc4YTRkOGMwODA2YjIzN2ExNDYxMTg1MjRlZTZlZDNlYzI3MmFjYjRkZmQ0NGRiMTNkYjYzYQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEVPRkFkZHJlc3NpbmcJNWQxMjk3MDU3MzgzYWM3MTNlMzAwMWNiZDU3Y2YyZGY3ZWE3NjUxM2U2MDRmNDg2NzgzZGFlNjRlZTAzNzQ1Nwpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEVhcmxpZXJIdW5rc05ldmVyU2hpZnRMYXRlckFkZHJlc3NlcwlmZjgxY2JmODVmODJiYjJiY2I0MTBhY2MzYmQyNjE0YTFiMzJmYTFlMmQ3OTFlZjcyMWY0YzkwNGYxNzRhYzZhCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0RWNob1BhZENsYW1wc0F0RU9GCThjYzQ0ODkwNzRhYmIwNjA5YzdkMzgzZGVlYjQ0NTY5MGE1ZjY4Y2NiYjdkMjRjNmE2ZmQzYTc3MGMwZDBjOTAKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RFY2hvUGFkRGVmYXVsdFByaW50c05vTGluZXMJYWEwZDdjZWIzMDk3NWRlMDQ4MDQ2ZjhhYTA5ZDM4NzcxMjFlYjVlOTdlNTRmMGQ5MTg2ZDVmNzI0OTBlYzFhNgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEVjaG9QYWROUHJpbnRzTk51bWJlcmVkTGluZXMJNzE5NjgxM2RhNjhhODNiOWEwNDkyZjFiNzYzNGU0ZjhiZTRiMzM0MmE3Mjc1OGQwZTkzZmVjOTBhMmEzZWNiMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEZhaWxlZEZpbGVzU3RpbGxBcHBlYXJJblRoZVJlY2VpcHQJOTBhNTIxMmE5MTgwMDEyZDI0Y2Y5N2QxNzE5N2Y2YjM4OGE4YzZlYWI4NDRhMDRmOGJlMDFlNzIxNmZhMWVmNgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEZpbGVQZXJtaXNzaW9uc1N1cnZpdmUJMmUyM2RlYzEyNzAwODRkOWE3YzVhMDVjNGVmNzBkYjdiOWFhYjA3NzQyNjc1Y2U5MTIyYTk5YTAxZWM0MjViMQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdEZvcmNlQnlwYXNzZXNUaGVHdWFyZAlhODJmNjliYjg5ZmVlNzlkYjI4NTEwNWI4MDY3Y2ZjZWRlMzZlZjcyOTM2MWIzZjUxZGIzNTY2NWE4ZDJlNGUyCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0R29vZFNoYUFuZENvdW50c1Bhc3MJMGM2NWM3Mzc2ODEzOWVjYTBjNzlmMzVjYTA5YjA4ZGM0MjAyZjEwYzliM2Q2OTlhNTQ0ZDE5YjdhNmY4YTZkYQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdE1hdGNoaW5nTmV0c09taXRCYWxhbmNlCTJhNGJmZmQ2NmM1OGIzNGFlMDRiZjljYmQwOTQzZjM4ZjhiYjAzNDg3NmU0YTlhNjE4NDcyOGUzNThjZmViNGEKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RNdWx0aXBsZUZpbGVzSW5PbmVSdW4JZmVjNWRkOGU1MWM0ZjczZTg0MDgwZjQyNTQ3NTc3YTY5MWMxZGUxYzI4MDdlMzlhZGUyODdlMjUyNGYxZGExMgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdE5pbExlZGdlckRpc2FibGVzVGhlR3VhcmQJZGMwNjU5NWYwOWJmMjdmNmE3MTMxYTRmOTQzMzJmNmFkMTczNWNkMmE4M2Q4MmZlNGZlMTdlZDhmNzZkOWI5YQpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdE9uZUJhZEh1bmtBYm9ydHNFdmVyeXRoaW5nQW5kU2F5c1doaWNoCWNmMTQzOWEyOTQxMWY5MTkzYThlM2VkZjM4Mjg1YmMxM2UyNjgzMWNkOGIxYjlmY2YxYjVmYzkwZTlmYjdmNmUKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RPbmx5RGVsZXRlUmVjb3Jkc0JvdW5kcwliMDc0N2Q2YWQyYjA1YzM3MGM1NjcxYTkwOWRkNTM3ZDYxNzZiNWJhNDVlODYxNmUyZjg2Y2Q4Yjk4YTk2MzU0CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0T3ZlcmxhcHBpbmdIdW5rc0FyZVJlamVjdGVkCTU5NzA2MWI0YzdkYWZiNzgyY2ZiMjRmYzk4ZmZmOGQ5NDczMGM4OTM3ODIxNzlhNGY3NGZlMWFlYTk2Mzk2Y2EKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RQcmVjb25kaXRpb25zCWE0YjU0YzNhYjA0MGU4OTMxYTQxNWM1YjFhNmM4MzkzYmZmNWM4YjhiNGI1MGY3MDZjMzIxZDVhNGNiNGE1MjYKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RQcm9zZU9taXRzQmFsYW5jZQlkMDFhNTdlNDkyOGE1M2EyOWQyMGVmNWVhZmUwZWI5NWFiYmI4YzU2NjIzMzI5MmU0OWE2OWViMmYxYmZmZDk0CmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0U3RyaWN0QmFsYW5jZUlzT2ZmQnlEZWZhdWx0CTAwN2EyNGQ0MTc1Nzc5OWU4ODZkYTQ5NTJhN2IzMjcwMWRmOTZhNGFkYzg0ZTFkZTZmMzNmMWU4ODgzNWY2YjMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RTdHJpY3RCYWxhbmNlTGVhdmVzQmFsYW5jZWRNdWx0aUxpbmVBbmRQcm9zZUFsb25lCWM0YWE2ZmIyMTYyMTVjZjRkNjFlMzhkMTEyMTM0OGM4NjAxYzcyZTE3YWMwZTNjYmEzNTFkM2JjYmUwZDcyZmMKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RTdHJpY3RCYWxhbmNlUmVmdXNlc1RoZVdyYXBUYWlsU2lnbmF0dXJlCWY1MmI0NjlmODcyYmI2YzQxYWE2MGZiOGNlMDgzOWRjNTg1NmQ5ZDE5MjAwYmU2MmNlY2M3YWQzMTE2YTExNTUKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RTdHJpY3RXb3VsZFJlZnVzZUlzQ291bnRlZE9ubHlXaGVuVGhlRmxhZ0lzT2ZmCTQ3MDFhMDUyMmM1M2E1ZWI3NzQ4MDg3ODg0OTVkN2U5ZTM3Njg2MTVkYmQzOGZiZmRhYTY1NDE3YmNmMTQ4NzIKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RUaGVFbmRQYXR0ZXJuSXNUaGVGaXJzdE1hdGNoQXRPckFmdGVyVGhlU3RhcnQJOTY3M2I4MGMwNTVjNTk4ZTk2YzYwNTBhNzYxMzJiNzdhODFiNWNiMDYxZDQ3MWJmOTVkMDk3MTAyZTE2M2E4Ngpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdFRoZUVuZ2luZVJlZnVzZXNBQm9keUxlc3NDcmVhdGUJY2NjNWY2NjA0MzRkMzc0ZTk1YzBjZDMxMmZjOGU1NzkwOTU0ZWM2NDA5NjE4NzI0NDI3Yzc0ZjE2NWEzMGE2Mgpib2R5CWludGVybmFsL2FwcGx5L2FwcGx5X3Rlc3QuZ28JVGVzdFRoZUVuZ2luZVJlZnVzZXNBUmVsYXRpdmVFbmRUaGVPcENhbm5vdEhvbm91cgkyMDNmMWYxOTJmMjI3MzJiYWYxZTc1MzE4ZmY0NzQ5OGE4NzYwYmQ4ZGEzNjlkNGM3NDY1YjE3ZTUyYjBmNmQzCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlUZXN0VGhlRW5naW5lUmVmdXNlc0V2ZXJ5U2hhcGVUaGVQYXJzZXJSZWZ1c2VzCTM3NjcxOWE4ZWIwNWM4Njc3ZWFlYWRiYjFmZGM5NzUyNWE5YmI2NDNhYzIwYmYzYmVlNGE3Yjk4OTM2ZjNlZGQKYm9keQlpbnRlcm5hbC9hcHBseS9hcHBseV90ZXN0LmdvCVRlc3RUcmFpbGluZ05ld2xpbmVJc1ByZXNlcnZlZAkxZTY3NmNmY2IzOTA2OTFmYTZiMmQyYTJjNjg1MWZhNjUwMjk5M2UxZWMxNjc1MWFmY2UxYTQ0OGQ0ZjMwMWEwCmJvZHkJaW50ZXJuYWwvYXBwbHkvYXBwbHlfdGVzdC5nbwlzeW1saW5rIGFsaWFzCWVmOTgwYmIwYTc0ZmFmN2VhNDU3MjlkYTk2ODU0NDQ4NWNlZjQxZjFkNzc3YzFhNGJiYTE2MzcxM2U1NzFmMzAKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFGYWlsZWRDb250ZW50Q29tbWl0UmVwb3J0c1dyaXR0ZW5IdW5rc09rQW5kVGhlUmVzdFNraXBwZWQJMDBiNTBiZjIxMzZiMjIxYTU1ZWFmZWQwNTBmMjVlZDE4NThlMzdiZjMwZDA0ZjlmODhkN2RkNWNmNDMyNTM2ZApib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QUZhaWxlZFBhdGhPcENvbW1pdFJlcG9ydHNOb3RoaW5nV3JpdHRlbkFmdGVyVGhlVW5kbwkzY2VhZjNhZDU0MGIyZTg3Mzc5YjNhNTViNDUwNDYyMjQzMzBiMWNlYzRmMzM2Y2ViNWI5ZTc5NTlkZDFlOGZkCmJvZHkJaW50ZXJuYWwvYXBwbHkvcGF0aG9wX2NvbW1pdF90ZXN0LmdvCVRlc3RBRmFpbGVkUmVuYW1lQWZ0ZXJBUmVwbGFjaW5nUmVuYW1lTG9zZXNOb0ZpbGUJNGUxZWFkYmY0MGU1YjNmMGQxYjYwY2ZiNDk3NjMzMzYwYjJiYjgwOTg4MTEyMjZlNzJmYWYwOGJlMjQ5MDc4NApib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QVJlbmFtZVdob3NlRGVzdGluYXRpb25EaXJlY3RvcnlDYW5ub3RCZUNyZWF0ZWRXcml0ZXNOb3RoaW5nCWEyNzc3YmEyNWFiMzcwODY0MTEyMzg5ZTQ2MTA2ZjM5YjJhNzc4ZmY5ODA0MWRhZGY4YjBlMmU0ZmZjOGU0ZTgKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFSZW5hbWVXaG9zZURlc3RpbmF0aW9uTmFtZUlzUmVqZWN0ZWRGYWlsc1ZhbGlkYXRpb24JMTIzYWFlMzk0ODE4OGZiOTQzMjkxNmMxZTI4YTc0OWI4MjY4OWI4NTE0MTc1MTgzZmVjYmM5M2JhOTliOTNhMgpib2R5CWludGVybmFsL2FwcGx5L3BhdGhvcF9jb21taXRfdGVzdC5nbwlUZXN0QVN0YWdlZFJlbmFtZURpcmVjdG9yeUlzVGFrZW5CYWNrV2hlbkFMYXRlclJlbmFtZUNhbm5vdFN0YWdlCTE4MDgwNmQzZDYwNDk2ZjZlYzU3ODJhMWU1YWNiYmJlNGY2NDIyYjI0ZTFjZDExYjk0YzFhMTBmZjcwZDY5ZTYKYm9keQlpbnRlcm5hbC9hcHBseS9wYXRob3BfY29tbWl0X3Rlc3QuZ28JVGVzdEFuVW5kb1RoYXRGYWlsc0xvc2VzTm9GaWxlCTI3OTU4YjVlNTk0MGRhNWNjYTM4NTAzNDFiNzljNTYzMjM5NGI0YzQ4YTU3YTgxODA1YWNkNmE1NTk3YmRiMDEKYm9keQlpbnRlcm5hbC9tY3AvcGFydGlhbF9yZWNlaXB0X3Rlc3QuZ28JVGVzdFRoZU1DUFJlY2VpcHRPZkFQYXJ0aWFsQ29tbWl0TmFtZXNFYWNoSHVuawk0OWQ2OTAwYWU3ZDIwOWVkNzNlNzRhNjk2ZTJjZjBjMzc3ZmJiYWMyZmM5MzI4ZDliYjdmMDY3OWNiNmJiZDQx
  ```
  --- last 10 line(s) of stdout (of 31 after folding 31 raw)
          wrote a.go  0L -> 0L  sha 
          2 hunk(s), 2 file(s), 1 failed, 0 advisories — NOTHING WRITTEN
  --- FAIL: TestAPartialCommitIsNotSummarisedAsApplied (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.010s
  === RUN   TestTheMCPReceiptOfAPartialCommitNamesEachHunk
  --- PASS: TestTheMCPReceiptOfAPartialCommitNamesEachHunk (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.010s
  FAIL
  ```
- 2026-09-24 · c3f5661* · exit 0 · `set -o pipefail …` · acceptance-sha256:c6d532c2df0edf0db5f1b38350bd209190cb16b701167b5710a1c982a6e3ee0e · ms:33211
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · ms:30573
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · ms:30215
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · ms:30431
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · ms:29944
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · ms:29393
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · ms:31804
- 2026-09-24 · b7df6e1* · exit 0 · `set -o pipefail …` · acceptance-sha256:a35f00f2d899289b4fd643e3d18fd61cbc2ba0cab56fc1d6930df3e941d94369 · ms:37513
- 2026-09-24 · a87ba4d* · exit 0 · `set -o pipefail …` · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · ms:29682
- 2026-09-24 · a87ba4d* · exit 0 · `set -o pipefail …` · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · ms:29113
- 2026-09-24 · a87ba4d* · exit 0 · `set -o pipefail …` · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · ms:29274
- 2026-09-24 · a87ba4d* · exit 0 · `set -o pipefail …` · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · ms:29496
- 2026-09-24 · a87ba4d* · exit 0 · `set -o pipefail …` · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · ms:29230
- 2026-09-24 · a87ba4d* · exit 0 · `set -o pipefail …` · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · ms:29135
- 2026-09-24 · a87ba4d* · exit 0 · `set -o pipefail …` · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · ms:29386
- 2026-09-24 · a87ba4d* · exit 0 · `set -o pipefail …` · acceptance-sha256:30855c209e9322ca88aaadbbaaa45f88fbfc81c5fa0a9cc2b6bc84a475722717 · ms:29061
