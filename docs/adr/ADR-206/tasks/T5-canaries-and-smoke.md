# Task ADR-206-T5: the canaries and the uncounted smoke

**Depends-on:** T1, T2, T3, T4
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `bench/stage-9/instructions.txt`; `docs/bench/stage-9/pilot.md`; four canaries; `docs/bench/stage-9/smoke/`
**Consumes:** everything T1 to T4 produce
**Data dependency:** needs Claude Code, both models and the two builds; held to $5 and about half an hour of attempt time (estimates)
**Proof map:** v1
**Rests-on:** `pilot.md names every pinned thing and its hashes are the files'`, `each smoke run carries its arm's binary and the frozen instructions`, `a scripted exposed write left a kept log in N`, `a run stopped on spend`

## Goal

ADR-206 Decision 9. `pilot.md` has one `key: value` line each for: `tag`, `claude code`, `bench revision`, `m sha256`, `n sha256`, `instructions sha256`, `classes sha256`, `script sha256`, `helper sha256`, `seed`, and the six figures. A canary per model for each of the two binaries, run with the unprofiled invocation, each recording its hashes. A smoke, uncounted, per model: two tasks in each of A, M and N; in N one further attempt of a fixture written to provoke an exposed write, so the kept log is not empty; and one run of two planned attempts with `--max-spend 0.01`, which stops after its first.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/stage-9/instructions.txt` | new | M's instructions, frozen |
| `docs/bench/stage-9/pilot.md`, `canary-*.json`, `smoke/**` | new | the pinned stage and its smoke |

## Ordered Steps

1. [S1] The red state: the fence fails with no `pilot.md`. [proof: acceptance]
2. [S2] Load the costly-runs skill. Build both binaries (T2), freeze the instructions, run the four canaries and then the smoke into `~/.local/share/mrw-bench/runs/stage9/smoke`; copy receipts, kept logs and `run.json`. [proof: human: the canaries' passes and the smoke's receipts read by Zy]
3. [S3] `pilot.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
R=docs/bench/stage-9
v() { sed -n "s/^$1: //p" $R/pilot.md; }
for k in tag 'claude code' 'bench revision' 'm sha256' 'n sha256' 'instructions sha256' 'classes sha256' 'script sha256' 'helper sha256' seed; do test -n "$(v "$k")" || exit 1; done
test "$(v 'instructions sha256')" = "$(shasum -a 256 bench/stage-9/instructions.txt | cut -d' ' -f1)" \
  && test "$(v 'classes sha256')" = "$(shasum -a 256 bench/stage-9/classes.json | cut -d' ' -f1)" \
  && test "$(v 'script sha256')" = "$(shasum -a 256 scripts/bench_stage9.py | cut -d' ' -f1)" \
  && test "$(v 'helper sha256')" = "$(shasum -a 256 scripts/bench_mechanism.py | cut -d' ' -f1)" \
  && test "$(v 'm sha256')" = "$(sed -n 's/^m sha256: //p' bench/stage-9/builds.md)" && test "$(v 'n sha256')" = "$(sed -n 's/^n sha256: //p' bench/stage-9/builds.md)" \
  && for m in claude-opus-5-5 claude-sonnet-5-5; do for a in m n; do jq -e --arg h "$(v "$a sha256")" --arg s "$(v 'instructions sha256')" --arg c "$(v 'claude code')" '.mrw_sha256 == $h and .mrw_instructions_sha256 == $s and .cli_version == $c' $R/smoke/$m/$a/run.json > /dev/null || exit 1; done; done \
  && for m in claude-opus-5-5 claude-sonnet-5-5; do cat $R/smoke/$m/n/*.json | jq -e -s '[.[] | select(.attempt != null)] | length > 0 and any(.[]; (.kept_state_file_sha256 // "") | test("^[0-9a-f]{64}$"))' > /dev/null || exit 1; test -s "$(ls $R/smoke/$m/n/*.jsonl | head -1)" || exit 1; done \
  && jq -e '.max_spend_usd == 0.01 and (.tasks | length) == 2' $R/smoke/stop/run.json > /dev/null \
  && [ "$(cat $R/smoke/stop/*.json | jq -s '[.[] | select(.attempt != null)] | length')" -eq 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence: `pilot.md`'s hashes are the files'; each smoke `run.json` of M and N carries its arm's binary hash, the frozen instruction hash and the pinned Claude Code; an N receipt carries a kept log that is not empty; the stopped run planned two attempts and holds one receipt | none | — |

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

- The unprofiled invocation may not work with the current Claude Code: the canary is where that shows, before anything is pinned.

## Stop Condition

The fence exits 0 and Zy has read `pilot.md`.

## Out of Scope

- The counted attempts (permanent: boundary: ADR-206 T6)
