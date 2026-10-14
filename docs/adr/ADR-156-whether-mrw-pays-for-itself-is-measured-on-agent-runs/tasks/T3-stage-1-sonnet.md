# Task ADR-156-T3: Stage 1, our own tasks on Sonnet 5.5

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (machine time: days, rate-limited)
**Owner:** Zy
**Produces:** `docs/bench/stage-1-sonnet/` receipts
**Consumes:** `bench run` (T2), `bench/tasks/` (T1)
**Data dependency:** real agent runs on the subscription; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `the pilot ran and found no harness defect`, `every registered attempt has a receipt`, `the receipts verify`

## Goal

The pilot, then the full Sonnet matrix on our own tasks, with every attempt's receipt kept.

**As run (2026-10-14):** the stage was started as `--stage stage1`, and every receipt and `run.json` carries that name; this fence was written for `stage-1-sonnet` and is changed to the name the receipts hold, since a receipt is never edited after its run. The first pilot (`pilot/`, CLI 2.1.299) found three harness defects, fixed in #413 and #414; its rerun (`pilot-2/`, CLI 2.1.300) found the prompt-cache asymmetry fixed in #415; the stage ran after #415 at 2d29f028 with the CLI pinned by its versioned path, mrw v1.66.0, seed 156. Each directory keeps the canary it ran under. Transcripts are not published: they stay on the machine that ran them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-1-sonnet/pilot/` | add | 5 tasks × 2 configurations × 1, not counted |
| `docs/bench/stage-1-sonnet/` | add | 30 tasks × 2 configurations × 2 repeats |

## Ordered Steps

1. [S1] Write the failing check and see it red: `bench verify docs/bench/stage-1-sonnet` exits non-zero while the directory is missing or incomplete. [proof: acceptance]
2. [S2] Zy's word to start, recorded. [proof: human: Zy says go for Stage 1]
3. [S3] The pilot; any harness defect it shows is fixed in T2 and the pilot rerun into a new directory, the old one kept. [proof: human: the executor reads every pilot receipt and transcript]
4. [S4] The full matrix, sequential, resumable. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go run ./cmd/bench verify --stage stage1 --tasks 30 --repeats 2 docs/bench/stage-1-sonnet
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerResumesWithoutRepeatingAStartedAttempt` | `internal/bench/runner_test.go` | the stage can stop and resume (T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench verify` and T6's analysis read them |
| 3 — the caller can discover it | `docs/bench/stage-1-sonnet/` |
| 4 — it is used | T6 |

## Mutation Log
(empty until execute)
- 2026-10-14 · 2d29f028* · mutant killed · exit 1 · `docs/bench/stage-1-sonnet/run.json` · the fence binds the receipts to the stage they were run as · acceptance-sha256:898be2e90082b0879fa93f82541e8d078b4b5989a56f5b9858e22b7d2681c9c5

## Verification Log
(empty until execute)
- 2026-10-14 · 2d29f028* · exit 0 · `set -o pipefail …` · acceptance-sha256:898be2e90082b0879fa93f82541e8d078b4b5989a56f5b9858e22b7d2681c9c5 · ms:987
- 2026-10-14 · human-observed · Zy's word to start observed: 'Pilot then full Stage 1' (2026-10-14), then 'okay, do it' after the pilot's rerun; the executor read every receipt of both pilots and of the stage, and the pilot transcripts of am-go-probe-k and cac-go-heartbeat. NOT done: reading every other pilot transcript
- 2026-10-14 · 2d29f028* · exit 1 · `set -o pipefail …` · acceptance-sha256:898be2e90082b0879fa93f82541e8d078b4b5989a56f5b9858e22b7d2681c9c5 · ms:356 · test-lock-sha256:55fab28246c591d80340b808b38c8a4e31347d53e1b6c73bfa0f2b38bed66049 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFGYWlsaW5nRXhpdElzQW5FcnJvckFuZEFuSW50ZXJydXB0U3RvcHNUaGVTdGFnZQkxMDQ1NmIzMWVlYTYzNjc5ZGEyZTQzYWZmODdmNGZjMzI5NzgzNGQ2MDg0MjBlYTY2ODg2NzUzYWRmODJjOWRlCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFuQXR0ZW1wdExlYXZlc05vdGhpbmdSdW5uaW5nCTM4ZDhlNWUyZTk2NTE5Y2UyMTA0ZDkzZDRmMGM1OTExMmZkNDg0ZGQ1ZjQ3NjZkNWRiYjM3OTdhYzlkODY5NmUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5FeGNsdWRlZEF0dGVtcHRJc1JlcGxhY2VkQW5kS2VwdAk3ZjE5MjY5YjJlNDg3ZjYyYzY1MmZjNTk4MWM2ODM5M2M1ZmJkYTM3ZmYxZDkzYWYxNWRhOWY5OTE5ZDJlNjI2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEJvdGhDb25maWd1cmF0aW9uc0tlZXBUaGVTeXN0ZW1Qcm9tcHRTdGFibGUJMzhmMjNjMzY5YTBiNDk5ZWVkYTM1ZGY1N2U0ODIzZDJjOGUyN2ZlZmU0ODY0MWQzYmUwNTM4ZTljMDNmNzZmNgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RDb25maWd1cmF0aW9uQUhhc05vTXJ3T25JdHNQYXRoCWYwNzRiOTUxNDZhNzliOGI4NzBhNDBjNzY0NDRjMjRkMjQ4MDc3ZGQ4NjJiZmRjMTNlNTU4YTZhZWFhMGFhOGYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ2FuYXJ5UnVuc0JvdGhDb25maWd1cmF0aW9uc0FuZFJlY29yZHNJdHNWZXJkaWN0CTVmNjBhOTkwNWE2OTNlNGUzZWIzNGFmZGVlODkyZTg5NDQxZGU4NmUxNGMwMWRlOWE4NWQ0ODY2ZGIzODNiODUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ29uZmlnRGlyZWN0b3J5SXNKdWRnZWRCeUlkZW50aXR5CTFlOWVmYzdkNjFhMWI2Y2QwMjYyZGIxYzhhMDE4NjNlYzU1MDJlNTA5YzNhZDc0MjkxOTY2MmE3NDdjNmJiMGQKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVjb3Jkc0FSZWNlaXB0UGVyQXR0ZW1wdEZyb21BU3R1YkNsYXVkZQkwNTA4YjMzNzZjNzZiODBkY2QyNzRiNWI1MzliN2VhYzRjNDRjNDY0MjZiNjBkYzkxZmQ0OWViZWM1M2VmOTA5CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNUb092ZXJ3cml0ZUFuRWFybGllclJ1bnNSZWNlaXB0cwk4ZGM0OWE3NmVjMGQ0NTI2MzQ5YjAxNDhkMWU2MzU0NGE2Mzc1NTUyMGViNzU4M2U5YzcxMWYyMzljMzc0NDg3CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNXaXRob3V0VGhlR2F0ZXNaeUhvbGRzCTVhYTE0ZDI2ZWU5YTNhYmMzMzg2OWM0NGNmOTBkYjA3OGE2ZTY1OWI3MDNhNjdjNDEyMDg0MDkyMWQ1NjkzZjgKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVzdW1lc1dpdGhvdXRSZXBlYXRpbmdBU3RhcnRlZEF0dGVtcHQJYjIwMGRjNGU5ODkzOThhZTVmN2ZlZGMwYmZiYTY0MTZhYzRjODQ5YjlhM2Q2ODJkODQ5ZmRiMDQwYTMyN2EyNApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlOYW1lc0FuQXR0ZW1wdFdpdGhOb1JlY2VpcHQJMzQyMDZkZGI0NjYwM2MzNjBiMGE5MDM1YzA4ODJiMTkwMzZmZTdjOTk5YmM2YTRmZTFhZWZiMDJjYWJkYmRkMgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlSZWZ1c2VzQVJlY2VpcHRUaGF0SXNOb3RJdHNBdHRlbXB0CWZmMjhhYmNiNWM5Njc3NDZkYWFmYjdkNzNlOWZiMDIyYWI4NGU5OTU4YzlkODNmMWIwZjE4MmRmZTczZjJhODU
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-1-sonnet/run.json: no such file or directory
  exit status 1
  ```
- 2026-10-14 · 2d29f028* · exit 0 · `set -o pipefail …` · acceptance-sha256:898be2e90082b0879fa93f82541e8d078b4b5989a56f5b9858e22b7d2681c9c5 · ms:158

## Invariants

- The mrw build and the CLI version are pinned for the whole stage and named in every receipt.

## Risks

- Rate limits stretch the stage over days: resumable by design.

## Out of Scope

- Changing a task after the pilot (permanent: boundary: a task changed after it was run is a new task, with a new id)

## Stop Condition

The fence exits 0; or the pilot shows the harness cannot measure what Decision 4 names, which goes back to Zy.
