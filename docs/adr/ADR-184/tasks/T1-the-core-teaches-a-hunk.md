# Task ADR-184-T1: the core's rule 4 shows the hunk header

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the new `guide.Core()` text; `TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf`; contract §320; the regenerated `.claude/skills/mrw/SKILL.md`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `rule 4 shows the three headers`

## Goal

Decisions 1-4 of the record: rule 4 shows `@@ PATH N-M replace anchor="TEXT"`, `@@ PATH N insert-after` and `@@ PATH N-M delete` and says the anchor is copied from line N; rules 3, 5 and 6 are shorter with every clause ADR-141 holds; the core is under 300 words; the checked-in skill equals `mrw skill`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/core184_test.go` | new | the red test; a new file, so ADR-141's and ADR-171's test files and their locks do not move |
| `internal/guide/guide.go` | edit | `Core()` (`:146-157`): rules 3, 4, 5 and 6 |
| `scripts/contract.sh` | edit | §320: the built binary's core, applied as printed, and the pair |
| `.claude/skills/mrw/SKILL.md` | regenerated | `mrw skill --dir .claude/skills`, held by `TestTheRepositorySkillIsTheBinarysSkill` |

## Ordered Steps

1. [S1] Write `TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf`: rule 4 of `Core()` contains `@@ PATH N-M replace anchor="TEXT"`, `@@ PATH N insert-after`, `@@ PATH N-M delete` and `copied from line N`; `CLI()` still contains `Ops: replace, insert-after, insert-before, delete` and `A multi-line replace requires anchor=`; the core is under 300 words. See it RED on the current code.
2. [S2] Rewrite rule 4 as Decision 1 and shorten rules 3, 5 and 6 as Decision 2. Mutant: rule 4 without the insert-after form — the test goes red. [proof: mutation]
3. [S3] Mutant: rule 4 says the anchor is copied "from the line your read printed" (the old wording) — the test goes red. [proof: mutation]
4. [S4] Regenerate the skill with the built binary: `mrw skill --dir .claude/skills`. [proof: acceptance]
5. [S5] Contract §320: take rule 4 from the built binary's `instructions --core`; check it names the three headers; build a plan from exactly those three forms with PATH, N, M and TEXT filled in for a five-line file, apply it, and compare the file; the pair, the same replace with an anchor from line N+1, is refused, exit 1, and the file is unchanged. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ ./cmd/mrw/ -count=1 -v -run '^(TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf|TestTheCoreIsEightShortRulesTheFullContractStillCarries|TestTheCoreTeachesTheBodyCountAsOptional|TestTheCoreSaysWhenToReRead|TestTheFullGuideSaysShaDoesNotStandInForAnAnchor|TestTheSkillBodyIsTheCoreAndTheRecipes|TestInstructionsCoreFlagPrintsTheCore|TestTheCoreNamesOnlyFlagsTheBinaryHas|TestTheCoreNamesTheSubWrite|TestTheRepositorySkillIsTheBinarysSkill)$' 2>&1) \
  && grep -qE '^--- PASS: TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreIsEightShortRulesTheFullContractStillCarries \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreTeachesTheBodyCountAsOptional \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreSaysWhenToReRead \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheFullGuideSaysShaDoesNotStandInForAnAnchor \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheSkillBodyIsTheCoreAndTheRecipes \(' <<<"$out" \
  && grep -qE '^--- PASS: TestInstructionsCoreFlagPrintsTheCore \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreNamesOnlyFlagsTheBinaryHas \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheCoreNamesTheSubWrite \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheRepositorySkillIsTheBinarysSkill \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 320\. ' scripts/contract.sh \
  && grep -qF 'PASS  the core names the three hunk headers' <<<"$c" \
  && grep -qF 'PASS  a plan of the three forms as the core prints them applies' <<<"$c" \
  && grep -qF 'PASS  the pair: an anchor not from line N is refused and writes nothing' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/read internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf` | `internal/guide/core184_test.go` | rule 4 shows the three headers and the anchor's source; the full text still carries the ops and the anchor rule; under 300 words | none | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `mrw instructions --core` and `mrw skill` print `Core()` (`TestInstructionsCoreFlagPrintsTheCore`, `TestTheSkillBodyIsTheCoreAndTheRecipes`) |
| 3 — the caller can discover it | `docs/agents.md:51` tells a host to run `mrw instructions --core` at session start |
| 4 — it is used | no telemetry (ADR-009); the evidence for building it is CORE-RESULTS.md run 2, where 8 of 10 core cells fetched the full guide |

## Mutation Log
- 2026-10-31 · 2c5a2abd* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: rule 4 without the insert-after form · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · covers:rule 4 shows the three headers
- 2026-10-31 · 2c5a2abd* · mutant killed · exit 1 · `internal/guide/guide.go` · S3: rule 4 keeps the old anchor wording, not line N · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · covers:rule 4 shows the three headers
- 2026-10-31 · 2c5a2abd* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: rule 4 prints a mangled op that a substring match would pass (Codex challenge) · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · covers:rule 4 shows the three headers

## Invariants

- Eight rules, under 300 words, only flags the binary has; every clause ADR-141's test holds.
- `mrw instructions` without the flag is unchanged.
- No engine package, flag or exit code changes.

## Risks

- §320 proves the forms as printed apply; it does not prove an agent reads them so. That is the deferred re-run's question.

## Stop Condition

The fence exits 0; or the new rule cannot fit under 300 words without dropping a held clause: stop and bring it to Zy.

## Out of Scope

- The full text's wording (permanent: boundary: ADR-141 Decision 3; `mrw instructions` is unchanged)

## Verification Log
- 2026-10-31 · 2c5a2abd* · exit 0 · `set -o pipefail …` · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · ms:82126
- 2026-10-31 · 2c5a2abd* · exit 0 · `set -o pipefail …` · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · ms:84075
- 2026-10-31 · 2c5a2abd* · exit 0 · `set -o pipefail …` · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · ms:83406
- 2026-10-31 · 2c5a2abd* · exit 0 · `set -o pipefail …` · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · ms:81520
- 2026-10-31 · f5e93f5f* · exit 1 · `set -o pipefail …` · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · ms:1199 · test-lock-sha256:0f939a8d39060806c2b0f1fc4e2e7c402a75cf880e971f39b64d13341c54a8ea · test-lock-file:docs/adr/locks/0f939a8d39060806c2b0f1fc4e2e7c402a75cf880e971f39b64d13341c54a8ea.lock
  ```
  ```
- 2026-10-31 · f5e93f5f* · exit 0 · `set -o pipefail …` · acceptance-sha256:14faf47d54df08768a6574f3206ce69dee09204b7f21cae993a9be83522eb35f · ms:84138
