# Task ADR-038-T1: `Record` holds the lock for load-merge-save

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `Record` holds the lock (T1)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every concurrent Record keeping its path`, `the unlocked mutant dropping paths`

## Goal

Make `seen.Record` the exclusive writer of one checkout's ledger for the duration of one
load-merge-save, without changing the format or `go.mod`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `Record` opens `seen.lock` via `state.Path`, locks, then does the existing load-merge-save. An unexported `afterLoad` hook lets the test force the interleaving the unlocked mutant needs. |
| `internal/seen/lock_unix.go` | add | `syscall.Flock` `LOCK_EX` / `LOCK_UN`. Selected by the `unix` build tag. |
| `internal/seen/lock_windows.go` | add | `syscall.LockFileEx` exclusive. Selected by `windows`. The file is compiled on Windows CI, not skipped. |
| `internal/seen/seen_test.go` | edit | `TestConcurrentRecordsKeepEveryPath` — 40 goroutines, each `Record`s a distinct path; after join the ledger holds all 40. The hook makes every goroutine load before any saves, so the unlocked mutant keeps 1. |

## Ordered Steps

1. [S1] Write `TestConcurrentRecordsKeepEveryPath` and confirm it is RED: without the lock, the
   hook lets all 40 load an empty ledger and the last save wins. ⚠ Do not assert the platform.
   [proof: mutation]
2. [S2] Implement `lock` / `unlock` on unix and windows. Wire `Record` to hold the exclusive lock
   around load-merge-save. [proof: mutation]
3. [S3] Confirm existing `internal/seen` tests still pass, including sequential merge of two
   `Record`s. [proof: acceptance]
4. [S4] Confirm `go.mod` still has exactly one `require`, and the ledger header is still
   `#mrw-seen v2`. [proof: acceptance]
5. [S5] Run `gofmt` and `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/seen/ -count=1 -v \
  -run 'TestConcurrentRecordsKeepEveryPath' 2>&1 | tee /tmp/adr038-t1.out \
  && grep -q '^--- PASS: TestConcurrentRecordsKeepEveryPath' /tmp/adr038-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr038-t1.out \
  && go test ./internal/seen/ -count=1 \
  && awk '/^require /{c++} END{exit !(c==1)}' go.mod \
  && grep -q '#mrw-seen v2' internal/seen/seen.go \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/seen/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestConcurrentRecordsKeepEveryPath` | `internal/seen/seen_test.go` | 40 concurrent `Record`s of distinct paths leave 40 entries; the unlocked mutant leaves fewer | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestConcurrentRecordsKeepEveryPath` |
| 2 — something selects it | `Record` is the only writer; every caller already calls it. Deleting the lock call fails S1 |
| 3 — the caller can discover it | T3 teaches it; this task does not change help text |
| 4 — it is used | §76 (T2) drives 40 CLI processes |

## Mutation Log

- 2026-09-09 · 0737f60* · mutant killed · exit 1 · `internal/seen/lock_unix.go` · unlocked Record lets concurrent saves clobber each other; TestConcurrentRecordsKeepEveryPath must keep fewer than 40 · acceptance-sha256:64684044e8e0d27ef63c877fc8ca652e671e80c31f109a35d7c7f397277b2a0d

## Invariants

- Sequential `Record` merge is unchanged: two calls about different paths keep both.
- Issue #47 / ADR-029 are untouched. This is who writes the file, not which observation is
  consumed.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/check`, `internal/state` stay
  byte-identical against the merge-base.
- `go.mod` declares exactly one requirement.

## Risks

- ⚠ A 40-goroutine test without the load/save hook is green against a missing lock. The hook is
  what makes S1's mutant fail. §76 is the process-shaped proof.
- A Windows `LockFileEx` that compiles and does nothing fails §76 on Linux only. The Windows file
  must lock, not stub.

## Stop Condition

Stop and ask if closing this requires a v3 ledger or a new `go.mod` requirement. Either is a
different record.

## Out of Scope

- The contract row — that is T2's job
- Teaching — that is T3's job
- A shared lock on `Load` (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-09-09 · 0737f60* · exit 0 · `set -o pipefail …` · acceptance-sha256:64684044e8e0d27ef63c877fc8ca652e671e80c31f109a35d7c7f397277b2a0d · ms:1076
