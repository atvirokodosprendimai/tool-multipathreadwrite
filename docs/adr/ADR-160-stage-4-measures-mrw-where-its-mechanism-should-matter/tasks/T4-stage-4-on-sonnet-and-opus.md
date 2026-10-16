# Task ADR-160-T4: Stage 4 on Sonnet 5.5 and Opus 5.5

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** L (machine time: hours per model, rate-limited)
**Owner:** Zy
**Produces:** `docs/bench/stage-4-sonnet/`, `docs/bench/stage-4-opus/`, `docs/bench/stage-4-swe/` receipts
**Consumes:** the calibrated corpus (T3), `bench/stage-4-tasks.txt` and `bench/swe-bench/selection-adr-160.json` (T2)
**Data dependency:** real agent runs on the subscription and SWE-bench images, on this Mac; each model's stage starts only on Zy's word
**Proof map:** v1
**Rests-on:** `every registered attempt has a receipt`, `the receipts verify`, `the stages ran the task list on the registered models`, `the slice ran the selection on both models`

## Goal

The registered Stage 4: 24 own tasks × A/B × 2 on each model, and the 20-instance `swe-multi-hunk` slice × A/B × 1 on each model, one image at a time.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-4-sonnet/` | add | own tasks on Sonnet 5.5, stage `stage4` |
| `docs/bench/stage-4-opus/` | add | own tasks on Opus 5.5, stage `stage4` |
| `docs/bench/stage-4-swe/` | add | the slice on both models, stage `stage-4-swe` |

## Ordered Steps

1. [S1] See the fence red: no stage receipts. [proof: acceptance]
2. [S2] Zy's word for each of the three stages — Sonnet's own tasks, Opus's own tasks, the slice on both models — given with its price. [proof: human: Zy says go for that Stage 4 stage]
3. [S3] Each stage with `--task` for every id in `bench/stage-4-tasks.txt` (the slice with `--selection bench/swe-bench/selection-adr-160.json`), sequential, resumable, with a canary per model; every slice transcript scanned for a read of the dataset file, as in Stage 3. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go run ./cmd/bench verify --stage stage4 --tasks 24 --repeats 2 docs/bench/stage-4-sonnet \
  && go run ./cmd/bench verify --stage stage4 --tasks 24 --repeats 2 docs/bench/stage-4-opus \
  && jq -e '.models == ["claude-sonnet-5-5"]' docs/bench/stage-4-sonnet/run.json > /dev/null \
  && jq -e '.models == ["claude-opus-5-5"]' docs/bench/stage-4-opus/run.json > /dev/null \
  && diff <(jq -r '.tasks[]' docs/bench/stage-4-sonnet/run.json | sort) <(jq -r '.tasks[]' docs/bench/stage-4-opus/run.json | sort) \
  && diff <(jq -r '.tasks[]' docs/bench/stage-4-sonnet/run.json | sort) \
          <(sort bench/stage-4-tasks.txt) \
  && go run ./cmd/bench verify --stage stage-4-swe --tasks 20 --repeats 1 docs/bench/stage-4-swe \
  && jq -e '(.models | sort) == ["claude-opus-5-5", "claude-sonnet-5-5"]' docs/bench/stage-4-swe/run.json > /dev/null \
  && diff <(jq -r '.tasks[]' docs/bench/stage-4-swe/run.json | sort) <(jq -r '.selected[]' bench/swe-bench/selection-adr-160.json | sort)
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | earlier stages' receipts survive Stage 4 (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench verify`; `bench analyze` |
| 3 — the caller can discover it | `docs/bench/stage-4-*/` |
| 4 — it is used | T5 |

## Mutation Log
(empty until execute)

## Invariants

- The mrw build, the CLI version and the host are one for each stage, named in every receipt.
- Pilot receipts are never in a stage directory.

## Risks

- Opus may be at ceiling even on the calibrated corpus; that is a result, published as computed.

## Stop Condition

The fence exits 0; or a stage shows the harness cannot measure what ADR-156 Decision 4 names, which goes back to Zy.

## Out of Scope

- Pooling with Stages 1 to 3 (permanent: boundary: one verdict per model, source and stage)

## Verification Log
(empty until execute)
- 2026-10-16 · 8c4b3f31 · exit 1 · `set -o pipefail …` · acceptance-sha256:0d3ce45dc6cecf0440c82831586db187a5ee88f146eea9f548b5ce0ccb1c74c2 · ms:940 · test-lock-sha256:3f528f47a4399e2d00c863d847852b3a72f1f7b26edde8d11544b6e00beec3b3 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFGYWlsaW5nRXhpdElzQW5FcnJvckFuZEFuSW50ZXJydXB0U3RvcHNUaGVTdGFnZQkxMDQ1NmIzMWVlYTYzNjc5ZGEyZTQzYWZmODdmNGZjMzI5NzgzNGQ2MDg0MjBlYTY2ODg2NzUzYWRmODJjOWRlCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdGFnZVdpdGhBTWlzc2luZ0RlY2xhcmVkU2VjdGlvbkRvZXNOb3RTdGFydAk1MGFjYjFiNzFjYmQ2MDUwYmE2N2MwNzE0ZjNmYzY0ZDE3MWZiZjgzZjc2OGM2NGJhNmI1MmRkMTQ3ZGM1NjQ2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdXBlcnNlZGVkVGFza0lzTm90UnVuQnlEZWZhdWx0CTNmMDg0NTc3YWFmYWQ5MDFkZmIxMTNkYjBkNTQwZjZkOWRmNDA5YmNiMzU4OGY4NDNkOTkyMTJlYmQ3MGEzNDMKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5BdHRlbXB0TGVhdmVzTm90aGluZ1J1bm5pbmcJMzhkOGU1ZTJlOTY1MTljZTIxMDRkOTNkNGYwYzU5MTEyZmQ0ODRkZDVmNDc2NmQ1ZGJiMzc5N2FjOWQ4Njk2ZQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBbkF0dGVtcHRSZWFkc05vbmVPZlRoZUhvc3RzU2hlbGxPckdpdFNldHVwCWZhYWU0YTk1YjBjODhiNDg1ZWIxM2M0ZmNkNGZmNTJjZmUxMWVhMThiNDI5N2NlYzFkMjdmMGM0NGU0ZGE2Y2IKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5FeGNsdWRlZEF0dGVtcHRJc1JlcGxhY2VkQW5kS2VwdAk3ZjE5MjY5YjJlNDg3ZjYyYzY1MmZjNTk4MWM2ODM5M2M1ZmJkYTM3ZmYxZDkzYWYxNWRhOWY5OTE5ZDJlNjI2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEJvdGhDb25maWd1cmF0aW9uc0tlZXBUaGVTeXN0ZW1Qcm9tcHRTdGFibGUJMzhmMjNjMzY5YTBiNDk5ZWVkYTM1ZGY1N2U0ODIzZDJjOGUyN2ZlZmU0ODY0MWQzYmUwNTM4ZTljMDNmNzZmNgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RDb25maWd1cmF0aW9uQUhhc05vTXJ3T25JdHNQYXRoCWYwNzRiOTUxNDZhNzliOGI4NzBhNDBjNzY0NDRjMjRkMjQ4MDc3ZGQ4NjJiZmRjMTNlNTU4YTZhZWFhMGFhOGYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ2FuYXJ5UnVuc0JvdGhDb25maWd1cmF0aW9uc0FuZFJlY29yZHNJdHNWZXJkaWN0CTVmNjBhOTkwNWE2OTNlNGUzZWIzNGFmZGVlODkyZTg5NDQxZGU4NmUxNGMwMWRlOWE4NWQ0ODY2ZGIzODNiODUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ29uZmlnRGlyZWN0b3J5SXNKdWRnZWRCeUlkZW50aXR5CTFlOWVmYzdkNjFhMWI2Y2QwMjYyZGIxYzhhMDE4NjNlYzU1MDJlNTA5YzNhZDc0MjkxOTY2MmE3NDdjNmJiMGQKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVjb3Jkc0FSZWNlaXB0UGVyQXR0ZW1wdEZyb21BU3R1YkNsYXVkZQkwNTA4YjMzNzZjNzZiODBkY2QyNzRiNWI1MzliN2VhYzRjNDRjNDY0MjZiNjBkYzkxZmQ0OWViZWM1M2VmOTA5CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNUb092ZXJ3cml0ZUFuRWFybGllclJ1bnNSZWNlaXB0cwk4ZGM0OWE3NmVjMGQ0NTI2MzQ5YjAxNDhkMWU2MzU0NGE2Mzc1NTUyMGViNzU4M2U5YzcxMWYyMzljMzc0NDg3CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNXaXRob3V0VGhlR2F0ZXNaeUhvbGRzCTVhYTE0ZDI2ZWU5YTNhYmMzMzg2OWM0NGNmOTBkYjA3OGE2ZTY1OWI3MDNhNjdjNDEyMDg0MDkyMWQ1NjkzZjgKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVzdW1lc1dpdGhvdXRSZXBlYXRpbmdBU3RhcnRlZEF0dGVtcHQJYjIwMGRjNGU5ODkzOThhZTVmN2ZlZGMwYmZiYTY0MTZhYzRjODQ5YjlhM2Q2ODJkODQ5ZmRiMDQwYTMyN2EyNApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlOYW1lc0FuQXR0ZW1wdFdpdGhOb1JlY2VpcHQJMzQyMDZkZGI0NjYwM2MzNjBiMGE5MDM1YzA4ODJiMTkwMzZmZTdjOTk5YmM2YTRmZTFhZWZiMDJjYWJkYmRkMgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlSZWZ1c2VzQVJlY2VpcHRUaGF0SXNOb3RJdHNBdHRlbXB0CWZmMjhhYmNiNWM5Njc3NDZkYWFmYjdkNzNlOWZiMDIyYWI4NGU5OTU4YzlkODNmMWIwZjE4MmRmZTczZjJhODU
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-4-sonnet/run.json: no such file or directory
  exit status 1
  ```
- 2026-10-16 · human-observed · S2 Zy says go for all three Stage 4 stages, in order (2026-10-16, answered 'All three, in order'): Sonnet's own tasks, 96 attempts, about 2.5 to 3 hours; Opus's own tasks, 96 attempts, about 3 to 4 hours; the swe-multi-hunk slice on both models, 80 attempts and 20 image pulls of about 10 GB one at a time, about 5 to 6 hours; on this Mac, CLI 2.1.300, mrw v1.66.0, subscription
