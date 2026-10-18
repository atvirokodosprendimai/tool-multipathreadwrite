# Task ADR-162-T5: ADR-106 amended, then Stage 5 block by block until the stop rule says it is complete

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** L (machine time: up to about 38 hours, rate-limited)
**Owner:** Zy
**Produces:** `docs/bench/stage-5/block-K/` receipts, `run.json` and `canary.json` per block, and `docs/bench/stage-5/looks/look-K.txt`, `bench look`'s output after block K
**Consumes:** the pilot's sign-off (T4), `selection-adr-162.json`, `--block`, `--configs-for` and `bench look` (T3)
**Data dependency:** quality-harness ADR-106 amended and its pre-registration committed there; real agent runs on the subscription and up to 191 SWE-bench images, on this Mac; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `ADR-106 was amended first`, `every attempt the stop rule scheduled has a receipt and no other does`, `one release, one plugin tag and one CLI ran the stage`, `the stage stopped where the rule says`, `no transcript read the dataset`

## Goal

Quality-harness ADR-106 takes this stage's rows before anything is counted. Then stage `stage5` runs block 1, looks, and runs each next block with the configurations `bench look` names, until `bench look` says the stage is complete (ADR-162 Decisions 5, 7, 8 and 10).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-5/` | add | `block-K/` (receipts, `run.json`, `canary.json`) and `looks/look-K.txt` |

## Ordered Steps

1. [S1] See the fence red: no stage receipts. [proof: acceptance]
2. [S2] Quality-harness ADR-106 amended in that repository, through its own PR and review, to take Stage 5's rows (k = 1, this pool, this stop rule), and its pre-registration committed there, on Zy's word. [proof: human: the quality-harness merge commit of the amendment and its pre-registration]
3. [S3] Zy's word for the stage, given with its price: up to 5 blocks of about 8 hours at both models and three configurations, 1,146 attempts at most; mrw at the current release, the plugin at the pilot's tag, the CLI the pilot ran. [proof: human: Zy says go for Stage 5]
4. [S4] Per block:
   - costly-runs (load under the core count) and the 25 GiB disk floor;
   - the canary per model;
   - `bench run --stage stage5 --profile stage5 --status-line --block K` with the configurations `bench look` printed;
   - every transcript scanned as Decision 8 says, where a hit stops the stage for Zy;
   - `bench look` into `looks/look-K.txt`, committed with the block's receipts.

   Stop when `bench look` exits 0. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-5
go run ./cmd/bench look $R > /tmp/adr162-look.txt \
  && diff /tmp/adr162-look.txt "$(ls $R/looks/look-*.txt | sort -V | tail -1)" \
  && for d in $R/block-*/; do n=40; [ "$(basename "$d")" = block-5 ] && n=31; go run ./cmd/bench verify --stage stage5 --tasks $n --repeats 1 "$d" || exit 1; done \
  && jq -e -s 'length > 0 and all(.[]; .stage == "stage5" and .profile == "stage5")' $R/block-*/run.json > /dev/null \
  && test "$(jq -r -s '[.[].cli_version] | unique | length' $R/block-*/*.receipt.json)" = 1 \
  && test "$(jq -r -s '[.[].mrw_version | select(. != null)] | unique | length' $R/block-*/*.receipt.json)" = 1 \
  && test "$(jq -r -s '[.[].plugin | select(. != null) | .commit] | unique | length' $R/block-*/*.receipt.json)" = 1
```

`bench look` exits 0 only when the stage is complete, refuses an attempt the rule did not schedule, and finds a missing scheduled one ineligible; `bench verify` checks each block's per-model plan.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | earlier stages' receipts survive Stage 5 (ADR-156 T2) | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench look`, `bench verify`, `bench analyze` |
| 3 — the caller can discover it | `docs/bench/stage-5/` |
| 4 — it is used | T6 |

## Mutation Log
(empty until execute)

## Invariants

- One mrw release, one plugin tag and one CLI executable for the whole stage; the profile in every block.
- No block starts before the previous look is committed.

## Risks

- A multi-day run meets a Claude Code release; the pinned executable is used, and a mismatch is refused rather than disclosed.
- The subscription's 7-day limit runs out mid-stage (overage is disabled on the account). Measured 2026-10-17 from the `rate_limit_event` in block 1's transcripts: 78% used at its first attempt, 81% at its 120th, reset 2026-10-20T07:00Z; a full block costs about 6 points, and the figure includes other sessions' use. Zy, 2026-10-17, chose to check between blocks: before each block after the first, read the newest transcript's `seven_day` utilization, and when the block could carry it past about 95%, hold the stage until the reset and say so, rather than run attempts against a capped account.
- Other work on this Mac contends with the run, so wall time is not attributable while it does. Measured 2026-10-17 during block 1: load 34.5 on 10 cores at 16:58Z, and 48.48 and 49.1 recorded by qh-check inside two P attempts, from quality-harness's builder worktrees and a memory-runtime test. Cost, resolve and the claim line do not depend on load. Zy, 2026-10-17: "continue, load is fine"; the results name block 1's wall time as taken under contention.
- Decision 8's scan meets names the agent did not read. At block 1 (ended 2026-10-17T21:07Z) it printed 35 hits and stopped the stage, as written. Every one was traced:
  - 23 were sympy FAIL_TO_PASS names already in the instance's base commit, so in the agent's own tree;
  - 12 were names the base lacks, which Opus wrote itself (a test it added, with upstream's exact name and docstring) before any tool output could have shown them: recall of the upstream fix from training, in A, B and P;
  - none was the dataset's, the probe's, Hugging Face's or a selection's name.
  Zy, 2026-10-18, chose "Continue, record it". The scan (`docs/bench/stage-5/scan.py`, the copy run from the run directory) now skips a name the base commit contains, prints a name whose first appearance is the agent's own writing as RECALL, and keeps every other appearance a hit. It judges by the test's bare identifier, since the runner's longer forms appear only in test output. A planted read of a name the base lacks still exits 1. Block 1 rescanned: 0 hits, 73 recalls in 19 instances (Opus A 17, B 11, P 19; Sonnet A 2, P 1). Recall inflates solved rates, and it is not equal across arms, so the results name it beside each verdict.

## Stop Condition

`bench look` exits 0 and the fence passes. These go back to Zy instead:
- the canary fails;
- a transcript hit;
- the disk floor stops the run and the prune cannot restore it;
- the subscription refuses for a day.

## Out of Scope

- Pooling with Stages 1 to 4b (permanent: boundary: one verdict per model, comparison, source and stage)

## Verification Log
(empty until execute)
- 2026-10-17 · 6de290ac · exit 1 · `set -o pipefail …` · acceptance-sha256:bf4eca9a150bf62a6f1d93f4aa387c5c6abf27f52567bc3c8f0fe680cc4329ea · ms:223 · test-lock-sha256:6ae5b3a5ee77688284edf6c2d0307f04da9b20fabff852c71825ffb1d6b7c7c2 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFBbmRCQ2FycnlOb1BsdWdpbkluQVN0YWdlV2l0aFAJOWU2NTUwMzkxNzNiNGZlNzM4OGZjMDc0ZTU3ZjMzODdlZTMwMDQ2NzY2ZTkxMzE4NDkzMTg1MzMxYzU0MjJlOQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBQmxvY2tSdW5zUGVyTW9kZWxDb25maWd1cmF0aW9ucwllNTc4OWFmNTZlZTAzNWRiMzk5YzkyZmNiZjNkMDc1N2QzZGEyNzljOWRiYWYyNzc5MTMyZDY4ZmY2MGQ2NzA0CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFGYWlsaW5nRXhpdElzQW5FcnJvckFuZEFuSW50ZXJydXB0U3RvcHNUaGVTdGFnZQkxMDQ1NmIzMWVlYTYzNjc5ZGEyZTQzYWZmODdmNGZjMzI5NzgzNGQ2MDg0MjBlYTY2ODg2NzUzYWRmODJjOWRlCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdGFnZVdpdGhBTWlzc2luZ0RlY2xhcmVkU2VjdGlvbkRvZXNOb3RTdGFydAk1MGFjYjFiNzFjYmQ2MDUwYmE2N2MwNzE0ZjNmYzY0ZDE3MWZiZjgzZjc2OGM2NGJhNmI1MmRkMTQ3ZGM1NjQ2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFTdXBlcnNlZGVkVGFza0lzTm90UnVuQnlEZWZhdWx0CTNmMDg0NTc3YWFmYWQ5MDFkZmIxMTNkYjBkNTQwZjZkOWRmNDA5YmNiMzU4OGY4NDNkOTkyMTJlYmQ3MGEzNDMKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5BdHRlbXB0TGVhdmVzTm90aGluZ1J1bm5pbmcJMzhkOGU1ZTJlOTY1MTljZTIxMDRkOTNkNGYwYzU5MTEyZmQ0ODRkZDVmNDc2NmQ1ZGJiMzc5N2FjOWQ4Njk2ZQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBbkF0dGVtcHRSZWFkc05vbmVPZlRoZUhvc3RzU2hlbGxPckdpdFNldHVwCWZhYWU0YTk1YjBjODhiNDg1ZWIxM2M0ZmNkNGZmNTJjZmUxMWVhMThiNDI5N2NlYzFkMjdmMGM0NGU0ZGE2Y2IKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5FeGNsdWRlZEF0dGVtcHRJc1JlcGxhY2VkQW5kS2VwdAk3ZjE5MjY5YjJlNDg3ZjYyYzY1MmZjNTk4MWM2ODM5M2M1ZmJkYTM3ZmYxZDkzYWYxNWRhOWY5OTE5ZDJlNjI2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEJvdGhDb25maWd1cmF0aW9uc0tlZXBUaGVTeXN0ZW1Qcm9tcHRTdGFibGUJMzhmMjNjMzY5YTBiNDk5ZWVkYTM1ZGY1N2U0ODIzZDJjOGUyN2ZlZmU0ODY0MWQzYmUwNTM4ZTljMDNmNzZmNgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RDb25maWd1cmF0aW9uQUhhc05vTXJ3T25JdHNQYXRoCWYwNzRiOTUxNDZhNzliOGI4NzBhNDBjNzY0NDRjMjRkMjQ4MDc3ZGQ4NjJiZmRjMTNlNTU4YTZhZWFhMGFhOGYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0Q29uZmlndXJhdGlvblBJc0FQbHVzVGhlUGx1Z2luQW5kU2tpbGxzCWJhNTdmN2VhMjQ2MWQzZTVjNTFjZTg0YWUxNDE4NzQwZjliMzIxNjVjNjZkN2E2ODVlZjVlZjkwNjYxY2YzZjIKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ2FuYXJ5UnVuc0JvdGhDb25maWd1cmF0aW9uc0FuZFJlY29yZHNJdHNWZXJkaWN0CTVmNjBhOTkwNWE2OTNlNGUzZWIzNGFmZGVlODkyZTg5NDQxZGU4NmUxNGMwMWRlOWE4NWQ0ODY2ZGIzODNiODUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlQ29uZmlnRGlyZWN0b3J5SXNKdWRnZWRCeUlkZW50aXR5CTFlOWVmYzdkNjFhMWI2Y2QwMjYyZGIxYzhhMDE4NjNlYzU1MDJlNTA5YzNhZDc0MjkxOTY2MmE3NDdjNmJiMGQKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlSGFybmVzc1ZlcmRpY3RTdXJ2aXZlc1RoZUJ1ZGdldAk1YTI1MzAxODMzMmY2M2Q3YTk5Y2RjNjU1MWYwNGUzNTM3MjhhMjhkZGVhZGFkNTM3ZjFlNWI5NjRiOTE3OTE0CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVByb2ZpbGVPdXRsaXZlc1AJYjhhZjAzM2QzNWZjNmE2N2VjM2UyYWZmOThlNTc0ZWFiZmQ3ZWY3ZWJiMGZkZDNiMmY2N2ZiMDIwYmM1YjFjYQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVSdW5TdG9wc0JlbG93VGhlRGlza0Zsb29yCWI3ZjRjN2QyZjlmYmFmNTliODUyYWE5NTkwZGE5YjE3OGMwNmI0ZTNhM2IxYTk0YmI1NDFhNjc0MGIyNDQzMzUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVjb3Jkc0FSZWNlaXB0UGVyQXR0ZW1wdEZyb21BU3R1YkNsYXVkZQkwNTA4YjMzNzZjNzZiODBkY2QyNzRiNWI1MzliN2VhYzRjNDRjNDY0MjZiNjBkYzkxZmQ0OWViZWM1M2VmOTA5CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNUb092ZXJ3cml0ZUFuRWFybGllclJ1bnNSZWNlaXB0cwk4ZGM0OWE3NmVjMGQ0NTI2MzQ5YjAxNDhkMWU2MzU0NGE2Mzc1NTUyMGViNzU4M2U5YzcxMWYyMzljMzc0NDg3CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNXaXRob3V0VGhlR2F0ZXNaeUhvbGRzCTVhYTE0ZDI2ZWU5YTNhYmMzMzg2OWM0NGNmOTBkYjA3OGE2ZTY1OWI3MDNhNjdjNDEyMDg0MDkyMWQ1NjkzZjgKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVzdW1lc1dpdGhvdXRSZXBlYXRpbmdBU3RhcnRlZEF0dGVtcHQJYjIwMGRjNGU5ODkzOThhZTVmN2ZlZGMwYmZiYTY0MTZhYzRjODQ5YjlhM2Q2ODJkODQ5ZmRiMDQwYTMyN2EyNApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVTdGF0dXNMaW5lSXNSZWFkRnJvbVRoZUZpbmFsTWVzc2FnZQlmZTBmYmM2OGI3OTIzMjkwYjNiNWRiMTJhNDkzYWMyNTU2YWNiNjZiMDI1Yzc2MjE5MzYwMGM2NjUzMGM0Yjc0CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFZlcmlmeU5hbWVzQW5BdHRlbXB0V2l0aE5vUmVjZWlwdAkzNDIwNmRkYjQ2NjAzYzM2MGIwYTkwMzVjMDg4MmIxOTAzNmZlN2M5OTliYzZhNGZlMWFlZmIwMmNhYmRiZGQyCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFZlcmlmeVJlZnVzZXNBUmVjZWlwdFRoYXRJc05vdEl0c0F0dGVtcHQJZmYyOGFiY2I1Yzk2Nzc0NmRhYWZiN2Q3M2U5ZmIwMjJhYjg0ZTk5NThjOWQ4M2YxYjBmMTgyZGZlNzNmMmE4NQ
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-5/block-1/run.json: no such file or directory
  exit status 2
  ```
- 2026-10-17 · human-observed · S2 approved: quality-harness ADR-106 is amended to take Stage 5's rows (k = 1, the 191-instance pool, this stop rule) through its own PRs and Codex reviews, merged as d157f072 (#24, the scorer and the amendment) and ac0b291e (#26, the aggregate verdict file and both pre-registrations, prereg-main-claude-sonnet-5-5.json and prereg-main-claude-opus-5-5.json, pinning scorer blob 558548fa). Merged at 2026-10-17T13:19Z, before Stage 5's first counted attempt. Zy's word: 'Go all, redo the benchmark', 'Amend it before Stage 5'.
- 2026-10-17 · human-observed · S3 approved: Zy said 'Go all, redo the benchmark' on 2026-10-17, given with Stage 5's price (up to 5 blocks, 1,146 attempts at most, about 38 hours quoted; the pilot measured 68 s per attempt with grading), and after the pilot 'Proceed as designed'. mrw v1.68.0, the plugin at v3.8.40 (the pilot's tag), Claude Code 2.1.302 (the pilot's CLI).
