# Task ADR-200-T2: the instructions stop inviting a stale number

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** rule 2 of `mrw instructions --core`, recipe 7 and a new recipe 21 in `mrw instructions` and `mrw skill`; the same in `AGENTS.md` and `README.md`
**Consumes:** ADR-200 Decision 4; ADR-161's recipe block; ADR-141's core and its 300-word limit
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `rule 2 says the numbers moved`, `recipe 7 points at recipe 21`, `recipe 21 addresses by content`

## Goal

ADR-200 Decision 4 says the instructions state what is true today. T1 added one sentence to the full text. The two places an agent meets first still granted the licence without the warning: rule 2 of the core ("except in a file mrw just wrote") and recipe 7 ("a file mrw just wrote needs no re-read"). That is the instruction that produces the chained numbered edit the record is about.

- Rule 2 says the file's numbers can move, inside the core's 300 words. It keeps the phrase ADR-141's test pins, "except in a file mrw just wrote", and so stays the short form: the exact licence is in recipe 7 and the full text.
- Recipe 7 says the file keeps a whole-file licence unless every hunk on it was a search_replace block, when the write adds the lines it wrote to what was licensed before; that its line numbers can move; and points at recipe 21. The full text's sentence on a written file carries the same exception.
- Recipe 21 is the second edit to a file just written: a hunk whose address is a pattern and which carries `anchor=`. It holds no line number.
- `AGENTS.md` rule 2 and its recipe list, and `README.md`'s per-line licence bullet, say the same.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/renumber200_test.go` | new | the red test |
| `internal/guide/guide.go` | edit | rule 2, recipe 7, recipe 21 |
| `internal/guide/guide_test.go` | edit | `TestInstructionsLeadWithTheRecipes` counts twenty-one recipes |
| `AGENTS.md`, `README.md` | edit | the same sentences |
| `.claude/skills/mrw/SKILL.md` | regenerate | it is the binary's own output (ADR-168) |

## Ordered Steps

1. [S1] Write `TestTheInstructionsDoNotInviteAStaleNumber` and see it RED on the three sentences. [proof: acceptance]
2. [S2] Change rule 2, recipe 7 and add recipe 21; update the recipe count in the existing test; regenerate the skill. Mutants, each killed: rule 2 loses its clause; recipe 7 says "needs no re-read" again; recipe 21's command addresses by number. [proof: mutation]
3. [S3] `AGENTS.md` and `README.md` say the same. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/guide ./cmd/mrw -run '^(TestTheInstructionsDoNotInviteAStaleNumber|TestInstructionsLeadWithTheRecipes|TestTheRepositorySkillIsTheBinarysSkill|TestRecipeTwentyOneAppliesToAFileJustWritten)$' > "${TMPDIR:-/tmp}/adr200t2-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr200t2-fence.out")" -eq 4 ] \
  && grep -q '^# 21\. a second edit to a file you just wrote' AGENTS.md \
  && grep -q 'needs no re-read, though its line numbers can move' AGENTS.md \
  && grep -q 'needs no re-read, though its line numbers can move' README.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheInstructionsDoNotInviteAStaleNumber` | `internal/guide/renumber200_test.go` | rule 2's clause and the core's word count; recipe 7's wording; recipe 21's label and a command that addresses by pattern with an anchor; both carried by the full text and the skill | none | S1, S2 |
| `TestInstructionsLeadWithTheRecipes` | `internal/guide/guide_test.go` | twenty-one recipes, each a label line and a command line | none | S2 |
| `TestTheRepositorySkillIsTheBinarysSkill` | `cmd/mrw/skill168_test.go` | the tracked skill is what the binary prints | none | S2 |
| `TestRecipeTwentyOneAppliesToAFileJustWritten` | `cmd/mrw/recipe200_test.go` | the command the instructions print, taken apart and its plan applied after a write that moved the line: the right line changes and nothing else | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test, and the three sentences |
| 2 — something selects it | `mrw instructions --core`, `mrw instructions` and `mrw skill` print them |
| 3 — the caller can discover it | they are the first things an agent reads |
| 4 — it is used | telemetry is refused by ADR-009; ADR-200's count, read on 2026-12-18, is taken while this text is what agents are told |

## Mutation Log
- 2026-11-18 · 77b5c30a* · mutant killed · exit 1 · `internal/guide/guide.go` · rule 2 grants the licence without the warning, so the core clause must go red · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · covers:rule 2 says the numbers moved
- 2026-11-18 · 77b5c30a* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 7 invites the chained edit again, so its wording check must go red · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · covers:recipe 7 points at recipe 21
- 2026-11-18 · 77b5c30a* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 21 addresses by number, so the pattern-and-anchor check must go red · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · covers:recipe 21 addresses by content
- 2026-11-18 · b854a8a9* · mutant killed · exit 1 · `internal/guide/guide.go` · rule 2 grants the licence without the warning, so the core clause must go red · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · covers:rule 2 says the numbers moved
- 2026-11-18 · b854a8a9* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 7 invites the chained edit again, so its wording check must go red · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · covers:recipe 7 points at recipe 21
- 2026-11-18 · b854a8a9* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 21 addresses by number, so the pattern-and-anchor check must go red · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · covers:recipe 21 addresses by content
- 2026-11-18 · b854a8a9* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 21 loses the backslash, so its pattern is not a regular expression and applying it must go red · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · covers:recipe 21 addresses by content
- 2026-11-19 · 29e154df* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 21 prints nothing into mrw write, so running it as printed must go red · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · covers:recipe 21 addresses by content
- 2026-11-19 · 29e154df* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 21 writes the line it found, so the fixed expected file must go red · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · covers:recipe 21 addresses by content
- 2026-11-19 · 29e154df* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 21 says a reread is the only way again, so its wording check must go red · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · covers:recipe 21 addresses by content

## Verification Log
- 2026-11-18 · 77b5c30a* · exit 1 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:589 · test-lock-sha256:6bb72f4ef7f4bbd999e85fc609068eac9c89a603c234ed3644c26e0de86ee03d · test-lock-file:docs/adr/locks/6bb72f4ef7f4bbd999e85fc609068eac9c89a603c234ed3644c26e0de86ee03d.lock
  ```
  ```
- 2026-11-18 · 77b5c30a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:472
- 2026-11-18 · 77b5c30a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:1949
- 2026-11-18 · 77b5c30a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:1922
- 2026-11-18 · 77b5c30a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:1789
- 2026-11-18 · 77b5c30a* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:0 · test-lock-sha256:175e08c629213d69e31c5be77ba332ead24b087f3fb105e0e74f80a4e76ebfc4 · test-lock-file:docs/adr/locks/175e08c629213d69e31c5be77ba332ead24b087f3fb105e0e74f80a4e76ebfc4.lock · test-lock-kind:replace
- 2026-11-18 · b854a8a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:899
- 2026-11-18 · b854a8a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:3331
- 2026-11-18 · b854a8a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:3373
- 2026-11-18 · b854a8a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:2565
- 2026-11-18 · b854a8a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:2581
- 2026-11-19 · b854a8a9* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:0 · test-lock-sha256:9ded7e8077e1508c5543e6ece12d91bba6ba75451452f6fe22542b90beca0e39 · test-lock-file:docs/adr/locks/9ded7e8077e1508c5543e6ece12d91bba6ba75451452f6fe22542b90beca0e39.lock · test-lock-kind:replace
- 2026-11-19 · 29e154df* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:1034
- 2026-11-19 · 29e154df* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:2649
- 2026-11-19 · 29e154df* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:2647
- 2026-11-19 · 29e154df* · exit 0 · `set -o pipefail …` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:2714
- 2026-11-19 · 29e154df* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:0 · test-lock-sha256:98fd7b17445fd9de0e9a88ebfd5a7e5ce3f487878ba664605526691d928c4b27 · test-lock-file:docs/adr/locks/98fd7b17445fd9de0e9a88ebfd5a7e5ce3f487878ba664605526691d928c4b27.lock · test-lock-kind:replace
- 2026-11-19 · 29e154df* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:0 · test-lock-sha256:98fd7b17445fd9de0e9a88ebfd5a7e5ce3f487878ba664605526691d928c4b27 · test-lock-file:docs/adr/locks/98fd7b17445fd9de0e9a88ebfd5a7e5ce3f487878ba664605526691d928c4b27.lock · test-lock-kind:replace
- 2026-11-19 · 6cf84b40* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:0 · test-lock-sha256:e33f9c267f141133a8b597625257676eff9192971187e4f0cb40e0000ba0162e · test-lock-file:docs/adr/locks/e33f9c267f141133a8b597625257676eff9192971187e4f0cb40e0000ba0162e.lock · test-lock-kind:replace
- 2026-11-20 · af5b4fa8* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:0 · test-lock-sha256:1bf5c3cb62a4e0ca0f4b9da64b6dac96498f8fa9d055c73bcf861dc8d2fb9499 · test-lock-file:docs/adr/locks/1bf5c3cb62a4e0ca0f4b9da64b6dac96498f8fa9d055c73bcf861dc8d2fb9499.lock · test-lock-kind:replace
- 2026-11-20 · 07234190* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:2 · test-lock-sha256:e7f38c57b18fc979e2989ab7df203aad9abf919e521a4b241e0d24b3e2b235f8 · test-lock-file:docs/adr/locks/e7f38c57b18fc979e2989ab7df203aad9abf919e521a4b241e0d24b3e2b235f8.lock · test-lock-kind:replace
- 2026-11-21 · 95b3955f* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:6 · test-lock-sha256:2d0c9edacbdb93106c1d16bc87f756967f528cdfc5bb117f5840f2a8c1c80de5 · test-lock-file:docs/adr/locks/2d0c9edacbdb93106c1d16bc87f756967f528cdfc5bb117f5840f2a8c1c80de5.lock · test-lock-kind:replace
- 2026-11-21 · 891ae7d5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:af0502a3551784b2a45332d33c8f8f77129edd5757540a8e69565e927029e652 · ms:0 · test-lock-sha256:c491f0eceb436f42f4397a8b7f0da346e3b60a356ffe05cad331ba8903e861f6 · test-lock-file:docs/adr/locks/c491f0eceb436f42f4397a8b7f0da346e3b60a356ffe05cad331ba8903e861f6.lock · test-lock-kind:replace

## Invariants

- The core stays under 300 words and keeps eight rules.
- No recipe's command changes but the new one; no flag, receipt or exit code changes.

## Risks

- Recipe 21's command was run by hand against a built binary (a file read, an insert above, then the recipe's hunk: applied, the right line changed). No test applies it.
- The skill file carries a development stamp until the next release's Status commit regenerates it from the tagged binary.
- The count ADR-200 pre-registered is read under this text from its release on; the bar's entry does not say which instructions the callers had, and the reading must.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- The refusal (deferred: docs/adr/BACKLOG.md)
