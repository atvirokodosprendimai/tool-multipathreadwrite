# ADR-183: `mrw version --check-latest` judges a `go install` pseudo-version by its base release

**Status:** Accepted
**Accepted:** 2026-10-29 by Zy — *"accepted"*
**Date:** 2026-10-29
**Amended:** 2026-10-29, before acceptance, from a Codex xhigh challenge (2 P2, 2 P3), each finding checked against the code: the `User-Agent` change had no case that told it from today's `mrw/dev`, so the seam test asserts `mrw/<reported version>` for the go-install cases and S3 gains the mutant that keeps `"mrw/"+version` (`cmd/mrw/latest149.go:72`); S7's README and AGENTS.md edits were in no fence clause, so the fence greps the phrase both must carry; the dependency clause named only `golang.org/x/mod`, so it holds `go.mod` and `go.sum` unchanged instead; the stamp sites are four commands, not three (Context). Codex found no fault in the base rule, the verdicts, the Invalidates line, the seam or §317's design.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-149, ADR-040, ADR-009, CONTRIBUTING.md, AGENTS.md, README.md
**Governs:** `cmd/mrw/latest149.go`, `cmd/mrw/main.go`, `README.md`, `AGENTS.md`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/latest183_test.go::TestCheckLatestJudgesAPseudoVersionByItsBaseRelease`
**Invalidates:** ADR-149 — the fourth bullet of its Decision 1, `cannot be compared` "when `version` is not a strict `vX.Y.Z`", which narrows to "when the version mrw reports is neither a strict `vX.Y.Z` nor a Go pseudo-version of the release form"; its User-Agent `mrw/<version>`, which takes the reported version. Checked and unchanged: ADR-149 Decisions 2-5 (could not ask is exit 2; no release count; nothing else makes a network call, held by `TestOnlyTheCheckLatestFileImportsTheNetwork`; `releasesAPI` moved only by `-X`), its current, newer and older lines and their exit codes; ADR-040 (`mrw version` prints what it printed); ADR-009 (nothing automatic is sent).
**Served-path change:** `mrw version --check-latest` on a `go install …/cmd/mrw@<commit>` build, which reports a pseudo-version such as `v1.77.2-0.20261028222602-46fc0cf9e067`, says `this mrw (<pseudo>) is older` and exits 1 when the latest release is above the release the commit follows, and `this mrw (<pseudo>) is a commit after vX.Y.Z` with exit 0 otherwise, where it said `cannot be compared` before; a `go install …@vX.Y.Z` build gets ADR-149's current/newer/older verdict; a checkout build and every other version keep `cannot be compared`.

## Context

`mrw version --check-latest` (ADR-149, `cmd/mrw/latest149.go:109-132`) compares the latest release tag with the linker variable `version` (`:114`) and gives no verdict, exit 0, for any `version` that is not a strict `vX.Y.Z` (`:115-117`). Since #561 (`cmd/mrw/main.go:105-117`, `moduleVersion`), a binary built by `go install github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw@<sha>` has no linker stamp, so `version` stays `dev`, but it reports the module version Go recorded: a pseudo-version (`v1.76.1-0.20261027131114-7c2c46c0504f`, a Windows session, 2026-10-27, quoted in `cmd/mrw/version_module_test.go:9-12`). `--check-latest` still reads `version`, so it prints `this build (dev) cannot be compared` under a version line that names the commit — the one build kind whose user is most likely to be behind (it is a snapshot of `main`, never updated by a release) gets no verdict. The same holds for `go install …@vX.Y.Z`: it reports the tag (`version_module_test.go:24`), and `--check-latest` still calls it `dev`. Owner direction, 2026-10-29: *"Two small ADRs"* — this is the second.

**Audit of the class** — *a build kind whose reported version names a release `--check-latest` could judge, while the value it judges is the raw linker stamp*. Enumerated by the branches of the function that produces the reported version, `mrw read 'cmd/mrw/main.go:/^func moduleVersion/,/^}/'`, and the table that pins them, `mrw read 'cmd/mrw/version_module_test.go:/for _, tc/,/} {/'` (six rows, at `e8a6a3da`), plus the stamp sites, `grep -rn 'main.version' .github/ scripts/` (four commands: `ci.yml:303`, `timing-123.yml:36` and `:37`, `contract.sh:9079`):

| Build kind | Reported version | Today | After |
|---|---|---|---|
| a release (`ci.yml:303`, `-X main.version=${GITHUB_REF_NAME}`) | strict `vX.Y.Z` | verdict | verdict, unchanged |
| hand-stamped (`timing-123.yml:36-37`, `contract.sh` §253) | the stamp | verdict if strict | verdict if strict or a release-form pseudo-version |
| `go install …@vX.Y.Z` | the tag | no verdict | verdict |
| `go install …@<sha>` | a pseudo-version | no verdict | verdict when it has a release base |
| a checkout build | `dev (rev[, dirty])`; judged as `dev` | no verdict | no verdict |
| no module version (`(devel)`, empty) | `dev` | no verdict | no verdict |

Two members stay without a verdict on purpose: a checkout build names a commit and no release (its module version is ignored by `moduleVersion` once `vcs.revision` is present), and `(devel)` names nothing.

## Existing Primitives Audit

- `moduleVersion(version, info.Main.Version, info.Settings)` (`cmd/mrw/main.go:105`): already the reported version before `versionFrom` adds `(rev[, dirty])`. Reused as the value to judge; its signature and its table test are unchanged.
- `versionString()` (`cmd/mrw/main.go:91-97`) calls `debug.ReadBuildInfo()` directly. A test binary always reports `Main.Version` `(devel)` (measured 2026-10-29, go1.27.3, a scratch module's `go test`), so no test can make the real build info carry a pseudo-version; the go-install wiring is reachable in a test only through a seam. Added: `var readBuildInfo = debug.ReadBuildInfo`, a package-level variable read by `versionString` and by the new `reportedVersion`, the same shape as `releasesAPI` (ADR-149 Decision 5). It moves nothing a caller can set.
- `semver` and `compareSemver` (`cmd/mrw/latest149.go:34-61`): reused for the base and the latest; no second comparison.
- `golang.org/x/mod/module` has `IsPseudoVersion` and `PseudoVersionBase` (`pseudo.go:123,164` at `golang.org/x/mod@v0.42.0`), but `go.mod` requires one module, `urfave/cli/v3`, and `CONTRIBUTING.md`'s Prerequisites row asks a new one to show the standard library cannot serve. It can: only the release form yields a base (Decision 2), and that is one regular expression over the shape x/mod's `pseudoVersionRE` (`pseudo.go:47`) accepts and the decrement its `-0` case does. Not imported (Alternatives).
- `strictTag` (`cmd/mrw/latest149.go:31`): the shape every release has; mrw has published no tag of any other shape (`git tag | grep -c -- -` → 0 at `e8a6a3da`).

## Decision

**1. `--check-latest` judges the version mrw reports.** `reportedVersion()` is `moduleVersion(version, info.Main.Version, info.Settings)` over `readBuildInfo()` (`version` when there is no build info); `versionString` composes `versionFrom` over it, so the version line is unchanged. `checkLatest` takes the reported version, and the request's `User-Agent` is `mrw/<reported version>` — equal to ADR-149's for every stamped build. A stamp still wins over the module version (`moduleVersion`, unchanged).

**2. A pseudo-version of the release form names its base.** A reported version matching `^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)-0\.\d{14}-[0-9A-Za-z]+$` with a patch above 0 follows the release `vX.Y.(Z-1)`: Go writes `vX.Y.(Z+1)-0.<timestamp>-<rev>` for a commit after tag `vX.Y.Z` (`pseudoVersionBase`, in `cmd/mrw/latest149.go`). Every other shape has no base mrw can judge, and keeps ADR-149's line `latest release: vL — this build (<reported>) cannot be compared`, exit 0:
   - `vX.0.0-<timestamp>-<rev>` — no tag before the commit;
   - `vX.Y.Z-<pre>.0.<timestamp>-<rev>` — a commit after a pre-release tag, which this repository has never published;
   - `vX.Y.0-0.<timestamp>-<rev>` — invalid (x/mod refuses it: there is no patch to decrement);
   - any build metadata (`+incompatible`, `+dirty`) — `+incompatible` cannot arise for this module path at v1, and `+dirty` is added only to a checkout build, which `moduleVersion` reports as `dev`;
   - `dev`, and anything else that is not strict.
   The timestamp is not checked as a date: the version is Go's record, not the caller's input, and a malformed one yields at worst a verdict about a base Go wrote.

**3. The verdict for a pseudo-version.** With base `vB` and latest `vL`:
   - `vL` above `vB`: `latest release: vL — this mrw (<pseudo>) is older; download <page>`, then `mrw: a newer release exists: vL` on stderr, exit 1 — ADR-149's older line and exit code, with the pseudo-version in the parentheses;
   - `vL` equal to `vB`: `latest release: vL — this mrw (<pseudo>) is a commit after vL`, exit 0. It is a commit after the latest release, so it is not older, and it is not that release, so it is not "current". `v1.78.1-0.<ts>-<rev>` against latest `v1.78.0` is this case;
   - `vL` below `vB` (a tag newer than the release GitHub calls latest): the same line, `… is a commit after vB`, exit 0 — not ADR-149's "is newer", which would claim the build is a release.

**4. A strict reported version is judged as ADR-149 judges it**, whichever source it came from: a `go install …@v1.77.1` build at latest `v1.77.1` is current, exit 0; at latest `v1.78.0` it is older, exit 1.

**5. Exit codes keep ADR-149's meanings.** 0: current, newer, a commit after its base, or no verdict; 1: a newer release than the build exists; 2: it could not ask. Only which builds get a verdict widens. Plain `mrw version` and every other command are unchanged and still make no request.

**What would make this decision wrong:** a release tag that is not strict (the base rule then names a tag that was never published), or Go changing its pseudo-version form. The first is refused by the build job (`ci.yml` builds only for a strict `vX.Y.Z` tag); the second would leave a new form unmatched, which is no verdict, never a false exit 1.

## Alternatives Considered

- **Import `golang.org/x/mod/module`** (`PseudoVersionBase`). Rejected: a second module requirement for one regular expression and a decrement, which `CONTRIBUTING.md` Prerequisites asks a record to justify and this one cannot; its extra coverage (pre-release bases, `+incompatible`) is exactly what Decision 2 leaves without a verdict.
- **Judge a pre-release base by semver precedence** (`vX.Y.Z-pre` is below `vX.Y.Z`, so latest ≥ `vX.Y.Z` is older). Rejected: this repository has published no pre-release tag (`git tag | grep -c -- -` → 0 at `e8a6a3da`), so the rule has no build to serve, and no verdict is never a false exit 1.
- **Report the equal-base case as "current".** Rejected: the build is a later commit, not the release; a script reading "current" would take an unreleased snapshot for the release.
- **Report the equal-base case as older.** Rejected: nothing released is newer than the build; exit 1 would tell its user to download a release that lacks the commits they have.
- **A contract row that runs `go install` against a `GOPROXY=file://` module proxy built in the script** (module zip, `.info`, `.mod` under a pseudo-version). Rejected: it reimplements Go's module-proxy layout in shell to reach what `TestCheckLatestJudgesTheVersionAGoInstallBuildReports` reaches through the `readBuildInfo` seam; the row instead stamps a pseudo-version with `-X`, which `moduleVersion` passes through, so the built binary's comparison is proved end to end and the go-install wiring by the seam test (T1 Risks).
- **Keep judging `version`, and add the module version only to the "cannot be compared" line.** Rejected: names the commit and still gives no verdict, which is the gap.

## Component / Boundary Impact

- `cmd/mrw` (`latest149.go`, `main.go`): the value `checkLatest` judges and the base rule. No engine package changes; the network boundary (`TestOnlyTheCheckLatestFileImportsTheNetwork`) is unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw version --check-latest` stdout | a pseudo-version gets `is older` or `is a commit after vB`; a go-install tag gets ADR-149's verdict | `cmd/mrw` (T1) | people, scripts |
| `mrw version --check-latest` exit code | 1 for a pseudo-version behind the latest; meanings unchanged | `cmd/mrw` (T1) | scripts |
| request `User-Agent` | `mrw/<reported version>` | `cmd/mrw` (T1) | GitHub |
| contract §317 | new rows | `scripts/contract.sh` (T1) | CI |
| `README.md`, `AGENTS.md` | which builds get a verdict | docs (T1) | agents, people |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** a `go install …@<sha>` user learns whether a release has passed their commit; a `go install …@vX.Y.Z` user gets the verdict a release download gets.
- **Negative:** a script that treated every `go install` build's exit 0 as "nothing to do" now sees exit 1 when it is behind — the meaning ADR-149 gave exit 1.
- **Neutral:** the `User-Agent` of a go-install build names its module version instead of `dev`.

## Out of Scope

- A verdict for a checkout build (`dev (rev)`) (permanent: boundary: it names a commit and no release; finding the release it follows needs git, which `--check-latest` does not run)
- A pre-release base (`vX.Y.Z-pre.0.<ts>-<rev>`) (permanent: fact: this repository has published no pre-release tag; citation: file `.github/workflows/ci.yml:303`)
- Checking the pseudo-version's timestamp or revision against the repository (permanent: boundary: ADR-149 Decision 3, one request; the revision is printed, not looked up)
- Importing `golang.org/x/mod` (permanent: boundary: Alternatives; the standard library serves the one form judged)
- The centralised `mrw` skill copy in memoryd `craft` (external: memoryd craft: refreshed by whoever merges, from `mrw skill`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A go-install user behind the latest release is told so where they were not before (exit 1) | Med | Low | that is ADR-149's exit 1; README names which builds get a verdict |
| The base rule misreads a pseudo-version and claims older | Low | Med | only the release form with patch above 0 yields a base; every other shape is no verdict, exit 0; T1's table covers each form named in Decision 2 |
| The wiring judges the stamp, not the reported version, and every test still passes | Med | Med | `TestCheckLatestJudgesTheVersionAGoInstallBuildReports` drives the real command with `version` `dev` and a fake build info through `readBuildInfo`; T1 S3's mutant |

## Rollback

Before a release: revert T1. After a release: a later release restores ADR-149's rule; nothing persistent changes.

## Follow-ups

- [ ] none at authoring
