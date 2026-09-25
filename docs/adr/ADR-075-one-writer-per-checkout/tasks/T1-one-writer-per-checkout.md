# Task ADR-075-T1: one writer per checkout

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `seen.LockWrites`, `seen.Snapshot`; `internal/writer.Apply` and `LedgerError`; both write surfaces through it
**Consumes:** `apply.Apply`, `seen.Drop`, `seen.Record` (ADR-038's lock primitive)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `no two writers are inside at once`, `a stale writer is refused, not rebased`, `what landed is recorded under the lock`, `the write lock is not the ledger lock`, `both surfaces write through writer.Apply`, `the built binary loses no edit`, `a contract row drives the binary`, `the packages vet for Windows`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

Eight writers off one read lost 45–53% of their edits with every loser printing "applied", exit 0.
Take a per-checkout write lock around apply and the ledger update, with the ledger loaded before it,
so a writer is refused or lands and is never told "applied" for an edit that is gone.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `LockWrites`; `withLock` shares its open-and-lock with it |
| `internal/writer/writer.go` | new | lock, apply, drop and record, release |
| `cmd/mrw/main.go` | edit | the write Action calls `writer.Apply`; its ledger loop moves there |
| `internal/mcp/tools.go` | edit | the write tool calls `writer.Apply`; its ledger loop moves there |
| `internal/seen/writelock_test.go` | new | the lock excludes, and is not the ledger's; a snapshot is never a half-saved ledger |
| `internal/writer/writer_test.go` | new | no overlap; a stale writer refused; what landed recorded |
| `internal/adversarial/concurrent_write_test.go` | new | eight processes of the built binary lose nothing |
| `scripts/contract.sh` | edit | §150 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN; the ADR-038 ledger test stays green. [proof: mutation]
   Mutants: `writer.Apply` takes no lock; `LockWrites` locks `seen.lock`; `writer.Apply` reloads the ledger under the lock; the ledger update removed; the MCP write tool back on `apply.Apply`; `Snapshot` loads without the lock.
3. [S3] Contract §150: eight writers off one read, each a different line — none exits 0 without its edit, every other exits 1 naming the change. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/writer/ ./internal/seen/ ./internal/adversarial/ -count=1 -timeout 300s -run 'TestNoTwoWritersAreInsideAtOnce|TestASnapshotNeverSeesAHalfSavedLedger|TestAWriterWhoseFileChangedWhileItWaitedIsRefused|TestWhatLandedIsRecordedBeforeTheLockIsReleased|TestAWriteLockExcludesASecondWriter|TestTheLedgerCanBeWrittenUnderTheWriteLock|TestConcurrentWritersOffOneReadLoseNothing|TestConcurrentRecordsKeepEveryPath' -v 2>&1 | tee /tmp/adr075-T1.out \
  && missing=$(for t in TestNoTwoWritersAreInsideAtOnce TestASnapshotNeverSeesAHalfSavedLedger TestAWriterWhoseFileChangedWhileItWaitedIsRefused TestWhatLandedIsRecordedBeforeTheLockIsReleased TestAWriteLockExcludesASecondWriter TestTheLedgerCanBeWrittenUnderTheWriteLock TestConcurrentWritersOffOneReadLoseNothing TestConcurrentRecordsKeepEveryPath; do grep -qE "^--- PASS: $t \(" /tmp/adr075-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q 'writer\.Apply(' cmd/mrw/main.go \
  && grep -q 'writer\.Apply(' internal/mcp/tools.go \
  && ! grep -q 'apply\.Apply(' cmd/mrw/main.go internal/mcp/tools.go \
  && grep -q '^# 150\. ' scripts/contract.sh \
  && GOOS=windows go vet ./internal/seen/ ./internal/writer/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/read internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc internal/seen ':(exclude)internal/seen/seen.go' ':(exclude)internal/seen/writelock_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/read internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc internal/seen ':(exclude)internal/seen/seen.go' ':(exclude)internal/seen/writelock_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestNoTwoWritersAreInsideAtOnce` | `internal/writer/writer_test.go` | six writers, each lingering inside; the peak is 1 and all land | — | S1, S2 |
| `TestAWriterWhoseFileChangedWhileItWaitedIsRefused` | `internal/writer/writer_test.go` | two writers off one snapshot: the second is "changed since", the file holds the first | — | S1, S2 |
| `TestWhatLandedIsRecordedBeforeTheLockIsReleased` | `internal/writer/writer_test.go` | a written file recorded wholly, an unlinked one dropped | — | S1, S2 |
| `TestAWriteLockExcludesASecondWriter` | `internal/seen/writelock_test.go` | a second holder waits; a release twice is harmless | — | S1, S2 |
| `TestTheLedgerCanBeWrittenUnderTheWriteLock` | `internal/seen/writelock_test.go` | `Record` completes while the write lock is held | — | S1, S2 |
| `TestASnapshotNeverSeesAHalfSavedLedger` | `internal/seen/writelock_test.go` | 2,000 snapshots against a writer rewriting the ledger never see fewer entries than it holds (review of #233) | — | S2 |
| `TestConcurrentWritersOffOneReadLoseNothing` | `internal/adversarial/concurrent_write_test.go` | eight processes, three rounds: exit 0 means the edit is there, exit 1 names the change | — | S1, S2 |
| `TestConcurrentRecordsKeepEveryPath` | `internal/seen/seen_test.go` | ADR-038, unchanged | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `writer.Apply` |
| 2 — something selects it | every `mrw write` and `mrw_write` |
| 3 — the caller can discover it | a stale writer's refusal names the change |
| 4 — it is used | the v1.25.1 round lost edits this way on two platforms |

## Verification Log
(empty until execute)
- 2026-09-26 · da2fd0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · ms:1297 · test-lock-sha256:149aec9ef23ac71702a92e5be72fc502157ad682007f05798b7c552610047030 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FkdmVyc2FyaWFsL2NvbmN1cnJlbnRfd3JpdGVfdGVzdC5nbwlUZXN0Q29uY3VycmVudFdyaXRlcnNPZmZPbmVSZWFkTG9zZU5vdGhpbmcJMjQ4NDY5ZGMzZTljZDQ2ZmQ2YjQwYTA3M2Q1ZmQ1YTJkYTM2YTczNzJjOTIxZjYxZTUyZDRiZGMwZDU4ODQzZgpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBTGVkZ2VyVGhpc0J1aWxkV3JvdGVMb2Fkc0JhY2sJZDFkNmI4OTRlNTFlZmZhZjAwNjI0NWQ3ZjdjZWExNTY4YTIzNmMyZTRhNWJhYjVjOTc2NDNiM2M2MTY3YTAwMApib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBTGVnYWN5UGF0aENvbnRhaW5pbmdBRG91YmxlU3BhY2VJc09uZVBhdGgJY2I3YjVjY2Q0NGMzMTk0YWY4OTVjMzhlMDlkNjJiODA0M2RmZjExZDcyZTBiNTIyNzU1OTg2MmFiYmEyYWYxNQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBUHJlVjJJblRyZWVMZWRnZXJJc0Rpc2NhcmRlZE5vdFRydXN0ZWQJOTYwY2YyOWViZjZmYTJkMmI5OGViYWZhZTZiMzU0ODc2ODNjMDM3Njc1YzU0ZjQ4YjExOGJmNzc0OTBlYWZlZQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBVjJMZWRnZXJJc0FjY2VwdGVkQW5kVGhlQnVtcElzRGVsaWJlcmF0ZQk0OWM1NWMzMzc4Zjk2NTcwZDljYzYwNmY1NWE5Y2Q3YTUwZTY5MGNkMjQ4MDUwYjhjYTQyMGNiOGIzZmZkOTdlCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdENvbmN1cnJlbnRSZWNvcmRzS2VlcEV2ZXJ5UGF0aAllYjlhNmM5NTAxMmVjNjM1YTRlNzUxOWI3NWM3NjEzNGNiOTdjMDA0Y2VkMGYyMTJmN2I4OWEyM2I0ODY5NTNhCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdExlZGdlcklzTm90V3JpdHRlbkludG9UaGVSb290CTQ0MWU5Zjc4YTRhOTMyYmY2MmI5NmIyMjA2MjI3N2EwMjdlODdiN2RjMjc4ZWVhOGU0ZDhjMjg2YThhMTg0YTYKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0TGVnYWN5SW5UcmVlTGVkZ2VySXNTdGlsbFJlYWQJYzRkYThjNzQzMmRhM2IzNTM0N2ZjOGUzMDI4YTNiMTY4ODQ5ZmM1MjM5Zjk5NGE1MzBiOWU4MTgzODU4YzhjNQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RNYWluCTM4M2EzNzQwZWI1YjEzNzI0MDQzZTZmN2RhMWI4YWJiYmU3MTVjYThjYTRjNTIxYzNjNzY1ZDkwMWM5ODJmZDQKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0TWlzc2luZ0xlZGdlcklzRW1wdHlOb3RBbkVycm9yCTU4NTczZDg0ZmNhZTVlNTBjYTEwMjMzMDY4ZDM2MjZkYTNhNmUyMWYzOWM1ZjljMTcyMjZkMjVkOTNlYzA1ZTgKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0UmVjb3JkTWVyZ2VzUmF0aGVyVGhhblJlcGxhY2VzCTIzN2I4MjA1ODNmZjliYmI4NmQ4YmYzYjQ1Zjc5NWNmNjVjMmU0YzQzMjcxNGIwYjliZDhhM2NhODYwMTIxYWMKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0UmVjb3JkT3ZlcndyaXRlc1RoZVNhbWVQYXRoCTViYzYwMDY0Y2RmMzE2ZTYxMmQ3Mjg4MDk0MTc5NzRlNGUwNTkzNDc0YzYwYmZiNWIwMmU2MDYzZWMxMmY5YTcKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0Um91bmRUcmlwCTFlMTgyOGYwMmUyMzhlY2ZjMDhmMGY1MmE4NmI2ZGQwZDhlYjgwODY2NjU5YmE0MDBhYmZlZTNmNjU3ODAyMmYKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0U0hBSXNTdGFibGUJMjlkM2VlYjRjOGYyNzFjNGU1OTUzMjFjN2FmZTdlOWYyMzZhY2IxNDEwMTJjOWM2NzQ0ZGYyNDEzZDYyYTE4ZQpib2R5CWludGVybmFsL3NlZW4vd3JpdGVsb2NrX3Rlc3QuZ28JVGVzdEFXcml0ZUxvY2tFeGNsdWRlc0FTZWNvbmRXcml0ZXIJNjU5ODA0NGE5MDcxNDFhNjExMDRkZTc3MTg5NTMzNDI4NWJkNjlmMjJiYzBmYjMzMTllYWM2NWFjMjcyZGFlMApib2R5CWludGVybmFsL3NlZW4vd3JpdGVsb2NrX3Rlc3QuZ28JVGVzdFRoZUxlZGdlckNhbkJlV3JpdHRlblVuZGVyVGhlV3JpdGVMb2NrCTVjNWFhYWU3MjA4MzVkZTM2Y2RjNjAzN2I1NDlmZjYwOGRmYzA3YjBmNDY4NTU0NWEzZjViYmVhMzZlZmIwYjAKYm9keQlpbnRlcm5hbC93cml0ZXIvd3JpdGVyX3Rlc3QuZ28JVGVzdEFXcml0ZXJXaG9zZUZpbGVDaGFuZ2VkV2hpbGVJdFdhaXRlZElzUmVmdXNlZAk5NmJmNzlhMDE2NWRiM2M2NzE3NjYzNTMyMDM0MWJmOTM5YWRjYjU1NGJmZWFjMjQzYzMyNDUyZWVhY2VkYzJmCmJvZHkJaW50ZXJuYWwvd3JpdGVyL3dyaXRlcl90ZXN0LmdvCVRlc3ROb1R3b1dyaXRlcnNBcmVJbnNpZGVBdE9uY2UJNzQ4NDhlNjAyMmZmM2U2MGIxYjVmNDk1ODQ0ZjNkZGJhNDQyYmJmZDA2MmMzODFmZjRhNzY4ZmUxZWJkNDY4Ywpib2R5CWludGVybmFsL3dyaXRlci93cml0ZXJfdGVzdC5nbwlUZXN0V2hhdExhbmRlZElzUmVjb3JkZWRCZWZvcmVUaGVMb2NrSXNSZWxlYXNlZAk0MzFkNGY4Y2ZlOWMwNDk3MTYzODJmN2M4YjJjNTk0OWYyYTZhYjVkZTJhMjVhZjVhNTJlNzhmZTE4NTBlZTA5
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
  internal/writer/writer_test.go:91:12: undefined: Apply
  internal/writer/writer_test.go:116:14: undefined: Apply
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer [build failed]
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen [build failed]
  === RUN   TestConcurrentWritersOffOneReadLoseNothing
      concurrent_write_test.go:85: round 0: writer(s) [1 3] exited 0 and their edit is gone: ["writer 0" "line 2" "line 3" "line 4" "line 5" "writer 5" "writer 6" "writer 7" "line 9" "line 10" "line 11" "line 12" "line 13" "line 14" "line 15" "line 16"]
  --- FAIL: TestConcurrentWritersOffOneReadLoseNothing (0.72s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial	0.914s
  FAIL
  ```
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · ms:1552
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · ms:1268
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · ms:971
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · ms:949
- 2026-09-26 · da2fd0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · ms:842
  ```
  --- last 10 line(s) of stdout (of 25 after folding 25 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen	0.369s
  === RUN   TestConcurrentWritersOffOneReadLoseNothing
      concurrent_write_test.go:78: round 2: writer 0 exited 1 without naming the change:
          FAIL f.txt 1 replace (plan line 1): f.txt has not been read: mrw does not know what it currently holds, and a line address means nothing without that. Run `mrw read f.txt` first, or pass --force
          1 hunk(s), 1 file(s), 1 failed, 0 advisories — NOTHING WRITTEN
          mrw: 1 hunk(s) failed — nothing was written
  --- FAIL: TestConcurrentWritersOffOneReadLoseNothing (0.39s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial	0.450s
  FAIL
  ```
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · ms:1176
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · ms:1529
- 2026-09-26 · 2463a7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1581
- 2026-09-26 · 2463a7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1093
- 2026-09-26 · 2463a7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1161
- 2026-09-26 · 2463a7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1150
- 2026-09-26 · 2463a7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1484
- 2026-09-26 · 2463a7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1215
- 2026-09-26 · 2463a7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1467
- 2026-09-26 · 2c23bd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1850
- 2026-09-26 · 0f9e789* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · ms:1916

## Mutation Log
(empty until execute)
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/writer/writer.go` · writer.Apply takes no lock, so two writers validate and commit at once · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · covers:no two writers are inside at once
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/writer/writer.go` · the ledger is reloaded under the lock, so a stale writer inherits the previous writer's whole-file licence and is rebased · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · covers:a stale writer is refused, not rebased
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/writer/writer.go` · what landed is not recorded before the lock is released · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · covers:what landed is recorded under the lock
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/seen/seen.go` · the write lock is the ledger's own lock, and Record under it waits on itself · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · covers:the write lock is not the ledger lock
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP write tool bypasses the write lock · acceptance-sha256:b14bc53a98cb54454c1fe1a960ed20813e73167059add05e104d1d7c9463dd12 · covers:both surfaces write through writer.Apply
- 2026-09-26 · 2463a7d* · mutant killed · exit 1 · `internal/seen/seen.go` · the snapshot loads without the ledger's lock and can see a half-saved ledger · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · covers:a stale writer is refused, not rebased
- 2026-09-26 · 2463a7d* · mutant killed · exit 1 · `internal/writer/writer.go` · writer.Apply takes no lock, so two writers validate and commit at once · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · covers:no two writers are inside at once
- 2026-09-26 · 2463a7d* · mutant killed · exit 1 · `internal/writer/writer.go` · the ledger is reloaded under the lock, so a stale writer inherits the previous writer's whole-file licence and is rebased · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · covers:a stale writer is refused, not rebased
- 2026-09-26 · 2463a7d* · mutant killed · exit 1 · `internal/writer/writer.go` · what landed is not recorded before the lock is released · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · covers:what landed is recorded under the lock
- 2026-09-26 · 2463a7d* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP write tool bypasses the write lock · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · covers:both surfaces write through writer.Apply
- 2026-09-26 · 2463a7d* · mutant killed · exit 1 · `internal/seen/seen.go` · the write lock is the ledger's own lock, and Record under it waits on itself · acceptance-sha256:cd32d7d7197ac22c63f1ce7eac50919277e309eb3013dec02f8b61c18fde9f5c · covers:the write lock is not the ledger lock

## Invariants

- A single writer on a checkout behaves exactly as before.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-075 Out of Scope)

## Stop Condition

Stop if the lock needs the target file or a package the record does not govern.
