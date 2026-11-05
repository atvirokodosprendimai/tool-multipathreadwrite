# Task ADR-192-T3: the release, the re-probe, the canary, a preflight and an uncounted pilot freeze the sentence

**Depends-on:** T1, T2
**Blocked-on:** a tag that contains ADR-191's merge (`git tag --contains 1f01dcdc | grep -q .`)
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** the frozen sentence and its sha, the re-probed built-ins, `docs/bench/stage-7/pilot.md` and the canary files
**Consumes:** configuration H and profile `stage7` (T1), comparisons `H:B` (T2)
**Data dependency:** needs the release binary, a pinned Claude Code of 2.1.316 or later, and 2 instances outside the pool
**Proof map:** v1
**Rests-on:** `the canary passed for A, B and H on both models`, `the sentence in the record is the one in the code`, `the pilot record exists`

## Goal

Before any counted attempt: the built-ins re-probed on the stage's CLI, the stage7 canary passing on both models,
a preflight showing the hook licenses, and an uncounted pilot showing whether the sentence moves the agent to the
host Read (ADR-192 Decisions 1, 2 and 6).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/profile.go` | edit | the built-in skills and plugins the stage's CLI lists (`:28-35`); the sentence if S5 changes it |
| `internal/bench/runner_test.go` | edit | the sentence if S5 changes it |
| `docs/adr/ADR-192-the-read-hook-is-measured-on-swe-bench.md` | edit | Decision 1's quoted sentence if S5 changes it |
| `docs/bench/stage-7/pilot.md` | add | the CLI version, the probe, the preflight, the pilot's counts, the frozen sentence verbatim and its sha |
| `docs/bench/stage-7/canary-<model>.json` | add | one canary per model |

## Ordered Steps

1. [S1] The red state: `docs/bench/stage-7/pilot.md` does not exist, so the fence fails. [proof: acceptance]
2. [S2] An uncounted probe on the stage's CLI records the built-in skills and plugins; `profile.go` takes them. [proof: human: the probe's init line against the edited lists]
3. [S3] The stage7 canary for A, B and H on each model. A failed canary stops here. [proof: acceptance]
4. [S4] A preflight in a scratch checkout through the stage's settings: H Reads a file, then applies a plan editing lines of it with no `mrw read` between; the same with the hook removed is refused as unread. [proof: human: both receipts, the applied one and the refused one]
5. [S5] H uncounted on the first 2 instances of `bench/swe-bench/selection.json` (outside the pool), both models: `host_reads`, `hook_runs`, the proxies. With no host Read on either model, change the sentence once (code, test and record together), run again, and record both. [proof: human: the pilot's counts and the sentence it froze]
6. [S6] Write `pilot.md` with a `cli: ` line holding the pinned Claude Code's version as receipts record it, a `sentence: ` line holding the frozen sentence and a `sentence sha256: ` line; commit before T4. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-7
s=$(sed -n 's/^sentence: //p' $R/pilot.md) && test -n "$s" \
  && test "$(printf '%s' "$s" | shasum -a 256 | cut -d' ' -f1)" = "$(sed -n 's/^sentence sha256: //p' $R/pilot.md)" \
  && grep -qF -- "$s" internal/bench/profile.go \
  && cli=$(sed -n 's/^cli: //p' $R/pilot.md) && test -n "$cli" \
  && for m in claude-sonnet-5-5 claude-opus-5-5; do jq -e --arg m "$m" --arg cli "$cli" '.pass == true and .profile == "stage7" and .model == $m and (.configs | sort) == ["A","B","H"] and .cli_version == $cli' "$R/canary-$m.json" > /dev/null || exit 1; done
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: the sentence recorded hashes to its sha and is the one in the code; both models' stage7 canaries passed with H | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the pilot record and canary files |
| 2 — something selects it | T4 runs with the sentence it froze |
| 3 — the caller can discover it | `run.json` `h_sentence_sha256` |
| 4 — it is used | T4 |

## Mutation Log
- 2026-11-03 · 76a2208d* · mutant killed · exit 1 · `docs/bench/stage-7/pilot.md` · S6: the recorded sha no longer hashes the recorded sentence · acceptance-sha256:6cb44f50e237e21f297671e14e48cb1f667d37957e1d92e842389d5d070e5523

## Invariants

- No pilot or preflight attempt is counted or kept in the stage's blocks.

## Risks

- Two instances say little about uptake over 191; the gate is the counted check.

## Stop Condition

The preflight's edit is refused with the hook in place, or the sentence moves no host Read on either model after one change: stop to Zy before T4.

## Out of Scope

- Any counted attempt (T4's job).

## Verification Log
- 2026-11-03 · 76a2208d* · exit 0 · `set -o pipefail …` · acceptance-sha256:6cb44f50e237e21f297671e14e48cb1f667d37957e1d92e842389d5d070e5523 · ms:56
- 2026-11-03 · 76a2208d* · exit 1 · `set -o pipefail …` · acceptance-sha256:6cb44f50e237e21f297671e14e48cb1f667d37957e1d92e842389d5d070e5523 · ms:36 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-file:docs/adr/locks/deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11.lock
  ```
  --- last 1 line(s) of stderr
  sed: docs/bench/stage-7/pilot.md: No such file or directory
  ```
- 2026-11-03 · 76a2208d* · exit 0 · `set -o pipefail …` · acceptance-sha256:6cb44f50e237e21f297671e14e48cb1f667d37957e1d92e842389d5d070e5523 · ms:55
- 2026-11-03 · 76a2208d* · exit 0 · `set -o pipefail …` · acceptance-sha256:6cb44f50e237e21f297671e14e48cb1f667d37957e1d92e842389d5d070e5523 · ms:103
