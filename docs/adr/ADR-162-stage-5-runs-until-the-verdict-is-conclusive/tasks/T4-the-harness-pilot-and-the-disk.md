# Task ADR-162-T4: the harness pilot, and room on the disk

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** S (a pilot run and a sign-off)
**Owner:** Zy
**Produces:** `docs/bench/stage-5-pilot/` receipts; the plugin worktree at its pinned tag; the time per attempt; the disk room
**Consumes:** configuration P and the profile (T1), `--status-line` and `export-paired` (T2), `--block`, `--configs-for` and `bench look` (T3)
**Data dependency:** real agent runs on the subscription and two SWE-bench images, on this Mac; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `P was engaged and nothing synced loaded`, `every pilot attempt has a receipt`, `the claim and harness fields are filled`, `the rows export`

## Goal

Before any counted attempt: see two instances run in A, B and P on both models, read P's transcript whole, measure the time per attempt, and leave room on the disk for the stage.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-5-pilot/` | add | the pilot's receipts, `run.json`, `canary.json`, stage `pilot-5` (never counted) |

## Ordered Steps

1. [S1] See the fence red: no pilot receipts. [proof: acceptance]
2. [S2] Zy's word for the pilot, given with its price (12 attempts, two image pulls). [proof: human: Zy says go for the Stage 5 pilot]
3. [S3] Prepare the machine and run the pilot.
   - **The disk.** `docker image prune -a` (Zy, 2026-10-17: "Prune other projects' unused images"); record free space before and after. Below 25 GiB, stop for Zy.
   - **The plugin.** A detached worktree of quality-harness at its latest tag.
   - **The canary.** With the profile, A, B and P on each model.
   - **The pilot.** The first 2 instances of `bench/swe-bench/selection-adr-160.json` (outside Stage 5's pool) × A/B/P × 1 on Sonnet and Opus, stage `pilot-5`, with `--profile stage5 --status-line`.
   - **What to record.** Read one P transcript whole: the plugin's skills and hook events it shows, and that no synced skill or plugin loaded. Record the mean time per attempt, and that `bench export-paired --pair A,P` writes rows for each model.

   [proof: human: the executor records free disk before and after, the canary's result, what P's transcript shows, the time per attempt and the export]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-5-pilot
go run ./cmd/bench verify --stage pilot-5 --tasks 2 --repeats 1 $R \
  && jq -e '.pass == true' $R/canary.json > /dev/null \
  && jq -e '(.configs | sort) == ["A","B","P"] and (.models | length) == 2 and .profile == "stage5" and (.plugin.tag | test("^v[0-9]")) and (.plugin.commit | test("^[0-9a-f]{40}$")) and (.status_line_sha256 | test("^[0-9a-f]{64}$"))' $R/run.json > /dev/null \
  && jq -e -s 'map(select(.exclusion == null)) | length >= 12 and all(.[]; (.status | IN("done","not done","unreadable")) and (.oracle.broke_tests | type) == "boolean" and (.oracle.harness_resolved | type) == "boolean" and ((.config == "P") == (.plugin != null)))' $R/*.receipt.json > /dev/null \
  && for m in claude-sonnet-5-5 claude-opus-5-5; do go run ./cmd/bench export-paired --pair A,P --model $m --scorer-sha pilot-before-the-scorer --selection bench/swe-bench/selection-adr-160.json --canary $R/canary.json $R | jq -e -s '(.[0] | has("config")) and length == 5' > /dev/null || exit 1; done
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | earlier stages' receipts survive the pilot (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the pilot receipts |
| 2 — something selects it | `bench run --profile stage5 --configs A,B,P --status-line` |
| 3 — the caller can discover it | `docs/bench/stage-5-pilot/` |
| 4 — it is used | T5 starts on its sign-off |

## Mutation Log
(empty until execute)
- 2026-10-17 · 8b435f8f* · mutant killed · exit 1 · `docs/bench/stage-5-pilot/astropy__astropy-8707.P.claude-opus-5-5.1.receipt.json` · claim field: a receipt whose STATUS is outside done/not done/unreadable is caught · acceptance-sha256:4a407a7842e63fb2cebad69aad15c58ad89131621ab880eb410fdbbf640df236

## Invariants

- Pilot receipts are never in the stage's directory, and no pilot instance is in Stage 5's pool.

## Risks

- The prune may free too little; then the stage waits for Zy (Decision 9's 25 GiB floor).

## Stop Condition

The fence exits 0 and S3's sign-off is recorded; or P's canary fails, or a synced entry loads, which goes back to Zy with the transcript.

## Out of Scope

- Removing Docker volumes or containers (permanent: boundary: Zy's word covers images no container uses, nothing else)

## Verification Log
(empty until execute)
- 2026-10-17 · 8b435f8f* · exit 1 · `set -o pipefail …` · acceptance-sha256:4a407a7842e63fb2cebad69aad15c58ad89131621ab880eb410fdbbf640df236 · ms:1057 · test-lock-sha256:6ae5b3a5ee77688284edf6c2d0307f04da9b20fabff852c71825ffb1d6b7c7c2 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFBbmRCQ2FycnlOb1BsdWdpbkluQVN0YWdlV2l0aFAJOWU2NTUwMzkxNzNiNGZlNzM4OGZjMDc0ZTU3ZjMzODdlZTMwMDQ2NzY2ZTkxMzE4NDkzMTg1MzMxYzU0MjJlOQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBQmxvY2tSdW5zUGVyTW9kZWxDb25maWd1cmF0aW9ucwllNTc4OWFmNTZlZTAzNWRiMzk5YzkyZmNiZjNkMDc1N2QzZGEyNzljOWRiYWYyNzc5MTMyZDY4ZmY2MGQ2NzA0CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFGYWlsaW5nRXhpdElzQW5FcnJvckFuZEFuSW50ZXJydXB0U3RvcHNUaGVTdGFnZQkxMDQ1NmIzMWVlYTYzNjc5ZGEyZTQzYWZmODdmNGZjMzI5NzgzNGQ2MDg0MjBlYTY2ODg2NzUzYWRmODJjOWRlCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdGFnZVdpdGhBTWlzc2luZ0RlY2xhcmVkU2VjdGlvbkRvZXNOb3RTdGFydAk1MGFjYjFiNzFjYmQ2MDUwYmE2N2MwNzE0ZjNmYzY0ZDE3MWZiZjgzZjc2OGM2NGJhNmI1MmRkMTQ3ZGM1NjQ2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdXBlcnNlZGVkVGFza0lzTm90UnVuQnlEZWZhdWx0CTNmMDg0NTc3YWFmYWQ5MDFkZmIxMTNkYjBkNTQwZjZkOWRmNDA5YmNiMzU4OGY4NDNkOTkyMTJlYmQ3MGEzNDMKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5BdHRlbXB0TGVhdmVzTm90aGluZ1J1bm5pbmcJMzhkOGU1ZTJlOTY1MTljZTIxMDRkOTNkNGYwYzU5MTEyZmQ0ODRkZDVmNDc2NmQ1ZGJiMzc5N2FjOWQ4Njk2ZQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBbkF0dGVtcHRSZWFkc05vbmVPZlRoZUhvc3RzU2hlbGxPckdpdFNldHVwCWZhYWU0YTk1YjBjODhiNDg1ZWIxM2M0ZmNkNGZmNTJjZmUxMWVhMThiNDI5N2NlYzFkMjdmMGM0NGU0ZGE2Y2IKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5FeGNsdWRlZEF0dGVtcHRJc1JlcGxhY2VkQW5kS2VwdAk3ZjE5MjY5YjJlNDg3ZjYyYzY1MmZjNTk4MWM2ODM5M2M1ZmJkYTM3ZmYxZDkzYWYxNWRhOWY5OTE5ZDJlNjI2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEJvdGhDb25maWd1cmF0aW9uc0tlZXBUaGVTeXN0ZW1Qcm9tcHRTdGFibGUJMzhmMjNjMzY5YTBiNDk5ZWVkYTM1ZGY1N2U0ODIzZDJjOGUyN2ZlZmU0ODY0MWQzYmUwNTM4ZTljMDNmNzZmNgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RDb25maWd1cmF0aW9uQUhhc05vTXJ3T25JdHNQYXRoCWYwNzRiOTUxNDZhNzliOGI4NzBhNDBjNzY0NDRjMjRkMjQ4MDc3ZGQ4NjJiZmRjMTNlNTU4YTZhZWFhMGFhOGYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0Q29uZmlndXJhdGlvblBJc0FQbHVzVGhlUGx1Z2luQW5kU2tpbGxzCWJhNTdmN2VhMjQ2MWQzZTVjNTFjZTg0YWUxNDE4NzQwZjliMzIxNjVjNjZkN2E2ODVlZjVlZjkwNjYxY2YzZjIKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ2FuYXJ5UnVuc0JvdGhDb25maWd1cmF0aW9uc0FuZFJlY29yZHNJdHNWZXJkaWN0CTVmNjBhOTkwNWE2OTNlNGUzZWIzNGFmZGVlODkyZTg5NDQxZGU4NmUxNGMwMWRlOWE4NWQ0ODY2ZGIzODNiODUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ29uZmlnRGlyZWN0b3J5SXNKdWRnZWRCeUlkZW50aXR5CTFlOWVmYzdkNjFhMWI2Y2QwMjYyZGIxYzhhMDE4NjNlYzU1MDJlNTA5YzNhZDc0MjkxOTY2MmE3NDdjNmJiMGQKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlSGFybmVzc1ZlcmRpY3RTdXJ2aXZlc1RoZUJ1ZGdldAk1YTI1MzAxODMzMmY2M2Q3YTk5Y2RjNjU1MWYwNGUzNTM3MjhhMjhkZGVhZGFkNTM3ZjFlNWI5NjRiOTE3OTE0CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVByb2ZpbGVPdXRsaXZlc1AJYjhhZjAzM2QzNWZjNmE2N2VjM2UyYWZmOThlNTc0ZWFiZmQ3ZWY3ZWJiMGZkZDNiMmY2N2ZiMDIwYmM1YjFjYQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVSdW5TdG9wc0JlbG93VGhlRGlza0Zsb29yCWI3ZjRjN2QyZjlmYmFmNTliODUyYWE5NTkwZGE5YjE3OGMwNmI0ZTNhM2IxYTk0YmI1NDFhNjc0MGIyNDQzMzUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVjb3Jkc0FSZWNlaXB0UGVyQXR0ZW1wdEZyb21BU3R1YkNsYXVkZQkwNTA4YjMzNzZjNzZiODBkY2QyNzRiNWI1MzliN2VhYzRjNDRjNDY0MjZiNjBkYzkxZmQ0OWViZWM1M2VmOTA5CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNUb092ZXJ3cml0ZUFuRWFybGllclJ1bnNSZWNlaXB0cwk4ZGM0OWE3NmVjMGQ0NTI2MzQ5YjAxNDhkMWU2MzU0NGE2Mzc1NTUyMGViNzU4M2U5YzcxMWYyMzljMzc0NDg3CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNXaXRob3V0VGhlR2F0ZXNaeUhvbGRzCTVhYTE0ZDI2ZWU5YTNhYmMzMzg2OWM0NGNmOTBkYjA3OGE2ZTY1OWI3MDNhNjdjNDEyMDg0MDkyMWQ1NjkzZjgKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVzdW1lc1dpdGhvdXRSZXBlYXRpbmdBU3RhcnRlZEF0dGVtcHQJYjIwMGRjNGU5ODkzOThhZTVmN2ZlZGMwYmZiYTY0MTZhYzRjODQ5YjlhM2Q2ODJkODQ5ZmRiMDQwYTMyN2EyNApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVTdGF0dXNMaW5lSXNSZWFkRnJvbVRoZUZpbmFsTWVzc2FnZQlmZTBmYmM2OGI3OTIzMjkwYjNiNWRiMTJhNDkzYWMyNTU2YWNiNjZiMDI1Yzc2MjE5MzYwMGM2NjUzMGM0Yjc0CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFZlcmlmeU5hbWVzQW5BdHRlbXB0V2l0aE5vUmVjZWlwdAkzNDIwNmRkYjQ2NjAzYzM2MGIwYTkwMzVjMDg4MmIxOTAzNmZlN2M5OTliYzZhNGZlMWFlZmIwMmNhYmRiZGQyCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFZlcmlmeVJlZnVzZXNBUmVjZWlwdFRoYXRJc05vdEl0c0F0dGVtcHQJZmYyOGFiY2I1Yzk2Nzc0NmRhYWZiN2Q3M2U5ZmIwMjJhYjg0ZTk5NThjOWQ4M2YxYjBmMTgyZGZlNzNmMmE4NQ
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-5-pilot/run.json: no such file or directory
  exit status 1
  ```
- 2026-10-17 · human-observed · S2 approved: Zy said 'Go all, redo the benchmark' on 2026-10-17, after being told the pilot's price (2 instances × A/B/P × Sonnet and Opus, 12 attempts, 2 image pulls, about 30–40 minutes) and the stage's (up to 5 blocks, about 38 hours, 1,146 attempts); the Docker prune was Zy's earlier word ('Prune other projects' unused images'), and Zy approved clearing the Go build cache when the prune left 17 GiB ('Clear the Go build cache')
- 2026-10-17 · 8b435f8f* · exit 0 · `set -o pipefail …` · acceptance-sha256:4a407a7842e63fb2cebad69aad15c58ad89131621ab880eb410fdbbf640df236 · ms:584
- 2026-10-17 · human-observed · S3 observed and approved by the executor, 2026-10-17. Disk (Docker Data volume): 6 GiB free before the prune; 144 GiB after docker image prune -a and the Go build-cache clear Zy approved; 142 GiB after the pilot. Canary: pass for claude-sonnet-5-5 and claude-opus-5-5, configs A, B, P, profile stage5. P transcript (astropy__astropy-8707.P.claude-opus-5-5.1) read through: init lists the 17 quality-harness skills plus the built-ins design and doctor, one plugin (quality-harness 3.8.40, inline, the pinned worktree), no MCP server; the plugin's hooks answered on SessionStart, UserPromptSubmit, PreToolUse and PostToolUse and Stop; no synced skill or plugin appears. Time: 12 attempts from 11:02:36Z to 11:16:07Z, 68 s each with grading; mean agent wall 36 s (A 31, B 28, P 50). Export: bench export-paired --pair A,P writes the config line and 4 rows per model, and ADR-106's scorer --plan reads both. Observed as well: the checkouts hold no test dependencies (numpy, pytest, built extensions), so the claim line read done on 5 of 12 attempts while all 12 resolved (A 1 of 4, B 4 of 4, P 0 of 4); Zy chose on 2026-10-17 to proceed as designed.
- 2026-10-17 · human-observed · S3 approved: the sign-off above is the executor's observation, and the hook name it lists last is the session-end hook event, not a verdict. The pilot is signed off.
- 2026-10-17 · 8b435f8f* · exit 0 · `set -o pipefail …` · acceptance-sha256:4a407a7842e63fb2cebad69aad15c58ad89131621ab880eb410fdbbf640df236 · ms:987
