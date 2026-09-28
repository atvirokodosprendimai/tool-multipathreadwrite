# Task ADR-089-T1: the plugin works, and CI proves it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `cmd/opencode/mrw-plugin/src/index.ts`; `test/smoke.test.mjs`; the CI job; the README section
**Consumes:** nothing
**Data dependency:** hermetic, except `npm ci` downloads the locked packages
**Proof map:** v1
**Rests-on:** `a cut page licenses only whole runs`, `a write reaches mrw`, `the binary resolves on this platform`, `CI runs the smoke test`

## Goal

The plugin as merged from `feat/opencode-adapter` could not spawn outside Windows and never delivered a plan to `mrw write`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/opencode/mrw-plugin/src/index.ts` | edit | binary resolution, stdin plan, `--root` order, `iter` verbs, exit codes, abort |
| `cmd/opencode/mrw-plugin/test/smoke.test.mjs` | new | drives the built tools against the built binary |
| `cmd/opencode/mrw-plugin/package.json`, `tsconfig.json` | edit | `npm test`; no unused locals |
| `.github/workflows/ci.yml`, `CONTRIBUTING.md`, `README.md` | edit | the job; the gate; what is supported |

## Ordered Steps

1. [S1] Write `test/smoke.test.mjs`; the fence fails on the merged branch, where a write never reaches mrw and nothing spawns off Windows. [proof: mutation]
2. [S2] Fix the plugin. [proof: mutation]
3. [S3] CI job, CONTRIBUTING gate, README section. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go build -o bin/mrw ./cmd/mrw \
  && (cd cmd/opencode/mrw-plugin && npm ci --no-audit --no-fund && npm run build && npm test) > /tmp/adr089-T1.out 2>&1 \
  && grep -qE '^# pass [1-9][0-9]*$' /tmp/adr089-T1.out \
  && grep -qE '^# fail 0$' /tmp/adr089-T1.out \
  && grep -qE '^ +run: npm test$' .github/workflows/ci.yml \
  && grep -q '^## opencode' README.md \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence runs `cmd/opencode/mrw-plugin/test/smoke.test.mjs`, which is JavaScript | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `cmd/opencode/mrw-plugin/dist/index.js` after `npm run build` |
| 2 — something selects it | `opencode.json`'s `plugin` entry |
| 3 — the caller can discover it | the README's opencode section |
| 4 — it is used | opencode sessions that load it; ADR-009 refuses telemetry |

## Verification Log
(empty until execute)
- 2026-09-27 · a2f9b22* · exit 2 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:36779 · test-lock-sha256:d0a88974807379b1a850b3ee812a98f67f902bffe32b9e36b9a17cac84f8be11 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMw
  ```
  ```
- 2026-09-27 · a2f9b22* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:5063
- 2026-09-27 · a2f9b22* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:7444
- 2026-09-27 · a2f9b22* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:13292
- 2026-09-27 · a2f9b22* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:6458
- 2026-09-27 · a2f9b22* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:3430
- 2026-09-27 · a2f9b22* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:3636
- 2026-09-28 · e4ff2d1* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:6079
- 2026-09-28 · e4ff2d1* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:3805
- 2026-09-28 · e4ff2d1* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:2892
- 2026-09-28 · e4ff2d1* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:3006
- 2026-09-28 · e4ff2d1* · exit 0 · `set -o pipefail …` · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · ms:2838

## Mutation Log
(empty until execute)
- 2026-09-27 · a2f9b22* · mutant survived · exit 0 · `cmd/opencode/mrw-plugin/src/index.ts` · the write tool stops handing its plan to mrw · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:a write reaches mrw
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-27 · a2f9b22* · mutant killed · exit 1 · `cmd/opencode/mrw-plugin/src/index.ts` · the binary is looked up as mrw.exe on every platform, as the branch had it · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:the binary resolves on this platform
- 2026-09-27 · a2f9b22* · mutant killed · exit 1 · `cmd/opencode/mrw-plugin/src/index.ts` · root goes after the subcommand, as the branch had it · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:root precedes the verb
- 2026-09-27 · a2f9b22* · mutant killed · exit 1 · `.github/workflows/ci.yml` · CI stops running the smoke test · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:CI runs the smoke test
- 2026-09-27 · a2f9b22* · mutant killed · exit 1 · `cmd/opencode/mrw-plugin/src/index.ts` · the write tool stops feeding its plan to mrw on stdin, as the branch had it · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:a write reaches mrw
- 2026-09-28 · e4ff2d1* · mutant killed · exit 1 · `cmd/opencode/mrw-plugin/src/index.ts` · the plugin acknowledges every run it served, cut or not · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:a cut page licenses only whole runs
- 2026-09-28 · e4ff2d1* · mutant killed · exit 1 · `cmd/opencode/mrw-plugin/src/index.ts` · the write tool stops sending its plan · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:a write reaches mrw
- 2026-09-28 · e4ff2d1* · mutant killed · exit 1 · `cmd/opencode/mrw-plugin/src/index.ts` · the binary is looked up as mrw.exe on every platform, as the branch had it · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:the binary resolves on this platform
- 2026-09-28 · e4ff2d1* · mutant killed · exit 1 · `.github/workflows/ci.yml` · CI stops running the smoke test · acceptance-sha256:96b3e5293bb3469f9c43c274655769f46065284c30b5dd6bcdf7debb175f7c35 · covers:CI runs the smoke test

## Invariants

- No Go package changes.

## Risks

- None beyond the record's.

## Out of Scope

- The record's Out of Scope (permanent: boundary: see the record)

## Stop Condition

The fence exits 0 and the CI job is green.
