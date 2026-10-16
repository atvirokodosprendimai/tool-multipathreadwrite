# Task ADR-161-T5: a paired B pilot — v1.66.0 against this record's build, on the same eight tasks

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** S (32 attempts, under an hour)
**Owner:** Zy
**Produces:** `docs/bench/adr-161-pilot/b-old/`, `docs/bench/adr-161-pilot/b-new/` receipts; ADR-160's T4 amended for the rerun
**Consumes:** the flags and the recipes (T1, T2, T3)
**Data dependency:** real agent runs on the subscription, on this Mac; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `the pilot ran B on the named calibration tasks`, `the two arms ran the two builds`

## Goal

Whether this record changes what B does and costs, paired on the same tasks, before Stage 4 runs again, choosing nothing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/adr-161-pilot/b-old/` | add | the eight tasks × B × 2, Sonnet 5.5, stage `pilot`, mrw v1.66.0 |
| `docs/bench/adr-161-pilot/b-new/` | add | the same, mrw built from T3's commit with `-X main.version=adr161-<short sha>` so the fence can name it |
| `docs/adr/ADR-160-stage-4-measures-mrw-where-its-mechanism-should-matter/tasks/T4-stage-4-on-sonnet-and-opus.md` | edit | the rerun on the release, registered before it runs |

The eight tasks: `am-wide-add-input`, `am-wide-migrations`, `am-wide-entry-room`, `am-wide-with-reranker`, `am-coupled-installer-bash`, `am-coupled-doctor-discarded`, `am-coupled-doctor-duplicated`, `am-coupled-agentkit-commands` — the last calibration round of each own category, never in `bench/stage-4-tasks.txt`, each with A receipts in `docs/bench/stage-4-pilot/calibration-2/` or `calibration-3/`.

## Ordered Steps

1. [S1] See the fence red: no pilot receipts. [proof: acceptance]
2. [S2] Zy's word to start, with its price. [proof: human: Zy says go for the ADR-161 pilot]
3. [S3] Both arms, sequential; every transcript read; per shape, mean cost, tool calls and output tokens for b-old, b-new and the calibration A of the same tasks, said to be descriptive. [proof: human: the executor records the comparison and what B did]
4. [S4] ADR-160's T4 amended to rerun Stage 4 on the release carrying this record, Sonnet's v1.66.0 stage kept as run. [proof: human: the executor records the amendment's commit]

## Acceptance

```bash
set -o pipefail
P=docs/bench/adr-161-pilot
ids='["am-coupled-agentkit-commands","am-coupled-doctor-discarded","am-coupled-doctor-duplicated","am-coupled-installer-bash","am-wide-add-input","am-wide-entry-room","am-wide-migrations","am-wide-with-reranker"]'
for arm in b-old b-new; do
  go run ./cmd/bench verify --stage pilot --tasks 8 --repeats 2 $P/$arm \
    && jq -e --argjson ids "$ids" '.configs == ["B"] and .models == ["claude-sonnet-5-5"] and (.tasks | sort) == $ids' $P/$arm/run.json > /dev/null || exit 1
done
jq -e '.mrw_version == "v1.66.0 (05ec3f8)"' $P/b-old/run.json > /dev/null \
  && jq -e '.mrw_version | startswith("adr161-")' $P/b-new/run.json > /dev/null
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPilotIsNeverCounted` | `internal/bench/analyze_test.go` | a pilot receipt is in no verdict | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the receipts |
| 2 — something selects it | `bench run --stage pilot --configs B --mrw <build>` |
| 3 — the caller can discover it | `docs/bench/adr-161-pilot/` |
| 4 — it is used | the Stage 4 rerun |

## Mutation Log
(empty until execute)
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `docs/bench/adr-161-pilot/b-new/run.json` · b-new ran the old build: the fence must tell the arms apart by build · acceptance-sha256:3e34169d883a998ceef138d361605143eea87b2dd46272ab2c16a2ccecead4d7
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `docs/bench/adr-161-pilot/b-old/run.json` · b-old ran a build other than v1.66.0 · acceptance-sha256:3e34169d883a998ceef138d361605143eea87b2dd46272ab2c16a2ccecead4d7 · covers:the two arms ran the two builds
- 2026-10-16 · 81668383* · mutant killed · exit 1 · `docs/bench/adr-161-pilot/b-new/run.json` · one arm ran a task outside the eight named calibration tasks · acceptance-sha256:3e34169d883a998ceef138d361605143eea87b2dd46272ab2c16a2ccecead4d7 · covers:the pilot ran B on the named calibration tasks

## Invariants

- No task of `bench/stage-4-tasks.txt` runs in the pilot (the fence checks the ids).

## Risks

- Eight tasks × 2 per arm cannot show a small difference; descriptive only.

## Stop Condition

b-new breaks a task b-old solved, for a reason in this record's code.

## Out of Scope

- Any verdict (permanent: boundary: a pilot chooses nothing)

## Verification Log
(empty until execute)
- 2026-10-16 · 9db0e7ba · exit 1 · `set -o pipefail …` · acceptance-sha256:3e34169d883a998ceef138d361605143eea87b2dd46272ab2c16a2ccecead4d7 · ms:105 · test-lock-sha256:7875947846e94d6921265c031a584cbf52caf35a23d6f94c00a98eb09c647251 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUGlsb3RJc05ldmVyQ291bnRlZAkxNzExNDE3YjczMzQ5NzMxMzgwMzNjODE4ZWM1NGFjM2Y2M2Q2NWUyYWUyYjFhNzhkOThiYTU1NjFlYjIwYTVhCmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBUm93V2l0aG91dFRoZVRhc2tzU3VwZXJzZWRlZFNpbmNlSXNEZXNjcmlwdGl2ZQkwYTc3OGYzZDBjZjE3NWRiMTkyNzEwOWI4YWMwNDRjM2JjOGJhMGJkNDM0OTE4YWY0YjA1ZWUyMTU1MjhlNzY0CmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RBbmFseXplUHJpbnRzQVN0YWdlNENhdGVnb3J5Um93CWU0MzJiODc4Njg0YjcyZjA3YjVjMzBkOGUzZDI0MWFmM2JhMjZhY2IwNjliZmE3MmJmY2FjZDBkZTEyMjY4ZjIKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdE9ubHlUaGVSZWdpc3RlcmVkRXhjbHVzaW9uc0FyZUFkbWl0dGVkCWRiODQ3NDI0Yzc2NzRlNzBhZTU0Nzc3ZTk5MzQ0NjFhZThkNTkwMmJmZDZlMmRjOTBiNGI0MTZiYWM4YmJmNWYKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFN0YWdlT25lc1B1Ymxpc2hlZFZlcmRpY3RJc1VuY2hhbmdlZAk5NjA5YmJlZTM2OTBiYTFkMWIxY2UyMjVjYmZmOTQzNTFmNmYwY2Q0NzNkOTljOTdhNTViNjFkMmU2NTJjYTQ1CmJvZHkJaW50ZXJuYWwvYmVuY2gvYW5hbHl6ZV90ZXN0LmdvCVRlc3RUaGVBbmFseXNpc1BhaXJzQnlUYXNrQW5kS2VlcHNFdmVyeUF0dGVtcHQJYjA5MDNmZGRmZjVmMzJmOWRjOGE0YWU1NjI5M2ZlMWUwNDI0MDIwZjUxNTU4ZWY5MDI2ZmM0NzcxMGMyNzAxYgpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VGhlUHVibGlzaGVkU3RhZ2VzV2l0aG91dFRoZVRhc2tzU3VwZXJzZWRlZFNpbmNlCTgwYWE5ZTk2ZjIyMGNiMDJjMzUwMTFiOGUzZGMyYTlmM2EyMDNkYWE3Yjk5OTc2ZmZkNGZiNmY1N2U4MDk2ZmIKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFRoZVZhcmllZE11bHRpU2l0ZVJvd0lzRGVzY3JpcHRpdmUJNmVlZjQzMWFhNmU3YzliZGJmYjk4ZmJlNzhjN2EzMTZiN2JjMGU0ZmNjYWFiMWNlNjA1ZWExZTZiN2UyOTU0MQpib2R5CWludGVybmFsL2JlbmNoL2FuYWx5emVfdGVzdC5nbwlUZXN0VGhlVmVyZGljdEZvbGxvd3NUaGVSZWdpc3RlcmVkQmFyCTUxNjUxNjM0MmUwNmE3YjYzOTVkZjA4NDJhNzQ5N2ZjMTdiNGQ1MDcwZDAxZDIzM2Y1NWZkMzIwYjg1MzM5ZTMKYm9keQlpbnRlcm5hbC9iZW5jaC9hbmFseXplX3Rlc3QuZ28JVGVzdFR3b1N0YWdlc0FyZU5ldmVyUG9vbGVkCTU4MTBmOTZmYTYyNmNmMzhiODg2NTlmMjljZDVjNTA5ZDg5NmQ2MWNkNmRmODZlYmUwYzhmZmQzZWZiZjFmMDY
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/adr-161-pilot/b-old/run.json: no such file or directory
  exit status 1
  ```
- 2026-10-16 · human-observed · S2 approved by Zy, 2026-10-16, in this session: "approved the use of sonet, since it uses subscription" — the ADR-161 pilot, 32 attempts on Sonnet 5.5 (8 tasks x B x 2, two arms: v1.66.0 and adr161-9db0e7ba), on the subscription.
- 2026-10-16 · human-observed · S3 observed by the executor, 2026-10-16: both arms ran on Sonnet 5.5, CLI 2.1.301, canary passed, 8 tasks x B x 2 each, every transcript read; b-old is mrw v1.66.0 (05ec3f8), b-new is adr161-9db0e7ba. Descriptive means per attempt: wide-rename, b-old 0.173 USD, 3.1 tool calls, 1848 output tokens, 6 of 8 solved; b-new 0.128 USD, 2.6 calls, 702 tokens, 6 of 8; calibration A 0.131 USD, 3.9 calls, 864 tokens, 6 of 8. check-coupled, b-old 0.153 USD, 7 of 8 solved; b-new 0.140 USD, 5 of 6 admitted; calibration A 0.136 USD, 7 of 8. b-new cost less than b-old on every task and solved every task b-old solved. The harness set aside 2 b-old and 3 b-new coupled attempts as setup (the .r1 receipts are the reruns); b-new doctor-discarded repeat 1 and installer-bash repeat 2 were set aside twice, each time the agent ran cd into a subdirectory before mrw, whose root then defaulted to that subdirectory and split the ledger: friction in root discovery, outside this record's code. What B did differently: b-new wrote wide-rename edits with mrw write --grep --sub in one call where b-old read every site and built the plan over several turns.
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:3e34169d883a998ceef138d361605143eea87b2dd46272ab2c16a2ccecead4d7 · ms:178
- 2026-10-16 · human-observed · S3 observed by the executor, 2026-10-16, superseding one sentence of the row above: per task over admitted attempts, b-new solved every task b-old solved except am-coupled-installer-bash, where b-old solved 1 of 2 and b-new 0 of 1 (its repeat 2 was set aside twice as setup). Both arms' unsolved installer-bash attempts made the same one-line edit and left clients/claude-code/installer.go still matching "bash " (receipts b-old 1.r1, b-new 1): the same partial edit on both builds, so the Stop Condition, a break caused by this record's code, is unmet and the pilot stands as recorded.
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:3e34169d883a998ceef138d361605143eea87b2dd46272ab2c16a2ccecead4d7 · ms:949
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:3e34169d883a998ceef138d361605143eea87b2dd46272ab2c16a2ccecead4d7 · ms:778
- 2026-10-16 · 81668383* · exit 0 · `set -o pipefail …` · acceptance-sha256:3e34169d883a998ceef138d361605143eea87b2dd46272ab2c16a2ccecead4d7 · ms:729
