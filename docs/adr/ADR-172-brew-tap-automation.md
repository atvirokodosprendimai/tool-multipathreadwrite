# ADR-172: brew tap formulas are generated and pushed by CI

**Status:** Proposed
**Accepted:** pending
**Date:** 2026-10-25
**Owner:** Marty
**Spec:** None — no spec stage
**Cross-references:** ADR-089, ADR-147, ADR-149, UPDATE.md, README.md (Install), CONTRIBUTING.md
**Governs:** `.github/workflows/brew-tap.yml`; the renderer it runs is `tools/gen-brew-formula.ts` in atvirokodosprendimai/skills, vendored at run time (ADR-172, Decision 4)
**Enforced-by:** CI only reaches the tap push on a strict `v*.*.*` tag whose Release the `release` job has already published
**Invalidates:** None — UPDATE.md's manual update recipe is kept and remains the fall-back when the push fails
**Served-path change:** none for callers of the binary; the only new artifact is a commit in atvirokodosprendimai/homebrew-tap at release time

## Context

mrw ships release archives (`mrw_<goos>_<goarch>.tar.gz`, `SHA256SUMS.txt`) but installs by curl from `releases/latest` (README). The org's tap (atvirokodosprendimai/homebrew-tap) exists and carries wgmesh, whose formula GoReleaser regenerates every release — one `brew install`/`brew upgrade` surface, no manual sha copy. mrw's pipeline is bespoke (one ci.yml, no goreleaser), so nothing regenerates an mrw formula; a hand-maintained one goes stale on the very next release, and a stale formula serves a stale sha that brew refuses at download time — discovered by a user, not by CI.

The skills repository (atvirokodosprendimai/skills `tools/gen-brew-formula.ts`) already carries a pure, tested generator that renders the GoReleaser-shaped formula from a release's own `SHA256SUMS.txt`. What was missing is the wiring: run it at tag time, against the tag, with write access to the tap.

## Decision

1. **CI renders and pushes the formula at release time.** A successful tag run of `ci` triggers `brew-tap`, which re-derives the tag from the run (workflow_run events carry no tag ref), confirms the Release already has assets — the precondition `release` owns — and pushes `Formula/mrw.rb` to the tap's default branch with a commit message naming the tag. The tap is a downstream of the Release, not a gate on it: a failed push turns `brew-tap` red and does not un-publish a tagged release.

2. **The formula serves the release archives, never a source build.** mrw's version contract (`mrw version` prints tag + SHA; UPDATE.md verifies it) survives a brew install only when the binary is the published one.

3. **The sha256 values are taken from the release's `SHA256SUMS.txt`, not recomputed.** The formula attests exactly what was published; before pushing, the workflow asserts the rendered shas equal the manifest's, so a half-published release cannot reach the tap.

4. **The generator is vendored from the skills repo at run time, not copied into this repo.** One copy of the convention; mrw keeps no second renderer to drift. Its tests live beside it, in the skills repo.

5. **A bootstrap formula landed by hand once** (tap PR #3, same bytes the generator renders) so `brew install atvirokodosprendimai/tap/mrw` works before the first automated push. The automation replaces it at the next tag.

6. **UPDATE.md gains the brew row** so the update story has one entry per install surface: `brew update && brew upgrade mrw`.

## Alternatives

- **Adopt GoReleaser** (the wgmesh path). Rejected: mrw's two asset shapes (raw + archive, chosen for the one-line curl install) and its contract-checked exit codes are load-bearing; rewriting the release for a formula is the tail wagging the dog.
- **Hand-maintained formula updated per release.** Rejected: it *will* be forgotten; the failure mode is a stale sha, and the discovery cost lands on a user.
- **A separate `brew-tap` repo fed by a shared workflow.** Rejected: one more repository to trust, for what a small job does.

## Consequences

- Every strict tag now writes one commit to the tap, directly to its default branch — the same contract wgmesh's GoReleaser push already sets there. If the tap later gains branch protection, the push fails loudly (`brew-tap` red) and the manual recipe in UPDATE.md is the recovery path; the workflow does not silently fall back to a PR nobody reviews.
- The tap's `mrw` formula is only as current as the last mrw tag.
- `brew bump-formula-pr` is not the update path here — regeneration is; the DO-NOT-EDIT header says so.
- The vendored generator's tests live in the skills repo; formula-shape changes are made there and picked up by every AKS release that vendors it.

## Experiences

- 2026-10-25: formula rendered locally from v1.75.0's SHA256SUMS.txt; `brew style` clean; installed from the rendered file on an M-series Mac; `brew test atvirokodosprendimai/tap/mrw` passed; `/opt/homebrew/bin/mrw version` → `v1.75.0 (c81cb11)`; all four shas cross-checked against the published manifest.
