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
  && for m in claude-sonnet-5-5 claude-opus-5-5; do go run ./cmd/bench export-paired --pair A,P --model $m $R | jq -e -s '(.[0] | has("config")) and length == 5' > /dev/null || exit 1; done
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
