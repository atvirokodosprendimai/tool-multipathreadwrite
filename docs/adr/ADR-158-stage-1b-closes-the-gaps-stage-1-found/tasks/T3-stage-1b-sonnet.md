# Task ADR-158-T3: Stage 1b on Sonnet 5.5, a pilot first

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (machine time: hours)
**Owner:** Zy
**Produces:** `docs/bench/stage-1b-sonnet/` receipts
**Consumes:** the attempt environment (T1), the revised corpus and per-stage verdicts (T2)
**Data dependency:** real agent runs on the subscription, on this Mac; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `the pilot ran and its transcripts were read`, `every registered attempt has a receipt`, `the receipts verify`, `the stage ran exactly the unsuperseded corpus`

## Goal

The registered Stage 1b: a pilot, then 40 tasks × 2 configurations × 2 repeats on Sonnet 5.5, on this Mac.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-1b-sonnet/pilot/` | add | 5 tasks (three `varied-multi-site`, at least one agentsmemory) × 2 × 1, stage `pilot`, not counted |
| `docs/bench/stage-1b-sonnet/` | add | the stage's receipts, `run.json` and canary |

## Ordered Steps

1. [S1] Write the failing check and see it red: the fence exits non-zero while the directory is missing. [proof: acceptance]
2. [S2] Zy's word to start, recorded. [proof: human: Zy says go for Stage 1b]
3. [S3] The canary on the pinned CLI, then the pilot; every pilot transcript is read. Any one-turn attempt with no tool call, or a harness defect, is fixed in T1 or T2 and the pilot rerun into a new directory, the old one kept; a task is superseded after the pilot only for an oracle defect shown on its base or its reference fix. [proof: human: the executor reads every pilot receipt and transcript]
4. [S4] The full matrix, `--stage stage1b`, sequential, on this Mac. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go run ./cmd/bench verify --stage stage1b --tasks 40 --repeats 2 docs/bench/stage-1b-sonnet \
  && diff <(jq -r '.tasks[]' docs/bench/stage-1b-sonnet/run.json | sort) \
          <(git ls-tree --name-only edb65629 bench/tasks/ | while read -r d; do git show "edb65629:$d/task.json" | jq -r 'select(.shape != "fixture" and (has("superseded_by") | not)) | .id'; done | sort) \
  && go test ./internal/bench/ -count=1 -run '^TestTheRunnerResumesWithoutRepeatingAStartedAttempt$' -v 2>&1 | grep '^--- PASS: TestTheRunnerResumesWithoutRepeatingAStartedAttempt ' > /dev/null
```

The corpus compared is the one at `edb65629`, the commit the stage ran from (ADR-159 Decision 3): a task superseded after the stage ran must not make its receipts look wrong.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerResumesWithoutRepeatingAStartedAttempt` | `internal/bench/runner_test.go` | the stage can stop and resume (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench verify`; `bench analyze` |
| 3 — the caller can discover it | `docs/bench/stage-1b-sonnet/` |
| 4 — it is used | ADR-156 T6's `RESULTS.md` |

## Mutation Log
(empty until execute)
- 2026-10-15 · edb65629* · mutant killed · exit 1 · `docs/bench/stage-1b-sonnet/run.json` · the fence binds the receipts to the stage they were run as · acceptance-sha256:1fe9795510492bd2d7bb5fcc04b0b6c844d2d53d9b7bf67c9ce14b639c43e3ed
- 2026-10-15 · ffbcfd04* · mutant killed · exit 1 · `docs/bench/stage-1b-sonnet/run.json` · the fence binds the receipts to the stage they were run as · acceptance-sha256:fc867387263f0676ee41371639d1179cd5e8f933b01a0a29453f4a5fba7e469d
- 2026-10-15 · ffbcfd04* · mutant killed · exit 1 · `docs/bench/stage-1b-sonnet/run.json` · the fence binds run.json to the corpus at the commit the stage ran from (ADR-159 Decision 3) · acceptance-sha256:fc867387263f0676ee41371639d1179cd5e8f933b01a0a29453f4a5fba7e469d
- 2026-10-15 · ffbcfd04* · mutant killed · exit 1 · `docs/bench/stage-1b-sonnet/run.json` · the fence binds the receipts to the stage they were run as · acceptance-sha256:a1f193663efe0afe7355e57c83919d35782d2352ffdeb08e93b04f3c252af795
- 2026-10-15 · ffbcfd04* · mutant killed · exit 1 · `docs/bench/stage-1b-sonnet/run.json` · the fence binds run.json to the corpus at the commit the stage ran from (ADR-159 Decision 3) · acceptance-sha256:a1f193663efe0afe7355e57c83919d35782d2352ffdeb08e93b04f3c252af795

## Verification Log
(empty until execute)
- 2026-10-15 · edb65629 · exit 1 · `set -o pipefail …` · acceptance-sha256:1fe9795510492bd2d7bb5fcc04b0b6c844d2d53d9b7bf67c9ce14b639c43e3ed · ms:382 · test-lock-sha256:3f528f47a4399e2d00c863d847852b3a72f1f7b26edde8d11544b6e00beec3b3 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFGYWlsaW5nRXhpdElzQW5FcnJvckFuZEFuSW50ZXJydXB0U3RvcHNUaGVTdGFnZQkxMDQ1NmIzMWVlYTYzNjc5ZGEyZTQzYWZmODdmNGZjMzI5NzgzNGQ2MDg0MjBlYTY2ODg2NzUzYWRmODJjOWRlCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdGFnZVdpdGhBTWlzc2luZ0RlY2xhcmVkU2VjdGlvbkRvZXNOb3RTdGFydAk1MGFjYjFiNzFjYmQ2MDUwYmE2N2MwNzE0ZjNmYzY0ZDE3MWZiZjgzZjc2OGM2NGJhNmI1MmRkMTQ3ZGM1NjQ2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdXBlcnNlZGVkVGFza0lzTm90UnVuQnlEZWZhdWx0CTNmMDg0NTc3YWFmYWQ5MDFkZmIxMTNkYjBkNTQwZjZkOWRmNDA5YmNiMzU4OGY4NDNkOTkyMTJlYmQ3MGEzNDMKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5BdHRlbXB0TGVhdmVzTm90aGluZ1J1bm5pbmcJMzhkOGU1ZTJlOTY1MTljZTIxMDRkOTNkNGYwYzU5MTEyZmQ0ODRkZDVmNDc2NmQ1ZGJiMzc5N2FjOWQ4Njk2ZQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBbkF0dGVtcHRSZWFkc05vbmVPZlRoZUhvc3RzU2hlbGxPckdpdFNldHVwCWZhYWU0YTk1YjBjODhiNDg1ZWIxM2M0ZmNkNGZmNTJjZmUxMWVhMThiNDI5N2NlYzFkMjdmMGM0NGU0ZGE2Y2IKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5FeGNsdWRlZEF0dGVtcHRJc1JlcGxhY2VkQW5kS2VwdAk3ZjE5MjY5YjJlNDg3ZjYyYzY1MmZjNTk4MWM2ODM5M2M1ZmJkYTM3ZmYxZDkzYWYxNWRhOWY5OTE5ZDJlNjI2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEJvdGhDb25maWd1cmF0aW9uc0tlZXBUaGVTeXN0ZW1Qcm9tcHRTdGFibGUJMzhmMjNjMzY5YTBiNDk5ZWVkYTM1ZGY1N2U0ODIzZDJjOGUyN2ZlZmU0ODY0MWQzYmUwNTM4ZTljMDNmNzZmNgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RDb25maWd1cmF0aW9uQUhhc05vTXJ3T25JdHNQYXRoCWYwNzRiOTUxNDZhNzliOGI4NzBhNDBjNzY0NDRjMjRkMjQ4MDc3ZGQ4NjJiZmRjMTNlNTU4YTZhZWFhMGFhOGYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ2FuYXJ5UnVuc0JvdGhDb25maWd1cmF0aW9uc0FuZFJlY29yZHNJdHNWZXJkaWN0CTVmNjBhOTkwNWE2OTNlNGUzZWIzNGFmZGVlODkyZTg5NDQxZGU4NmUxNGMwMWRlOWE4NWQ0ODY2ZGIzODNiODUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ29uZmlnRGlyZWN0b3J5SXNKdWRnZWRCeUlkZW50aXR5CTFlOWVmYzdkNjFhMWI2Y2QwMjYyZGIxYzhhMDE4NjNlYzU1MDJlNTA5YzNhZDc0MjkxOTY2MmE3NDdjNmJiMGQKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVjb3Jkc0FSZWNlaXB0UGVyQXR0ZW1wdEZyb21BU3R1YkNsYXVkZQkwNTA4YjMzNzZjNzZiODBkY2QyNzRiNWI1MzliN2VhYzRjNDRjNDY0MjZiNjBkYzkxZmQ0OWViZWM1M2VmOTA5CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNUb092ZXJ3cml0ZUFuRWFybGllclJ1bnNSZWNlaXB0cwk4ZGM0OWE3NmVjMGQ0NTI2MzQ5YjAxNDhkMWU2MzU0NGE2Mzc1NTUyMGViNzU4M2U5YzcxMWYyMzljMzc0NDg3CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNXaXRob3V0VGhlR2F0ZXNaeUhvbGRzCTVhYTE0ZDI2ZWU5YTNhYmMzMzg2OWM0NGNmOTBkYjA3OGE2ZTY1OWI3MDNhNjdjNDEyMDg0MDkyMWQ1NjkzZjgKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVzdW1lc1dpdGhvdXRSZXBlYXRpbmdBU3RhcnRlZEF0dGVtcHQJYjIwMGRjNGU5ODkzOThhZTVmN2ZlZGMwYmZiYTY0MTZhYzRjODQ5YjlhM2Q2ODJkODQ5ZmRiMDQwYTMyN2EyNApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlOYW1lc0FuQXR0ZW1wdFdpdGhOb1JlY2VpcHQJMzQyMDZkZGI0NjYwM2MzNjBiMGE5MDM1YzA4ODJiMTkwMzZmZTdjOTk5YmM2YTRmZTFhZWZiMDJjYWJkYmRkMgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlSZWZ1c2VzQVJlY2VpcHRUaGF0SXNOb3RJdHNBdHRlbXB0CWZmMjhhYmNiNWM5Njc3NDZkYWFmYjdkNzNlOWZiMDIyYWI4NGU5OTU4YzlkODNmMWIwZjE4MmRmZTczZjJhODU
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-1b-sonnet/run.json: no such file or directory
  exit status 1
  ```
- 2026-10-15 · human-observed · S2 approved: Zy said go for Stage 1b on 2026-10-15 ('run'), after the record, T1 and T2 merged as edb65629
- 2026-10-15 · human-observed · S3 observed 2026-10-15 on this Mac, CLI 2.1.300, mrw v1.66.0 (05ec3f8), bench at edb65629: canary passed; pilot of 5 tasks x A/B x 1 (am-vms-stamp-builds, qh-vms-listing-timeout, site-vms-social-metadata, am-go-sha-message-2, site-html-cost-faq-2) read receipt by receipt and every transcript's tool calls and results: 10/10 solved, no silent mis-edit, no exclusion, no slipped write, empty stderr, no one-turn attempt; both agentsmemory attempts worked past the dropped AGENTS.md sections; B's two hook refusals were git stash in the tree and sed -i on /tmp/plan.txt, both the registered rule (only a redirection or tee may write outside the tree); in am-go-sha-message-2, B left the asserting test unchanged and A changed it, both graded solved, as the replacement intends. No harness or oracle defect: the full stage proceeds
- 2026-10-15 · edb65629* · exit 0 · `set -o pipefail …` · acceptance-sha256:1fe9795510492bd2d7bb5fcc04b0b6c844d2d53d9b7bf67c9ce14b639c43e3ed · ms:808
- 2026-10-15 · edb65629* · exit 0 · `set -o pipefail …` · acceptance-sha256:1fe9795510492bd2d7bb5fcc04b0b6c844d2d53d9b7bf67c9ce14b639c43e3ed · ms:2098
- 2026-10-15 · ffbcfd04* · exit 0 · `set -o pipefail …` · acceptance-sha256:fc867387263f0676ee41371639d1179cd5e8f933b01a0a29453f4a5fba7e469d · ms:792
- 2026-10-15 · ffbcfd04* · exit 0 · `set -o pipefail …` · acceptance-sha256:fc867387263f0676ee41371639d1179cd5e8f933b01a0a29453f4a5fba7e469d · ms:791
- 2026-10-15 · ffbcfd04* · exit 141 · `set -o pipefail …` · acceptance-sha256:cd36389883f3ca7e9033fab4f346d2e5027c10bc456cab6d7d456bee89d57a5b · ms:3152
  ```
  ```
- 2026-10-15 · ffbcfd04* · exit 141 · `set -o pipefail …` · acceptance-sha256:cd36389883f3ca7e9033fab4f346d2e5027c10bc456cab6d7d456bee89d57a5b · ms:2827
  ```
  ```
- 2026-10-15 · ffbcfd04* · exit 0 · `set -o pipefail …` · acceptance-sha256:a1f193663efe0afe7355e57c83919d35782d2352ffdeb08e93b04f3c252af795 · ms:3775
- 2026-10-15 · ffbcfd04* · exit 0 · `set -o pipefail …` · acceptance-sha256:a1f193663efe0afe7355e57c83919d35782d2352ffdeb08e93b04f3c252af795 · ms:6363

## Invariants

- The mrw build, the CLI version and the host are one for the whole stage, and the first two are named in every receipt.

## Risks

- Rate limits stretch the stage: resumable by design.

## Out of Scope

- Changing a task after the pilot (permanent: boundary: a task changed after it was run is a new task, with a new id)

## Stop Condition

The fence exits 0; or the pilot shows the harness cannot measure what ADR-156 Decision 4 names, which goes back to Zy.
