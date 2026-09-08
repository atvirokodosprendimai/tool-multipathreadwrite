# Task ADR-034-T5: The prune addresses a handle, and only ErrNotExist means gone

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** M (one file rewritten in `internal/state`, one line in `state.go`, one contract section)
**Owner:** Zy
**Produces:** `state.Entry.Name`, `state.Entry.Dead`, `state.openBase`, `state.resolveThroughAncestors`, contract §72
**Consumes:** T1's `state.Entries`/`state.Prune`, T3's §71 — both must keep passing
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the base opened is the base verified, by identity`, `a refusal names the base and says it is a symlink`, `a marker is read back exactly as Dir wrote it`, `only ErrNotExist means the checkout is gone`, `the self guard survives the live-to-missing transition`, `a removal is reported only when this walk saw the entry`, `the current answer vetoes the caller's stale one`, `a symlinked entry is reported, not silently dropped`

## Goal

Close the findings of the 2026-09-07 Codex review on PR #137 — and record which one did not hold,
because a review credited for something it did not find is a false citation
(`.claude/rules/reviews.md`).

## What the review found, and what measurement said

| # | Finding | Verdict |
|---|---------|---------|
| 1 | HIGH — enumeration and removal are not bound to one securely-opened base handle; `os.RemoveAll` follows a symlinked base | **held.** Reproduced before any fix: the prune removed a directory outside the base |
| 2 | MEDIUM — `TrimSpace` corrupts a marker whose path ends in a space; every stat error is read as "gone" | **held**, both halves. The same trim defect is also on the WRITE side at `state.go:61`, which the review did not name |
| 3 | MEDIUM — `selfDir` diverges from `Dir` across the live→missing transition | **held.** Probed before writing a fixture: alive `/private/var/…` key `a27de214…`, gone `/var/…` key `c6a44a46…` |
| 4 | MEDIUM — three fixtures leave `self` alive, so the self-survival assertions pass with the self guard deleted | **DID NOT HOLD.** `plant` removes `selfGo` at `prune_test.go:64`, so `self` IS dead and `e.Dir == self` is the only thing keeping it. The mutant is killed by `TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned` and `TestADryRunPruneReportsTheSameAndRemovesNothing`, exit 1. No work was done for this finding, and none was needed |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/prune.go` | rewrite | `openBase` opens the base through the parent handle and THEN verifies by `os.SameFile` that the name still resolves to the object it opened. `Prune` re-enumerates AND re-describes under its own handle, and removes a child NAME via `os.Root.RemoveAll` only where the current answer agrees with the caller's. `Entry.Live` becomes `Entry.Dead`, whose zero value is *keep*. `isDead` treats only `fs.ErrNotExist` as gone. `selfNames` returns three spellings. `children` returns symlinks so `describe` can report them unfollowed. |
| `internal/state/state.go` | edit | `:61` compares the marker with `TrimSuffix(…, "\n")`, matching exactly what `:62` writes |
| `internal/state/prune_test.go` | edit | `Live` → `Dead` at four assertion sites; the assertions keep their meaning |
| `internal/state/prune_findings_test.go` | create | Five tests, each red before its fix |
| `scripts/contract.sh` | edit | §72, pairing a refused symlinked base with a legitimate base that still prunes |

## Ordered Steps

1. [S1] Write the RED tests first, one per held finding, and confirm each fails for its OWN reason
   rather than merely making the suite red: a symlinked base, a marker with a trailing space, a root
   behind a denied parent, a self whose checkout is gone under a symlinked path. Reproduce finding
   3's key divergence with a scratch probe BEFORE building a fixture around it — a fixture written
   from review text that never reaches the branch is `testing.md`'s named failure — and run finding
   4's mutant to see whether it is real. [proof: acceptance]
2. [S2] `openBase`, `children`, and removal by name through the handle. [proof: mutation]
3. [S3] `TrimSuffix` on both the read and the write side, and `isDead` narrowed to `fs.ErrNotExist`.
   [proof: mutation]
4. [S4] `selfNames`, including the ancestor-resolved spelling. ⚠ The two obvious candidates are NOT
   enough: once the leaf is gone `absReal` and `filepath.Abs` return the same literal path, so the
   spelling the entry was created under has to be rebuilt from the ancestors that still exist. This
   cost a red test to learn, which is why it is its own step. [proof: mutation]
5. [S5] Delete the `validName` guard rather than keep it, and replace it with `present[e.Name]`,
   which can fail. [proof: mutation]
6. [S6] §72, paired: a base that must be refused and a base that must still be pruned. [proof: mutation]
7. [S7] Run the full gate list stand-alone, each command unpiped. [proof: acceptance]

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestABaseThatIsASymlinkIsRefusedRatherThanFollowed` | `internal/state/prune_findings_test.go` | A `<state>/mrw` that is a symlink to a directory holding a plausible dead entry is refused; the entry and everything beside it survive; and the refusal NAMES the base and the word `symlink` rather than leaving `os.Root`'s generic escape message to explain itself | — | S1, S2 |
| `TestAMarkerIsReadBackExactlyAsDirWroteIt` | `internal/state/prune_findings_test.go` | A live checkout whose directory name ends in a space is reported with that exact path, is identified, and is not removed | — | S1, S3 |
| `TestARootThatCannotBeStattedIsKept` | `internal/state/prune_findings_test.go` | A root behind a parent denying traversal stats with a permission error, not `ErrNotExist`, and its entry is kept | — | S1, S3 |
| `TestSelfSurvivesWhenItsCheckoutIsGoneUnderASymlinkedPath` | `internal/state/prune_findings_test.go` | A checkout named through a symlinked component, deleted underneath the run, keeps its own state directory — the case `Prune`'s doc comment promises and the existing fixture cannot see, because `plant` pre-resolves every root | — | S1, S4 |
| `TestAnEntryThatVanishedBetweenTheWalkAndTheRemovalIsNotReported` | `internal/state/prune_findings_test.go` | An entry removed between the walk and the prune is not reported as removed, because `os.Root.RemoveAll` succeeds silently on a name that is not there and ADR-008 says a delete says what it removed | — | S5 |
| `TestABaseThatIsARelativeSymlinkInsideTheStateHomeIsRefused` | `internal/state/prune_findings_test.go` | A relative `mrw -> other` INSIDE the state home — which `os.Root` follows, unlike an escaping one — is refused by the identity check, and the sibling's entry survives | — | S2 |
| `TestAnEntryWhoseCheckoutCameBackIsNotRemoved` | `internal/state/prune_findings_test.go` | A checkout restored between the caller's walk and the prune keeps its state: the current answer vetoes the stale verdict | — | S5 |
| `TestASymlinkedEntryIsReportedRatherThanSilentlyDropped` | `internal/state/prune_findings_test.go` | A symlinked entry is returned unidentified and not dead, so it is reported and kept, which is what ADR-034:132 promises | — | S5 |
| `TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned` | `internal/state/prune_test.go` | Unchanged behaviour under the rewrite, with `Live` read as `Dead` | — | S2, S3, S4 |
| `TestThePruneStaysInsideTheStateBase` | `internal/state/prune_test.go` | Unchanged: a symlinked ENTRY under a real base is still neither followed nor removed | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `openBase`, `isDead`, `selfNames`, `resolveThroughAncestors` in `internal/state/prune.go` |
| 2 — something selects it | `Entries` and `Prune` both call `openBase`; `Prune` calls `selfNames`; `describe` calls `isDead` |
| 3 — the caller can discover it | Unchanged from T2 — `mrw seen --prune` and its usage string. This task adds no surface a caller has to find |
| 4 — it is used | Contract §72 drives the refusal AND the successful prune through the built binary; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-08 · c60d7d2* · mutant killed · exit 1 · `internal/state/prune.go` · the symlinked-base refusal goes, so the only thing left is os.Root reporting that a path escaped — a message that names neither the base nor what is wrong with it. This is the mutant that SURVIVED until the test was made to assert the message rather than the mere presence of an error. · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · covers:a refusal names the base and says it is a symlink
- 2026-09-08 · c60d7d2* · mutant killed · exit 1 · `internal/state/prune.go` · the marker is trimmed wider than Dir writes it, so a checkout whose directory name ends in a space reads back as a different path — one that does not exist, so a LIVE checkout is classified dead and its state removed. · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · covers:a marker is read back exactly as Dir wrote it
- 2026-09-08 · c60d7d2* · mutant killed · exit 1 · `internal/state/prune.go` · every stat error is read as "the checkout is gone", so a root behind a denied parent, an unmounted volume or a timed-out network filesystem has its state removed on an answer about the LOOKUP rather than about the checkout. ⚠ The obvious spelling — return true — leaves errors and fs unused and does not compile, and a mutant that does not build proves nothing. · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · covers:only ErrNotExist means the checkout is gone
- 2026-09-08 · c60d7d2* · mutant killed · exit 1 · `internal/state/prune.go` · the ancestor-resolved spelling is dropped, leaving the two candidates that are IDENTICAL once the leaf is gone — so the self guard stops matching for a checkout reached through a symlinked component and the prune removes the state of the run in progress. ⚠ Deleting the block outright leaves real unused and does not compile. · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · covers:the self guard survives the live-to-missing transition
- 2026-09-08 · c60d7d2* · mutant killed · exit 1 · `internal/state/prune.go` · the walk-scoped check goes, so an entry that vanished between the walk and the prune is reported as removed — os.Root.RemoveAll succeeds silently on a name that is not there, and a delete that says it removed what it did not is ADR-008 broken. · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · covers:a removal is reported only when this walk saw the entry
- 2026-09-08 · fff5b94* · mutant killed · exit 1 · `internal/state/prune.go` · the identity check goes, leaving check-then-open — the ordering the first fix round shipped and its own comment wrongly called race-free. os.Root FOLLOWS a symlink that stays inside its root, so a relative mrw -> other under the same state home is opened and its sibling enumerated and pruned. · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · covers:the base opened is the base verified, by identity
- 2026-09-08 · fff5b94* · mutant killed · exit 1 · `internal/state/prune.go` · the escaping-symlink case falls through to os.Root own message, which says only that a path escaped and names neither the directory nor the remedy (ADR-015). · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · covers:a refusal names the base and says it is a symlink
- 2026-09-08 · fff5b94* · mutant killed · exit 1 · `internal/state/prune.go` · the marker is trimmed wider than Dir writes it, so a checkout whose directory name ends in a space reads back as a different path and a LIVE checkout is classified dead. · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · covers:a marker is read back exactly as Dir wrote it
- 2026-09-08 · fff5b94* · mutant killed · exit 1 · `internal/state/prune.go` · every stat error is read as gone, so a denied parent, an unmounted volume or a timed-out mount loses its state on an answer about the LOOKUP. The obvious spelling return true leaves errors and fs unused and does not build. · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · covers:only ErrNotExist means the checkout is gone
- 2026-09-08 · fff5b94* · mutant killed · exit 1 · `internal/state/prune.go` · the ancestor-resolved spelling is dropped, leaving two candidates that are IDENTICAL once the leaf is gone, so the self guard stops matching for a checkout reached through a symlinked component and the run prunes its own state. Deleting the block outright leaves real unused and does not build. · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · covers:the self guard survives the live-to-missing transition
- 2026-09-08 · fff5b94* · mutant killed · exit 1 · `internal/state/prune.go` · the freshness lookup is forced true, so an entry that vanished between the walk and the prune is REPORTED as removed — os.Root RemoveAll succeeds silently on a name that is not there, and a delete that says it removed what it did not is ADR-008 broken. · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · covers:a removal is reported only when this walk saw the entry
- 2026-09-08 · fff5b94* · mutant killed · exit 1 · `internal/state/prune.go` · the current answer loses its veto, so a checkout restored between the caller walk and the prune has its live state deleted on evidence that has expired. · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · covers:the current answer vetoes the caller's stale one
- 2026-09-08 · fff5b94* · mutant killed · exit 1 · `internal/state/prune.go` · symlinked entries are silently omitted from the walk again, so they are skipped but never REPORTED, and ADR-034:132 promises reported and skipped. An entry nothing reports cannot be told from one nothing looked at. · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · covers:a symlinked entry is reported, not silently dropped

## Verification Log

- 2026-09-08 · c60d7d2* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · ms:26088
- 2026-09-08 · c60d7d2* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · ms:33493
- 2026-09-08 · c60d7d2* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · ms:25625
- 2026-09-08 · c60d7d2* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · ms:27293
- 2026-09-08 · c60d7d2* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · ms:23342
- 2026-09-08 · c60d7d2* · exit 0 · `set -o pipefail …` · acceptance-sha256:2b8b9ef51d7aca7e9b57d8a30ee233230425f1a060ecc84bcccd966317df6be9 · ms:23289
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:26437
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:23540
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:22958
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:23086
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:20821
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:23029
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:23302
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:24434
- 2026-09-08 · fff5b94* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:22908
- 2026-09-08 · 69e6a1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:154265c939657a0d301500eefdab2cd0b31a2693973c71a91a36c889c552e3de · ms:23482

## Invariants

- `mrw seen --prune` keeps its exit codes, its output shape and the state directory as the FIRST
  line of `mrw seen`. §71 asserts all three and must keep passing unchanged.
- A dry run still performs no filesystem write and returns the identical slice a real run would.
- `Dir()`, `Path()`, `LegacyPath()`, `Migrate()`, `stateHome()`, `absReal()` and `key()` keep their
  behaviour. The one edit to `state.go` narrows a comparison that was wider than the value it
  compares against; it changes no key and no path.
- ⚠ **No state directory on any existing machine changes its key.** `absReal` is untouched, so a
  root that resolves today resolves to the same string tomorrow. The ancestor-resolved spelling is
  an ADDITIONAL candidate consulted only when matching self, never a new way to compute a key.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` and `internal/check` stay
  byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- ⚠ **A guard that cannot fail.** M1 survived its first form: the test asserted only that an error
  came back, and `os.Root` produces one on its own. The guard was scored by a test that would have
  passed without it. Both directions are now taken deliberately — the symlink check kept because it
  buys a message, with the message asserted; `validName` deleted because it bought nothing.
- ⚠ **THE FIRST FIX FOR THE HIGH WAS WRONG, AND ITS OWN COMMENT SAID SO CONFIDENTLY.** It ordered
  `Lstat("mrw")` → reject a symlink → `OpenRoot("mrw")`, and claimed "the thing that was checked is
  the thing that was opened". That is false: they are two operations on a NAME. The second Codex
  review said so and it reproduces — `os.Root` CONFINES to its root but FOLLOWS a symlink that stays
  inside it, so a relative `mrw -> other` under the same state home opens and enumerates `other`.
  The escaping-symlink fixture could never show this, because `os.Root` rejects that one on its own.
  The order is now open-then-verify-by-identity, and
  `TestABaseThatIsARelativeSymlinkInsideTheStateHomeIsRefused` is the case the first round missed.
- ⚠ **DECLARED UNCOVERED: removing by NAME rather than by PATH.** No mutant binds it and none is
  claimed. Swapping `base.RemoveAll(cur.Name)` back to `os.RemoveAll(cur.Dir)` leaves every test and
  §72 green, because the base refusal above it already turns those fixtures away. What the handle
  buys is a window a fixture cannot force: the base is opened once and used later, and a component
  swapped in between redirects a path but cannot redirect an open handle. Kept as the correct
  construction and stated here as unproven, rather than given a mutant chosen to satisfy a counter.
- ⚠ **`TestARootThatCannotBeStattedIsKept` skips rather than fails on platforms where mode 0 does
  not deny traversal, and as root.** A skip is not a pass; the property is unproven there, and it is
  said here rather than left for a reader to discover from a green run.
- The ancestor walk in `resolveThroughAncestors` terminates at the volume root and reports false, so
  a path where nothing resolves costs one pass and adds no candidate.

## Stop Condition

Stop and ask if closing these findings would require changing `absReal`, or any other function that
computes a state key. That would orphan every state directory already on disk under a symlinked
parent — a migration, not a fix, and a different decision from the one this record took.

## Acceptance

```bash
set -o pipefail
go test ./internal/state/ ./cmd/mrw/ -count=1 \
    -run 'TestABaseThatIsASymlinkIsRefusedRatherThanFollowed|TestABaseThatIsARelativeSymlinkInsideTheStateHomeIsRefused|TestAMarkerIsReadBackExactlyAsDirWroteIt|TestARootThatCannotBeStattedIsKept|TestAnEntryThatVanishedBetweenTheWalkAndTheRemovalIsNotReported|TestAnEntryWhoseCheckoutCameBackIsNotRemoved|TestASymlinkedEntryIsReportedRatherThanSilentlyDropped|TestSelfSurvivesWhenItsCheckoutIsGoneUnderASymlinkedPath|TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned|TestADryRunPruneReportsTheSameAndRemovesNothing|TestAnUnidentifiableEntryIsKeptAndReported|TestThePruneStaysInsideTheStateBase' \
  && grep -q '^# 72\. ' scripts/contract.sh \
  && grep -q 'Dead bool' internal/state/prune.go \
  && grep -q 'func openBase' internal/state/prune.go \
  && grep -q 'func resolveThroughAncestors' internal/state/prune.go \
  && ! grep -q 'func validName' internal/state/prune.go \
  && ! grep -q 'TrimSpace(string(b))' internal/state/prune.go \
  && ! grep -q 'TrimSpace(string(existing))' internal/state/state.go \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr034-t5c.out \
  && grep -q '^contract holds$' /tmp/adr034-t5c.out \
  && go test ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./... \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Out of Scope

- Making `absReal` itself resolve through missing ancestors — it would make `Dir` and every later
  lookup agree by construction rather than by three candidates, but it changes the KEY for any root
  under a symlinked parent that does not yet exist, orphaning state directories already on disk,
  which is a migration and the Stop Condition above names it (deferred: `docs/adr/BACKLOG.md`)
- An automatic reaper (permanent: boundary: ADR-004's reason stands — an absent root is also an unmounted volume, and only an operator can tell them apart).
- Removing state whose marker mrw cannot read (permanent: boundary: unknown provenance is the one mistake here that looking again cannot undo).
- Proving the permission-error case on Windows (deferred: `docs/adr/BACKLOG.md`).
