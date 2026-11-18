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

- Rule 2 says the file's numbers moved, inside the core's 300 words.
- Recipe 7 says the file is wholly licensed and its line numbers have moved, and points at recipe 21.
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
go test -count=1 -v ./internal/guide ./cmd/mrw -run '^(TestTheInstructionsDoNotInviteAStaleNumber|TestInstructionsLeadWithTheRecipes|TestTheRepositorySkillIsTheBinarysSkill)$' > "${TMPDIR:-/tmp}/adr200t2-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr200t2-fence.out")" -eq 3 ] \
  && grep -q '^# 21\. a second edit to a file you just wrote' AGENTS.md \
  && grep -q 'needs no re-read, though its line numbers have moved' AGENTS.md \
  && grep -q 'needs no re-read, though its line numbers have moved' README.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheInstructionsDoNotInviteAStaleNumber` | `internal/guide/renumber200_test.go` | rule 2's clause and the core's word count; recipe 7's wording; recipe 21's label and a command that addresses by pattern with an anchor; both carried by the full text and the skill | none | S1, S2 |
| `TestInstructionsLeadWithTheRecipes` | `internal/guide/guide_test.go` | twenty-one recipes, each a label line and a command line | none | S2 |
| `TestTheRepositorySkillIsTheBinarysSkill` | `cmd/mrw/skill168_test.go` | the tracked skill is what the binary prints | none | S2 |

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

## Verification Log
- 2026-11-18 · 77b5c30a* · exit 1 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:589 · test-lock-sha256:6bb72f4ef7f4bbd999e85fc609068eac9c89a603c234ed3644c26e0de86ee03d · test-lock-file:docs/adr/locks/6bb72f4ef7f4bbd999e85fc609068eac9c89a603c234ed3644c26e0de86ee03d.lock
  ```
  ```
- 2026-11-18 · 77b5c30a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:472
- 2026-11-18 · 77b5c30a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:1949
- 2026-11-18 · 77b5c30a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:1922
- 2026-11-18 · 77b5c30a* · exit 0 · `set -o pipefail …` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:1789
- 2026-11-18 · 77b5c30a* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d075696f2056b1a9104fffcfdfb9e546d2bdd12e92aaf836d8fa1b3a94155851 · ms:0 · test-lock-sha256:175e08c629213d69e31c5be77ba332ead24b087f3fb105e0e74f80a4e76ebfc4 · test-lock-file:docs/adr/locks/175e08c629213d69e31c5be77ba332ead24b087f3fb105e0e74f80a4e76ebfc4.lock · test-lock-kind:replace

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
