# Task ADR-193-T3: the canary and an uncounted smoke freeze the paragraph

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** the frozen paragraph and its sha, `docs/bench/stage-8/pilot.md` and the canary files
**Consumes:** configuration S and profile `stage8` (T1), `bench_edits.py count` (T2)
**Data dependency:** needs mrw v1.83.0's release asset, Claude Code 2.1.317 (`~/.local/share/mrw-bench/claude-2.1.317`), both models, and the 2 instances ADR-192 T3 used
**Proof map:** v1
**Rests-on:** `the paragraph is recorded`, `the sha hashes the recorded paragraph`, `the paragraph is the one in the code`, `the CLI is recorded`, `the canary passed for B and S on both models`

## Goal

Before any counted attempt: the stage8 canary passing on both models, and an uncounted smoke showing whether the
paragraph moves the agent to SEARCH/REPLACE writes (ADR-193 Decisions 1 and 6).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/profile.go` | edit | `EditParagraph` if S3 changes it |
| `internal/bench/runner_test.go` | edit | the paragraph if S3 changes it |
| `docs/adr/ADR-193-claude-edits-lead-with-search-replace.md` | edit | Decision 1's quoted paragraph if S3 changes it |
| `docs/bench/stage-8/pilot.md` | add | the CLI and mrw versions, the canaries, the smoke's counts, the frozen paragraph verbatim and its sha |
| `docs/bench/stage-8/canary-<model>.json` | add | one canary per model |

## Ordered Steps

1. [S1] The red state: `docs/bench/stage-8/pilot.md` does not exist, so the fence fails. [proof: acceptance]
2. [S2] The stage8 canary for B and S on each model. A failed canary stops here. [proof: acceptance]
3. [S3] A preflight in a scratch git checkout through the stage's settings and bench's Bash hook: S writes a SEARCH/REPLACE document to `$TMPDIR` by heredoc and runs `mrw write --format=search_replace` on it with no `mrw read` of the file; the write applies, the hook does not refuse it, and `bench_edits.py count` counts one landed, attributed SEARCH/REPLACE write call. [proof: human: the receipt and the count]
4. [S4] B and S, uncounted, on `matplotlib__matplotlib-22719` and `astropy__astropy-13033` (the first 2 of `bench/swe-bench/selection.json`, outside the pool), both models; `bench_edits.py count` over the 8 transcripts. When S makes no attributed SEARCH/REPLACE write call on at least one of the two models, change the paragraph once (code, test and record together, still directive), run again, and record both rounds. [proof: human: the smoke's counts and the paragraph it froze]
5. [S5] Write `pilot.md` with a `cli: ` line holding the CLI's version as receipts record it, a `paragraph: ` line holding the frozen paragraph and a `paragraph sha256: ` line; commit before T4. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-8
p=$(sed -n 's/^paragraph: //p' $R/pilot.md) && test -n "$p" \
  && test "$(printf '%s' "$p" | shasum -a 256 | cut -d' ' -f1)" = "$(sed -n 's/^paragraph sha256: //p' $R/pilot.md)" \
  && grep -qF -- "$p" internal/bench/profile.go \
  && cli=$(sed -n 's/^cli: //p' $R/pilot.md) && test -n "$cli" \
  && for m in claude-sonnet-5-5 claude-opus-5-5; do jq -e --arg m "$m" --arg cli "$cli" '.pass == true and .profile == "stage8" and .model == $m and (.configs | sort) == ["B","S"] and .cli_version == $cli' "$R/canary-$m.json" > /dev/null || exit 1; done
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: the recorded paragraph hashes to its sha and is the one in the code; both models' stage8 canaries passed with B and S | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the pilot record and canary files |
| 2 — something selects it | T4 runs with the paragraph it froze |
| 3 — the caller can discover it | `run.json` `edit_paragraph_sha256` |
| 4 — it is used | T4, and T5 on SHIP |

## Mutation Log

## Invariants

- No canary or smoke attempt is counted or kept in T4's directory.

## Risks

- Two instances say little about uptake over 18; (U) in T4 is the counted check.

## Stop Condition

The preflight's write is refused by bench's hook or counted as anything but a landed, attributed SEARCH/REPLACE write, or after one change S still makes no attributed SEARCH/REPLACE write call on at least one model: stop to Zy before T4.

## Out of Scope

- Any counted attempt (T4's job).

## Verification Log
