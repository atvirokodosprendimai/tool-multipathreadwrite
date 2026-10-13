# ADR-147: Every language the tree runs is analysed, not only Go

**Status:** Accepted
**Accepted:** 2026-10-13 by Zy — "accept. here our goal is not to silence issues, but fix em."
**Date:** 2026-10-13
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-088, ADR-089, ADR-092, ADR-131, docs/adr/BACKLOG.md
**Governs:** `scripts/static.sh`, `.claude/rules/static-analysis.md`, `.github/workflows/ci.yml`
**Enforced-by:** None — the gate is a shell script, not Go: `scripts/static.sh`'s own `self-test` step plants one defect per analyser this record adds and fails unless each analyser reports it, every step refuses to pass over zero files, and CI and the after-commit hook (ADR-088) run the script; fence-prose (ADR-088 T2) re-checks T2's and T3's greps over `scripts/static.sh`, `ci.yml` and the docs on every run, so deleting a step turns the gate red
**Invalidates:** none — checked: ADR-088 T2's fence greps six step names, `static analysis clean`, the `run: ./scripts/static.sh` line in `ci.yml` and `scripts/static.sh` in CONTRIBUTING.md; ADR-088 T3's greps `.claude/rules/static-analysis.md`; ADR-089 T1's greps `ci.yml` and runs `npm ci`, `npm run build`, `npm test` — every line they match is kept, and T3 edits those files only by adding
**Served-path change:** None — this ADR changes only tooling; no file under `internal/` or `cmd/mrw/` changes

## Context

**What prompted it.** Zy asked on 2026-10-13 whether the repository runs a full set of linters,
static analysers, formatters and dead-code scanners. For Go it does (ADR-088). For everything else
it runs nothing beyond `tsc`.

**The class**: every tracked source file in a language the tree executes that no analyser in
`scripts/static.sh` reads. Enumerated 2026-10-13 at `f94694d` with
`git ls-files | sed -n 's/.*\.\([A-Za-z0-9]*\)$/\1/p' | sort | uniq -c`:
- Python, 9 files (`git ls-files '*.py'`): the two Claude Code hooks, `scripts/` and three
  `docs/break/*/stress.py` harnesses. No analyser.
- Shell, 5 files (`git ls-files '*.sh'`): `scripts/contract.sh` (9,126 lines) and four more. No
  analyser.
- GitHub Actions workflows, 2 files (`.github/workflows/*.yml`). No analyser.
- TypeScript, 1 file (`cmd/opencode/mrw-plugin/src/index.ts`, ADR-089): `tsc` strict with
  `noUnusedLocals` and `noUnusedParameters`; nothing checks unused files, exports or dependencies.
  Its one `.mjs` test is outside `tsc` (`tsconfig.json` includes `src/**/*` only).
- Code that is not a file of its own: Python in shell — 108 `<<'PY'` heredocs and 52 `python3 -c`
  sites, all in `scripts/contract.sh` and `scripts/break-campaign.sh` (`mrw read --grep "<<.?PY"` and
  `--grep 'python3 -c'` over `scripts/`); the task Acceptance fences under `docs/adr/`; the inline
  commands in `.claude/settings.json` and `.quality-harness.json`. Out of Scope says what becomes of each.

What each candidate finds, measured 2026-10-13 at `f94694d` on the tree above:
- `ruff check --select F,E9` (0.15.6): 2 — `scripts/chaos.py:39` imports `hashlib` unused (F401),
  `scripts/chaos.py:802` assigns `wr` unused (F841).
- `shellcheck -S warning` (0.11.0): 9 — `scripts/contract.sh:75` a `cd` with no failure branch
  (SC2164), `:870` `SHA` assigned and never read (SC2034), six `ls | grep` (SC2010: 7421, 8109, 8633
  three times, 8975), `scripts/measure.sh:229` `search_cmd=grep` read as a command (SC2209).
- `actionlint` (v1.7.12, with `SHELLCHECK_OPTS='-S warning'`): 1 — `.github/workflows/timing-123.yml`
  line 60 (ADR-131's timing workflow), a loop counter `i` never read (SC2034).
- `knip` (6.38.0) in `cmd/opencode/mrw-plugin` with its `node_modules`: 0. Without `node_modules` it
  reports `tsc` as an unlisted binary, a false finding, so it needs `npm ci` first.

Cost, measured 2026-10-13: `scripts/static.sh` as it stands runs in 6.3 s warm at load 2.5 on 10
cores; at that load shellcheck over the five scripts takes 6.9 s, and at load 23.6 ruff took 0.1 s,
actionlint 0.6 s and knip 2.3 s. T2 measures the whole new gate.

## Existing Primitives Audit

- `scripts/static.sh` (ADR-088) is the gate: `step` runs each analyser and the script exits 1 when any
  found anything. This record adds steps to it; it adds no second script.
- The pinning pattern stays ADR-088's: a Go tool runs as `go run module@version` (actionlint is Go), a
  binary that cannot be run that way is required on PATH and named with its install when missing, and
  a version different from the pin prints a note (golangci-lint's `case` at `scripts/static.sh:48-53`).
  A Node tool is pinned the way Node pins: an exact devDependency in `package-lock.json`.
- `scripts/fence-prose.py --self-test` (ADR-088 T2) is the precedent for a gate proving it can fail
  inside the gate itself.
- CI's `opencode` job (ADR-089) already sets up node 24 and runs `npm ci`; the `test` job, which runs
  `static.sh`, has Go only.

## Decision

1. **`scripts/static.sh` runs four more analysers**, each a `step`, each pinned:
   - `ruff check --select F,E9` over `git ls-files '*.py'` — pyflakes (unused imports and names,
     undefined names) and syntax errors; ruff 0.15.6, required on PATH.
   - `shellcheck -S warning` over `git ls-files '*.sh'`; shellcheck 0.11.0, required on PATH.
   - `actionlint` over `.github/workflows/`, as `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`,
     with `SHELLCHECK_OPTS='-S warning'` so a workflow's `run:` scripts meet the bar `scripts/` meets.
   - `knip` in `cmd/opencode/mrw-plugin` — unused files, exports and dependencies — pinned as an exact
     devDependency `knip@6.38.0`, so `package-lock.json` pins its whole dependency tree (knip 6.38.0
     declares 10 of its 13 dependencies as `^` ranges, which `npx -y knip@6.38.0` would resolve afresh
     on every cold cache). It runs from that directory's `node_modules`; when they are missing the step
     fails and names `npm ci`, rather than passing on a run that could not look. 6.38.0 (published
     2026-09-23) and not the newest 6.41.0 (2026-10-09): a pin should have been out for two weeks.

   Each step lists its files first, prints `<step>: N file(s)`, and fails when N is 0: on this Mac
   `printf '' | xargs -0 shellcheck` exits 0 having run nothing, so a glob that stopped matching would
   otherwise read as a clean pass.
2. **A `self-test` step proves each of the four can fail.** In a temporary directory, removed on every
   exit, it plants:
   - for ruff, a `.py` that imports `os` and never uses it — the output must name F401;
   - for shellcheck, a `.sh` whose `cd` has no failure branch — SC2164;
   - for actionlint, a workflow with a `run:` loop whose counter is never read and a
     `${{ github.nosuch }}` expression — the output must name both SC2034 (its shellcheck path) and
     `nosuch` (its expression checker). Amended in T2: the first plant, `run: cd x`, drew no SC2164
     from actionlint 1.7.12, since a `run:` step's default shell is `bash -e`, under which a failed
     `cd` stops the step; an unused counter is the class the tree had (`timing-123.yml`);
   - for knip, a project skeleton: a `package.json` whose `main` is `src/index.ts`, that file, and an
     unreferenced `src/dead.ts`, run with the plugin's knip binary — the output must name `dead.ts`.

   Each analyser must exit non-zero AND name its defect, so a gate that never reached its tool, or a
   tool that exits 1 for a configuration error, does not pass.
3. **Every finding above is fixed**, none silenced. The four `ls | grep` rows are rewritten over a
   name list a glob produces, because a wildcard glob reads the directory and keeps each entry's own
   spelling: measured 2026-10-13 on case-folding APFS with bash 3.2.57 and only `a230.txt` on disk,
   `[ -e A230.TXT ]` is true while `*` lists `a230.txt`, so the two case-folding rows (8633, 8975) still
   see which spelling the directory holds. Row 8109 looks for mrw's temp file, a dotfile —
   `internal/state/write.go:60` names it `"." + base + ".tmp-*"` — so its glob sets `dotglob`.
4. **CI installs them** in the `test` job before `scripts/static.sh`: ruff with `pipx` (preinstalled
   on GitHub's `ubuntu-latest` image); shellcheck from its v0.11.0 release tarball, checked against
   sha256 `8c3be12b05d5c177a04c29e3c78ce89ac86f1595681cab149b65b97c4e227198` (computed 2026-10-13 from
   `https://github.com/koalaman/shellcheck/releases/download/v0.11.0/shellcheck-v0.11.0.linux.x86_64.tar.xz`)
   and put first on `PATH`; node 24 with `actions/setup-node` and the plugin's `npm ci`.

What would make the decision fail: an analyser that stops reporting, and the self-test is the check
that says so. Data that produces such a failure exists now: each planted defect was reported on
2026-10-13 by the pinned version (ruff F401, shellcheck SC2164, actionlint `property "nosuch" is not
defined` and SC2034, knip `Unused files`).

## Alternatives Considered

- **ruff's default rule set** (E4, E7, E9, F): 134 findings, 132 of them E701/E702 (several
  statements on one line) — style, which ADR-088 declined for Go. Rejected.
- **ruff with flake8-bugbear (`B`)**: 7 more. The three B023 (a closure over a loop variable), all
  read on 2026-10-13, are closures called inside the same iteration (`scripts/chaos.py:590`, threads
  joined in-loop; `:747`, `send` called in-loop). The other four are B007 (an unused loop variable,
  three) and B905 (`zip` without `strict=`), each a style rule. Rejected.
- **vulture**: 0 at `--min-confidence 80`, where it reports what ruff F already does; at 60, 4, all
  read as false (two `shlex` attributes, a `unittest` class, a registry constant). Rejected.
- **shellcheck at `-S info` with SC2015 excluded**: 69 findings. The 11 SC2086 and SC2059 were read on
  2026-10-13 and are deliberate word splitting of a flag list or a `printf` format the row builds; the
  other 58 (SC2016, SC2012 and smaller codes) were judged by their code, not read line by line.
  Rejected: ADR-088's bar is defects, not style.
- **knip through `npx -y knip@6.38.0`**: nothing added to the plugin's manifest. Rejected: `npx`
  pins knip and not its ranged dependencies, while the plugin needs `npm ci` for knip anyway, so the
  devDependency costs no extra step and pins the whole tree.
- **knip in CI's `opencode` job instead of `static.sh`**: no new install in the `test` job. Rejected:
  the after-commit hook runs `static.sh`, so a finding would reach CI and never the model.

## Component / Boundary Impact

None — internal to the repository's tooling: `scripts/`, `.github/workflows/`, `.claude/rules/`, the
plugin's devDependencies, the docs that list the gate. No package under `internal/` or `cmd/mrw/`
changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `scripts/static.sh` | five more steps; requires ruff, shellcheck and the plugin's `node_modules` | T2 | developers, `.claude/hooks/static-after-commit.py`, CI `test` job, `mrw check --then static` (ADR-092) |
| `cmd/opencode/mrw-plugin/package.json` | devDependency `knip` 6.38.0, exact | T2 | `npm ci` in CI's `test` and `opencode` jobs |
| CI `test` job | installs ruff, shellcheck, node 24 and the plugin's dependencies | T3 | `scripts/static.sh` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| a tree every new analyser passes | T1 | T2 | No — T2's gate is green only on it |
| the new `static.sh` steps | T2 | T3 | No — CI installs what they require |

## Implementation

See `tasks/`. The record and its tasks land in one pull request: fence-prose reads every task's
fence, pending ones included, so T3's greps over the docs are red until T3 is done.

## Consequences

- **Positive:** a Python, shell, workflow or plugin defect of the classes above reaches the model after
  the commit that introduced it, as a Go one does.
- **Negative:** `static.sh` runs four more analysers and the self-test runs them again on the fixture.
  Measured in T2, 2026-10-13: the whole gate ran clean in 20.7 s warm at load 12.5 on 10 cores, against
  6.3 s for ADR-088's steps alone at load 2.5 — the loads differ, so not all of the difference is this
  record's; the 280 s hook bound holds either way. A developer without ruff or shellcheck, or without
  the plugin's `node_modules`, sees the gate fail and name the install — on the after-commit hook too.
- **Neutral:** a `--then static` (ADR-092) pays the same cost. The `opencode` job's `npm ci` installs
  knip as well.

## Out of Scope

- Python embedded in shell: the 108 heredocs and 52 `python3 -c` sites in `scripts/contract.sh` and `scripts/break-campaign.sh` (deferred: docs/adr/BACKLOG.md, "Python embedded in shell is not analysed")
- The task Acceptance fences (permanent: boundary: `adr-verify` runs each, and fence-prose re-runs their greps over tracked files on every gate run, ADR-088 T2)
- Inline commands in `.claude/settings.json` and `.quality-harness.json` (permanent: boundary: each is a one-line invocation of a tracked script or `go test`, and those are analysed)
- `cmd/opencode/mrw-plugin/test/smoke.test.mjs` under a type checker (permanent: boundary: knip reads it as an entry, `node --test` runs it in CI, and type-checking JavaScript is a `checkJs` change to the plugin ADR-089 owns)
- The recorded fixture trees under `docs/blind/` and `docs/break/` that are not executed harnesses — the 16 `.yaml` and the `.tsx` (permanent: boundary: they are kept byte-for-byte as the trees a reading or break campaign ran against)
- `.golangci.yml` and other configuration (permanent: boundary: each is read by the tool it configures, which refuses one it cannot parse)
- Formatters for Python, shell or TypeScript (permanent: boundary: ADR-088 gates defects, and a formatter's findings are style)
- Info- and style-level shellcheck findings, B and E7 ruff rules, vulture (permanent: boundary: the Alternatives give each count and why)
- A contract row (permanent: boundary: static analysis is not a promise the binary makes, as ADR-088 records)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| ubuntu-latest's own shellcheck shadows the pinned one | Medium | Low | CI puts the pinned binary first on `PATH`, and `static.sh` notes a version that differs |
| `pipx` leaves a future `ubuntu-latest` image | Low | Low | the install step fails loudly; T3's run is the check |
| `go run` or `npm ci` needs the network on a cold cache | Low | Low | the failure names the tool, as govulncheck's does under ADR-088 |
| actionlint without shellcheck on PATH skips its `run:` checks silently | Low | Medium | the shellcheck step fails first when it is missing, and the self-test requires actionlint to report SC2034 |

## Rollback

Revert the tasks: the steps are lines in `scripts/static.sh`, the installs are steps in `ci.yml`, knip
is one devDependency.

## Follow-ups

None.
