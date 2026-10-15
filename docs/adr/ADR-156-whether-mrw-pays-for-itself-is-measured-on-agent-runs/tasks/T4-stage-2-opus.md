# Task ADR-156-T4: Stage 2, our own tasks on Opus 5.5

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** M (machine time: hours, rate-limited)
**Owner:** Zy
**Produces:** `docs/bench/stage-2-opus/` receipts
**Consumes:** `bench run` (T2), `bench/tasks/` (T1)
**Data dependency:** real agent runs on the subscription, on this Mac; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `every registered attempt has a receipt`, `the receipts verify`, `the stage ran exactly the unsuperseded corpus on Opus`

## Goal

The registered Stage 2 (BACKLOG, "Pre-registered for ADR-156 T4", Zy 2026-10-15): a pilot, then every measured task not superseded — 40 — × 2 configurations × 2 repeats on Opus 5.5, on this Mac.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-2-opus/pilot/` | add | Stage 1b's five pilot tasks × 2 × 1 on Opus, stage `pilot`, not counted |
| `docs/bench/stage-2-opus/` | add | the stage's receipts, `run.json` and canary |

## Ordered Steps

1. [S1] Write the failing check and see it red: the fence exits non-zero while the directory is missing or incomplete. [proof: acceptance]
2. [S2] Zy's word to start, recorded. [proof: human: Zy says go for Stage 2]
3. [S3] The canary on the pinned CLI, then the pilot; every pilot transcript is read. A harness defect is fixed and the pilot rerun into a new directory, the old one kept; a task is superseded after the pilot only for an oracle defect shown on its base or its reference fix. [proof: human: the executor reads every pilot receipt and transcript]
4. [S4] The full matrix, `--stage stage2`, sequential, resumable, on this Mac. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go run ./cmd/bench verify --stage stage2 --tasks 40 --repeats 2 docs/bench/stage-2-opus \
  && jq -e '.models == ["claude-opus-5-5"]' docs/bench/stage-2-opus/run.json > /dev/null \
  && diff <(jq -r '.tasks[]' docs/bench/stage-2-opus/run.json | sort) \
          <(for f in bench/tasks/*/task.json; do jq -r 'select(.shape != "fixture" and (has("superseded_by") | not)) | .id' "$f"; done | sort)
```

The corpus compared is the one in the tree. If a task is superseded after Stage 2 runs, this fence is pinned to the commit the stage ran from, as ADR-159 Decision 3 did for Stage 1b.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | earlier stages' receipts survive Stage 2 (T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench verify`; T6 |
| 3 — the caller can discover it | `docs/bench/stage-2-opus/` |
| 4 — it is used | T6 |

## Mutation Log
(empty until execute)
- 2026-10-15 · fe000db8* · mutant killed · exit 1 · `docs/bench/stage-2-opus/run.json` · the fence binds the receipts to the stage they were run as · acceptance-sha256:7bb3a825e52cd5331b4b6408a3ec8d3313f7825a9d930eaec22ff19e06290b4f
- 2026-10-15 · fe000db8* · mutant killed · exit 1 · `docs/bench/stage-2-opus/run.json` · the fence binds run.json to the unsuperseded corpus in the tree · acceptance-sha256:7bb3a825e52cd5331b4b6408a3ec8d3313f7825a9d930eaec22ff19e06290b4f

## Verification Log
(empty until execute)
- 2026-10-15 · fb76bd07* · exit 1 · `set -o pipefail …` · acceptance-sha256:7bb3a825e52cd5331b4b6408a3ec8d3313f7825a9d930eaec22ff19e06290b4f · ms:648 · test-lock-sha256:3f528f47a4399e2d00c863d847852b3a72f1f7b26edde8d11544b6e00beec3b3 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFGYWlsaW5nRXhpdElzQW5FcnJvckFuZEFuSW50ZXJydXB0U3RvcHNUaGVTdGFnZQkxMDQ1NmIzMWVlYTYzNjc5ZGEyZTQzYWZmODdmNGZjMzI5NzgzNGQ2MDg0MjBlYTY2ODg2NzUzYWRmODJjOWRlCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdGFnZVdpdGhBTWlzc2luZ0RlY2xhcmVkU2VjdGlvbkRvZXNOb3RTdGFydAk1MGFjYjFiNzFjYmQ2MDUwYmE2N2MwNzE0ZjNmYzY0ZDE3MWZiZjgzZjc2OGM2NGJhNmI1MmRkMTQ3ZGM1NjQ2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdXBlcnNlZGVkVGFza0lzTm90UnVuQnlEZWZhdWx0CTNmMDg0NTc3YWFmYWQ5MDFkZmIxMTNkYjBkNTQwZjZkOWRmNDA5YmNiMzU4OGY4NDNkOTkyMTJlYmQ3MGEzNDMKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5BdHRlbXB0TGVhdmVzTm90aGluZ1J1bm5pbmcJMzhkOGU1ZTJlOTY1MTljZTIxMDRkOTNkNGYwYzU5MTEyZmQ0ODRkZDVmNDc2NmQ1ZGJiMzc5N2FjOWQ4Njk2ZQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBbkF0dGVtcHRSZWFkc05vbmVPZlRoZUhvc3RzU2hlbGxPckdpdFNldHVwCWZhYWU0YTk1YjBjODhiNDg1ZWIxM2M0ZmNkNGZmNTJjZmUxMWVhMThiNDI5N2NlYzFkMjdmMGM0NGU0ZGE2Y2IKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5FeGNsdWRlZEF0dGVtcHRJc1JlcGxhY2VkQW5kS2VwdAk3ZjE5MjY5YjJlNDg3ZjYyYzY1MmZjNTk4MWM2ODM5M2M1ZmJkYTM3ZmYxZDkzYWYxNWRhOWY5OTE5ZDJlNjI2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEJvdGhDb25maWd1cmF0aW9uc0tlZXBUaGVTeXN0ZW1Qcm9tcHRTdGFibGUJMzhmMjNjMzY5YTBiNDk5ZWVkYTM1ZGY1N2U0ODIzZDJjOGUyN2ZlZmU0ODY0MWQzYmUwNTM4ZTljMDNmNzZmNgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RDb25maWd1cmF0aW9uQUhhc05vTXJ3T25JdHNQYXRoCWYwNzRiOTUxNDZhNzliOGI4NzBhNDBjNzY0NDRjMjRkMjQ4MDc3ZGQ4NjJiZmRjMTNlNTU4YTZhZWFhMGFhOGYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ2FuYXJ5UnVuc0JvdGhDb25maWd1cmF0aW9uc0FuZFJlY29yZHNJdHNWZXJkaWN0CTVmNjBhOTkwNWE2OTNlNGUzZWIzNGFmZGVlODkyZTg5NDQxZGU4NmUxNGMwMWRlOWE4NWQ0ODY2ZGIzODNiODUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ29uZmlnRGlyZWN0b3J5SXNKdWRnZWRCeUlkZW50aXR5CTFlOWVmYzdkNjFhMWI2Y2QwMjYyZGIxYzhhMDE4NjNlYzU1MDJlNTA5YzNhZDc0MjkxOTY2MmE3NDdjNmJiMGQKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVjb3Jkc0FSZWNlaXB0UGVyQXR0ZW1wdEZyb21BU3R1YkNsYXVkZQkwNTA4YjMzNzZjNzZiODBkY2QyNzRiNWI1MzliN2VhYzRjNDRjNDY0MjZiNjBkYzkxZmQ0OWViZWM1M2VmOTA5CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNUb092ZXJ3cml0ZUFuRWFybGllclJ1bnNSZWNlaXB0cwk4ZGM0OWE3NmVjMGQ0NTI2MzQ5YjAxNDhkMWU2MzU0NGE2Mzc1NTUyMGViNzU4M2U5YzcxMWYyMzljMzc0NDg3CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNXaXRob3V0VGhlR2F0ZXNaeUhvbGRzCTVhYTE0ZDI2ZWU5YTNhYmMzMzg2OWM0NGNmOTBkYjA3OGE2ZTY1OWI3MDNhNjdjNDEyMDg0MDkyMWQ1NjkzZjgKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVzdW1lc1dpdGhvdXRSZXBlYXRpbmdBU3RhcnRlZEF0dGVtcHQJYjIwMGRjNGU5ODkzOThhZTVmN2ZlZGMwYmZiYTY0MTZhYzRjODQ5YjlhM2Q2ODJkODQ5ZmRiMDQwYTMyN2EyNApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlOYW1lc0FuQXR0ZW1wdFdpdGhOb1JlY2VpcHQJMzQyMDZkZGI0NjYwM2MzNjBiMGE5MDM1YzA4ODJiMTkwMzZmZTdjOTk5YmM2YTRmZTFhZWZiMDJjYWJkYmRkMgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlSZWZ1c2VzQVJlY2VpcHRUaGF0SXNOb3RJdHNBdHRlbXB0CWZmMjhhYmNiNWM5Njc3NDZkYWFmYjdkNzNlOWZiMDIyYWI4NGU5OTU4YzlkODNmMWIwZjE4MmRmZTczZjJhODU
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-2-opus/run.json: no such file or directory
  exit status 1
  ```
- 2026-10-15 · human-observed · S2 approved: Zy said go for Stage 2 on 2026-10-15, choosing 'ADR-156 T4–T6' and then 'Current 40' (every measured task not superseded, 40, on Opus 5.5), registered in BACKLOG before any Stage 2 run
- 2026-10-15 · human-observed · S3 observed 2026-10-15 on this Mac, CLI 2.1.300, mrw v1.66.0 (05ec3f8), bench at 5b77a44c (a detached worktree, so branch work in the checkout leaves the corpus pinned): canary passed on claude-opus-5-5; pilot of Stage 1b's five pilot tasks x A/B x 1 on Opus, every receipt and every transcript's tool calls, errors and result read: 10/10 solved, no silent mis-edit, no exclusion, empty stderr, no one-turn attempt; am-go-sha-message-2 B's automatic check found the test that asserts the old wording and the attempt left it, as the instruction says; B's one hook refusal was git stash in the tree (qh-vms-listing-timeout), the registered rule, and that attempt went on to solve; no harness defect, no task superseded
- 2026-10-15 · fe000db8* · exit 0 · `set -o pipefail …` · acceptance-sha256:7bb3a825e52cd5331b4b6408a3ec8d3313f7825a9d930eaec22ff19e06290b4f · ms:1026
- 2026-10-15 · fe000db8* · exit 0 · `set -o pipefail …` · acceptance-sha256:7bb3a825e52cd5331b4b6408a3ec8d3313f7825a9d930eaec22ff19e06290b4f · ms:671

## Invariants

- The mrw build, the CLI version and the host are one for the whole stage, and the first two are named in every receipt.

## Risks

- Opus costs more of the subscription's limits per run: the stage is paused, never cut, when a limit hits.

## Out of Scope

- Pooling with Stage 1 or Stage 1b (permanent: boundary: the bar gives one verdict per model and stage)

## Stop Condition

The fence exits 0; or the pilot shows the harness cannot measure what ADR-156 Decision 4 names, which goes back to Zy.
