# Task ADR-196-T7: a plan or list refusal names the reason that held

**Depends-on:** T6
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `rooted.OwnStateRefusal` (replaces `rooted.InStateOrLinked`), `errNoIdentity`; `TestARefusalForOwnStateSaysWhichReasonHeld`, `TestAPathThatNamesNothingPastItsPrefixSaysSo`
**Consumes:** `judgeOn` (T6); ADR-134's `stateLinks.refusal`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the own-state refusal names its reason`, `nothing past the prefix is named as nothing`

## Goal

Decision 10, from the Windows field test of v1.86.4 (2026-11-11, four Windows 11 peers). Two refusals were right but gave the wrong reason:
- `write C:\$MFT` said "inside mrw's own state directory, or is a hard link to a file in it". Neither was true. Windows would not open `$MFT` to read its identity, so ADR-134's check could not finish and refused (Decision 4, kept). `stateLinks.refusal` reported that as "has more than one name", and `InStateOrLinked` then reduced every reason to one bool and one sentence.
- `\\.\C:\..` said `"" names a Windows object`, because the `..` climbed above the volume and left nothing to name.

Now an identity that cannot be read is named as such, `OwnStateRefusal` returns the reason it found and the CLI prints it, and a path that names nothing past its prefix says so.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/hardlink.go` | edit | `errNoIdentity` for an identity that cannot be read; `OwnStateRefusal` returns the reason in place of `InStateOrLinked`'s bool |
| `cmd/mrw/main.go` | edit | both open sites print the reason `OwnStateRefusal` returns |
| `cmd/mrw/keep196.go` | edit | `judgeOn` says a path names nothing past its prefix |
| `internal/rooted/refusal196_test.go`, `cmd/mrw/keep196t7_test.go` | add | both refusals, on every platform |

## Ordered Steps

1. [S1] Write the two tests:
   - `TestARefusalForOwnStateSaysWhichReasonHeld`: an identity that cannot be read is named "cannot read the file identity", not "more than one name", and is still `ErrOwnState`. A file in the state directory is named as inside, not as a link. A plain file is not refused.
   - `TestAPathThatNamesNothingPastItsPrefixSaysSo`: `\\.\C:\..`, `\\.\Volume{…}\..` and `\\.\` say "names nothing past its prefix", with no `""`. An object is still named.

   See both RED.
2. [S2] Change `refusal`, add `OwnStateRefusal`, and change both CLI sites and `judgeOn`. Mutants, each killed: the identity failure back to `errNotComparable`; `judgeOn`'s empty case back to the object message. [proof: mutation]
3. [S3] `GOOS=windows go vet ./...`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/rooted/ ./cmd/mrw/ -count=1 -v -run '^(TestARefusalForOwnStateSaysWhichReasonHeld|TestAPathThatNamesNothingPastItsPrefixSaysSo|TestAPlanOrListNamingAWindowsObjectIsRefused|TestAHardLinkToMrwsStateIsRefused)$' 2>&1) \
  && grep -qE '^--- PASS: TestARefusalForOwnStateSaysWhichReasonHeld \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAPathThatNamesNothingPastItsPrefixSaysSo \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAHardLinkToMrwsStateIsRefused \(' <<<"$out" \
  && [ "$(grep -c 'OwnStateRefusal(' cmd/mrw/main.go)" -eq 2 ] \
  && GOOS=windows go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARefusalForOwnStateSaysWhichReasonHeld` | `internal/rooted/refusal196_test.go` | the own-state refusal names the reason that held | none | S1, S2 |
| `TestAPathThatNamesNothingPastItsPrefixSaysSo` | `cmd/mrw/keep196t7_test.go` | a path that names nothing past its prefix says so | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `OwnStateRefusal`, `errNoIdentity` |
| 2 — something selects it | both CLI open sites; the fence counts `OwnStateRefusal(` in main.go |
| 3 — the caller can discover it | the refusal names its reason |
| 4 — it is used | Windows peers met both messages on 2026-11-11; no telemetry (ADR-009) |

## Mutation Log
- 2026-11-11 · 322844cf* · mutant killed · exit 1 · `internal/rooted/hardlink.go` · an identity that cannot be read named as a second name again · acceptance-sha256:fc839a9e57b5a137a7f836e1232b615286a8513e127fe0515aea10a982f35227 · covers:the own-state refusal names its reason
- 2026-11-11 · 322844cf* · mutant killed · exit 1 · `cmd/mrw/keep196.go` · nothing past the prefix named as the object "" again · acceptance-sha256:fc839a9e57b5a137a7f836e1232b615286a8513e127fe0515aea10a982f35227 · covers:nothing past the prefix is named as nothing

## Verification Log
- 2026-11-11 · 322844cf* · exit 1 · `set -o pipefail …` · acceptance-sha256:fc839a9e57b5a137a7f836e1232b615286a8513e127fe0515aea10a982f35227 · ms:1033 · test-lock-sha256:1c7562b3e1a84bcc606a2859b3b2f51bfb3afdec12c83e0fb656e6208c6da7d3 · test-lock-file:docs/adr/locks/1c7562b3e1a84bcc606a2859b3b2f51bfb3afdec12c83e0fb656e6208c6da7d3.lock
  ```
  ```
- 2026-11-11 · 322844cf* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:fc839a9e57b5a137a7f836e1232b615286a8513e127fe0515aea10a982f35227 · ms:0 · test-lock-sha256:da6bf84dfddf40d9bb1ebf88528657bce12ea8abe09f3484a8263bac4865fc4a · test-lock-file:docs/adr/locks/da6bf84dfddf40d9bb1ebf88528657bce12ea8abe09f3484a8263bac4865fc4a.lock · test-lock-kind:replace
- 2026-11-11 · 322844cf* · exit 0 · `set -o pipefail …` · acceptance-sha256:fc839a9e57b5a137a7f836e1232b615286a8513e127fe0515aea10a982f35227 · ms:793
- 2026-11-11 · 322844cf* · exit 0 · `set -o pipefail …` · acceptance-sha256:fc839a9e57b5a137a7f836e1232b615286a8513e127fe0515aea10a982f35227 · ms:2705
- 2026-11-11 · 322844cf* · exit 0 · `set -o pipefail …` · acceptance-sha256:fc839a9e57b5a137a7f836e1232b615286a8513e127fe0515aea10a982f35227 · ms:2639

## Invariants

- What is refused is unchanged: ADR-134 Decision 4 still refuses a comparison that cannot finish, and every T3–T6 refusal stands.
- Every own-state refusal is still `ErrOwnState` and still says "own state directory" (contract §236, `ownstate157_test.go`).

## Risks

- None new: the change is in wording and in how a reason reaches the CLI.

## Stop Condition

The fence exits 0.

## Out of Scope

- Reading `$MFT` (permanent: boundary: Windows refuses its identity, and ADR-134 refuses what it cannot compare)
