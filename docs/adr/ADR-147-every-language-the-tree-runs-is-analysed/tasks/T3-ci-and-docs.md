# Task ADR-147-T3: CI installs them and the docs name them

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** none
**Consumes:** the new `static.sh` steps (T2)
**Data dependency:** hermetic for the fence; the CI run itself needs GitHub Actions
**Proof map:** v1
**Rests-on:** `CI installs every analyser in the test job`, `the tarball is checked`, `the workflow lints clean`, `the docs name the gate's tools`, `the engine is untouched`, `go.mod keeps one requirement`

## Goal

CI's `test` job installs ruff, shellcheck (checksummed), node and the plugin's dependencies before `scripts/static.sh`, and the docs that list the gate name the new tools.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `.github/workflows/ci.yml` | edit | the installs in the `test` job |
| `AGENTS.md`, `CONTRIBUTING.md`, `.claude/rules/static-analysis.md` | edit | the gate's tools and what to install |

## Ordered Steps

1. [S1] The fence fails at T2's head: `ci.yml` installs no ruff. [proof: acceptance]
2. [S2] The CI installs in the `test` job, pinned to `static.sh`'s versions, the tarball checked with `sha256sum -c`. [proof: mutation]
3. [S3] The docs. [proof: mutation]
4. [S4] The pushed branch's CI `test` job passes. [proof: human: the reviewer reads the green `test` job on the PR, which `gh pr checks` shows]

## Acceptance

```bash
set -o pipefail
grep -q 'pipx install ruff==0.15.6' .github/workflows/ci.yml \
  && grep -q '8c3be12b05d5c177a04c29e3c78ce89ac86f1595681cab149b65b97c4e227198' .github/workflows/ci.yml \
  && grep -q 'name: Install the plugin.s dependencies for knip' .github/workflows/ci.yml \
  && awk '/^  test:$/,/^  race:$/' .github/workflows/ci.yml > /tmp/adr147-T3.job \
  && for s in 'actions/setup-node' 'pipx install ruff==0.15.6' 'sha256sum -c' 'npm ci' './scripts/static.sh'; do grep -qF "$s" /tmp/adr147-T3.job || exit 1; done \
  && SHELLCHECK_OPTS='-S warning' go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/ci.yml \
  && grep -q 'ruff, shellcheck, actionlint, knip (ADR-147)' AGENTS.md \
  && grep -q 'ruff, shellcheck, actionlint and knip' CONTRIBUTING.md \
  && grep -q 'ruff 0.15.6' .claude/rules/static-analysis.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal cmd/mrw \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence greps the installs inside the `test` job and lints the workflow; CI's run is the rest | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the CI steps |
| 2 — something selects it | the `test` job, which `build` needs |
| 3 — the caller can discover it | AGENTS.md §Build and check, CONTRIBUTING.md, `.claude/rules/static-analysis.md` |
| 4 — it is used | every push and pull request |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- The `test` job still runs `scripts/static.sh`, then `go test ./...`, then `./scripts/contract.sh`.
- Every line ADR-088 T2, ADR-088 T3 and ADR-089 T1 grep in these files is kept.

## Risks

- The runner's preinstalled shellcheck is found before the pinned one. Mitigation: the tarball's directory goes first on `GITHUB_PATH`, and `static.sh` notes a differing version in the log.

## Out of Scope

- The Windows job (permanent: boundary: CI analyses on Linux, as ADR-088 records; the after-commit hook runs the gate wherever the developer is)

## Stop Condition

The fence exits 0 and the PR's `test` job is green.
