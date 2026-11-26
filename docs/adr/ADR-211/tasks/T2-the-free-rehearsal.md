# Task ADR-211-T2: a free rehearsal of everything the paid tasks rely on

**Depends-on:** T1, T3
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `bench/stage-9/instructions.txt`, committed; `bench/stage-9/rehearse.sh` and its stub agent `bench/stage-9/stub-agent`; `docs/bench/stage-9/rehearsal.md`
**Consumes:** T1's instrument and plans files; T3's `gate`, `review`, `project` and amended paid tasks; ADR-206 T2's `build.sh`
**Data dependency:** the pilot Mac, a clean checkout at the revision `pilot.md` will name, the pinned tag, every task's local repository; no model and no spend; four stub canaries and the 240 planned attempts of the six registered plans run against the stub (wall-clock minutes: load the costly-runs skill first)
**Proof map:** v1
**Rests-on:** `the frozen instructions are mrw-m's`, `the build test and the oracle test ran unskipped`, `the bench build is clean and identified`, `the tasks are pinned`, `the six registered plans run through the gate`, `count reads runner-written receipts and sees N's event`, `the paid fences fail where they must`, `the rehearsal's negatives failed`, `the record is bound to its revision and inputs`, `each input digest is present once and verified`, `no engine package changed`

## Goal

ADR-211 Decision 10. First, `bench/stage-9/instructions.txt` committed, byte-equal to `mrw-m instructions` from the build `builds.md` names. Then on the pilot Mac, on the day the canaries will run, `rehearse.sh` runs each check below and writes `rehearsal.md`: `date:`, `revision:` (`git rev-parse HEAD`, the tree clean), one `<input> sha256:` line for each of `bench/stage-9/instructions.txt`, `plans.json`, `smoke-plans.json`, `classes.json`, `scripts/bench_stage9.py`, `scripts/bench_mechanism.py`, `bench/stage-9/rehearse.sh`, `bench/stage-9/stub-agent`, and `m`, `n` and `tasks`, each line present exactly once: `m sha256:` and `n sha256:` the digests `builds.md` names for the two builds, and `tasks sha256:` the SHA-256 of each id's `bench/tasks/<id>/task.json` then `reference.patch`, concatenated in the order `jq -r '[.[] | .[]] | .[]' bench/stage-9/classes.json` prints the ids, which `rehearse.sh` and the fence compute with the same command, three more lines ADR-206 T5's fence reads, `plans sha256:`, `smoke plans sha256:` and `instructions sha256:`, each the digest of the file of that name above, and one line `check <name>: exit <code>` per check, the code the command returned:

- `instructions frozen`: `cmp` of the committed file and `mrw-m instructions`;
- `build test`: `TestArmNDiffersFromMInOneRefusal`, with a `--- PASS` line and no `SKIP`;
- `oracle test`: `TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit`, the same way;
- `bench build`: `go build` of `cmd/bench` from the clean tree, its `bench_revision` `bench@` and the revision, with no `+modified`;
- `stub canaries`: the four canaries with their registered figures, `--claude` the stub, each passing `gate`;
- `stub plans`: the six runs of `plans.json` in order, every `--task` of each list, the stub as the CLI and its reported cost a fixed fraction of a cent, `gate` between steps; the stub makes one numbered write to a line not shown since its own write in each M and N follow-on and control task, and does nothing else;
- `count`: `count` and `report` with `plans.json` on the stub's output;
- `n event counted`: every stub N follow-on and control attempt `exposed` in the counts, every M one too, and no A one;
- `paid fences`: `test_the_paid_fences_pass_a_fixture_and_fail_each_broken_one`.
- and three negatives, each recorded as `check <name> negative: exit <code>` and required non-zero: `n event counted` with the stub making no exposed write, `stub plans` with one plan's task list cut to one id, and `build test` with the pinned tag hidden from the clone, which makes the test skip.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/stage-9/instructions.txt` | new | frozen before the rehearsal, consumed by ADR-206 T5 |
| `bench/stage-9/rehearse.sh`, `bench/stage-9/stub-agent` | new | the rehearsal and the stub CLI it drives |
| `docs/bench/stage-9/rehearsal.md` | new | the record, written by `rehearse.sh` |

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_the_rehearsal_fence_refuses_a_missing_or_wrong_digest` | `scripts/test_bench_stage9.py` | this task's fence, extracted from the task file and run with `REHEARSAL` at a generated record: exit 0 on a complete record; non-zero with each of the `m`, `n` and `tasks` digests missing, given twice or wrong, a file digest missing, a check missing, and a negative recorded as exit 0 | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rehearse.sh` and its record |
| 2 — something selects it | ADR-206 T5 depends on it and its fence reads the record |
| 3 — the caller can discover it | `docs/bench/stage-9/rehearsal.md` |
| 4 — it is used | ADR-206 T5 |

## Ordered Steps

1. [S1] The red state: the fence fails with no `rehearsal.md`. [proof: acceptance]
2. [S2] `instructions.txt` committed after `cmp` against `mrw-m instructions`. [proof: acceptance]
3. [S3] `rehearse.sh` and the stub, with the three negatives the rehearsal itself runs and records beside each check; the fence's digest test green. [proof: acceptance]
4. [S4] The rehearsal on the pilot Mac on the canaries' day. [proof: human: Zy reads rehearsal.md]

## Acceptance

```bash
set -o pipefail
I='bench/stage-9 bench/tasks scripts/bench_stage9.py scripts/bench_mechanism.py internal/bench cmd/bench'
p=$(python3 -m unittest -v scripts.test_bench_stage9 2>&1) && grep -qE '^test_the_rehearsal_fence_refuses_a_missing_or_wrong_digest .* ok$' <<<"$p" \
  && R=${REHEARSAL:-docs/bench/stage-9/rehearsal.md} && test -f "$R" && git ls-files --error-unmatch bench/stage-9/instructions.txt > /dev/null \
  && for k in 'instructions frozen' 'build test' 'oracle test' 'bench build' 'stub canaries' 'stub plans' 'count' 'n event counted' 'paid fences'; do grep -qxF "check $k: exit 0" "$R" || exit 1; done \
  && [ "$(grep -c '^check ' "$R")" -eq 12 ] && ! grep -q '^subset:' "$R" && grep -qE '^date: [0-9]{4}-[0-9]{2}-[0-9]{2}$' "$R" \
  && for k in 'n event counted' 'stub plans' 'build test'; do grep -qE "^check $k negative: exit [1-9][0-9]*$" "$R" || exit 1; done \
  && rev=$(sed -n 's/^revision: \([0-9a-f]\{40\}\)$/\1/p' "$R") && [ -n "$rev" ] && git merge-base --is-ancestor "$rev" HEAD \
  && git diff --quiet "$rev" HEAD -- $I && git diff --quiet HEAD -- $I && [ -z "$(git ls-files --others --exclude-standard -- $I)" ] \
  && for f in bench/stage-9/instructions.txt bench/stage-9/plans.json bench/stage-9/smoke-plans.json bench/stage-9/classes.json scripts/bench_stage9.py scripts/bench_mechanism.py bench/stage-9/rehearse.sh bench/stage-9/stub-agent; do [ "$(grep -c "^$f sha256: " "$R")" -eq 1 ] && grep -qxF "$f sha256: $(shasum -a 256 $f | cut -d' ' -f1)" "$R" || exit 1; done \
  && for a in 'plans:bench/stage-9/plans.json' 'smoke plans:bench/stage-9/smoke-plans.json' 'instructions:bench/stage-9/instructions.txt'; do k=${a%%:*}; f=${a#*:}; [ "$(grep -c "^$k sha256: " "$R")" -eq 1 ] && grep -qxF "$k sha256: $(shasum -a 256 $f | cut -d' ' -f1)" "$R" || exit 1; done \
  && for d in m n; do [ "$(grep -c "^$d sha256: " "$R")" -eq 1 ] && grep -qxF "$d sha256: $(sed -n "s/^$d sha256: //p" bench/stage-9/builds.md)" "$R" || exit 1; done \
  && ids=$(jq -r '[.[] | .[]] | .[]' bench/stage-9/classes.json) && [ -n "$ids" ] \
  && t=$(for id in $ids; do cat bench/tasks/$id/task.json bench/tasks/$id/reference.patch || exit 1; done | shasum -a 256 | cut -d' ' -f1) \
  && [ "$(grep -c '^tasks sha256: ' "$R")" -eq 1 ] && grep -qxF "tasks sha256: $t" "$R" \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state internal/writer cmd/mrw ':(exclude)*_test.go'
```

## Mutation Log
- 2026-11-21 · cd646822* · mutant killed · exit 1 · `docs/bench/stage-9/rehearsal.md` · The record says the deliberately shortened plan was accepted. T2 must refuse a successful negative control. · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · covers:the rehearsal's negatives failed

## Verification Log
- 2026-11-20 · 751edd9b* · exit 1 · `set -o pipefail …` · acceptance-sha256:99feeb10ac9a59ddcc568f89bac971c10554f29929fb2009378e9e64b6819c54 · ms:87234 · test-lock-sha256:cd8d3106982901184cd9985d751d2771072ed2ef4a16cc36af00b2a038f67748 · test-lock-file:docs/adr/locks/cd8d3106982901184cd9985d751d2771072ed2ef4a16cc36af00b2a038f67748.lock
  ```
  ```
- 2026-11-20 · 751edd9b* · exit 1 · `set -o pipefail …` · acceptance-sha256:edbfaffc09ce799683e0a9ed894176ec62e02818e4b449bdab30256cfc9cc39c · ms:53602
  ```
  ```
- 2026-11-21 · 2125b715* · exit 0 · `set -o pipefail …` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:60199
- 2026-11-21 · human-observed · S4: Zy read docs/bench/stage-9/rehearsal.md for revision 47564433 and confirmed the review on 2026-11-21. · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2
- 2026-11-21 · cd646822* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:0 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-21 · cd646822* · exit 0 · `set -o pipefail …` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:74172
- 2026-11-26 · 3657eb95* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:45 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-26 · 9eb2f968* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:35 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-26 · 837587b0* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:42 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-26 · ea298826* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:47 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-26 · e60d3102* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:36 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-26 · e2718628* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:45 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-26 · 0de0a0bf* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:38 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-26 · 62c9fe8a* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:34 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace
- 2026-11-26 · ce74e857* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:10e067b309fc772e30f8b2423a5a504aeb1072819201f9641211cc90d9d075d2 · ms:69 · test-lock-sha256:8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa · test-lock-file:docs/adr/locks/8971458df87729582319a76e376ed44206e06cd48358a1dec1dfed1166f112fa.lock · test-lock-kind:replace

## Invariants

- Nothing here spends money: the stub is the only CLI the rehearsal starts.

## Risks

- The rehearsal needs the tag's checkout and every task's repository on this machine; without them a check records a non-zero exit and the task is not done.
- A change to an input after the rehearsal turns this fence red; the rehearsal is run again, not edited.

## Stop Condition

The fence exits 0 and Zy has read `rehearsal.md`.

## Out of Scope

- Any paid attempt (permanent: boundary: ADR-206 T5)
