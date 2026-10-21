# Task ADR-156-T6: analysis against the registered bar, and README

**Depends-on:** T3, T4, T5
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `bench analyze`, `docs/bench/RESULTS.md`, the README sentence
**Consumes:** the receipts of T3, T4 and T5
**Data dependency:** the stage receipts
**Proof map:** v1
**Rests-on:** `pairs by task and keeps every attempt`, `admits only by the registered rules`, `the verdict follows the bar`, `README states no more than the verdict`, `a descriptive row leaves out the tasks red at base and is never the verdict`

## Goal

One verdict per model and task source, computed exactly as BACKLOG registered it, and README saying that and no more.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/analyze.go` | add | pairing, the bootstrap (seed 156, 10,000 resamples), the verdict |
| `internal/bench/analyze_test.go` | add | the tests |
| `cmd/bench/main.go` | edit | `analyze` |
| `docs/bench/RESULTS.md` | add | every verdict, every silent mis-edit, every exclusion |
| `README.md` | edit | one sentence per verdict, with the link |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestTheAnalysisPairsByTaskAndKeepsEveryAttempt`, `TestOnlyTheRegisteredExclusionsAreAdmitted`, `TestTheVerdictFollowsTheRegisteredBar` (constructed receipts that must yield PAYS, LOSES and INCONCLUSIVE, and one at each boundary).
2. [S2] `bench analyze`. [proof: mutation]
3. [S3] `docs/bench/RESULTS.md` from `bench analyze` output, not by hand. [proof: acceptance]
4. [S4] README, one sentence per verdict, a LOSES or INCONCLUSIVE as plainly as a PAYS. [proof: human: Zy reads the README sentence against RESULTS.md]
5. [S5] Zy, 2026-10-14 ("Keep, add node, declare it"): where a model's and source's attempts include tasks flagged `check_red_at_base`, `bench analyze` adds one row computed without them, marked descriptive and printed with its verdict in parentheses; the verdict row still counts every task, and a group with no flagged attempt gets no second row. The registered bar is not changed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/bench/ -count=1 -run 'TestTheAnalysisPairsByTaskAndKeepsEveryAttempt|TestOnlyTheRegisteredExclusionsAreAdmitted|TestTheVerdictFollowsTheRegisteredBar|TestADescriptiveRowLeavesOutTasksWhoseCheckIsRedAtBase' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheAnalysisPairsByTaskAndKeepsEveryAttempt \(' "$out" \
  && grep -qE '^--- PASS: TestOnlyTheRegisteredExclusionsAreAdmitted \(' "$out" \
  && grep -qE '^--- PASS: TestTheVerdictFollowsTheRegisteredBar \(' "$out" \
  && grep -qE '^--- PASS: TestADescriptiveRowLeavesOutTasksWhoseCheckIsRedAtBase \(' "$out" \
  && go run ./cmd/bench analyze docs/bench > "$out.results" \
  && cmp -s "$out.results" docs/bench/RESULTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheAnalysisPairsByTaskAndKeepsEveryAttempt` | `internal/bench/analyze_test.go` | pairing within model and source; nothing dropped | none | S1, S2 |
| `TestOnlyTheRegisteredExclusionsAreAdmitted` | `internal/bench/analyze_test.go` | only the registered exclusions | none | S1, S2 |
| `TestTheVerdictFollowsTheRegisteredBar` | `internal/bench/analyze_test.go` | PAYS, LOSES, INCONCLUSIVE and their boundaries | none | S1, S2 |
| `TestADescriptiveRowLeavesOutTasksWhoseCheckIsRedAtBase` | `internal/bench/redbase_test.go` | a group with flagged attempts gets one descriptive row without them; the verdict counts every task; a group without gets none | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `bench analyze` |
| 2 — something selects it | the fence regenerates RESULTS.md and compares |
| 3 — the caller can discover it | README's sentence and link |
| 4 — it is used | README readers |

## Mutation Log
(empty until execute)
- 2026-10-14 · human-observed · mutant killed · test exit 1 · `internal/bench/analyze.go` · line 87 · from `d.Descriptive = withoutRedAtBase` · to `d.Descriptive = ""` · test `TestADescriptiveRowLeavesOutTasksWhoseCheckIsRedAtBase` · T6 fence cannot run to completion until docs/bench/RESULTS.md exists (S3, after Stage 1)
- 2026-10-15 · f98fc6d9* · mutant killed · exit 1 · `docs/bench/RESULTS.md` · a verdict edited by hand in RESULTS.md · acceptance-sha256:42bb7bf03a0c61b136a4a5d60140f9dd9ecfdeb60bb604ce93b857ee985d5463 · covers:the verdict follows the bar

## Verification Log
(empty until execute)
- 2026-10-15 · f98fc6d9* · exit 0 · `set -o pipefail …` · acceptance-sha256:42bb7bf03a0c61b136a4a5d60140f9dd9ecfdeb60bb604ce93b857ee985d5463 · ms:1199
- 2026-10-15 · f98fc6d9* · exit 2 · `set -o pipefail …` · acceptance-sha256:42bb7bf03a0c61b136a4a5d60140f9dd9ecfdeb60bb604ce93b857ee985d5463 · ms:683 · test-lock-sha256:8a6a783cef83f0f0393c390bccd40b074b7ac2e5b17185d6647388de186ad015 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUGlsb3RJc05ldmVyQ291bnRlZAkxNzExNDE3YjczMzQ5NzMxMzgwMzNjODE4ZWM1NGFjM2Y2M2Q2NWUyYWUyYjFhNzhkOThiYTU1NjFlYjIwYTVhCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUm93V2l0aG91dFRoZVRhc2tzU3VwZXJzZWRlZFNpbmNlSXNEZXNjcmlwdGl2ZQkwYTc3OGYzZDBjZjE3NWRiMTkyNzEwOWI4YWMwNDRjM2JjOGJhMGJkNDM0OTE4YWY0YjA1ZWUyMTU1MjhlNzY0CmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBbmFseXplUHJpbnRzQVN0YWdlNENhdGVnb3J5Um93CWU0MzJiODc4Njg0YjcyZjA3YjVjMzBkOGUzZDI0MWFmM2JhMjZhY2IwNjliZmE3MmJmY2FjZDBkZTEyMjY4ZjIKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdE9ubHlUaGVSZWdpc3RlcmVkRXhjbHVzaW9uc0FyZUFkbWl0dGVkCWRiODQ3NDI0Yzc2NzRlNzBhZTU0Nzc3ZTk5MzQ0NjFhZThkNTkwMmJmZDZlMmRjOTBiNGI0MTZiYWM4YmJmNWYKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFN0YWdlT25lc1B1Ymxpc2hlZFZlcmRpY3RJc1VuY2hhbmdlZAk5NjA5YmJlZTM2OTBiYTFkMWIxY2UyMjVjYmZmOTQzNTFmNmYwY2Q0NzNkOTljOTdhNTViNjFkMmU2NTJjYTQ1CmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RUaGVBbmFseXNpc1BhaXJzQnlUYXNrQW5kS2VlcHNFdmVyeUF0dGVtcHQJYjA5MDNmZGRmZjVmMzJmOWRjOGE0YWU1NjI5M2ZlMWUwNDI0MDIwZjUxNTU4ZWY5MDI2ZmM0NzcxMGMyNzAxYgpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VGhlUHVibGlzaGVkU3RhZ2VzV2l0aG91dFRoZVRhc2tzU3VwZXJzZWRlZFNpbmNlCTgwYWE5ZTk2ZjIyMGNiMDJjMzUwMTFiOGUzZGMyYTlmM2EyMDNkYWE3Yjk5OTc2ZmZkNGZiNmY1N2U4MDk2ZmIKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFRoZVZhcmllZE11bHRpU2l0ZVJvd0lzRGVzY3JpcHRpdmUJNmVlZjQzMWFhNmU3YzliZGJmYjk4ZmJlNzhjN2EzMTZiN2JjMGU0ZmNjYWFiMWNlNjA1ZWExZTZiN2UyOTU0MQpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VGhlVmVyZGljdEZvbGxvd3NUaGVSZWdpc3RlcmVkQmFyCTUxNjUxNjM0MmUwNmE3YjYzOTVkZjA4NDJhNzQ5N2ZjMTdiNGQ1MDcwZDAxZDIzM2Y1NWZkMzIwYjg1MzM5ZTMKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFR3b1N0YWdlc0FyZU5ldmVyUG9vbGVkCTU4MTBmOTZmYTYyNmNmMzhiODg2NTlmMjljZDVjNTA5ZDg5NmQ2MWNkNmRmODZlYmUwYzhmZmQzZWZiZjFmMDYKYm9keQlpbnRlcm5hbC9iZW5jaC9yZWRiYXNlX3Rlc3QuZ28JVGVzdEFEZXNjcmlwdGl2ZVJvd0xlYXZlc091dFRhc2tzV2hvc2VDaGVja0lzUmVkQXRCYXNlCTdjNGVlYjQzMjNmNWE5ZWM2Y2FhZGE3NThiYTFkYjIwM2M2YzliODgxMGY2OTA1ZTQ3MGFlMDZhN2IyZjFmZDMKYm9keQlpbnRlcm5hbC9iZW5jaC9yZWRiYXNlX3Rlc3QuZ28JVGVzdEFSZWNlaXB0Q2Fycmllc0l0c1Rhc2tzUmVkQ2hlY2sJMzRiYjJlNDI0OGU5MmRhNTc3YWJmNWM4ZjdhYzVkODRhNTNhNWEwMzQ0ZTgxYTI5Y2ZhOTQ5N2M0NWIxZDk1MQpib2R5CWludGVybmFsL2JlbmNoL3JlZGJhc2VfdGVzdC5nbwlUZXN0VGhlQXR0ZW1wdFBhdGhMaW5rc1RoZVRvb2xjaGFpbkFUYXNrQ2hlY2tOZWVkcwlkYWRkODYwMWIwYzFmYWY5YjAyYWQ5NzY0NDJiYjJiMGEyZGJiYTRjMzQ1ODRmZDI3ODllZmFmNzE1Njc1MWRmCmJvZHkJaW50ZXJuYWwvYmVuY2gvcmVkYmFzZV90ZXN0LmdvCVRlc3RUaGVUYXNrc1dob3NlQ2hlY2tJc1JlZEF0QmFzZVNheVdoeQkzZTJmMDFiNmZlYjE3MGYzM2U2OTYyNzU5YzZkMDA0MjMwYzE3NTUyNDhmNmYxMjA2M2VmYjMwY2UxMzI5MjI4
  ```
  --- last 10 line(s) of stdout
  === RUN   TestTheAnalysisPairsByTaskAndKeepsEveryAttempt
  --- PASS: TestTheAnalysisPairsByTaskAndKeepsEveryAttempt (0.00s)
  === RUN   TestOnlyTheRegisteredExclusionsAreAdmitted
  --- PASS: TestOnlyTheRegisteredExclusionsAreAdmitted (0.00s)
  === RUN   TestTheVerdictFollowsTheRegisteredBar
  --- PASS: TestTheVerdictFollowsTheRegisteredBar (0.00s)
  === RUN   TestADescriptiveRowLeavesOutTasksWhoseCheckIsRedAtBase
  --- PASS: TestADescriptiveRowLeavesOutTasksWhoseCheckIsRedAtBase (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench	0.191s
  ```
- 2026-10-15 · f98fc6d9* · exit 0 · `set -o pipefail …` · acceptance-sha256:42bb7bf03a0c61b136a4a5d60140f9dd9ecfdeb60bb604ce93b857ee985d5463 · ms:386
- 2026-10-15 · human-observed · S4 approved: Zy read the README sentences against docs/bench/RESULTS.md on 2026-10-15 and picked 'Approve as written' — five INCONCLUSIVE verdicts, each with its cost interval, and the two stages where B made more silent mis-edits than A said plainly
- 2026-10-21 · 6715a44c* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:42bb7bf03a0c61b136a4a5d60140f9dd9ecfdeb60bb604ce93b857ee985d5463 · ms:0 · test-lock-sha256:077c69723b08e9f299b503ddc1a98acd38a2c944340291151244ff64164cb2ad · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUGlsb3RJc05ldmVyQ291bnRlZAkxNzExNDE3YjczMzQ5NzMxMzgwMzNjODE4ZWM1NGFjM2Y2M2Q2NWUyYWUyYjFhNzhkOThiYTU1NjFlYjIwYTVhCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUm93V2l0aG91dFRoZVRhc2tzU3VwZXJzZWRlZFNpbmNlSXNEZXNjcmlwdGl2ZQkwYTc3OGYzZDBjZjE3NWRiMTkyNzEwOWI4YWMwNDRjM2JjOGJhMGJkNDM0OTE4YWY0YjA1ZWUyMTU1MjhlNzY0CmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBbmFseXplUHJpbnRzQVN0YWdlNENhdGVnb3J5Um93CWU0MzJiODc4Njg0YjcyZjA3YjVjMzBkOGUzZDI0MWFmM2JhMjZhY2IwNjliZmE3MmJmY2FjZDBkZTEyMjY4ZjIKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdEFuYWx5emVQcmludHNQQWdhaW5zdEEJZjVlY2I1Mzk0NGZmZjE1YzUyOGZmM2ZmNGRjMjhhNjdlMjJiZmIzZjg4Zjg2ODUxNzRiNjE5MWY3MDYzNzQxZApib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0T25seVRoZVJlZ2lzdGVyZWRFeGNsdXNpb25zQXJlQWRtaXR0ZWQJZGI4NDc0MjRjNzY3NGU3MGFlNTQ3NzdlOTkzNDQ2MWFlOGQ1OTAyYmZkNmUyZGM5MGI0YjQxNmJhYzhiYmY1Zgpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0U3RhZ2VPbmVzUHVibGlzaGVkVmVyZGljdElzVW5jaGFuZ2VkCTk2MDliYmVlMzY5MGJhMWQxYjFjZTIyNWNiZmY5NDM1MWY2ZjBjZDQ3M2Q5OWM5N2E1NWI2MWQyZTY1MmNhNDUKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFRoZUFuYWx5c2lzUGFpcnNCeVRhc2tBbmRLZWVwc0V2ZXJ5QXR0ZW1wdAliMDkwM2ZkZGZmNWYzMmY5ZGM4YTRhZTU2MjkzZmUxZTA0MjQwMjBmNTE1NThlZjkwMjZmYzQ3NzEwYzI3MDFiCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RUaGVQdWJsaXNoZWRTdGFnZXNXaXRob3V0VGhlVGFza3NTdXBlcnNlZGVkU2luY2UJODBhYTllOTZmMjIwY2IwMmMzNTAxMWI4ZTNkYzJhOWYzYTIwM2RhYTdiOTk5NzZmZmQ0ZmI2ZjU3ZTgwOTZmYgpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VGhlVmFyaWVkTXVsdGlTaXRlUm93SXNEZXNjcmlwdGl2ZQk2ZWVmNDMxYWE2ZTdjOWJkYmZiOThmYmU3OGM3YTMxNmI3YmMwZTRmY2NhYWIxY2U2MDVlYTFlNmI3ZTI5NTQxCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RUaGVWZXJkaWN0Rm9sbG93c1RoZVJlZ2lzdGVyZWRCYXIJNTE2NTE2MzQyZTA2YTdiNjM5NWRmMDg0MmE3NDk3ZmMxN2I0ZDUwNzBkMDFkMjMzZjU1ZmQzMjBiODUzMzllMwpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VHdvU3RhZ2VzQXJlTmV2ZXJQb29sZWQJNTgxMGY5NmZhNjI2Y2YzOGI4ODY1OWYyOWNkNWM1MDlkODk2ZDYxY2Q2ZGY4NmViZTBjOGZmZDNlZmJmMWYwNgpib2R5CWludGVybmFsL2JlbmNoL3JlZGJhc2VfdGVzdC5nbwlUZXN0QURlc2NyaXB0aXZlUm93TGVhdmVzT3V0VGFza3NXaG9zZUNoZWNrSXNSZWRBdEJhc2UJN2M0ZWViNDMyM2Y1YTllYzZjYWFkYTc1OGJhMWRiMjAzYzZjOWI4ODEwZjY5MDVlNDcwYWUwNmE3YjJmMWZkMwpib2R5CWludGVybmFsL2JlbmNoL3JlZGJhc2VfdGVzdC5nbwlUZXN0QVJlY2VpcHRDYXJyaWVzSXRzVGFza3NSZWRDaGVjawkzNGJiMmU0MjQ4ZTkyZGE1NzdhYmY1YzhmN2FjNWQ4NGE1M2E1YTAzNDRlODFhMjljZmE5NDk3YzQ1YjFkOTUxCmJvZHkJaW50ZXJuYWwvYmVuY2gvcmVkYmFzZV90ZXN0LmdvCVRlc3RUaGVBdHRlbXB0UGF0aExpbmtzVGhlVG9vbGNoYWluQVRhc2tDaGVja05lZWRzCTVkZDg3MWE2MDNmYTA4Yjk3MjhlM2RiOGZiMDllMTliNjQ4ZjA1ZDZkZjhkMmI5OTgxNmY2NWQ2ZGRlYmI3NDEKYm9keQlpbnRlcm5hbC9iZW5jaC9yZWRiYXNlX3Rlc3QuZ28JVGVzdFRoZVRhc2tzV2hvc2VDaGVja0lzUmVkQXRCYXNlU2F5V2h5CTNlMmYwMWI2ZmViMTcwZjMzZTY5NjI3NTljNmQwMDQyMzBjMTc1NTI0OGY2ZjEyMDYzZWZiMzBjZTEzMjkyMjg · test-lock-kind:replace

## Invariants

- The bar in BACKLOG is never edited after T3 starts; an amendment is a new registration before a new run, as ADR-119's were.

## Risks

- A verdict read as a general claim: README names the models, the hosts and the task sources it holds for.

## Out of Scope

- A claim about hosts other than Claude Code (permanent: boundary: Decision 1 runs one host)

## Stop Condition

The fence exits 0 and Zy has read the README sentence.
