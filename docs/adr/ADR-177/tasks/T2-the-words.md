# Task ADR-177-T2: the instructions teach `mrw run`, its exit codes and its trust

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** a `mrw run` recipe in `mrw instructions` and `mrw skill`; `.claude/skills/mrw/SKILL.md` regenerated; README, AGENTS.md and `docs/agents.md` text
**Consumes:** `mrw run`, its receipt lines and exit codes (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guide test runs`, `the guide test passed`, `the guide and cmd packages pass`, `the binary builds`, `mrw skill prints the recipe`, `mrw instructions prints the recipe`, `the repository skill is the binary's`, `README names the subcommand`, `README gives its exit codes`, `AGENTS.md gives its exit codes`, `docs/agents.md names its trust`, `docs/agents.md teaches it for a long output`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 5, 6 and 9 reach the caller: `mrw instructions` (its recipes, which `mrw skill` prints too) teaches `mrw run 'CMD'` in place of `CMD | tail` and the read of the log by pattern; README (the subcommand, its receipt and log, "Exit status" at `README.md:565`, "What allowing mrw allows"), AGENTS.md ("Exit codes are the contract", §5) and `docs/agents.md` (§6 a long output, §7 trust) say what it does, which exit codes are its own, and that allowing mrw allows arbitrary shell through it. `mrw instructions --core` is unchanged: it holds eight rules in 300 words (ADR-141) and running a command is not one of them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/run177_test.go` | new | the red test: the recipes carry the `mrw run` recipe and its read form |
| `internal/guide/guide.go` | edit | a recipe in `Recipes()` (`:42`), which both `mrw instructions` and `Skill()` (`:153-160`) print |
| `.claude/skills/mrw/SKILL.md` | edit | regenerated from `mrw skill`, as `TestTheRepositorySkillIsTheBinarysSkill` (`cmd/mrw/skill168_test.go:301`) requires |
| `README.md` | edit | the subcommand, the receipt and log (and the state-home-in-the-tree case), "Exit status", "What allowing mrw allows" (`:157`) |
| `AGENTS.md` | edit | "Exit codes are the contract": `mrw run`'s codes are the command's, 124 and 130 its own, as `version --check-latest`'s are |
| `docs/agents.md` | edit | §6 "The read habits that cost" (a long output) and §7 "What allowing mrw allows" |
| `internal/guide/cost171_test.go` | edit, only if its byte budget for the guide must move | ADR-171 bounds what the guide costs; the change names the new figure |

## Ordered Steps

1. [S1] Write `TestTheGuideTeachesMrwRun` (`Recipes()` holds `mrw run '` and `- < '`, and `Skill("x")` holds the same recipe) and see it RED.
2. [S2] The recipe. Mutant: the recipe removed from `Recipes()`. [proof: mutation]
3. [S3] README, AGENTS.md, `docs/agents.md`, the regenerated skill. [proof: acceptance]
4. [S4] Recipe 12 is `mrw run` alone, so no later command's status replaces the run's, and the guide says a log is eligible by age or by count. Mutant: recipe 12's command gains a `; mrw read …` after the run. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ -count=1 -v -run '^(TestTheGuideTeachesMrwRun)$' 2>&1) \
  && grep -qE '^--- PASS: TestTheGuideTeachesMrwRun \(' <<<"$out" \
  && go test ./internal/guide/ ./cmd/mrw/ -count=1 >/dev/null \
  && go build -o bin/mrw ./cmd/mrw \
  && ./bin/mrw skill | grep -qF "mrw run '" \
  && ./bin/mrw instructions | grep -qF "mrw run '" \
  && grep -qF "mrw run '" .claude/skills/mrw/SKILL.md \
  && grep -qF '`mrw run' README.md \
  && sed -n '/^## Exit status/,/^## /p' README.md | grep -qE 'mrw run.*124|124.*mrw run' \
  && sed -n '/^## Exit codes are the contract/,/^## /p' AGENTS.md | grep -qE 'mrw run.*130|130.*mrw run' \
  && sed -n '/^## 7\./,/^## 8\./p' docs/agents.md | grep -qF '`mrw run' \
  && sed -n '/^## 6\./,/^## 7\./p' docs/agents.md | grep -qF "mrw run '" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheGuideTeachesMrwRun` | `internal/guide/run177_test.go` | the recipe is in `Recipes()` and in `Skill()` | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `mrw instructions` and `mrw skill` print `Recipes()` |
| 3 — the caller can discover it | the binary's own text, each surface checked on its own; the regenerated repository skill; README, AGENTS.md, `docs/agents.md` §6 and §7 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `internal/guide/guide.go` · the mrw run recipe is removed from Recipes() (S2) · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · covers:the guide test passed
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `README.md` · README Exit status loses 124 (S3) · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · covers:README gives its exit codes
- 2026-10-27 · e9186d50* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md exit codes lose 130 (S3) · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · covers:AGENTS.md gives its exit codes
- 2026-10-27 · 5e342885* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 12 chains a read after the run, so the read's status replaces the run's (S4) · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · covers:the guide test passed

## Invariants

- `mrw instructions --core` is unchanged (ADR-141).
- No recipe pipes an exit code.

## Risks

- The guide grows; ADR-171's cost test names the bound and this task moves it only with the figure.

## Stop Condition

The fence exits 0.

## Out of Scope

- The centralised `mrw` skill copy in memoryd `craft` (external: memoryd craft: refreshed by whoever merges, from `mrw skill`)

## Verification Log
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:49369
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:53427
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:53474
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:53256
- 2026-10-27 · e9186d50* · exit 1 · `set -o pipefail …` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:988 · test-lock-sha256:7ea0d62fbd2b99751993c5bd64e2adb4eeb864f4471a7dee3dedaabdd47364f2 · test-lock-file:docs/adr/locks/7ea0d62fbd2b99751993c5bd64e2adb4eeb864f4471a7dee3dedaabdd47364f2.lock
  ```
  ```
- 2026-10-27 · e9186d50* · exit 0 · `set -o pipefail …` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:50202
- 2026-10-27 · 5e342885* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:0 · test-lock-sha256:205221c025f083113c25cd403b32765dc40f494c80dae86738e7c9beb3d338ed · test-lock-file:docs/adr/locks/205221c025f083113c25cd403b32765dc40f494c80dae86738e7c9beb3d338ed.lock · test-lock-kind:replace
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:64505
- 2026-10-27 · 5e342885* · exit 0 · `set -o pipefail …` · acceptance-sha256:7a12e43efcd43d84ddc5d4e4eb3d1a5b345d20d644f36c7f715db24805af8847 · ms:57114
