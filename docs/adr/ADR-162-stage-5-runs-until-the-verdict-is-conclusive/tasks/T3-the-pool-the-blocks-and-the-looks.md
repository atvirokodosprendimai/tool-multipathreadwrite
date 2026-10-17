# Task ADR-162-T3: the pool, the blocks and the looks

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** L (multi-file)
**Owner:** Zy
**Produces:** `bench/swe-bench/selection-adr-162.json`, the pool's clones in `bench/repos.json`, `bench run --block --configs-for`, `bench look`, `bench analyze`'s stage levels, Stage 5's table and P against A
**Consumes:** the receipt's `status`, `oracle.broke_tests` and `oracle.harness_resolved` (T2), configuration P and the profile (T1)
**Data dependency:** the SWE-bench Verified dataset file ADR-156 T5 recorded, for the selection; network, for the clones; the rest hermetic
**Proof map:** v1
**Rests-on:** `the pool is the salted 2-hunk Verified set less earlier slices`, `a look reads the stage's level and only when eligible`, `a model stops when both comparisons close`, `P is compared with A by ADR-156's rule`, `a block is a directory with per-model configurations`

## Goal

The registered pool is a file a reader can regenerate, and a block is a fixed range of it in a directory of its own. At each of the five looks, `bench look` gives every model's and comparison's verdict at the stage's 99%, whether the look was eligible, and what each model runs next. `bench analyze` prints Stage 5 in its own table beside the earlier stages, unchanged (ADR-162 Decisions 4 to 7).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/swebench.go` | edit | the selection takes several exclusion files |
| `internal/bench/analyze.go` | edit | the level and looks from each stage's `run.json`; P against A by the same rule; P's silent mis-edits counted as P's (`analyze.go:241-244`); Stage 5's own table, whose header names 99% and the closing look; the 5+ hunk descriptive row; reading across `block-K/` directories |
| `internal/bench/look.go` | add | `Look`: eligibility, the verdict at the stage's level, the look a comparison closed at, the configurations each model runs next, completeness |
| `internal/bench/look_test.go` | add | the stop rule |
| `internal/bench/analyze_test.go` | edit | the level and P against A |
| `internal/bench/swebench_test.go` | edit | the pool |
| `internal/bench/runner.go` | edit | `--block K` and `--configs-for MODEL=LIST`; `claimOut` and `plan` per block directory; three consecutive start-errors stop the run (exit 2, resumable); a block whose CLI version differs from block 1's is refused |
| `internal/bench/runner_test.go` | edit | per-model configurations; the start-error stop; the CLI pin |
| `cmd/bench/main.go` | edit | `--exclude` and `--selection` repeatable, `--block`, `--configs-for`, `bench look`; `bench verify` with per-model configurations; the usage text |
| `bench/swe-bench/selection-adr-162.json` | add | the 191 ids in registered order, with the 5+ hunk subset and no `unnamed` |
| `bench/repos.json` | edit | the repositories the 191 need that no earlier stage cloned |
| `docs/bench/README.md` | edit | the blocks, the looks, eligibility and the stop rule |

## Ordered Steps

1. [S1] Write the five failing tests and see them red. [proof: acceptance]
2. [S2] The pool: `bench swebench-select --dataset … --salt adr-162 --min-hunks 2 --exclude bench/swe-bench/selection.json --exclude bench/swe-bench/selection-adr-160.json --n 191` (the CLI appends the colon), with the 5+ hunk subset and no `unnamed`. Clone every repository the 191 need and add it to `bench/repos.json`. [proof: mutation]
3. [S3] `bench analyze` takes the level and looks from each stage's `run.json`, so earlier stages read 95%. In Stage 5 the solved interval is Tango's score interval for a paired difference, and cost keeps the percentile bootstrap. P against A is ADR-156's rule with P in B's place. Stage 5 prints its own table. `--selection` repeats. [proof: mutation]
4. [S4] `bench look DIR`, per model and comparison:
   - A look is eligible when every scheduled attempt has a receipt, exclusions are at most 5% per configuration, and at least 95% of the instances run are paired.
   - The verdict is read at 99% at each eligible look. The comparison closes at the first one that is PAYS or LOSES, and later attempts of its second configuration are not read.
   - It prints the configurations each model runs next.
   - It exits 0 when the stage is complete (after the fifth look, or when no model runs on) and 1 when a block is due. It refuses attempts the rule did not schedule.

   [proof: mutation]
5. [S5] `bench run --block K --configs-for MODEL=LIST` runs instances 40(K−1)+1 to 40K (block 5: 161 to 191) into `block-K/`, each instance's attempts together with its image pulled once. Three consecutive start-errors stop the run, and a block whose CLI version differs from block 1's is refused. Wire the flags, `bench verify`'s per-model configurations, and the usage text. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestTheStage5PoolIsTheSaltedTwoHunkSetLessEarlierSlices|TestALookUsesTheStageLevelAndOnlyWhenEligible|TestAModelStopsWhenBothComparisonsAreConclusive|TestAnalyzePrintsPAgainstA|TestABlockRunsPerModelConfigurations)$' 2>&1) \
  && for t in TestTheStage5PoolIsTheSaltedTwoHunkSetLessEarlierSlices TestALookUsesTheStageLevelAndOnlyWhenEligible TestAModelStopsWhenBothComparisonsAreConclusive TestAnalyzePrintsPAgainstA TestABlockRunsPerModelConfigurations; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go run ./cmd/bench swebench-select --dataset "$HOME/.local/share/mrw-bench/swe-probe/verified-78f471bf.json" --salt adr-162 --min-hunks 2 --exclude bench/swe-bench/selection.json --exclude bench/swe-bench/selection-adr-160.json --n 191 > /tmp/adr162-sel.json \
  && jq -S . /tmp/adr162-sel.json > /tmp/adr162-regen.json \
  && jq -S . bench/swe-bench/selection-adr-162.json > /tmp/adr162-committed.json \
  && diff /tmp/adr162-regen.json /tmp/adr162-committed.json \
  && test "$(jq '.selected | length' bench/swe-bench/selection-adr-162.json)" = 191 \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'bench look'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSWEBenchSliceIsTheLowestTwentyHashes` | `internal/bench/swebench_test.go` | placeholder until this task's tests land: Stage 3's selection does not move | none | — |

Added by this task, and rowed here when they exist:
- `TestTheStage5PoolIsTheSaltedTwoHunkSetLessEarlierSlices` (`internal/bench/swebench_test.go`; S2): the `adr-162:` hash, two exclusion files, the 2-hunk floor, the 5+ subset, no `unnamed`; Stage 3's and 4's selections unchanged.
- `TestALookUsesTheStageLevelAndOnlyWhenEligible` (`internal/bench/look_test.go`; S3, S4): a fixture whose 95% cost interval excludes 0 and whose 99% one does not is open at a look. 40 paired instances with no discordant solved outcome and a −20% cost do not close PAYS: the score interval is not [0, 0]. A look with a missing receipt, 6% exclusions or 94% pairing closes nothing. An earlier stage still reads 95% with the bootstrap.
- `TestAModelStopsWhenBothComparisonsAreConclusive` (`internal/bench/look_test.go`; S4): a comparison closes at the first conclusive eligible look and its later attempts are unread. A model with one open comparison runs A and that configuration. The stage is complete after the fifth look. An unscheduled attempt is refused.
- `TestAnalyzePrintsPAgainstA` (`internal/bench/analyze_test.go`; S3): a stage with P prints B against A and P against A in its own table, never P against B. P's silent mis-edits are P's.
- `TestABlockRunsPerModelConfigurations` (`internal/bench/runner_test.go`; S5): `--configs-for`, a block in its own directory, three start-errors stopping the run, and a CLI version mismatch refused.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests and the selection file |
| 2 — something selects it | `--block`, `--configs-for` and `bench look` in `cmd/bench/main.go`; the fence regenerates the selection and runs `go run ./cmd/bench` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T5 runs block by block on `bench look`'s answer |

## Mutation Log
(empty until execute)

## Invariants

- Earlier stages' verdicts and descriptive rows are unchanged: their level stays 95%, they have no looks, and their selection files are still read.
- No look, level, block edge or eligibility rule is chosen after a block has run.

## Risks

- A repository the pool needs may not build its SWE-bench image under emulation; T4's pilot does not reach it, so its first instance in block 1 is the probe, and a setup exclusion there counts under the 5% rule.

## Stop Condition

The fence exits 0; or the dataset file's revision is not `78f471bf…`, or a needed repository cannot be cloned, either of which goes back to Zy.

## Out of Scope

- A group-sequential boundary other than Bonferroni (permanent: boundary: Decision 5 registers Bonferroni because its level needs no share of the final information, which exclusions and closed comparisons move)

## Verification Log
(empty until execute)
