# Task ADR-079-T1: one lock per state file

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `state.Hold`; `iter.Update`; the tally's `locked` wrappers; `mrw seen` through `seen.Snapshot`
**Consumes:** `seen.Snapshot`, the platform flock
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a second holder waits`, `racing updates keep every entry`, `racing records keep every count`, `a reader never sees a half-saved file`, `mrw seen reads under the lock`, `a writer that cannot lock writes nothing`, `every tally call waits for the lock`, `migration takes the destination's lock`, `a contract row drives the binary`, `the packages vet for Windows`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

The working set and the tally were rewritten with no lock, so racing processes lost entries and counts
— or read an emptied file and wiped the lot — and `mrw seen` could print an empty ledger.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/lock.go`, `lock_unix.go`, `lock_windows.go` | new | `Hold`, moved from `seen` |
| `internal/seen/seen.go`, `lock_unix.go`, `lock_windows.go` | edit, removed | `seen` uses `state.Hold` |
| `internal/iter/iter.go` | edit | `Load` locked; `Update` |
| `internal/state/state.go` | edit | `Migrate` holds each destination's lock across its check and copy |
| `internal/authoring/authoring.go` | edit | `locked`; exported wrappers over unexported bodies |
| `cmd/mrw/main.go` | edit | `mrw iter` through `Update`; `mrw seen` through `Snapshot` |
| `*079_test.go` | new | the tests below |
| `scripts/contract.sh` | edit | §160 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: `Hold` takes no lock; `Update` takes no lock; `locked` takes no lock; `mrw seen` back on `seen.Load`; `locked` runs a writer it could not lock; `Migrate` takes no lock.
3. [S3] Contract §160. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/state/ ./internal/iter/ ./internal/authoring/ ./internal/seen/ ./cmd/mrw/ -count=1 -timeout 240s -run 'TestAStateLockExcludesASecondHolder|TestConcurrentUpdatesKeepEveryEntry|TestAnUpdateThatFailsWritesNothing|TestConcurrentRecordsCountEveryOutcome|TestATallyReadNeverSeesAHalfSavedTally|TestSeenNeverPrintsAHalfSavedLedger|TestConcurrentRecordsKeepEveryPath|TestAWriterThatCannotLockWritesNothing|TestEveryTallyCallWaitsForTheLock|TestALoadWaitsWhileTheSetIsHeld|TestMigrateWaitsForTheDestinationsLock' -v 2>&1 | tee /tmp/adr079-T1.out \
  && missing=$(for t in TestAStateLockExcludesASecondHolder TestConcurrentUpdatesKeepEveryEntry TestAnUpdateThatFailsWritesNothing TestConcurrentRecordsCountEveryOutcome TestATallyReadNeverSeesAHalfSavedTally TestSeenNeverPrintsAHalfSavedLedger TestConcurrentRecordsKeepEveryPath TestAWriterThatCannotLockWritesNothing TestEveryTallyCallWaitsForTheLock TestALoadWaitsWhileTheSetIsHeld TestMigrateWaitsForTheDestinationsLock; do grep -qE "^--- PASS: $t \(" /tmp/adr079-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 160\. ' scripts/contract.sh \
  && GOOS=windows go vet ./internal/state/ ./internal/seen/ ./internal/iter/ ./internal/authoring/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/check internal/lines internal/subproc internal/rooted internal/read \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/check internal/lines internal/subproc internal/rooted internal/read)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAStateLockExcludesASecondHolder` | `internal/state/lock079_test.go` | a second holder waits; a release twice is harmless | — | S1, S2 |
| `TestConcurrentUpdatesKeepEveryEntry` | `internal/iter/update079_test.go` | 24 racing updates keep 24 entries | — | S1, S2 |
| `TestAnUpdateThatFailsWritesNothing` | `internal/iter/update079_test.go` | the change's own error comes back and nothing is written | — | S1, S2 |
| `TestConcurrentRecordsCountEveryOutcome` | `internal/authoring/lock079_test.go` | 16 × 20 racing records count 320 | — | S1, S2 |
| `TestATallyReadNeverSeesAHalfSavedTally` | `internal/authoring/lock079_test.go` | a reader's count never falls while a writer records | — | S1, S2 |
| `TestSeenNeverPrintsAHalfSavedLedger` | `cmd/mrw/bookkeeping079_test.go` | 400 `mrw seen` beside a rewriting writer all list the file | — | S1, S2 |
| `TestConcurrentRecordsKeepEveryPath` | `internal/seen/seen_test.go` | ADR-038's ledger lock, now `state.Hold`, unchanged | — | S2 |
| `TestAWriterThatCannotLockWritesNothing` | `internal/authoring/lock079_test.go` | with the lock unopenable and the tally writable, no writer writes, a reader reads, a reset fails | — | S2 |
| `TestEveryTallyCallWaitsForTheLock` | `internal/authoring/lock079_test.go` | each of the eight exported calls waits while another holder has the lock | — | S2 |
| `TestALoadWaitsWhileTheSetIsHeld` | `internal/iter/update079_test.go` | `iter.Load` waits while the set's lock is held | — | S2 |
| `TestMigrateWaitsForTheDestinationsLock` | `internal/state/lock079_test.go` | a legacy copy waits for the destination's lock, and the live set wins | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the functions under Produces |
| 2 — something selects it | every write, every `mrw iter`, every `mrw stats` and `mrw seen` |
| 3 — the caller can discover it | `mrw stats` and `mrw iter` show counts and entries that no longer go missing |
| 4 — it is used | the v1.25.1 round's peers ran racing writers on one checkout |

## Verification Log
(empty until execute)
- 2026-09-26 · 686cb60* · exit 1 · `set -o pipefail …` · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · ms:1630 · test-lock-sha256:2adf7ec5da927722ee5c783c5289cae7a8c0e8eb5c128363ec930eb23eef97f9 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYm9va2tlZXBpbmcwNzlfdGVzdC5nbwlUZXN0QUNsZWFuRHJ5UnVuUmVjb3Jkc05vdGhpbmcJNzNlMTdhMDg2MDIwOTQ3YWQ0YTk2YmY1YmE2ZjE4MjJhM2NkYTBmODY1YTg0MTA5ZjkxMDEwZTQyNjI0NzEyMwpib2R5CWNtZC9tcncvYm9va2tlZXBpbmcwNzlfdGVzdC5nbwlUZXN0U2Vlbk5ldmVyUHJpbnRzQUhhbGZTYXZlZExlZGdlcglkNGFlNDI3ZmZjMjliZTg2ODFlYjQ3MDljOTQxM2VjM2ZkM2RmN2RjNTNjOTllM2I3MTJiODU3NzdkMWZjMzRiCmJvZHkJaW50ZXJuYWwvYXV0aG9yaW5nL2xvY2swNzlfdGVzdC5nbwlUZXN0QVRhbGx5UmVhZE5ldmVyU2Vlc0FIYWxmU2F2ZWRUYWxseQlmODIyZWViMmUxOWExMDkzOTY4MGJjMmM3NDAzNzMwMmM2ZGNkZDQyYTA1MDU3M2RlOTA0ZmFmMmU3ZmQzYWM5CmJvZHkJaW50ZXJuYWwvYXV0aG9yaW5nL2xvY2swNzlfdGVzdC5nbwlUZXN0Q29uY3VycmVudFJlY29yZHNDb3VudEV2ZXJ5T3V0Y29tZQk5NGMwYTgyOWY2YWVkNGRhYzA0ODIyYjU2MThmZDk4MTA0ZTBlNzU5ODEwNTRlMWRjNjY4ZWM2Nzc0YTViNjIyCmJvZHkJaW50ZXJuYWwvaXRlci91cGRhdGUwNzlfdGVzdC5nbwlUZXN0QW5VcGRhdGVUaGF0RmFpbHNXcml0ZXNOb3RoaW5nCTc4NmE3NGZkMjBmZDYxNzg2MWI0ODE3YTFmYWYwY2VkOTExYjg3NTQwNTNkMDYyNTNkOTNiNmVlZGY2ZjM0MDAKYm9keQlpbnRlcm5hbC9pdGVyL3VwZGF0ZTA3OV90ZXN0LmdvCVRlc3RDb25jdXJyZW50VXBkYXRlc0tlZXBFdmVyeUVudHJ5CWEyMmM5OGFhMWFiNjU5OTU4NjNjYjdmZmQyOTA3MjExNjAzODY5OTgzODNlMDNiOTQ0ZjBiNjU1Njc3Y2RhOTAKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QUxlZGdlclRoaXNCdWlsZFdyb3RlTG9hZHNCYWNrCWQxZDZiODk0ZTUxZWZmYWYwMDYyNDVkN2Y3Y2VhMTU2OGEyMzZjMmU0YTViYWI1Yzk3NjQzYjNjNjE2N2EwMDAKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QUxlZ2FjeVBhdGhDb250YWluaW5nQURvdWJsZVNwYWNlSXNPbmVQYXRoCWNiN2I1Y2NkNDRjMzE5NGFmODk1YzM4ZTA5ZDYyYjgwNDNkZmYxMWQ3MmUwYjUyMjc1NTk4NjJhYmJhMmFmMTUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QVByZVYySW5UcmVlTGVkZ2VySXNEaXNjYXJkZWROb3RUcnVzdGVkCTk2MGNmMjllYmY2ZmEyZDJiOThlYmFmYWU2YjM1NDg3NjgzYzAzNzY3NWM1NGY0OGIxMThiZjc3NDkwZWFmZWUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QVYyTGVkZ2VySXNBY2NlcHRlZEFuZFRoZUJ1bXBJc0RlbGliZXJhdGUJNDljNTVjMzM3OGY5NjU3MGQ5Y2M2MDZmNTVhOWNkN2E1MGU2OTBjZDI0ODA1MGI4Y2E0MjBjYjhiM2ZmZDk3ZQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RDb25jdXJyZW50UmVjb3Jkc0tlZXBFdmVyeVBhdGgJZWI5YTZjOTUwMTJlYzYzNWE0ZTc1MTliNzVjNzYxMzRjYjk3YzAwNGNlZDBmMjEyZjdiODlhMjNiNDg2OTUzYQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RMZWRnZXJJc05vdFdyaXR0ZW5JbnRvVGhlUm9vdAk0NDFlOWY3OGE0YTkzMmJmNjJiOTZiMjIwNjIyNzdhMDI3ZTg3YjdkYzI3OGVlYThlNGQ4YzI4NmE4YTE4NGE2CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdExlZ2FjeUluVHJlZUxlZGdlcklzU3RpbGxSZWFkCWM0ZGE4Yzc0MzJkYTNiMzUzNDdmYzhlMzAyOGEzYjE2ODg0OWZjNTIzOWY5OTRhNTMwYjllODE4Mzg1OGM4YzUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0TWFpbgkzODNhMzc0MGViNWIxMzcyNDA0M2U2ZjdkYTFiOGFiYmJlNzE1Y2E4Y2E0YzUyMWMzYzc2NWQ5MDFjOTgyZmQ0CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdE1pc3NpbmdMZWRnZXJJc0VtcHR5Tm90QW5FcnJvcgk1ODU3M2Q4NGZjYWU1ZTUwY2ExMDIzMzA2OGQzNjI2ZGEzYTZlMjFmMzljNWY5YzE3MjI2ZDI1ZDkzZWMwNWU4CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJlY29yZE1lcmdlc1JhdGhlclRoYW5SZXBsYWNlcwkyMzdiODIwNTgzZmY5YmJiODZkOGJmM2I0NWY3OTVjZjY1YzJlNGM0MzI3MTRiMGI5YmQ4YTNjYTg2MDEyMWFjCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJlY29yZE92ZXJ3cml0ZXNUaGVTYW1lUGF0aAk1YmM2MDA2NGNkZjMxNmU2MTJkNzI4ODA5NDE3OTc0ZTRlMDU5MzQ3NGM2MGJmYjViMDJlNjA2M2VjMTJmOWE3CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJvdW5kVHJpcAkxZTE4MjhmMDJlMjM4ZWNmYzA4ZjBmNTJhODZiNmRkMGQ4ZWI4MDg2NjY1OWJhNDAwYWJmZWUzZjY1NzgwMjJmCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFNIQUlzU3RhYmxlCTI5ZDNlZWI0YzhmMjcxYzRlNTk1MzIxYzdhZmU3ZTlmMjM2YWNiMTQxMDEyYzljNjc0NGRmMjQxM2Q2MmExOGUKYm9keQlpbnRlcm5hbC9zdGF0ZS9sb2NrMDc5X3Rlc3QuZ28JVGVzdEFTdGF0ZUxvY2tFeGNsdWRlc0FTZWNvbmRIb2xkZXIJOWRhYzM5ZDNmNjU1YzA4MGZmYTIxMjA1ZjVmNTBjODE3NTFkZDIyNDQzOTBlYzI4ZjJmN2NlMDIwMDlkOWQwNw
  ```
  --- last 10 line(s) of stdout (of 30 after folding 30 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen	0.221s
  === RUN   TestSeenNeverPrintsAHalfSavedLedger
      bookkeeping079_test.go:40: run 37: `mrw seen` printed a ledger without a.txt (exit 0):
          /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestSeenNeverPrintsAHalfSavedLedger62960981/001/mrw/276078ecbfb1b4ca
          # 1 state directory under /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestSeenNeverPrintsAHalfSavedLedger62960981/001/mrw
          0 file(s) seen
  --- FAIL: TestSeenNeverPrintsAHalfSavedLedger (0.02s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.205s
  FAIL
  ```
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · ms:1629
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · ms:1514
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · ms:900
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · ms:967
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · ms:941
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · ms:963
- 2026-09-26 · 0b665d5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:0 · test-lock-sha256:17a5dbd37cf818b8dbb8e63500d9f3d76f3b692da1c9344978a813eb14c82bfc · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYm9va2tlZXBpbmcwNzlfdGVzdC5nbwlUZXN0QUNsZWFuRHJ5UnVuUmVjb3Jkc05vdGhpbmcJNzNlMTdhMDg2MDIwOTQ3YWQ0YTk2YmY1YmE2ZjE4MjJhM2NkYTBmODY1YTg0MTA5ZjkxMDEwZTQyNjI0NzEyMwpib2R5CWNtZC9tcncvYm9va2tlZXBpbmcwNzlfdGVzdC5nbwlUZXN0U2Vlbk5ldmVyUHJpbnRzQUhhbGZTYXZlZExlZGdlcglkNGFlNDI3ZmZjMjliZTg2ODFlYjQ3MDljOTQxM2VjM2ZkM2RmN2RjNTNjOTllM2I3MTJiODU3NzdkMWZjMzRiCmJvZHkJaW50ZXJuYWwvYXV0aG9yaW5nL2xvY2swNzlfdGVzdC5nbwlUZXN0QVRhbGx5UmVhZE5ldmVyU2Vlc0FIYWxmU2F2ZWRUYWxseQlmODIyZWViMmUxOWExMDkzOTY4MGJjMmM3NDAzNzMwMmM2ZGNkZDQyYTA1MDU3M2RlOTA0ZmFmMmU3ZmQzYWM5CmJvZHkJaW50ZXJuYWwvYXV0aG9yaW5nL2xvY2swNzlfdGVzdC5nbwlUZXN0QVdyaXRlclRoYXRDYW5ub3RMb2NrV3JpdGVzTm90aGluZwk0YzNiNGM5MDVkNWYwMzk5YWVjMzQ2NDEyM2NiODc4ODc0YjdjYTQwY2EwYzk0ZmYxZjg3OGMxNTViZWIxNmQzCmJvZHkJaW50ZXJuYWwvYXV0aG9yaW5nL2xvY2swNzlfdGVzdC5nbwlUZXN0Q29uY3VycmVudFJlY29yZHNDb3VudEV2ZXJ5T3V0Y29tZQk5NGMwYTgyOWY2YWVkNGRhYzA0ODIyYjU2MThmZDk4MTA0ZTBlNzU5ODEwNTRlMWRjNjY4ZWM2Nzc0YTViNjIyCmJvZHkJaW50ZXJuYWwvYXV0aG9yaW5nL2xvY2swNzlfdGVzdC5nbwlUZXN0RXZlcnlUYWxseUNhbGxXYWl0c0ZvclRoZUxvY2sJNjI5ODJlNDVhNWNjZWJmYWUzMzU2ZTc0N2VjMDVmNWY0ZDYwOWM0Y2MwZTA4MjAxNDNkYTJjYmJkZmQ5YzY0Nwpib2R5CWludGVybmFsL2l0ZXIvdXBkYXRlMDc5X3Rlc3QuZ28JVGVzdEFMb2FkV2FpdHNXaGlsZVRoZVNldElzSGVsZAlhY2E4Y2U5YzRhYWVmOWY2YjI4OTkwYTkyMGY0MzNjYzc2ZTA3MTU1NzcxOGFlNTRmZGNiN2MyNmI2NTQ0M2FiCmJvZHkJaW50ZXJuYWwvaXRlci91cGRhdGUwNzlfdGVzdC5nbwlUZXN0QW5VcGRhdGVUaGF0RmFpbHNXcml0ZXNOb3RoaW5nCTc4NmE3NGZkMjBmZDYxNzg2MWI0ODE3YTFmYWYwY2VkOTExYjg3NTQwNTNkMDYyNTNkOTNiNmVlZGY2ZjM0MDAKYm9keQlpbnRlcm5hbC9pdGVyL3VwZGF0ZTA3OV90ZXN0LmdvCVRlc3RDb25jdXJyZW50VXBkYXRlc0tlZXBFdmVyeUVudHJ5CWEyMmM5OGFhMWFiNjU5OTU4NjNjYjdmZmQyOTA3MjExNjAzODY5OTgzODNlMDNiOTQ0ZjBiNjU1Njc3Y2RhOTAKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QUxlZGdlclRoaXNCdWlsZFdyb3RlTG9hZHNCYWNrCWQxZDZiODk0ZTUxZWZmYWYwMDYyNDVkN2Y3Y2VhMTU2OGEyMzZjMmU0YTViYWI1Yzk3NjQzYjNjNjE2N2EwMDAKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QUxlZ2FjeVBhdGhDb250YWluaW5nQURvdWJsZVNwYWNlSXNPbmVQYXRoCWNiN2I1Y2NkNDRjMzE5NGFmODk1YzM4ZTA5ZDYyYjgwNDNkZmYxMWQ3MmUwYjUyMjc1NTk4NjJhYmJhMmFmMTUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QVByZVYySW5UcmVlTGVkZ2VySXNEaXNjYXJkZWROb3RUcnVzdGVkCTk2MGNmMjllYmY2ZmEyZDJiOThlYmFmYWU2YjM1NDg3NjgzYzAzNzY3NWM1NGY0OGIxMThiZjc3NDkwZWFmZWUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QVYyTGVkZ2VySXNBY2NlcHRlZEFuZFRoZUJ1bXBJc0RlbGliZXJhdGUJNDljNTVjMzM3OGY5NjU3MGQ5Y2M2MDZmNTVhOWNkN2E1MGU2OTBjZDI0ODA1MGI4Y2E0MjBjYjhiM2ZmZDk3ZQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RDb25jdXJyZW50UmVjb3Jkc0tlZXBFdmVyeVBhdGgJZWI5YTZjOTUwMTJlYzYzNWE0ZTc1MTliNzVjNzYxMzRjYjk3YzAwNGNlZDBmMjEyZjdiODlhMjNiNDg2OTUzYQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RMZWRnZXJJc05vdFdyaXR0ZW5JbnRvVGhlUm9vdAk0NDFlOWY3OGE0YTkzMmJmNjJiOTZiMjIwNjIyNzdhMDI3ZTg3YjdkYzI3OGVlYThlNGQ4YzI4NmE4YTE4NGE2CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdExlZ2FjeUluVHJlZUxlZGdlcklzU3RpbGxSZWFkCWM0ZGE4Yzc0MzJkYTNiMzUzNDdmYzhlMzAyOGEzYjE2ODg0OWZjNTIzOWY5OTRhNTMwYjllODE4Mzg1OGM4YzUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0TWFpbgkzODNhMzc0MGViNWIxMzcyNDA0M2U2ZjdkYTFiOGFiYmJlNzE1Y2E4Y2E0YzUyMWMzYzc2NWQ5MDFjOTgyZmQ0CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdE1pc3NpbmdMZWRnZXJJc0VtcHR5Tm90QW5FcnJvcgk1ODU3M2Q4NGZjYWU1ZTUwY2ExMDIzMzA2OGQzNjI2ZGEzYTZlMjFmMzljNWY5YzE3MjI2ZDI1ZDkzZWMwNWU4CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJlY29yZE1lcmdlc1JhdGhlclRoYW5SZXBsYWNlcwkyMzdiODIwNTgzZmY5YmJiODZkOGJmM2I0NWY3OTVjZjY1YzJlNGM0MzI3MTRiMGI5YmQ4YTNjYTg2MDEyMWFjCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJlY29yZE92ZXJ3cml0ZXNUaGVTYW1lUGF0aAk1YmM2MDA2NGNkZjMxNmU2MTJkNzI4ODA5NDE3OTc0ZTRlMDU5MzQ3NGM2MGJmYjViMDJlNjA2M2VjMTJmOWE3CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJvdW5kVHJpcAkxZTE4MjhmMDJlMjM4ZWNmYzA4ZjBmNTJhODZiNmRkMGQ4ZWI4MDg2NjY1OWJhNDAwYWJmZWUzZjY1NzgwMjJmCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFNIQUlzU3RhYmxlCTI5ZDNlZWI0YzhmMjcxYzRlNTk1MzIxYzdhZmU3ZTlmMjM2YWNiMTQxMDEyYzljNjc0NGRmMjQxM2Q2MmExOGUKYm9keQlpbnRlcm5hbC9zdGF0ZS9sb2NrMDc5X3Rlc3QuZ28JVGVzdEFTdGF0ZUxvY2tFeGNsdWRlc0FTZWNvbmRIb2xkZXIJOWRhYzM5ZDNmNjU1YzA4MGZmYTIxMjA1ZjVmNTBjODE3NTFkZDIyNDQzOTBlYzI4ZjJmN2NlMDIwMDlkOWQwNwpib2R5CWludGVybmFsL3N0YXRlL2xvY2swNzlfdGVzdC5nbwlUZXN0TWlncmF0ZVdhaXRzRm9yVGhlRGVzdGluYXRpb25zTG9jawk3NmQ3NGFkNjkzMDNiMmQ2NWE4YWE2NzVjOTdkOWIwYTk3OTFlOGZlMWI3MTlmZTVlOTZjZWRmOWM3NmJlZjVl · test-lock-kind:replace
- 2026-09-26 · 0b665d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:4732
- 2026-09-26 · 0b665d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:4003
- 2026-09-26 · 0b665d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:3920
- 2026-09-26 · 0b665d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:3344
- 2026-09-26 · 0b665d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:2753
- 2026-09-26 · 0b665d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:3917
- 2026-09-26 · 0b665d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:4352
- 2026-09-26 · 0b665d5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · ms:4188

## Mutation Log
(empty until execute)
- 2026-09-26 · 686cb60* · mutant killed · exit 1 · `internal/state/lock_unix.go` · Hold takes no lock · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · covers:a second holder waits
- 2026-09-26 · 686cb60* · mutant killed · exit 1 · `internal/iter/iter.go` · Update takes no lock · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · covers:racing updates keep every entry
- 2026-09-26 · 686cb60* · mutant killed · exit 1 · `internal/authoring/authoring.go` · locked takes no lock · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · covers:racing records keep every count
- 2026-09-26 · 686cb60* · mutant killed · exit 1 · `cmd/mrw/main.go` · mrw seen back on seen.Load · acceptance-sha256:560d97734e970c7cf80f397df034834a38d6158c5942697728800291cd3cc111 · covers:mrw seen reads under the lock
- 2026-09-26 · 0b665d5* · mutant killed · exit 1 · `internal/state/lock_unix.go` · Hold takes no lock · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · covers:a second holder waits
- 2026-09-26 · 0b665d5* · mutant killed · exit 1 · `internal/iter/iter.go` · Update takes no lock · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · covers:racing updates keep every entry
- 2026-09-26 · 0b665d5* · mutant killed · exit 1 · `cmd/mrw/main.go` · mrw seen back on seen.Load · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · covers:mrw seen reads under the lock
- 2026-09-26 · 0b665d5* · mutant killed · exit 1 · `internal/authoring/authoring.go` · locked runs a writer it could not lock · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · covers:a writer that cannot lock writes nothing
- 2026-09-26 · 0b665d5* · mutant killed · exit 1 · `internal/state/state.go` · Migrate takes no lock the writers share · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · covers:migration takes the destination's lock
- 2026-09-26 · 0b665d5* · mutant killed · exit 1 · `internal/authoring/authoring.go` · RecordPricing takes no lock · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · covers:every tally call waits for the lock
- 2026-09-26 · 0b665d5* · mutant killed · exit 1 · `internal/authoring/authoring.go` · locked takes no lock · acceptance-sha256:5dc06969c5abd5fe0533384dfc693b0f1316cc3ca19b90a4f42a625b0b14cf1e · covers:racing records keep every count

## Invariants

- No mrw process reads a state file another is rewriting, and none loses another's update.

## Risks

- A stopped holder makes the next process wait (Out of Scope in the record).

## Out of Scope

- Rename-over saves (deferred: ADR-043; docs/adr/BACKLOG.md "Torn `Load`")

## Stop Condition

The fence exits 0 and every Mutation Log row reads `killed`.
