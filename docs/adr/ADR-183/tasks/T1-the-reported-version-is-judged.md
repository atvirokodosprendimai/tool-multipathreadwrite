# Task ADR-183-T1: `--check-latest` judges the version mrw reports

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `reportedVersion`, `readBuildInfo` in `cmd/mrw/main.go`; `pseudoVersionBase` in `cmd/mrw/latest149.go`; contract §317; `TestCheckLatestJudgesAPseudoVersionByItsBaseRelease`, `TestCheckLatestJudgesTheVersionAGoInstallBuildReports`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a pseudo-version behind the latest exits 1`, `an equal base is a commit after it`, `no base is no verdict`, `the reported version is judged`

## Goal

Decisions 1-5 of the record: `checkLatest` judges `reportedVersion()`; a release-form pseudo-version is judged by its base, an equal base being a commit after the latest (exit 0); every other non-strict version keeps no verdict; ADR-149's lines, exit codes and network boundary are otherwise unchanged.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/latest183_test.go` | new | the red tests; a new file, so `cmd/mrw/latest149_test.go` and ADR-149 T1's test lock do not move |
| `cmd/mrw/latest149.go` | edit | `pseudoVersionBase`; `checkLatest` (`:109-132`) and `latestTag` (`:65`, the `User-Agent` at `:72`) take the reported version; the pseudo-version branch of Decision 3 |
| `cmd/mrw/main.go` | edit | `var readBuildInfo = debug.ReadBuildInfo`; `reportedVersion()` beside `versionString` (`:91-97`), which calls it; `versionCmd` (`:625`) passes it to `checkLatest`; the `check-latest` flag Usage (`:607-608`) unchanged unless it names which builds get a verdict |
| `scripts/contract.sh` | edit | §317 (after §253, `:9054-9100`): a stamped pseudo-version against a local server |
| `README.md` | edit | `:111-114`, "0 for a local build, which it names without a verdict": a go-install build gets a verdict, a checkout build none |
| `AGENTS.md` | edit | `:668-670`: one clause, a go-install build is judged by the release its commit follows |

## Ordered Steps

1. [S1] Write `TestCheckLatestJudgesAPseudoVersionByItsBaseRelease` (an `httptest` server as in `latest149_test.go`, `version` stamped through a local restore helper, each case a line and an exit: `v1.78.1-0.20261029000000-abcdefabcdef` at latest `v1.78.0` → 0, `this mrw (v1.78.1-0.20261029000000-abcdefabcdef) is a commit after v1.78.0`; at `v1.78.1` and at `v1.79.0` → 1, `is older; download`, error `a newer release exists`; at `v1.77.0` → 0, `is a commit after v1.78.0`; `v1.10.1-0.…` at `v1.9.9` → 0, `is a commit after v1.10.0` (numeric, not lexical: `v1.10.0` sorts below `v1.9.9` as text); no verdict, exit 0, `this build (<v>) cannot be compared`, for `v0.0.0-20261029000000-abcdefabcdef`, `v1.78.0-rc.1.0.20261029000000-abcdefabcdef`, `v1.78.0-0.20261029000000-abcdefabcdef` (patch 0), `v1.78.1-0.20261029000000-abcdefabcdef+dirty`, `v1.78.1-0.2026102900000-abcdefabcdef` (13 digits) and `v01.78.1-0.20261029000000-abcdefabcdef`; every case's `User-Agent` is `mrw/<v>`) and `TestCheckLatestJudgesTheVersionAGoInstallBuildReports` (through `runVersion`, `version` `dev`, `readBuildInfo` replaced: `Main.Version` `v1.77.2-0.20261028222602-46fc0cf9e067` and no `vcs.*` at latest `v1.78.0` → the version line is that pseudo-version and the verdict is `is older`, exit 1; `Main.Version` `v1.77.1` and no `vcs.*` at latest `v1.77.1` → current, exit 0, and at `v1.78.0` → older, exit 1; the pseudo-version with a `vcs.revision` (a checkout) → `this build (dev) cannot be compared`, exit 0; `(devel)` → the same; every case's `User-Agent` is `mrw/<the reported version>`, so `mrw/v1.77.2-0.20261028222602-46fc0cf9e067` and `mrw/v1.77.1`, not `mrw/dev`); see both RED on the current code.
2. [S2] `pseudoVersionBase`: the release form with patch above 0 names `vX.Y.(Z-1)`; `checkLatest` compares that base with the latest. Mutant: return `vX.Y.Z` without the decrement — the `v1.78.1` at `v1.78.1` case goes from exit 1 to 0. [proof: mutation]
3. [S3] `readBuildInfo`, `reportedVersion`, `versionCmd` passes it, and `latestTag` sends it as the `User-Agent`. Mutant: `versionCmd` passes `version` — the go-install cases give no verdict and go red; second mutant: `latestTag` keeps `"mrw/"+version` — the go-install cases' `User-Agent` is `mrw/dev` and goes red. [proof: mutation]
4. [S4] Every other shape keeps no verdict. Mutant: drop the patch-above-0 condition (a `vX.Y.0-0.…` gets a base) — the patch-0 case goes red; second mutant: drop the `$` anchor — the `+dirty` case goes red. [proof: mutation]
5. [S5] The equal-base line. Mutant: an equal base reports older — the `v1.78.1-0.…` at `v1.78.0` case goes red. [proof: mutation]
6. [S6] Contract §317: one binary stamped `-X main.version=v1.78.1-0.20261029000000-abcdefabcdef -X main.releasesAPI=<local>` against a local server: tag `v1.78.0` → exit 0 and `is a commit after v1.78.0`; the pair, tag `v1.78.1` → exit 1 and `mrw: a newer release exists: v1.78.1`; a second binary stamped `v0.0.0-20261029000000-abcdefabcdef` → exit 0 and `cannot be compared`. The server and binaries are removed at the end, as §253 does. [proof: acceptance]
7. [S7] `README.md:111-114`, `AGENTS.md:668-670`: each says a `go install` build is judged by the release its commit follows, and README no longer calls every local build unjudged; the fence greps that phrase in both. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestCheckLatestJudgesAPseudoVersionByItsBaseRelease|TestCheckLatestJudgesTheVersionAGoInstallBuildReports|TestCheckLatestSaysWhetherANewerReleaseExists|TestVersionWithoutTheFlagAsksNothing|TestOnlyTheCheckLatestFileImportsTheNetwork|TestAGoInstallBuildReportsItsModuleVersion)$' 2>&1) \
  && grep -qE '^--- PASS: TestCheckLatestJudgesAPseudoVersionByItsBaseRelease \(' <<<"$out" \
  && grep -qE '^--- PASS: TestCheckLatestJudgesTheVersionAGoInstallBuildReports \(' <<<"$out" \
  && grep -qE '^--- PASS: TestCheckLatestSaysWhetherANewerReleaseExists \(' <<<"$out" \
  && grep -qE '^--- PASS: TestVersionWithoutTheFlagAsksNothing \(' <<<"$out" \
  && grep -qE '^--- PASS: TestOnlyTheCheckLatestFileImportsTheNetwork \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAGoInstallBuildReportsItsModuleVersion \(' <<<"$out" \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum \
  && grep -qF 'judged by the release its commit follows' README.md \
  && grep -qF 'judged by the release its commit follows' AGENTS.md \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 317\. ' scripts/contract.sh \
  && grep -qF 'PASS  a pseudo-version whose base is the latest release is a commit after it' <<<"$c" \
  && grep -qF 'PASS  the pair: a release after its base makes it older' <<<"$c" \
  && grep -qF 'PASS  a pseudo-version with no base release names the latest with no verdict' <<<"$c" \
  && grep -qF 'PASS  naming the latest with no verdict' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestCheckLatestJudgesAPseudoVersionByItsBaseRelease` | `cmd/mrw/latest183_test.go` | each pseudo-version form's line and exit code against a local server; the equal-base case exit 0; every shape without a base no verdict | none | S1, S2, S4, S5 |
| `TestCheckLatestJudgesTheVersionAGoInstallBuildReports` | `cmd/mrw/latest183_test.go` | the real command judges `moduleVersion` over the build info, not the stamp: a go-install pseudo-version and tag get a verdict, a checkout and `(devel)` none | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | every `mrw version --check-latest`; the value judged is `reportedVersion()`, which `versionCmd` passes (S3's mutant) |
| 3 — the caller can discover it | the stdout line itself; README, AGENTS.md |
| 4 — it is used | no telemetry (ADR-009); the evidence for building it is the go-install report of 2026-10-27 quoted in `version_module_test.go`, whose build #561 made name its commit and `--check-latest` still could not judge |

## Mutation Log
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · S2: the base is vX.Y.Z, not the release before it · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: versionCmd judges the linker stamp, not the reported version · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · S3: the User-Agent names the linker stamp, not the reported version · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · S4: a patch-0 release-form pseudo-version gets a base · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · S4: the pattern is unanchored, so build metadata gets a base · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · S5: an equal base is reported older · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · S2: the base is vX.Y.Z, not the release before it · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · covers:a pseudo-version behind the latest exits 1
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · S5: an equal base is reported older · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · covers:an equal base is a commit after it
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · S4: a patch-0 release-form pseudo-version gets a base · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · covers:no base is no verdict
- 2026-10-29 · 7b4ebb2b* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: versionCmd judges the linker stamp, not the reported version · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · covers:the reported version is judged

## Invariants

- ADR-149's current, newer and older lines, its could-not-ask exit 2, and the one network file are unchanged.
- No verdict is never a false exit 1: a shape the base rule does not match is `cannot be compared`.
- The stamp wins over the module version; `moduleVersion` is unchanged.
- No new module requirement; no engine package changes.

## Risks

- The go-install path itself (`go install …@<sha>`, a real module-mode build) is not driven by a contract row: §317 stamps the pseudo-version, and the wiring from `debug.ReadBuildInfo` is proved by the seam test. Record Alternatives say why the `GOPROXY=file://` row was not taken.

## Stop Condition

The fence exits 0; or the base rule would need a form Decision 2 leaves without a verdict (a pre-release tag is published, or Go changes its pseudo-version form): stop and bring it to Zy.

## Out of Scope

- Changing ADR-149's lines for a strict version (permanent: boundary: ADR-149 Decision 1 owns them)

## Verification Log
- 2026-10-29 · 7b4ebb2b* · exit 1 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:363 · test-lock-sha256:76cb0fab836f98d63d1dd1fa0a1c55d4e53648847369a6bfbdf783d1b7ef7425 · test-lock-file:docs/adr/locks/76cb0fab836f98d63d1dd1fa0a1c55d4e53648847369a6bfbdf783d1b7ef7425.lock
  ```
  ```
- 2026-10-29 · 7b4ebb2b* · exit 1 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:1304
  ```
  ```
- 2026-10-29 · 7b4ebb2b* · exit 1 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:1369
  ```
  ```
- 2026-10-29 · 7b4ebb2b* · exit 1 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:1300
  ```
  ```
- 2026-10-29 · 7b4ebb2b* · exit 1 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:1322
  ```
  ```
- 2026-10-29 · 7b4ebb2b* · exit 1 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:1307
  ```
  ```
- 2026-10-29 · 7b4ebb2b* · exit 1 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:1375
  ```
  ```
- 2026-10-29 · 7b4ebb2b* · exit 1 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:1389
  ```
  ```
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:86629
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:86885
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:81944
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:81123
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:81357
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:80532
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:79829
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:80545
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:81140
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:89864
- 2026-10-29 · 7b4ebb2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:55726b5b57842f2c7777ef884a8f1a5c9cc6b88a73e79c59232a30f16949be4b · ms:91939
