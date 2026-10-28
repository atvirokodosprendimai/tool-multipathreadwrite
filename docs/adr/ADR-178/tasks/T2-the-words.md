# Task ADR-178-T2: the instructions teach `--json-path`, and that it licenses nothing

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** a `--json-path` recipe in `mrw instructions` and `mrw skill`; `.claude/skills/mrw/SKILL.md` regenerated; README, AGENTS.md and `docs/agents.md` text
**Consumes:** `--json-path` and its output lines (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guide test runs`, `the guide test passed`, `the guide and cmd packages pass`, `the binary builds`, `the skill text is captured`, `mrw skill carries the recipe`, `mrw skill carries its warning`, `mrw instructions carries the recipe`, `mrw read --help names the flag`, `the repository skill is the binary's`, `README names it`, `AGENTS.md names it`, `docs/agents.md teaches it`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

`Recipes()` — which `mrw instructions` and `mrw skill` both print — teaches `CMD --json | mrw read --json-path '$.a,$.b' -`, with the warning that a projection licenses nothing and the path address is what to read before editing; README, AGENTS.md (beside its JSON path paragraph and its measured quoting) and `docs/agents.md` §6 say the same. `mrw instructions --core` is unchanged: its eight rules stay within 300 words (ADR-141), and a projection is not one of them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/jsonpath178_test.go` | new | the red test |
| `internal/guide/guide.go` | edit | a recipe and its warning in `Recipes()` (`:42`) |
| `.claude/skills/mrw/SKILL.md` | edit | regenerated from `mrw skill` (`TestTheRepositorySkillIsTheBinarysSkill`, `cmd/mrw/skill168_test.go:301`) |
| `README.md` | edit | the flag beside the JSON path address, with the quoting forms |
| `AGENTS.md` | edit | §1 beside the JSON path address paragraph and its quoting table |
| `docs/agents.md` | edit | §6 "The read habits that cost" |
| `internal/guide/cost171_test.go` | edit, only if its bound must move | ADR-171 |

## Ordered Steps

1. [S1] Write `TestTheGuideTeachesJSONPathProjection` (`Recipes()` holds `--json-path` with `-` and the words `licenses nothing`; `Skill("x")` holds both) and see it RED.
2. [S2] The recipe and its warning. Mutant: the warning removed from the recipe. [proof: mutation]
3. [S3] README, AGENTS.md, `docs/agents.md`, the regenerated skill. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ -count=1 -v -run '^(TestTheGuideTeachesJSONPathProjection)$' 2>&1) \
  && grep -qE '^--- PASS: TestTheGuideTeachesJSONPathProjection \(' <<<"$out" \
  && go test ./internal/guide/ ./cmd/mrw/ -count=1 >/dev/null \
  && go build -o bin/mrw ./cmd/mrw \
  && s=$(./bin/mrw skill) && grep -qF -- '--json-path' <<<"$s" && grep -qF 'licenses nothing' <<<"$s" \
  && i=$(./bin/mrw instructions) && grep -qF -- '--json-path' <<<"$i" \
  && h=$(./bin/mrw read --help) && grep -qF -- '--json-path' <<<"$h" \
  && grep -qF -- '--json-path' .claude/skills/mrw/SKILL.md \
  && grep -qF -- '--json-path' README.md \
  && grep -qF -- '--json-path' AGENTS.md \
  && d=$(sed -n '/^## 6\./,/^## 7\./p' docs/agents.md) && grep -qF -- '--json-path' <<<"$d" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheGuideTeachesJSONPathProjection` | `internal/guide/jsonpath178_test.go` | the recipe and its warning, in `Recipes()` and in `Skill()` | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `mrw instructions` and `mrw skill` print `Recipes()` |
| 3 — the caller can discover it | the skill, the instructions and `read --help`, each checked on its own; the regenerated repository skill; the three documents |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the warning that a projection licenses nothing is removed from the recipe · acceptance-sha256:16ddc70b09dd7dea8e052ea7ee1537e2575d70af3c33722291c4b03329c9159a · covers:mrw skill carries its warning

## Invariants

- `mrw instructions --core` is unchanged (ADR-141).

## Risks

- None beyond ADR-178's.

## Stop Condition

The fence exits 0.

## Out of Scope

- The centralised `mrw` skill copy in memoryd `craft` (external: memoryd craft: refreshed by whoever merges, from `mrw skill`)

## Verification Log
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:061c7f2573f3d863c4403b95758b6a6e502b9e8ecbc1d653185e1db91018deb9 · ms:213 · test-lock-sha256:0380345350b258910c2531677d7082a1658630b8ade462b38da29c9ae860b9eb · test-lock-file:docs/adr/locks/0380345350b258910c2531677d7082a1658630b8ade462b38da29c9ae860b9eb.lock
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:061c7f2573f3d863c4403b95758b6a6e502b9e8ecbc1d653185e1db91018deb9 · ms:0 · test-lock-sha256:7992c71db3490e568e1213ad99eab4460b9d02a825766375f5269c51215b7793 · test-lock-file:docs/adr/locks/7992c71db3490e568e1213ad99eab4460b9d02a825766375f5269c51215b7793.lock · test-lock-kind:replace
- 2026-10-27 · 5952c1ce* · exit 141 · `set -o pipefail …` · acceptance-sha256:061c7f2573f3d863c4403b95758b6a6e502b9e8ecbc1d653185e1db91018deb9 · ms:77394
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:16ddc70b09dd7dea8e052ea7ee1537e2575d70af3c33722291c4b03329c9159a · ms:72971
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:16ddc70b09dd7dea8e052ea7ee1537e2575d70af3c33722291c4b03329c9159a · ms:55828
- 2026-10-27 · 4a80c019* · exit 0 · `set -o pipefail …` · acceptance-sha256:16ddc70b09dd7dea8e052ea7ee1537e2575d70af3c33722291c4b03329c9159a · ms:55086
