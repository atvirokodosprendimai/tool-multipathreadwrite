# Safety evidence for the write path

Checked 2026-10-26 against source at `0ec1b97f`. This is a bounded map of five
write-path guarantees, their existing evidence and their limits. It adds no
contract and is not a complete security or crash-recovery audit. The ADRs remain
the decision authority; [the README failure matrix](../../README.md#what-a-failure-leaves-on-disk)
describes recovery.

## Production path

Both `cmd/mrw.writeCmd` and `internal/mcp.writeTool` call
[`writer.Prepare`, `Prepared.Land`, `Landed.Verify` and `Landed.Settle`](../../internal/writer/flow.go).
`Land` snapshots the read ledger before
[`applyCounted`](../../internal/writer/writer.go) acquires the checkout's write lock.
Under that lock it calls `apply.Apply` and records the files that actually landed.
The project check runs after the lock is released.

## Guarantee → evidence → limit

Contract section numbers below refer to [`scripts/contract.sh`](../../scripts/contract.sh),
which drives the built binary. Reading a row is not evidence that it passed in
this session.

| Guarantee and authority | Production enforcement | Focused tests | Binary contract rows | Limit |
|---|---|---|---|---|
| A validation failure writes nothing; valid siblings say `skip` (ADR-001). | [`apply.apply`](../../internal/apply/apply.go) returns on `res.Failed > 0` before staging, clearing siblings' write details. | `TestOneBadHunkAbortsEverythingAndSaysWhich` in [apply_test.go](../../internal/apply/apply_test.go). | §1 rejects one bad anchor among valid siblings; §2 applies a valid multi-file plan. | Validation atomicity does not make the later filesystem commit crash-atomic. |
| An unread range or a stale observed version is refused (ADR-002, ADR-005). | `Land` supplies `opts.Seen`; [`apply.planFile`](../../internal/apply/apply.go) checks the recorded SHA and covered lines. | `TestARangedReadLicensesOnlyTheLinesItServed`, `TestAWholeFileReadLicensesTheWholeFile` in [ledger_test.go](../../internal/adversarial/ledger_test.go); `TestAFileChangedBehindMrwsBackCannotBeEdited` in [apply_test.go](../../internal/apply/apply_test.go). | §6 covers unseen/stale files; §9 pairs unread-range refusal with a ranged re-read that licenses the edit. | The checkout's ledger records service, not comprehension or a separate observation for each agent. CLI `--force` bypasses the ordinary Seen guard; `from=` source licensing has its own rule (ADR-167). |
| mrw writers take turns, validating against the snapshot taken before waiting (ADR-075). | [`applyCounted`](../../internal/writer/writer.go) holds `seen.LockWrites` across apply and ledger updates; `Land` loads the snapshot before calling it. | `TestNoTwoWritersAreInsideAtOnce`, `TestAWriterWhoseFileChangedWhileItWaitedIsRefused`, `TestWhatLandedIsRecordedBeforeTheLockIsReleased` in [writer_test.go](../../internal/writer/writer_test.go). | §150 checks eight CLI writers: every successful writer's edit survives; stale refusals name their cause. | The lock coordinates mrw writers, not arbitrary editors. It does not cover the project check; the shared ledger is not a caller-specific version pin. Use `sha=` when callers share a checkout. |
| A commit failure reports and records what actually landed (ADR-066, ADR-102). | `apply.apply` stages content before committing, reports content renames already completed, and invokes `commitPathOps`; `applyCounted` records written files even for a partial result and retains a commit error if the ledger also fails. | `TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped`, `TestAFailedPathOpCommitReportsNothingWrittenAfterTheUndo` in [pathop_commit_test.go](../../internal/apply/pathop_commit_test.go); `TestAPartialCommitRecordsWhatLanded`, `TestAPartialCommitWhoseLedgerFailsKeepsItsCommitError` in [partial102_test.go](../../internal/writer/partial102_test.go). | §119 pairs a replacing rename with a refusal now caught at staging; actual commit undo is exercised by the unit seam above. §196 pairs a partial commit with a successful retry of the remaining unlink; §186 covers a landed write whose ledger cannot be saved. | Permission fixtures skip on Windows or when privilege ignores the permission bits. A killed process may produce no receipt. A full landing followed by ledger failure remains a changed tree, exit 2. |
| A check's verdict follows the process, not printed success text (ADR-003, ADR-054). | [`check.Run`](../../internal/check/check.go) records the process status; `Landed.Verify` runs it after landing. The surfaces map a failed check to exit 3 / the MCP check result. | `TestRunReportsTheRealExitCode`, `TestRunPasses` in [check_test.go](../../internal/check/check_test.go). | §5 prints PASS but exits 1; §2 passes a real check; §3 retains the write after a failed check. | A passing command proves only what that command exercises. The write is not rolled back, and another writer can land during the check (ADR-112, ADR-127). |

The content identity checks before commit narrow external-editor races; the README
also names the remaining window after the last check, same-size/same-mtime rewrites,
and the different treatment of unlink/rename sources. Process death and power loss
remain the documented limitations in its failure matrix. No crash or power-loss
experiment was run for this note.

## Focused verification

On 2026-10-26 all 13 selected tests below ran and passed, with no skips, against the
source revision above on macOS. The command used `GOMAXPROCS=2`, `-p 1`, `-count=1`
and a 120-second outer timeout. This was a focused run, not the whole suite,
race detector, Windows CI or a run of `contract.sh`.

Run these from the repository root; use an outer timeout appropriate to the host:

```sh
go test ./internal/apply -count=1 -v -run '^(TestOneBadHunkAbortsEverythingAndSaysWhich|TestAFileChangedBehindMrwsBackCannotBeEdited|TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped|TestAFailedPathOpCommitReportsNothingWrittenAfterTheUndo)$'
go test ./internal/writer -count=1 -v -run '^(TestNoTwoWritersAreInsideAtOnce|TestAWriterWhoseFileChangedWhileItWaitedIsRefused|TestWhatLandedIsRecordedBeforeTheLockIsReleased|TestAPartialCommitRecordsWhatLanded|TestAPartialCommitWhoseLedgerFailsKeepsItsCommitError)$'
go test ./internal/adversarial -count=1 -v -run '^(TestARangedReadLicensesOnlyTheLinesItServed|TestAWholeFileReadLicensesTheWholeFile)$'
go test ./internal/check -count=1 -v -run '^(TestRunReportsTheRealExitCode|TestRunPasses)$'
```

The graph's coverage metadata matched the inspected paths with no recorded gap;
that is a best-effort discovery signal. Source and tests were also read directly.

## What this changes next

These scenarios already have production enforcement and focused tests; this pass
found no new reproduced engine failure among them. No engine extraction is selected
on file length alone. A future change should identify its concrete failure or the
responsibility it can separate while preserving these outcomes.

Whether the safeguards reduce mistakes in ordinary agent sessions remains a
measurement question. The [batching pilot proposal](batching-pilot.md) separates a
batching cue from the mrw treatment; it does not isolate individual guards.
