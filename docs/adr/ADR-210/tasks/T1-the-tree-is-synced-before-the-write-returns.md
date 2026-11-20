# Task ADR-210-T1: the tree is synced before the write returns

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the file and directory syncs in `internal/apply`, behind seams; `writer.SyncError` and its rendering on the CLI and `mrw_write`; contract §361; Decision 6's words; the end-to-end cost in the record
**Consumes:** `stageFile`, the commit loop, `commitPathOps`, `(*tree)`, `writer.LedgerError`'s handling in `Land`, `appliedButUnreportable`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `every temp is synced before the first rename`, `every changed directory is synced once after the last change`, `the drain runs once per device per phase on macOS`, `a Windows directory is synced like any other and a flush the volume does not support is a note`, `a sync failure before the renames writes nothing`, `a sync failure after the renames is a landed write with an error`, `a sync failure and a ledger failure keep both errors`, `the check and steps do not run after a sync failure`, `a dry run syncs nothing`, `every subtest ran and none skipped`, `the CLI and mrw_write report it`, `the contract row exists with its trace`, `the new words are in the documents`, `the engine packages are untouched, tracked and untracked`

## Goal

ADR-210 Decisions 1 to 6.

**`TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns`** (`internal/apply`), one subtest per row, the seams recording every sync, drain, rename, removal and directory creation in order:

| Subtest | Plan and seam | Outcome |
|---------|---------------|---------|
| `edit_create_rename` (every platform) | edit `a.txt`; create `n/m/k.txt` in new directories; rename `s/b.txt` to `t/b.txt` (both parents exist and differ) | every temp's sync before the first rename; on macOS one drain after the last temp sync and before the first rename; after the last change one sync each for `.`, `n`, `n/m`, `s` and `t`, then on macOS one drain; nothing synced after `Apply` returns |
| `two_devices` (macOS) | edit a file on each of two devices, the second faked through the seam's device number | one drain per device in each phase |
| `create_under_existing` (every platform) | create `s/new/k.txt` where `s` exists | `s` and `new` synced and the root not; each temp is synced at the size of its content |
| `rename_only` (every platform) | rename `s/b.txt` to `t/b.txt` | no file sync; `s` and `t` synced |
| `unlink_only` (every platform) | unlink `s/b.txt` | `s` synced |
| `dir_no_barrier` (every platform) | the flush of `n` answers `ErrNoBarrier` (a Windows directory over SMB) | a clean landing with `sync_note`; every other directory still synced; exit 0 |
| `drain_no_barrier` (macOS) | the first drain answers `ErrNoBarrier` (`F_FULLFSYNC` not supported, the fallback `fsync` fine) | a clean landing with `sync_note` |
| `temp_sync_fails` (every platform) | the second temp's sync fails | an error; nothing renamed; every temp removed |
| `drain_before_fails` (macOS) | the first drain fails | as `temp_sync_fails` |
| `dir_sync_fails` (Linux and macOS) | the sync of `n` fails | `Applied` true; the error is a sync error naming `n`; every other directory is still synced and drained, and every file is in place |
| `drain_after_fails` (macOS) | the second drain fails | as `dir_sync_fails` |
| `dry_run` (every platform) | `DryRun` | no sync recorded |

A row for another platform is not registered at all (a build tag or a `runtime.GOOS` test before `t.Run`), never registered and skipped, so the fence can require every row its platform names and refuse any skip.

**`TestTheCLIReportsASyncThatFailedAfterLandingAsAWriteThatLanded`** (`cmd/mrw`, through the writer with the seam): exit 2; `--json` has `applied: true` and `error` naming the directory and saying the files are written but not known to be on stable storage; the ledger records the file; with `--check` and `--then`, no check runs and the steps are reported not run; `mrw stats` counts `check_not_run`; with the ledger also failing, `error` names both.

**`TestMrwWriteReportsASyncThatFailedAfterLanding`** (`internal/mcp`): `isError` with the landed receipt; under a result ceiling too small for the receipt, the fallback sentence carries the same words.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the file sync in `stageFile`, the drain, the directory syncs after the last change, the sync error |
| `internal/apply/sync210.go` | new | the seams, `SyncError`, the directory set, the drain per device |
| `internal/apply/sync_darwin.go`, `sync_other.go`, `sync_windows.go`, `sync_darwin_test.go` | new | the per-platform file sync, drain and directory sync; the EINTR retry on macOS |
| `internal/writer/writer.go`, `internal/writer/flow.go` | edit | `SyncError`, classified as a landed write as `LedgerError` is, both errors kept |
| `internal/mcp/tools.go`, `internal/mcp/schema.go` | edit | `isError` with the landed receipt; the fallback sentence; the `error` key's description |
| `internal/curve/score.go`, `.github/workflows/ci.yml` | edit | a sync failure after landing scores as applied; `strace` for §361 in CI |
| `internal/apply/sync210_test.go`, `cmd/mrw/sync210_test.go`, `internal/mcp/sync210_test.go`, `internal/curve/sync210_test.go` | new | the three tests, and the scorer's |
| `scripts/contract.sh` | edit | §361 |
| `README.md`, `AGENTS.md` | edit | Decision 6 |

## Ordered Steps

1. [S1] The three tests; RED on every subtest but `dry_run`, and on the CLI and MCP tests. [proof: acceptance]
2. [S2] The syncs. Mutants, each killed: no temp sync (`edit_create_rename`); a temp synced after the first rename (`edit_create_rename`, by order); no directory sync (`edit_create_rename`); a directory synced once per rename (`edit_create_rename`, by count); the drain dropped on macOS (`edit_create_rename`); one drain for all devices (`two_devices`); a flush answered not-supported counted as a failure (`dir_no_barrier`); a failed temp sync ignored (`temp_sync_fails`); a failed directory sync ignored (`dir_sync_fails`); the sync error classified as `refused_apply` (the CLI test's stats); the check run after a sync failure (the CLI test); the ledger's error dropping the sync's (the CLI test); the dry run syncing (`dry_run`). [proof: mutation]
3. [S3] Contract §361, as the record's Wiring table says: `strace -f -y -e trace=fsync,fdatasync,rename,renameat,renameat2` over `$MRW write` of a plan editing two files in two directories, decoding each descriptor's path (`-y`) and taking only calls that returned 0: every temp's `fsync` precedes the first tree rename; each changed directory's `fsync` follows the last tree change; the ledger's rename follows those. Paired with `--dry-run` of the same plan on a fresh fixture, which must exit 0 and show no `fsync` on a tree path. Where `strace` is absent the row says `skip` and why; the CI job that proves it installs `strace`. The words. [proof: acceptance]
4. [S4] The cost: ADR-105 T3's protocol (a 500-file, 5,000-hunk write and a one-file write, 5 alternating runs each, the old build against the new, each on a fresh copy with its own `XDG_STATE_HOME`) on the owner's Mac and the Linux and Windows CI runners; the medians, the load and the filesystem go into the record's Consequences, for Zy's decision under Decision 7. [proof: human: the medians, the load, the filesystem and the machine of each run, recorded in the record]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./internal/apply ./cmd/mrw ./internal/mcp -run '^(TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns|TestTheCLIReportsASyncThatFailedAfterLandingAsAWriteThatLanded|TestMrwWriteReportsASyncThatFailedAfterLanding)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 3 ] \
  && out2="$(mktemp)" \
  && go test -count=1 -v ./internal/apply ./internal/curve ./internal/mcp -run '^(TestAPlanWhoseSyncFailedAfterLandingIsScoredAsApplied|TestAStagingSyncFailureIsTheHarnessesNotARefusal|TestTheDrainDoesNotReopenAStagedTempForReading|TestADirectorySwappedAfterTheRenamesIsNotSynced|TestARetargetedLinkDoesNotMoveTheDirectorySync|TestAnInterruptedFsyncIsRetried|TestAnUnsupportedDrainFallsBackToFsyncAndSaysSo|TestMrwWriteReportsAVolumeWithNoBarrierAsANote|TestAWindowsDirectoryFlushThatIsNotSupportedIsNotAFailure|TestAWindowsDirectoryIsOpenedWithWriteAccessForTheFlush|TestAWindowsDirectoryPathOver260CharactersIsOpenedInExtendedLengthForm|TestAWindowsDirectoryOpenedByPathMustBeTheRecordedOne|TestADirectoryWithoutARecordedIdentityIsNotFlushed)$' > "$out2" \
  && grep -q '^--- PASS: TestAPlanWhoseSyncFailedAfterLandingIsScoredAsApplied ' "$out2" \
  && grep -q '^--- PASS: TestAStagingSyncFailureIsTheHarnessesNotARefusal ' "$out2" \
  && grep -q '^--- PASS: TestTheDrainDoesNotReopenAStagedTempForReading ' "$out2" \
  && { [ "$(go env GOOS)" = windows ] || { grep -q '^--- PASS: TestADirectorySwappedAfterTheRenamesIsNotSynced ' "$out2" && grep -q '^--- PASS: TestARetargetedLinkDoesNotMoveTheDirectorySync ' "$out2"; }; } \
  && grep -q '^--- PASS: TestMrwWriteReportsAVolumeWithNoBarrierAsANote ' "$out2" \
  && grep -q '^--- PASS: TestADirectoryWithoutARecordedIdentityIsNotFlushed ' "$out2" \
  && grep -q '^    --- PASS: TestTheCLIReportsASyncThatFailedAfterLandingAsAWriteThatLanded/no_barrier_proceeds_with_a_note ' "$out" \
  && grep -q '^    --- PASS: TestTheCLIReportsASyncThatFailedAfterLandingAsAWriteThatLanded/no_barrier_is_said_once_in_the_human_form ' "$out" \
  && { [ "$(go env GOOS)" != windows ] || { grep -q '^--- PASS: TestAWindowsDirectoryPathOver260CharactersIsOpenedInExtendedLengthForm ' "$out2" && grep -q '^--- PASS: TestAWindowsDirectoryOpenedByPathMustBeTheRecordedOne ' "$out2"; }; } \
  && { [ "$(go env GOOS)" != windows ] || { grep -q '^--- PASS: TestAWindowsDirectoryFlushThatIsNotSupportedIsNotAFailure ' "$out2" && grep -q '^--- PASS: TestAWindowsDirectoryIsOpenedWithWriteAccessForTheFlush ' "$out2"; }; } \
  && { [ "$(go env GOOS)" != darwin ] || { grep -q '^--- PASS: TestAnInterruptedFsyncIsRetried ' "$out2" && grep -q '^--- PASS: TestAnUnsupportedDrainFallsBackToFsyncAndSaysSo ' "$out2"; }; } \
  && case "$(go env GOOS)" in darwin) want='edit_create_rename create_under_existing rename_only unlink_only temp_sync_fails dry_run dir_sync_fails dir_no_barrier two_devices two_devices_first_drain_fails second_device_drain_fails drain_before_fails drain_after_fails drain_no_barrier no_barrier_then_eio' ;; *) want='edit_create_rename create_under_existing rename_only unlink_only temp_sync_fails dry_run dir_sync_fails dir_no_barrier' ;; esac \
  && missing=0 && for c in $want; do grep -q "^    --- PASS: TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns/$c " "$out" || missing=1; done && [ "$missing" -eq 0 ] \
  && ! grep -q -- '--- SKIP' "$out" \
  && grep -q '^# 361\. ' scripts/contract.sh \
  && s361="$(sed -n '/^# 361\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh)" \
  && grep -qF 'strace -f -y' <<<"$s361" \
  && grep -qF -- '--dry-run' <<<"$s361" \
  && grep -q 'not known to be on stable storage' internal/writer/writer.go \
  && readme="$(tr -s '\n ' '  ' < README.md)" \
  && grep -qF 'synced before the ledger' <<<"$readme" \
  && grep -qF 'is said so in `sync_note`' <<<"$readme" \
  && grep -qF 'the files are written but not known to be on stable storage: do not send the plan again' <<<"$readme" \
  && grep -q 'the files could not be synced' <(tr -s '\n ' '  ' < AGENTS.md) \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state)" ]
```

The cases each platform must pass are named in the fence: eight on Linux and on Windows, fourteen on macOS. The fence runs where the test runs; CI runs it on Linux and Windows, and macOS's rows are proved on the owner's Mac.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns` | `internal/apply/sync210_test.go` | the first table, by the order and count of the recorded calls, the error's class and the files on disk | none | S1, S2 |
| `TestTheCLIReportsASyncThatFailedAfterLandingAsAWriteThatLanded` | `cmd/mrw/sync210_test.go` | exit 2, the `--json` receipt, the ledger, no check, steps not run, the tally, both errors kept | none | S1, S2 |
| `TestMrwWriteReportsASyncThatFailedAfterLanding` | `internal/mcp/sync210_test.go` | `isError` with the landed receipt, and the fallback's words | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §361 |
| 2 — something selects it | every write that lands, on both surfaces |
| 3 — the caller can discover it | the README's failure table; `AGENTS.md`; the after-landing error's own words |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is the power-loss row the README already prints |

## Mutation Log
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/apply/apply.go` · no temp sync · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:every temp is synced before the first rename
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/apply/sync210.go` · no directory sync · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:every changed directory is synced once after the last change
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/apply/sync210.go` · a directory synced twice (once per rename) · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:every changed directory is synced once after the last change
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the drain dropped on macOS · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/apply/sync210.go` · one drain for all devices · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/apply/apply.go` · a failed temp sync ignored · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:a sync failure before the renames writes nothing
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/apply/sync210.go` · a failed directory sync ignored · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/writer/flow.go` · the sync error classified as refused_apply · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the check and steps do not run after a sync failure
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `cmd/mrw/main.go` · the check run after a sync failure · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the check and steps do not run after a sync failure
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/writer/writer.go` · the ledger's error dropping the sync's · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:a sync failure and a ledger failure keep both errors
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/apply/apply.go` · the dry run syncing · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:a dry run syncs nothing
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `internal/mcp/tools.go` · the terminal sentence drops the sync words · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the CLI and mrw_write report it
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `README.md` · the README claims Windows syncs directories · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the new words are in the documents
- 2026-11-20 · 4c053211* · mutant survived · exit 0 · `scripts/contract.sh` · the contract row drops -y · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the contract row exists with its trace
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-20 · 4c053211* · mutant survived · exit 0 · `scripts/contract.sh` · the contract row drops -y (the row's one strace command) · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the contract row exists with its trace
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-20 · 4c053211* · mutant killed · exit 1 · `scripts/contract.sh` · the row's strace command drops -y, which it must carry (second attempt: the comment no longer repeats the literal) · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · covers:the contract row exists with its trace
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/apply.go` · no temp sync · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:every temp is synced before the first rename
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync210.go` · no directory sync · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:every changed directory is synced once after the last change
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync210.go` · a directory synced twice (once per rename) · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:every changed directory is synced once after the last change
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the drain dropped on macOS · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync210.go` · one drain for all devices · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/apply.go` · a failed temp sync ignored · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:a sync failure before the renames writes nothing
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync210.go` · a failed directory sync ignored · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/writer/flow.go` · the sync error classified as refused_apply · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:the check and steps do not run after a sync failure
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `cmd/mrw/main.go` · the check run after a sync failure · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:the check and steps do not run after a sync failure
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/writer/writer.go` · the ledger's error dropping the sync's · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:a sync failure and a ledger failure keep both errors
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/apply.go` · the dry run syncing · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:a dry run syncs nothing
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/mcp/tools.go` · the terminal sentence drops the sync words · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:the CLI and mrw_write report it
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `README.md` · the README claims Windows syncs directories · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:the new words are in the documents
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `scripts/contract.sh` · the row's strace command drops -y · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:the contract row exists with its trace
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/apply.go` · a temp synced before its bytes are written (review P2-1, mutant early) · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:every temp is synced before the first rename
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync210.go` · a created directory's parent not synced (review P2-2, m3) · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:every changed directory is synced once after the last change
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync210.go` · syncChanged stops at the first failing directory (review P3) · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:every changed directory is synced once after the last change
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · an interrupted fsync is not retried (review P3) · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:every temp is synced before the first rename
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/curve/score.go` · a landed sync failure scored as a refusal (review P3) · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · a79532f0* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · an interrupted fsync is not retried (review P3; errors.Is form) · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · covers:every temp is synced before the first rename
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the drain falls back to Go File.Sync, which swallows ENOTSUP (Codex P1) · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync210.go` · the drain reopens each temp for reading (Codex P2, mode 0244) · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:every temp is synced before the first rename
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync210.go` · the drain stops at the first failing device (Codex P2) · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/curve/score.go` · a staging sync failure scored as a refusal (Codex P2) · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:a sync failure before the renames writes nothing
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync210.go` · one directory under two spellings synced twice (Codex P3) · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:every changed directory is synced once after the last change
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/apply.go` · no temp sync · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:every temp is synced before the first rename
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync210.go` · no directory sync · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:every changed directory is synced once after the last change
- 2026-11-20 · f2d91c78* · mutant survived · exit 0 · `internal/apply/sync210.go` · a directory synced twice (once per rename) · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:every changed directory is synced once after the last change
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the drain dropped on macOS · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync210.go` · one drain for all devices · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/apply.go` · a failed temp sync ignored · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:a sync failure before the renames writes nothing
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync210.go` · a failed directory sync ignored · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/writer/flow.go` · the sync error classified as refused_apply · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the check and steps do not run after a sync failure
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `cmd/mrw/main.go` · the check run after a sync failure · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the check and steps do not run after a sync failure
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/writer/writer.go` · the ledger's error dropping the sync's · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:a sync failure and a ledger failure keep both errors
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/apply.go` · the dry run syncing · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:a dry run syncs nothing
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/mcp/tools.go` · the terminal sentence drops the sync words · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the CLI and mrw_write report it
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `README.md` · the README claims Windows syncs directories · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the new words are in the documents
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `scripts/contract.sh` · the row's strace command drops -y · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:the contract row exists with its trace
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync210.go` · a created directory's parent not synced · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:every changed directory is synced once after the last change
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync210.go` · syncChanged stops at the first failing directory · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:every changed directory is synced once after the last change
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · an interrupted fsync is not retried · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:every temp is synced before the first rename
- 2026-11-20 · f2d91c78* · mutant killed · exit 1 · `internal/curve/score.go` · a landed sync failure scored as a refusal · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · a directory opened blocking: a FIFO swapped in hangs the sync (final P2-1) · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every changed directory is synced once after the last change
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · the recorded identity is not checked (final P2-1) · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every changed directory is synced once after the last change
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · the source directory is resolved again at sync time instead of recorded (final P2-2) · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every changed directory is synced once after the last change
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/apply.go` · a later device drain failure is attributed to the first file (final P3) · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:a sync failure before the renames writes nothing
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · one directory under two spellings recorded twice · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every changed directory is synced once after the last change
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/apply.go` · no temp sync · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every temp is synced before the first rename
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · no directory sync · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every changed directory is synced once after the last change
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the drain dropped on macOS · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · one drain for all devices · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/apply.go` · a failed temp sync ignored · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:a sync failure before the renames writes nothing
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · a failed directory sync ignored · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/writer/flow.go` · the sync error classified as refused_apply · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the check and steps do not run after a sync failure
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `cmd/mrw/main.go` · the check run after a sync failure · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the check and steps do not run after a sync failure
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/writer/writer.go` · the ledger's error dropping the sync's · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:a sync failure and a ledger failure keep both errors
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/apply.go` · the dry run syncing · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:a dry run syncs nothing
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/mcp/tools.go` · the terminal sentence drops the sync words · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the CLI and mrw_write report it
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `README.md` · the README claims Windows syncs directories · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the new words are in the documents
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `scripts/contract.sh` · the row's strace command drops -y · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the contract row exists with its trace
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · a created directory's parent not synced · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every changed directory is synced once after the last change
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · an interrupted fsync is not retried · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every temp is synced before the first rename
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/curve/score.go` · a landed sync failure scored as a refusal · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/curve/score.go` · a staging sync failure scored as a refusal · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:a sync failure before the renames writes nothing
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the drain falls back to Go File.Sync · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · the drain reopens each temp for reading · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every temp is synced before the first rename
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · the drain stops at the first failing device · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 2f3f930c* · mutant killed · exit 1 · `internal/apply/sync210.go` · syncChanged stops at the first failing directory · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · covers:every changed directory is synced once after the last change
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · an unsupported drain is an error again, not a fallback (Zy 2026-11-20) · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the fallback fsync failing is ignored · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · a drain with no barrier is a failure again · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · a directory flush the volume cannot do is a failure again · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:a Windows directory is synced like any other and a flush the volume does not support is a note
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/apply.go` · the receipt does not say the volume gives no barrier · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the CLI and mrw_write report it
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `cmd/mrw/main.go` · the human form does not say it on stderr · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the CLI and mrw_write report it
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/apply.go` · no temp sync · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every temp is synced before the first rename
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · no directory sync · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every changed directory is synced once after the last change
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the drain dropped on macOS · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · one drain for all devices · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/apply.go` · a failed temp sync ignored · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:a sync failure before the renames writes nothing
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · a failed directory sync ignored · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/writer/flow.go` · the sync error classified as refused_apply · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the check and steps do not run after a sync failure
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `cmd/mrw/main.go` · the check run after a sync failure · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the check and steps do not run after a sync failure
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/writer/writer.go` · the ledger's error dropping the sync's · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:a sync failure and a ledger failure keep both errors
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/apply.go` · the dry run syncing · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:a dry run syncs nothing
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/mcp/tools.go` · the terminal sentence drops the sync words · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the CLI and mrw_write report it
- 2026-11-20 · 24beedfe* · mutant survived · exit 0 · `README.md` · the README claims every volume gives a barrier · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the new words are in the documents
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `scripts/contract.sh` · the row's strace command drops -y · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the contract row exists with its trace
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · a created directory's parent not synced · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every changed directory is synced once after the last change
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · an interrupted fsync is not retried · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every temp is synced before the first rename
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/curve/score.go` · a landed sync failure scored as a refusal · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:a sync failure after the renames is a landed write with an error
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/curve/score.go` · a staging sync failure scored as a refusal · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:a sync failure before the renames writes nothing
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync_darwin.go` · the drain uses Go File.Sync again · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · the drain reopens each temp for reading · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every temp is synced before the first rename
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · the drain stops at the first failing device · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the drain runs once per device per phase on macOS
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · a directory opened blocking · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every changed directory is synced once after the last change
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · the recorded identity is not checked · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every changed directory is synced once after the last change
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · the source directory is resolved again at sync time · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every changed directory is synced once after the last change
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · syncChanged stops at the first failing directory · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every changed directory is synced once after the last change
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `internal/apply/sync210.go` · one directory under two spellings recorded twice · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:every changed directory is synced once after the last change
- 2026-11-20 · 24beedfe* · mutant killed · exit 1 · `README.md` · the README does not say where a no-barrier volume is reported · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · covers:the new words are in the documents

## Verification Log
- 2026-11-20 · 91a611c9* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:1262 · test-lock-sha256:f9402ae33123588c1c5166b48985591e71f03f6d556a3b9a46caf07a78bd7268 · test-lock-file:docs/adr/locks/f9402ae33123588c1c5166b48985591e71f03f6d556a3b9a46caf07a78bd7268.lock
  ```
  ```
- 2026-11-20 · 4c053211 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:4235
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:5022
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3960
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3818
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3833
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3404
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3620
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3956
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:4066
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3295
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3619
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:4025
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3476
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:4096
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:4023
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:4860
- 2026-11-20 · 4c053211* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:3945
- 2026-11-20 · 5a6d7cc5* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:2184
- 2026-11-20 · human-observed · S4 measured by the implementer on the owner's Mac and in a Docker Desktop container on it; Windows not measured; no CI runner measured. Protocol: ADR-105 T3's (500 files of 100 lines in 20 directories, 10 single-line replaces per file, every file read first, `mrw write --no-check` timed alone, each run on a fresh copy with its own `XDG_STATE_HOME`), plus a one-file one-hunk write; main at `91a611c9` against this build, alternating (the order flips each run), medians; the timer is Python's perf_counter around the subprocess. macOS series A (Apple silicon, APFS, 10 cores; 1-minute load 7.08 before and 6.85 after, 5-minute 10.1 to 9.8, other sessions' tests running): one file, 21 runs: main 12.7 ms, T1 21.2 ms, +8.5 ms (+66.9 %); 500 files, 21 runs: main 324.8 ms, T1 364.9 ms, +40.0 ms (+12.3 %). Samples (s): old samples: 0.0157 0.0129 0.0124 0.0125 0.0130 0.0121 0.0122 0.0109 0.0110 0.0122 0.0127 0.0195 0.0156 0.0144 0.0138 0.0119 0.0150 0.0126 0.0285 0.0119 0.0127 | new samples: 0.0451 0.0203 0.0347 0.0273 0.0221 0.0191 0.0215 0.0200 0.0192 0.0211 0.0201 0.0212 0.0211 0.0231 0.0230 0.0195 0.0264 0.0210 0.0204 0.0248 0.0256 | old samples: 0.3177 0.3281 0.3289 0.3288 0.3545 0.3790 0.3103 0.2966 0.3888 0.2925 0.2957 0.3807 0.3818 0.4101 0.4084 0.3106 0.2995 0.3248 0.3145 0.2862 0.2790 | new samples: 0.3593 0.4080 0.3319 0.3312 0.4180 0.3680 0.3994 0.3394 0.3895 0.3383 0.3649 0.3838 0.3589 0.4087 0.3989 0.3501 0.3744 0.3228 0.3711 0.3360 0.3564. macOS series B (an hour earlier, 1-minute load 7.36 to 7.52): one file, 21 runs: main 13.1 ms, T1 20.0 ms (+53.1 %); 500 files, 9 runs: main 354.5 ms, T1 344.7 ms (-2.8 %, inside the spread of its samples). Samples (s): old samples: 0.0134 0.0169 0.0143 0.0131 0.0154 0.0126 0.0170 0.0110 0.0121 0.0109 0.0110 0.0146 0.0140 0.0119 0.0167 0.0110 0.0132 0.0129 0.0131 0.0120 0.0124 | new samples: 0.0197 0.0194 0.0234 0.0200 0.0200 0.0197 0.0220 0.0193 0.0191 0.0189 0.0190 0.0231 0.0278 0.0236 0.0232 0.0209 0.0216 0.0191 0.0204 0.0189 0.0200 | old samples: 0.2713 0.3872 0.3639 0.4795 0.4737 0.3302 0.3107 0.3545 0.2834 | new samples: 0.3480 0.3269 0.4936 0.4528 0.3304 0.3403 0.3447 0.3442 0.4104. Linux (a `rust:1-bookworm` container, linuxkit 7.0.14 aarch64, 10 CPUs visible, `--cpus 4 --memory 4g`, overlay filesystem on the VM disk, the work directory on that overlay; the host's load 6.70 at the start, the container's 0.04 to 0.52): one file, 21 runs: main 11.3 ms, T1 14.4 ms, +3.1 ms (+27.0 %); 500 files, 21 runs: main 142.2 ms, T1 704.1 ms, +561.9 ms (+395.3 %). Samples (s): old samples: 0.0227 0.0134 0.0139 0.0149 0.0094 0.0113 0.0160 0.0110 0.0071 0.0107 0.0119 0.0154 0.0116 0.0102 0.0100 0.0093 0.0095 0.0087 0.0102 0.0116 0.0120 | new samples: 0.0885 0.0176 0.0196 0.0157 0.0181 0.0174 0.0144 0.0124 0.0133 0.0162 0.0111 0.0327 0.0117 0.0143 0.0121 0.0144 0.0157 0.0130 0.0124 0.0138 0.0115 | old samples: 0.3716 0.2599 0.1768 0.1654 0.1582 0.1533 0.1151 0.1422 0.1274 0.1416 0.2425 0.1696 0.1664 0.1133 0.1110 0.1567 0.1344 0.1192 0.1220 0.1095 0.1111 | new samples: 0.8196 0.8975 0.8551 0.7987 0.7551 0.6302 0.7392 0.7041 0.6524 0.7786 0.5970 0.5697 0.6848 0.5866 0.5124 0.5540 0.8614 0.5686 0.8842 0.6744 0.8513. Confounds: the macOS load was above the 5 the owner's earlier probe ran at and the 5-minute average was about 10 throughout; the Linux disk is a virtual one; S4 asks for the CI runners and none was measured. The scratch harness is not committed. · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed
- 2026-11-20 · human-observed · S4 measured on GitHub's hosted runners, 2026-11-20, by a temporary workflow that built main's binary (91a611c9) and T1's, then ran the same harness as the macOS and Docker series (ADR-105 T3's fixture, a one-file write, 21 alternating runs each, medians; Python's perf_counter around the subprocess); the workflow also ran the task's three tests on both runners and all passed, the windows row on windows-latest and dir_sync_fails on ubuntu-latest. GitHub Actions run 44774475420 (workflow measure210 on the throwaway branch measure-210-ci, which was deleted; no PR): || [ubuntu-latest] platform: Linux-7.0.0-1019-azure-x86_64-with-glibc2.43 cpus: 4 ; load before: (0.63720703125, 0.2607421875, 0.09912109375) ; one-file write (1 file, 1 hunk): runs=21 old median 0.0055 s, new median 0.0070 s, delta +0.0015 s (+26.5 %); load1 before 0.64, last 0.64 ; old samples: 0.0097 0.0055 0.0060 0.0055 0.0055 0.0055 0.0054 0.0056 0.0056 0.0054 0.0053 0.0052 0.0052 0.0094 0.0054 0.0061 0.0092 0.0054 0.0192 0.0055 0.0053 ; new samples: 0.0060 0.0079 0.0067 0.0133 0.0090 0.0092 0.0060 0.0061 0.0060 0.0231 0.0059 0.0097 0.0060 0.0059 0.0086 0.0124 0.0080 0.0092 0.0067 0.0060 0.0070 ; 500 files, 20 dirs, 5000 hunks: runs=21 old median 0.2606 s, new median 0.4740 s, delta +0.2134 s (+81.9 %); load1 before 0.64, last 0.96 ; old samples: 0.2607 0.2751 0.2606 0.2571 0.2571 0.2609 0.2536 0.2580 0.2684 0.2587 0.2902 0.2579 0.2866 0.2550 0.2584 0.2567 0.2595 0.5803 0.3392 0.2688 0.2703 ; new samples: 0.6738 0.6803 0.5214 0.4494 0.4947 0.5613 0.4079 0.4740 0.3940 0.3946 0.4922 0.3878 0.5444 0.4344 0.3736 0.4343 0.4370 1.1166 0.4769 0.5195 0.4632 || [windows-latest] platform: Windows-2025Server-10.0.26100-SP0 cpus: 4 ; load before: (-1.0, -1.0, -1.0) ; one-file write (1 file, 1 hunk): runs=21 old median 0.0253 s, new median 0.0255 s, delta +0.0002 s (+0.8 %); load1 before -1.00, last -1.00 ; old samples: 0.0257 0.0246 0.0252 0.0241 0.0255 0.0260 0.0247 0.0248 0.0249 0.0287 0.0242 0.0254 0.0248 0.0253 0.0266 0.0281 0.0250 0.0254 0.0262 0.0244 0.0260 ; new samples: 0.0255 0.0248 0.0272 0.0268 0.0268 0.0266 0.0257 0.0238 0.0260 0.0264 0.0240 0.0243 0.0265 0.0249 0.0250 0.0251 0.0253 0.0257 0.0248 0.0255 0.0268 ; 500 files, 20 dirs, 5000 hunks: runs=21 old median 1.7260 s, new median 1.8964 s, delta +0.1704 s (+9.9 %); load1 before -1.00, last -1.00 ; old samples: 1.7321 1.7075 1.7191 1.7205 1.7528 1.7260 1.7227 1.7103 1.7298 1.7543 1.7206 1.7343 1.7207 1.7147 1.6996 1.7307 1.7425 1.7692 1.7470 1.7334 1.6953 ; new samples: 1.9028 1.9288 1.8728 1.9026 1.9000 1.8980 1.9040 1.8812 1.8861 1.8964 1.8908 1.9108 1.8809 1.8805 1.8764 1.9193 1.9215 1.9658 1.8701 1.8775 1.8466 || GitHub Actions run 44775224235 (workflow measure210 on the throwaway branch measure-210-ci, which was deleted; no PR): || [ubuntu-latest] platform: Linux-7.0.0-1019-azure-x86_64-with-glibc2.43 cpus: 4 ; load before: (1.05517578125, 0.2744140625, 0.0869140625) ; one-file write (1 file, 1 hunk): runs=21 old median 0.0064 s, new median 0.0072 s, delta +0.0008 s (+11.9 %); load1 before 1.06, last 1.06 ; old samples: 0.0072 0.0064 0.0068 0.0063 0.0066 0.0065 0.0065 0.0065 0.0063 0.0063 0.0064 0.0063 0.0065 0.0064 0.0064 0.0063 0.0067 0.0063 0.0063 0.0073 0.0067 ; new samples: 0.0077 0.0074 0.0072 0.0078 0.0070 0.0071 0.0094 0.0095 0.0092 0.0071 0.0070 0.0073 0.0073 0.0078 0.0070 0.0071 0.0067 0.0067 0.0069 0.0072 0.0074 ; 500 files, 20 dirs, 5000 hunks: runs=21 old median 0.3064 s, new median 0.4488 s, delta +0.1423 s (+46.4 %); load1 before 1.06, last 1.28 ; old samples: 0.3101 0.3355 0.3089 0.3111 0.3111 0.3084 0.3053 0.3087 0.3043 0.3064 0.3045 0.3112 0.3092 0.3102 0.3038 0.3063 0.3060 0.3033 0.3054 0.3047 0.3052 ; new samples: 0.6353 0.6546 0.4582 0.4596 0.4741 0.4158 0.4514 0.4276 0.4757 0.4325 0.4340 0.4177 0.4175 0.4101 0.4267 0.4328 0.5038 0.4241 1.9138 0.4488 0.4638 || [windows-latest] platform: Windows-2025Server-10.0.26100-SP0 cpus: 4 ; load before: (-1.0, -1.0, -1.0) ; one-file write (1 file, 1 hunk): runs=21 old median 0.0174 s, new median 0.0173 s, delta -0.0001 s (-0.7 %); load1 before -1.00, last -1.00 ; old samples: 0.0185 0.0195 0.0177 0.0176 0.0170 0.0174 0.0176 0.0174 0.0167 0.0169 0.0166 0.0180 0.0174 0.0173 0.0169 0.0168 0.0164 0.0195 0.0303 0.0171 0.0185 ; new samples: 0.0197 0.0180 0.0173 0.0171 0.0169 0.0169 0.0172 0.0168 0.0167 0.0175 0.0170 0.0175 0.0179 0.0173 0.0171 0.0169 0.0173 0.0204 0.0176 0.0171 0.0190 ; 500 files, 20 dirs, 5000 hunks: runs=21 old median 1.3947 s, new median 1.4613 s, delta +0.0666 s (+4.8 %); load1 before -1.00, last -1.00 ; old samples: 1.4133 1.3719 1.4490 1.3863 1.4068 1.4083 1.3863 1.3959 1.4186 1.4139 1.4095 1.3828 1.4611 1.4447 1.3808 1.3508 1.3887 1.3887 1.3869 1.3947 1.3837 ; new samples: 1.4565 1.4613 1.4529 1.4984 1.6554 1.4492 1.4443 1.5024 1.4870 1.4735 1.5017 1.5017 1.4706 1.4567 1.4144 1.4226 1.4599 1.5113 1.5035 1.4417 1.4349. Confounds: shared hosted runners; the filesystem of $RUNNER_TEMP was not recorded; the two runs of one workflow differ by more than the Windows difference; Windows has no load average. · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed
- 2026-11-20 · 09625199* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:1983
- 2026-11-20 · 169661bb · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:446b4d3d89fbedf4f08646e769d14fa4ffdabdfb40575d5e13ff28f952cb5bed · ms:2023
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:3433
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5224
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5147
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:4703
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5715
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5244
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5384
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5243
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5297
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5357
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5972
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:6132
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:7526
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:8221
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5024
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:8448
- 2026-11-20 · a79532f0* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:3882
  ```
  --- last 4 line(s) of stderr
  github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test: open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/go-build1095619830/b260/importcfg: no space left on device
  github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp: open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/go-build1095619830/b259/vet.cfg: no space left on device
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test]
  can't create $WORK/b170/_pkg_.a: open $WORK/b170/_pkg_.a: no space left on device
  ```
- 2026-11-20 · a79532f0* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:3959
  ```
  --- last 4 line(s) of stderr
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  compile: writing output: write $WORK/b259/_pkg_.a: no space left on device
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test]
  can't create $WORK/b170/_pkg_.a: open $WORK/b170/_pkg_.a: no space left on device
  ```
- 2026-11-20 · a79532f0* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:781
  ```
  --- last 4 line(s) of stderr
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test
  /opt/homebrew/Cellar/go/1.27.4/libexec/pkg/tool/darwin_arm64/link: mapping output file failed: no space left on device
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test
  /opt/homebrew/Cellar/go/1.27.4/libexec/pkg/tool/darwin_arm64/link: mapping output file failed: no space left on device
  ```
- 2026-11-20 · a79532f0* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:2274
  ```
  --- last 2 line(s) of stderr
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test
  /opt/homebrew/Cellar/go/1.27.4/libexec/pkg/tool/darwin_arm64/link: mapping output file failed: no space left on device
  ```
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5801
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:4794
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:4815
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5403
- 2026-11-20 · a79532f0* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:738
  ```
  --- last 2 line(s) of stderr
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply
  internal/apply/sync_darwin.go:30:20: undefined: errors
  ```
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:5752
- 2026-11-20 · a79532f0* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:bfe9b76f784ba1d239af603db3907c185812844f341677c2547845ccef16750a · ms:2618
- 2026-11-20 · f2d91c78 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:3338
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:5027
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4750
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:5505
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4677
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4593
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4987
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:5980
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:6087
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:5160
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:6168
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:6251
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:5331
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4350
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4024
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:5135
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4360
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:3819
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:3966
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4167
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:4041
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:3627
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:6564
- 2026-11-20 · f2d91c78* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:5965
- 2026-11-20 · 9847b698 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:98c962f0822c80c35f393f487f47a87a17b2c8cfd2fc6444a7d4b0df60338edd · ms:2524
- 2026-11-20 · 2f3f930c · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:10161
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:17469
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:13091
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:13894
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:9471
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:9517
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:17934
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:8249
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:9865
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:5617
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:5729
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:6506
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:6460
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:7908
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:6880
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:6977
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:8171
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:7684
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:6496
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:7724
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:7715
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:7190
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:6447
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:4969
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:6006
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:5689
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:5734
- 2026-11-20 · 2f3f930c* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:2473
- 2026-11-20 · f97365c8 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb · ms:2708
- 2026-11-20 · human-observed · S4 measured on real Windows hardware by a Windows peer, reported by the lead, 2026-11-20 (the implementer did not run it): Windows 11 26200, Samsung 980 PRO NVMe, NTFS, Defender on, go1.27.4; 21 alternating runs (5 over SMB); PowerShell 7 Stopwatch around the whole process including its start; plans written with anchor= hunks because bare line numbers after --quiet are refused; main against T1. Local NVMe: one file, main 90.2 ms, T1 91.8 ms median (+1.5 ms, 1.02x); 500 files, main 12677.7 ms, T1 13962.8 ms (+1285 ms, 1.10x), spread about +-20 %, main alone about 25 ms per file on Windows (cf. #650). Loopback SMB (\\localhost\C$), 5 runs, every write exited 0: one file, main 663.9 ms, T1 647.2 ms (0.97x); 500 files, main 132743 ms, T1 133744 ms (1.01x). Decision 3's premise, measured non-elevated: on local NTFS os.Open(dir).Sync() gives Access is denied (5); but CreateFile(dir, FILE_WRITE_DATA|SYNCHRONIZE or FILE_APPEND_DATA|SYNCHRONIZE or GENERIC_WRITE, share read|write|delete, OPEN_EXISTING, FILE_FLAG_BACKUP_SEMANTICS) followed by FlushFileBuffers succeeds; GENERIC_READ and FILE_WRITE_ATTRIBUTES give 5. Over SMB every write-capable handle opens and the flush fails with ERROR_INVALID_FUNCTION (1). Whether an NTFS directory flush makes a rename durable was not tested (no power-loss test). Whether Windows should sync directories, treating errno 1 as not supported here, is open for Zy. · acceptance-sha256:49ee3a8465c0bf92b4b7410e83e65e3b1c827e48dfc38d6d14922c9fadf780bb
- 2026-11-20 · 24beedfe · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:3135
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6290
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6541
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:8044
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:8382
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6808
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:5736
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:12094
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:12651
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:13674
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:8632
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:7008
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:5854
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6516
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:7096
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:5835
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:7411
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6358
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6941
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6688
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6390
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:6555
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:5005
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:4243
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:4761
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:4287
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:4438
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:4498
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:4309
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:4012
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:5389
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:16443
- 2026-11-20 · 24beedfe* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:7184
- 2026-11-20 · b95b3ce1 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d00cff50a0ce00350dc42ba04f47fae1e1f3fa917ac559e98e6b0b4a65b73f0c · ms:3804
- 2026-11-20 · d5eba789 · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:d651985478f7f19eb3b1fc4a34fa8ceb6240e948a60182aa4339411b57743c7c · ms:3950

## Invariants

- mrw's state files keep ADR-105's rule.
- No receipt key is removed or renamed; `sync_note` is the one added (ADR-111, Zy's amendment of 2026-11-20).
- Each file still lands by one rename.

## Risks

- The fence proves §361's text, not that it ran: §361 is proved by `scripts/contract.sh` on a runner with `strace` (`lifecycle.md` §6); on one without, the row is a visible skip and supplies no evidence, which the Stop Condition does not accept.
- The fence's replacement-word clauses fix the phrases T1 must use: `not known to be on stable storage` (the error), the three README phrases of the record's Decision 6, and `the files could not be synced` (AGENTS.md).
- The mutant `a directory synced twice (once per rename)` survives the current fence, and is equivalent: the directory set is now deduplicated by identity (`os.SameFile`), so a list that names a directory twice still syncs it once. The mutant that removes that deduplication is the one that matters, and the `one_directory_two_spellings` row kills it where the filesystem folds case (macOS, Windows).

## Stop Condition

The fence exits 0, `./scripts/contract.sh` exits 0 on a runner with `strace` with §361's rows printed and not skipped, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes on Linux and Windows CI, S4's numbers are in the record, and the record is Accepted by Zy.

## Out of Scope

- Syncing mrw's state directory (permanent: boundary: the record's Out of Scope; ADR-105 stands for it)
