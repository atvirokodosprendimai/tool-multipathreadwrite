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
