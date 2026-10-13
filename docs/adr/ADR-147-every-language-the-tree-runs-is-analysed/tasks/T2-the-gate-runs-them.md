# Task ADR-147-T2: the gate runs them and proves each can fail

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the new `static.sh` steps
**Consumes:** a tree every new analyser passes (T1)
**Data dependency:** needs `npm ci` run in `cmd/opencode/mrw-plugin`; `go run` fetches actionlint on a cold cache
**Proof map:** v1
**Rests-on:** `the gate is clean`, `static.sh runs every new analyser`, `each step refuses zero files`, `each analyser reports a planted defect`, `the engine is untouched`, `go.mod keeps one requirement`

## Goal

`scripts/static.sh` runs ruff, shellcheck, actionlint and knip, each over a file list it refuses when empty, and a `self-test` step that fails unless each reports a defect planted for it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/static.sh` | edit | four analyser steps, the self-test step, the pins and the missing-tool messages |
| `cmd/opencode/mrw-plugin/package.json`, `cmd/opencode/mrw-plugin/package-lock.json` | edit | devDependency `knip` 6.38.0, exact (`npm install --save-dev --save-exact knip@6.38.0`) |

## Ordered Steps

1. [S1] The fence fails at T1's head: `static.sh` has no `step ruff`. [proof: acceptance]
2. [S2] The four steps, written `step ruff …`, `step shellcheck …`, `step actionlint …`, `step knip …`, pinned, each printing `<name>: N file(s)` and failing at 0; each names its install when its tool is missing, knip's names `npm ci` when `node_modules` is absent. [proof: mutation]
3. [S3] `step self-test …`: the plants of the record's Decision 2 in a `mktemp -d` directory removed by a trap, each analyser required to exit non-zero with its defect named. [proof: mutation]
4. [S4] Time the whole gate once, warm, and write the number and the load into the record's Consequences. [proof: human: the executor pastes the `time` line into the PR]

## Acceptance

```bash
set -o pipefail
grep -q '^step ruff ' scripts/static.sh \
  && grep -q '^step shellcheck ' scripts/static.sh \
  && grep -q '^step actionlint ' scripts/static.sh \
  && grep -q '^step knip ' scripts/static.sh \
  && grep -q '^step self-test ' scripts/static.sh \
  && grep -q '"knip": "6.38.0"' cmd/opencode/mrw-plugin/package.json \
  && ./scripts/static.sh > /tmp/adr147-T2.out 2>&1 \
  && grep -q '^static analysis clean$' /tmp/adr147-T2.out \
  && for s in ruff shellcheck actionlint knip; do grep -qE "^$s: [1-9][0-9]* file" /tmp/adr147-T2.out || exit 1; done \
  && grep -q '^self-test: ruff, shellcheck, actionlint and knip each reported its planted defect$' /tmp/adr147-T2.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal cmd/mrw \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence runs the gate, and the gate runs its own self-test | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the steps in `scripts/static.sh` |
| 2 — something selects it | `.claude/hooks/static-after-commit.py`, `mrw check --then static`, and T3's CI step; fence-prose keeps the `^step` greps checked |
| 3 — the caller can discover it | T3's docs |
| 4 — it is used | every CI run once T3 lands |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- The six ADR-088 steps and fence-prose still run, in their order, before the new ones.
- The self-test leaves nothing behind: its directory is removed on every exit.

## Risks

- A planted defect that a future analyser version stops reporting turns the gate red. That is the self-test working; update the fixture or the pin.

## Out of Scope

- Installing the tools in CI (T3's job)

## Stop Condition

The fence exits 0.
