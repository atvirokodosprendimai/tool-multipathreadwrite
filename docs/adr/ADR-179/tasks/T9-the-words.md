# Task ADR-179-T9: the instructions teach the shorter reads and the terse receipt

**Depends-on:** T1, T2, T3, T4, T5, T6, T7, T8
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** recipes in `mrw instructions` and `mrw skill`; `.claude/skills/mrw/SKILL.md` regenerated; README, AGENTS.md and `docs/agents.md` text
**Consumes:** the hint line and `MRW_MAX_COLS` (T1), `--outline` (T2), `--if-served` (T3), `--fold` (T4), the once-per-invocation rule (T5), `--terse` (T6), `--max-matches` (T7), `--no-comments` (T8)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guide test runs`, `the guide test passed`, `the guide and cmd packages pass`, `the binary builds`, `the surfaces are captured`, `the skill names every flag`, `the instructions name every flag`, `README names every flag`, `AGENTS.md names every flag`, `the repository skill is the binary's`, `the loop stops at the first missing name`, `read --help names the read flags`, `the read-help loop stops at the first missing flag`, `write --help names --terse`, `docs/agents.md §3 names MRW_MAX_COLS`, `docs/agents.md §6 names the read flags`, `the §6 loop stops at the first missing flag`, `docs/agents.md says when --if-served is safe`, `the section is cut from the start of its heading`, `the next heading ends it`, `each grep reads the cut section only`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

`Recipes()` — printed by `mrw instructions` and by `mrw skill` — teaches `--outline` before a whole-file read, `--max-matches` beside `--count`, `--fold` and `--no-comments` for long reads, `MRW_MAX_COLS` for a setup, `--terse` on a write, and `--if-served PATH=SHA` with its condition (a session alone in its checkout, a host that runs `mrw seen --forget` on start, resume, clear and compaction, reads made under `--max-chars` below the host's limit). README, AGENTS.md and `docs/agents.md` (§3 the variable in a setup, §6 the habits, beside §8 `--if-served`) say the same. Each read flag's `CLI()` line was its own task's. `mrw instructions --core` is unchanged: its eight rules stay within 300 words (ADR-141), and none of these flags is one of them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/shorter179_test.go` | new | the red test |
| `internal/guide/guide.go` | edit | recipes in `Recipes()` (`:42`) |
| `.claude/skills/mrw/SKILL.md` | edit | regenerated from `mrw skill` (`TestTheRepositorySkillIsTheBinarysSkill`, `cmd/mrw/skill168_test.go:301`) |
| `README.md` | edit | the read flags table, the variable, the write flags |
| `AGENTS.md` | edit | §1 read flags; §4 the write receipt |
| `docs/agents.md` | edit | §3 `MRW_MAX_COLS` in a setup; §6 the habits; beside §8 `--if-served` |
| `internal/guide/cost171_test.go` | edit, only if its bound must move | ADR-171 |

## Ordered Steps

1. [S1] Write `TestTheGuideTeachesTheShorterReads` (`Recipes()` names each of `--outline`, `--max-matches`, `--fold`, `--no-comments`, `--if-served`, `--terse` and `MRW_MAX_COLS`; the `--if-served` recipe names `seen --forget` and `--max-chars`; `Skill("x")` holds every one) and see it RED.
2. [S2] The recipes. Mutant: the `--if-served` recipe loses its `seen --forget` clause. [proof: mutation]
3. [S3] README, AGENTS.md, `docs/agents.md`, the regenerated skill. **Pre-declared:** if T2 or T8 stopped at its measurement, its flag is removed from this task's test and from the fence's lists in the same change that records the stop, and nothing else in the fence changes. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ -count=1 -v -run '^(TestTheGuideTeachesTheShorterReads)$' 2>&1) \
  && grep -qE '^--- PASS: TestTheGuideTeachesTheShorterReads \(' <<<"$out" \
  && go test ./internal/guide/ ./cmd/mrw/ -count=1 >/dev/null \
  && go build -o bin/mrw ./cmd/mrw \
  && s=$(./bin/mrw skill) && i=$(./bin/mrw instructions) && rh=$(./bin/mrw read --help) && wh=$(./bin/mrw write --help) \
  && for f in --outline --max-matches --fold --no-comments --if-served --terse MRW_MAX_COLS; do grep -qF -- "$f" <<<"$s" && grep -qF -- "$f" <<<"$i" && grep -qF -- "$f" README.md && grep -qF -- "$f" AGENTS.md && grep -qF -- "$f" .claude/skills/mrw/SKILL.md || exit 1; done \
  && for f in --outline --max-matches --fold --no-comments --if-served; do grep -qF -- "$f" <<<"$rh" || exit 1; done \
  && grep -qF -- '--terse' <<<"$wh" \
  && sed -n '/^## 3\./,/^## 4\./p' docs/agents.md | grep -qF 'MRW_MAX_COLS' \
  && for f in --outline --max-matches --fold --no-comments; do sed -n '/^## 6\./,/^## 7\./p' docs/agents.md | grep -qF -- "$f" || exit 1; done \
  && grep -qE -- '--if-served.*seen --forget|seen --forget.*--if-served' docs/agents.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheGuideTeachesTheShorterReads` | `internal/guide/shorter179_test.go` | every flag and the variable are in `Recipes()` and `Skill()`; `--if-served` names its condition | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `mrw instructions` and `mrw skill` print `Recipes()` |
| 3 — the caller can discover it | the skill, the instructions, `read --help` and `write --help`, each checked on its own; the regenerated repository skill; README, AGENTS.md, `docs/agents.md` §3, §6 and beside §8 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/guide/guide.go` · the --if-served recipe loses its seen --forget clause (S2) · acceptance-sha256:be40923a2d08463dc523eb92578c5472aa39d88aace132bb7a08e0a0a00230aa · covers:the guide test passed

## Invariants

- `mrw instructions --core` is unchanged (ADR-141).
- `--if-served` is never taught without its condition.

## Risks

- The guide grows; ADR-171's cost test moves only with its figure.

## Stop Condition

The fence exits 0. A flag whose task stopped at its measurement is dropped as S3 pre-declares, and only so.

## Out of Scope

- The centralised `mrw` skill copy in memoryd `craft` (external: memoryd craft: refreshed by whoever merges, from `mrw skill`)

## Verification Log
- 2026-10-28 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:be40923a2d08463dc523eb92578c5472aa39d88aace132bb7a08e0a0a00230aa · ms:192 · test-lock-sha256:27e451656f1dbdd427fc746766c17f853badbea56fa4572d789f4d9d2b3c8dda · test-lock-file:docs/adr/locks/27e451656f1dbdd427fc746766c17f853badbea56fa4572d789f4d9d2b3c8dda.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:be40923a2d08463dc523eb92578c5472aa39d88aace132bb7a08e0a0a00230aa · ms:61576
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:be40923a2d08463dc523eb92578c5472aa39d88aace132bb7a08e0a0a00230aa · ms:57187
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:be40923a2d08463dc523eb92578c5472aa39d88aace132bb7a08e0a0a00230aa · ms:57147
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:be40923a2d08463dc523eb92578c5472aa39d88aace132bb7a08e0a0a00230aa · ms:65626
