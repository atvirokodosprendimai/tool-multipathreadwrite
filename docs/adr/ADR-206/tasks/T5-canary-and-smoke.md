# Task ADR-206-T5: the canary and the uncounted smoke

**Depends-on:** T1, T2, T3, T4
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `docs/bench/stage-9/` canaries, the instructions file and its hash, `pilot.md` naming every pinned thing, the smoke's receipts
**Consumes:** everything T1 to T4 produce
**Data dependency:** needs Claude Code, both models and the M release asset; about $2 and twenty minutes of attempt time
**Proof map:** v1
**Rests-on:** `every pinned hash is written down before a counted attempt`, `the smoke shows the kept log in N`, `the smoke shows M and N with one instruction hash`, `the stop works on real receipts`

## Goal

ADR-206 Decision 9. `docs/bench/stage-9/pilot.md` names the Claude Code release, the models, M's tag and asset hash, N's hash, the instruction file's hash, the class list's hash, the script's and helper's hashes, the seed and the three `--max-spend` figures. A canary per model for A and for B. A smoke of two tasks per arm per model, uncounted, whose `run.json` and receipts show the hashes, whose N receipts carry the kept log's hash, and one run started with `--max-spend 0.01` that stops after its first attempt.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/stage-9/pilot.md`, `canary-*.json`, `smoke/` | new | the pinned stage and its smoke |
| `bench/stage-9/instructions.txt` | new | M's instructions, frozen |

## Ordered Steps

1. [S1] Load the costly-runs skill. Build N (T2), download and verify M, freeze the instructions. [proof: acceptance]
2. [S2] The canaries, then the smoke, into `~/.local/share/mrw-bench/runs/stage9/smoke`; copy receipts and `run.json`. [proof: human: the smoke's receipts and the stop's message]
3. [S3] `pilot.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-9
for k in 'claude code' 'm sha256' 'n sha256' 'instructions sha256' 'classes sha256' 'script sha256' 'helper sha256' 'seed'; do grep -qi "^$k: " $R/pilot.md || exit 1; done \
  && test "$(sed -n 's/^instructions sha256: //p' $R/pilot.md)" = "$(shasum -a 256 bench/stage-9/instructions.txt | cut -d' ' -f1)" \
  && test "$(sed -n 's/^script sha256: //p' $R/pilot.md)" = "$(shasum -a 256 scripts/bench_stage9.py | cut -d' ' -f1)" \
  && for d in $R/smoke/*/m $R/smoke/*/n; do jq -e --arg s "$(sed -n 's/^instructions sha256: //p' $R/pilot.md)" '.mrw_instructions_sha256 == $s and (.mrw_sha256 | length) == 64' "$d/run.json" > /dev/null || exit 1; done \
  && ls $R/smoke/*/n/*.json | xargs jq -e 'select(.attempt != null) | .kept_state_file_sha256 // "none"' > /dev/null
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: `pilot.md` names every pinned thing and its hashes are the files'; each smoke `run.json` of M and N carries the frozen instruction hash and a binary hash | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the smoke ran |
| 2 — something selects it | T6 |
| 3 — the caller can discover it | `docs/bench/stage-9/pilot.md` |
| 4 — it is used | T6 |

## Mutation Log

## Verification Log

## Invariants

- Nothing in the smoke is counted.

## Risks

- The Claude Code release moves under the stage: it is pinned at the canary and T6 refuses another.

## Stop Condition

The fence exits 0 and Zy has read `pilot.md`.

## Out of Scope

- The counted attempts (deferred: docs/adr/BACKLOG.md)
