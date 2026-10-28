# ADR-172: brew tap formulas are generated and pushed by CI

**Status:** Accepted
**Accepted:** 2026-10-28 by Zy — "Finish the existing PRs: Homebrew trigger fix and dependency PR blockers (Recommended)"
**Date:** 2026-10-25
**Amended:** 2026-10-28 — repair the tag trigger, pin the renderer and verify publication.
**Owner:** Marty; maintenance execution by Codex for Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-089, ADR-147, ADR-149, UPDATE.md
**Governs:** `.github/workflows/brew-tap.yml`, `scripts/brew_tap.py`, `scripts/test_brew_tap.py`
**Enforced-by:** `scripts/test_brew_tap.py::test_untrusted_runs_refused`, `scripts/test_brew_tap.py::test_publish_and_retry`, `scripts/test_brew_tap.py::test_workflow_reaches_publisher`
**Invalidates:** none — checked: binary release assets, the installer and engine contracts stay unchanged.
**Served-path change:** Homebrew installs and upgrades use the four prebuilt release archives through a generated tap formula.

## Context

mrw's bespoke release pipeline publishes archives and SHA256SUMS.txt. The org tap
has a shared renderer in the skills repository, but no merged mrw formula:
on 2026-10-28 tap PR #3 was still open. The original bootstrap claim in this
record was premature; its v1.75.0 local installation experience was not evidence
that the tap's default branch served it.

The first workflow filtered workflow_run to main. The successful v1.77.1 ci run
40676139276 instead has head_branch=v1.77.1, so that filter excludes the intended
publisher. Removing it requires explicit checks on the triggering run.

The governed class is the mrw tap publisher and its test; enumerate with
`git ls-files .github/workflows/brew-tap.yml scripts/*brew*tap*.py` (three files
after this change). ci.yml selects the tests; other release publishers are not changed.

## Existing Primitives Audit

Reuse ci.yml's successful release run, the published manifest, gh authentication,
and the skills renderer. Pin the renderer to commit
a9c2d508f840694cbd601cdbb3b313661fd51bc9 rather than executing mutable main.
The renderer retains ownership of Ruby formatting and archive selection; this
repository owns release authorization, attestation and publication.

## Decision

1. A completed ci workflow can trigger the publisher without a main branch filter.
   Before rendering or pushing, require a successful push run of this repository's
   ci.yml whose strict vX.Y.Z tag and head SHA match the canonical tag commit.
   Reject forks, PR runs, prereleases and mismatched commits.
2. Require the tag to be the latest published, non-draft, non-prerelease release,
   with the manifest and all four darwin/linux amd64/arm64 archives present.
   Recheck latest immediately before pushing, so an old replay cannot downgrade
   the tap. A concurrent advance can still reject a non-force push; retry safely.
3. Execute the pinned renderer with network permission only. Check each OS/CPU
   URL and its checksum against the manifest, including the version, rather than
   comparing an unordered set of hashes.
4. Use the tap's default branch. Keep PUSH_TOKEN out of argv and the remote URL;
   pass it through GH_TOKEN to gh's Git credential helper for the push.
   A byte-identical retry makes no commit. Read back and compare the exact formula
   bytes after publication, including on a no-op retry. Any failure turns this
   downstream workflow red without changing the released assets.
5. workflow_dispatch accepts a strict tag for repair or bootstrap. It requires the
   same successful release CI and current-release checks, so v1.77.1 can be
   published after this workflow merges without moving its tag or republishing assets.

These checks are falsifiable today: the hermetic suite supplies bad run metadata,
incomplete releases, incorrect URL/checksum pairs and a rejecting real Git remote.

## Alternatives Considered

- **Adopt GoReleaser:** rejected; the existing binary/archive release contract does
  not need replacing to publish one formula.
- **Keep shell logic inline:** rejected; a Python stdlib coordinator makes the
  production authorization and retry path executable against a local Git remote.
- **Keep a mutable renderer:** rejected; unrelated skills changes would execute
  with a tap credential in the environment.

## Component / Boundary Impact

The release job owns archive publication; the downstream publisher owns one tap
formula. No Go engine boundary changes. The integration crosses GitHub Actions,
the skills repository and the tap; no additional service or daemon is introduced.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| workflow_run / workflow_dispatch | trusted-run or tag input | ci / maintainer | brew-tap job |
| PUSH_TOKEN | tap contents write credential through credential helper | repository secret | Git push |
| Formula/mrw.rb | generated prebuilt URL/checksum pairs | pinned renderer and publisher | Homebrew |
| brew-tap harness step and ci test step | execute regression tests | scripts/test_brew_tap.py | local checks and CI |

## Inter-task Contracts

None — one implementation task.

## Implementation

See [tasks/README.md](ADR-172/tasks/README.md) and
[T1](ADR-172/tasks/T1-publish-the-release-formula.md).

## Consequences

- **Positive:** tags and safe manual replays update the tap; retries do not create
  empty commits and success means the checked formula is actually served.
- **Negative:** the renderer pin and Deno version need deliberate maintenance.
- **Neutral:** only the latest stable release is publishable; old releases remain
  available through their existing GitHub assets.

## Out of Scope

- Source builds and Go version stamping (permanent: boundary: Homebrew serves the published binary).
- Other tap formulas and the separate bootstrap PR #3 (external: homebrew-tap: PR #3).
- Binary telemetry (permanent: boundary: ADR-009 prohibits it).

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Missing or insufficient PUSH_TOKEN | Medium | High | fail the downstream job and verify through a real manual replay |
| Another release or tap writer races | Low | Medium | latest check, non-force push and idempotent retry |
| Renderer changes its output shape | Low | Medium | immutable pin, per-target validation and real-manifest render |
| Local doubles miss GitHub authentication behavior | Medium | Medium | post-merge workflow_dispatch and exact served-byte verification |

## Rollback

Disable brew-tap.yml and revert the generated formula through the tap's normal
review process. Release assets and the curl installer remain available.

## Follow-ups

None outside the task's explicit post-merge integration sign-off.

## Experiences

- 2026-10-25: original author reported local rendering and installation from
  v1.75.0's manifest. This does not prove the tap's default branch was updated.
- 2026-10-28: the pinned Deno 2.9.8 renderer produced the v1.77.1 formula from
  the real manifest; all four OS/CPU URL/checksum pairs validated and Ruby syntax
  passed. The ten-test local-remote suite passed, with trigger, trust, checksum
  and served-byte mutants killed. Actual Actions authentication/publication is
  the task's remaining post-merge sign-off.
- 2026-10-28: fresh review found that closing Ruby blocks were not tracked and
  subprocess stderr was hidden. Both were reproduced; two new asserting tests
  failed before their fixes. The twelve-test suite now passes, with paired
  scope-regression and diagnostic-suppression mutants killed. The original ten
  locked test bodies are unchanged; native relock added the two new bodies.
- 2026-10-28: actual [Actions run 40718218358](https://github.com/atvirokodosprendimai/tool-multipathreadwrite/actions/runs/40718218358)
  published v1.77.1 to tap commit `1ec84f0ac89658644b9f2e72059c7be75a82bca3`. An independent
  readback matched the rendered formula byte for byte (sha256
  `89679a5c283479316f76ebfc2ca6480fb3a878f46166b9dd4641f46c658ce7f7`) and all four
  OS/CPU URL/checksum pairs. T1's integration sign-off is complete.
