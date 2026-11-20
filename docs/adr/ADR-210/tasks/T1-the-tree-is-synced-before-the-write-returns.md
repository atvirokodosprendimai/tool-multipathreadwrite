# Task ADR-210-T1: the tree is synced before the write returns

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the file and directory syncs in `internal/apply`, behind seams; `writer.SyncError` and its rendering on the CLI and `mrw_write`; contract §361; Decision 6's words; the end-to-end cost in the record
**Consumes:** `stageFile`, the commit loop, `commitPathOps`, `(*tree)`, `writer.LedgerError`'s handling in `Land`, `appliedButUnreportable`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `every temp is synced before the first rename`, `every changed directory is synced once after the last change`, `the drain runs once per device per phase on macOS`, `no directory is synced on Windows`, `a sync failure before the renames writes nothing`, `a sync failure after the renames is a landed write with an error`, `a sync failure and a ledger failure keep both errors`, `the check and steps do not run after a sync failure`, `a dry run syncs nothing`, `every subtest ran and none skipped`, `the CLI and mrw_write report it`, `the contract row exists with its trace`, `the new words are in the documents`, `the engine packages are untouched, tracked and untracked`

## Goal

ADR-210 Decisions 1 to 6.

**`TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns`** (`internal/apply`), one subtest per row, the seams recording every sync, drain, rename, removal and directory creation in order:

| Subtest | Plan and seam | Outcome |
|---------|---------------|---------|
| `edit_create_rename` (every platform) | edit `a.txt`; create `n/m/k.txt` in new directories; rename `s/b.txt` to `t/b.txt` (both parents exist and differ) | every temp's sync before the first rename; on macOS one drain after the last temp sync and before the first rename; outside Windows, after the last change one sync each for `.`, `n`, `n/m`, `s` and `t`, then on macOS one drain; on Windows no directory sync; nothing synced after `Apply` returns |
| `two_devices` (macOS) | edit a file on each of two devices, the second faked through the seam's device number | one drain per device in each phase |
| `rename_only` (every platform) | rename `s/b.txt` to `t/b.txt` | no file sync; outside Windows `s` and `t` synced; on Windows nothing synced |
| `unlink_only` (every platform) | unlink `s/b.txt` | outside Windows `s` synced; on Windows nothing synced |
| `windows` (Windows) | as `edit_create_rename`, with the directory-sync seam set to fail the test if called | the file syncs; the directory seam never called |
| `temp_sync_fails` (every platform) | the second temp's sync fails | an error; nothing renamed; every temp removed |
| `drain_before_fails` (macOS) | the first drain fails | as `temp_sync_fails` |
| `dir_sync_fails` (Linux and macOS) | the sync of `n` fails | `Applied` true; the error is a sync error naming `n`; every file in place |
| `drain_after_fails` (macOS) | the second drain fails | as `dir_sync_fails` |
| `dry_run` (every platform) | `DryRun` | no sync recorded |

A row for another platform is not registered at all (a build tag or a `runtime.GOOS` test before `t.Run`), never registered and skipped, so the fence can require every row its platform names and refuse any skip.

**`TestTheCLIReportsASyncThatFailedAfterLandingAsAWriteThatLanded`** (`cmd/mrw`, through the writer with the seam): exit 2; `--json` has `applied: true` and `error` naming the directory and saying the files are written but not known to be on stable storage; the ledger records the file; with `--check` and `--then`, no check runs and the steps are reported not run; `mrw stats` counts `check_not_run`; with the ledger also failing, `error` names both.

**`TestMrwWriteReportsASyncThatFailedAfterLanding`** (`internal/mcp`): `isError` with the landed receipt; under a result ceiling too small for the receipt, the fallback sentence carries the same words.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the file sync in `stageFile`, the drain, the directory syncs after the last change, the sync error |
| `internal/apply/tree.go` | edit | open a directory through the root for its sync |
| `internal/apply/sync_darwin.go`, `sync_other.go`, `sync_windows.go` | new | the per-platform drain and directory sync |
| `internal/writer/writer.go`, `internal/writer/flow.go` | edit | `SyncError`, classified as a landed write as `LedgerError` is, both errors kept |
| `internal/mcp/tools.go`, `internal/mcp/schema.go` | edit | `isError` with the landed receipt; the fallback sentence; the `error` key's description |
| `cmd/mrw/main.go` | edit | the CLI's rendering of the sync error |
| `internal/apply/sync210_test.go`, `cmd/mrw/sync210_test.go`, `internal/mcp/sync210_test.go` | new | the three tests |
| `scripts/contract.sh` | edit | §361 |
| `README.md`, `AGENTS.md` | edit | Decision 6 |

## Ordered Steps

1. [S1] The three tests; RED on every subtest but `dry_run`, and on the CLI and MCP tests. [proof: acceptance]
2. [S2] The syncs. Mutants, each killed: no temp sync (`edit_create_rename`); a temp synced after the first rename (`edit_create_rename`, by order); no directory sync (`edit_create_rename`); a directory synced once per rename (`edit_create_rename`, by count); the drain dropped on macOS (`edit_create_rename`); one drain for all devices (`two_devices`); a directory sync on Windows (`windows`); a failed temp sync ignored (`temp_sync_fails`); a failed directory sync ignored (`dir_sync_fails`); the sync error classified as `refused_apply` (the CLI test's stats); the check run after a sync failure (the CLI test); the ledger's error dropping the sync's (the CLI test); the dry run syncing (`dry_run`). [proof: mutation]
3. [S3] Contract §361, as the record's Wiring table says: `strace -f -y -e trace=fsync,fdatasync,rename,renameat,renameat2` over `$MRW write` of a plan editing two files in two directories, decoding each descriptor's path (`-y`) and taking only calls that returned 0: every temp's `fsync` precedes the first tree rename; each changed directory's `fsync` follows the last tree change; the ledger's rename follows those. Paired with `--dry-run` of the same plan on a fresh fixture, which must exit 0 and show no `fsync` on a tree path. Where `strace` is absent the row says `skip` and why; the CI job that proves it installs `strace`. The words. [proof: acceptance]
4. [S4] The cost: ADR-105 T3's protocol (a 500-file, 5,000-hunk write and a one-file write, 5 alternating runs each, the old build against the new, each on a fresh copy with its own `XDG_STATE_HOME`) on the owner's Mac and the Linux and Windows CI runners; the medians, the load and the filesystem go into the record's Consequences, for Zy's decision under Decision 7. [proof: human: the medians, the load, the filesystem and the machine of each run, recorded in the record]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./internal/apply ./cmd/mrw ./internal/mcp -run '^(TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns|TestTheCLIReportsASyncThatFailedAfterLandingAsAWriteThatLanded|TestMrwWriteReportsASyncThatFailedAfterLanding)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 3 ] \
  && case "$(go env GOOS)" in darwin) want='edit_create_rename rename_only unlink_only temp_sync_fails dry_run dir_sync_fails two_devices drain_before_fails drain_after_fails' ;; windows) want='edit_create_rename rename_only unlink_only temp_sync_fails dry_run windows' ;; *) want='edit_create_rename rename_only unlink_only temp_sync_fails dry_run dir_sync_fails' ;; esac \
  && missing=0 && for c in $want; do grep -q "^    --- PASS: TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns/$c " "$out" || missing=1; done && [ "$missing" -eq 0 ] \
  && ! grep -q -- '--- SKIP' "$out" \
  && grep -q '^# 361\. ' scripts/contract.sh \
  && s361="$(sed -n '/^# 361\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh)" \
  && grep -qF 'strace -f -y' <<<"$s361" \
  && grep -qF -- '--dry-run' <<<"$s361" \
  && grep -q 'not known to be on stable storage' internal/writer/writer.go \
  && readme="$(tr -s '\n ' '  ' < README.md)" \
  && grep -qF 'synced before the ledger' <<<"$readme" \
  && grep -qF 'on Windows only the file contents are synced, not the directory entries' <<<"$readme" \
  && grep -qF 'the files are written but not known to be on stable storage: do not send the plan again' <<<"$readme" \
  && grep -q 'the files could not be synced' <(tr -s '\n ' '  ' < AGENTS.md) \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state)" ]
```

The cases each platform must pass are named in the fence: six on Linux, nine on macOS, six on Windows. The fence runs where the test runs; CI runs it on Linux and Windows, and macOS's rows are proved on the owner's Mac.

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

## Verification Log

## Invariants

- mrw's state files keep ADR-105's rule.
- No receipt key is added, removed or renamed.
- Each file still lands by one rename.

## Risks

- The fence proves §361's text, not that it ran: §361 is proved by `scripts/contract.sh` on a runner with `strace` (`lifecycle.md` §6); on one without, the row is a visible skip and supplies no evidence, which the Stop Condition does not accept.
- The fence's replacement-word clauses fix the phrases T1 must use: `not known to be on stable storage` (the error), the three README phrases of the record's Decision 6, and `the files could not be synced` (AGENTS.md).

## Stop Condition

The fence exits 0, `./scripts/contract.sh` exits 0 on a runner with `strace` with §361's rows printed and not skipped, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes on Linux and Windows CI, S4's numbers are in the record, and the record is Accepted by Zy.

## Out of Scope

- Syncing mrw's state directory (permanent: boundary: the record's Out of Scope; ADR-105 stands for it)
