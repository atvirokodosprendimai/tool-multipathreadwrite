# Task ADR-185-T1: rule 6 says `--stat` lists the matching files

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the new `guide.Core()` text; `TestTheCoreSaysStatListsTheMatchingFiles`; contract §321; the regenerated `.claude/skills/mrw/SKILL.md`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `rule 6 names which files --stat lists`

## Goal

Decisions 1-3 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/core185_test.go` | new | the red test; a new file, so the older tests and their locks do not move |
| `internal/guide/guide.go` | edit | `Core()`: rules 1 and 6 |
| `scripts/contract.sh` | edit | §321 |
| `.claude/skills/mrw/SKILL.md` | regenerated | `mrw skill --dir .claude/skills` |

## Ordered Steps

1. [S1] Write `TestTheCoreSaysStatListsTheMatchingFiles`: rule 6 of `Core()` contains `--stat lists the matching files`; rule 1 still contains `mrw write --grep P --sub R`; the core is under 300 words. See it RED on the current code.
2. [S2] Reword rules 1 and 6 as Decisions 1 and 2. Mutant: rule 6 keeps `--stat lists files` — the test goes red. [proof: mutation]
3. [S3] Regenerate the skill. [proof: acceptance]
4. [S4] Contract §321: the built binary's core names `--stat lists the matching files`; `mrw read --grep P --stat .` on a two-file fixture lists the one matching file and exits 0; the pair, a bare `mrw read --stat`, exits 2 naming that a read needs a path. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ ./cmd/mrw/ -count=1 -v -run '^(TestTheCoreSaysStatListsTheMatchingFiles|TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf|TestTheCoreIsEightShortRulesTheFullContractStillCarries|TestTheCoreNamesTheSubWrite|TestTheCoreNamesOnlyFlagsTheBinaryHas|TestTheRepositorySkillIsTheBinarysSkill)$' 2>&1) \
  && grep -qE '^--- PASS: TestTheCoreSaysStatListsTheMatchingFiles \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreIsEightShortRulesTheFullContractStillCarries \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreNamesTheSubWrite \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreNamesOnlyFlagsTheBinaryHas \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheRepositorySkillIsTheBinarysSkill \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 321\. ' scripts/contract.sh \
  && grep -qF 'PASS  the core says --stat lists the matching files' <<<"$c" \
  && grep -qF 'PASS  and a grep with --stat lists the one matching file' <<<"$c" \
  && grep -qF 'PASS  the pair: a bare --stat is refused, exit 2' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/read internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCoreSaysStatListsTheMatchingFiles` | `internal/guide/core185_test.go` | rule 6's `--stat` clause names the matching files; rule 1 keeps the sub write; under 300 words | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `mrw instructions --core` and `mrw skill` print `Core()` |
| 3 — the caller can discover it | `docs/agents.md` tells a host to run `mrw instructions --core` at session start |
| 4 — it is used | no telemetry (ADR-009); the evidence is the five bare `--stat` calls in runs 3-5's core cells |

## Mutation Log
- 2026-11-01 · 891702e8* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: rule 6 keeps the short wording · acceptance-sha256:2659fb90ad4aa50d960166428ef99b2315da5e5f8b18c601e6822e88346b56d1 · covers:rule 6 names which files --stat lists

## Invariants

- Eight rules, under 300 words, only flags the binary has; every held clause.
- `mrw instructions` without the flag is unchanged.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0; or the wording cannot fit under 300 words: stop and bring it to Zy.

## Out of Scope

- The full text's wording (permanent: boundary: ADR-141 Decision 3)

## Verification Log
- 2026-11-01 · 891702e8* · exit 1 · `set -o pipefail …` · acceptance-sha256:2659fb90ad4aa50d960166428ef99b2315da5e5f8b18c601e6822e88346b56d1 · ms:416 · test-lock-sha256:2e62263a0efbefeeaeb63382fb7708ebc4328dec7266ab2e5540aa7ac3f2cc50 · test-lock-file:docs/adr/locks/2e62263a0efbefeeaeb63382fb7708ebc4328dec7266ab2e5540aa7ac3f2cc50.lock
  ```
  ```
- 2026-11-01 · 891702e8* · exit 0 · `set -o pipefail …` · acceptance-sha256:2659fb90ad4aa50d960166428ef99b2315da5e5f8b18c601e6822e88346b56d1 · ms:81843
- 2026-11-01 · 891702e8* · exit 0 · `set -o pipefail …` · acceptance-sha256:2659fb90ad4aa50d960166428ef99b2315da5e5f8b18c601e6822e88346b56d1 · ms:82482
