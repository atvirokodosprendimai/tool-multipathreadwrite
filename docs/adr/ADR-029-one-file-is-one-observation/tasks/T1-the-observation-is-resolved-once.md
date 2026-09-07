# Task ADR-029-T1: The observation is resolved once and `covered()` consumes it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one lookup hoisted, one deleted, and the two-sided fixture that pins both)
**Owner:** Zy
**Produces:** the per-line ledger check consuming the alias-resolved observation (T2)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an alias-spelled hunk being refused for lines the caller was never served`, `a whole read still licensing an alias-spelled write`

## Goal

Make one file one observation to the per-line ledger, whatever the plan calls it, without refusing
the alias spellings issue #47 exists to accept.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | The ledger lookup and its `sameFileEntry` recovery are hoisted above the file-level block; `covered()` closes over that value and the second exact-key lookup at `:553` is deleted. Two lookups of one fact is what let them disagree. |
| `internal/adversarial/ledger_test.go` | edit | `TestAnAliasSpellingIsTheSameFileToThePerLineLedger` lives with the other ledger-boundary tests, beside ADR-028's, because what it asserts is an ADR-002/005 property rather than an apply mechanic. |

## Ordered Steps

1. [S1] Write `TestAnAliasSpellingIsTheSameFileToThePerLineLedger` and confirm it is RED. ⚠ **Both halves in the one test.** Half A: read `real.txt:1`, write `link.txt 4 replace`, and assert the refusal AND that the file is unchanged. Half B: read `real.txt` WHOLE, write `link.txt 4 replace`, and assert it APPLIES — without it, "refuse every alias" passes and undoes issue #47. ⚠ Skip on `os.Symlink` returning an error rather than asserting the platform: Windows CI may refuse to create one, and a test that fails there fails for a reason unrelated to this defect.
2. [S2] Hoist the lookup and the `sameFileEntry` recovery above the file-level block, and delete the second lookup. [proof: mutation]
3. [S3] Add the case-only half, guarded by a runtime probe: create `real.txt`, `os.Stat("REAL.txt")`, and skip unless the filesystem answers. It runs on Windows CI and on a developer's macOS; it cannot run on Linux, where the two names are two files. [proof: mutation]
4. [S4] Confirm ADR-028's `TestAFailedAnchorDoesNotReadBackAnUnservedLine` still passes, and that the anchor leak through an alias is now closed too — the same fixture with an `anchor=` and no served line must print no file content. [proof: acceptance]
5. [S5] Confirm a `create` into a path with no ledger entry is unaffected, since the hoisted resolution now runs when the file does not exist. [proof: acceptance]
6. [S6] Run the package, the adversarial package, `go test -race ./...`, `gofmt` and `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/adversarial/ -count=1 -v \
  -run 'TestAnAliasSpellingIsTheSameFileToThePerLineLedger' 2>&1 | tee /tmp/adr029-t1.out \
  && grep -q '^--- PASS: TestAnAliasSpellingIsTheSameFileToThePerLineLedger' /tmp/adr029-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr029-t1.out \
  && go test ./internal/adversarial/ -count=1 -v \
       -run 'TestAFailedAnchorDoesNotReadBackAnUnservedLine|TestARangedReadLicensesOnlyTheLinesItServed' 2>&1 | tee /tmp/adr029-t1n.out \
  && grep -q '^--- PASS: TestAFailedAnchorDoesNotReadBackAnUnservedLine' /tmp/adr029-t1n.out \
  && grep -q '^--- PASS: TestARangedReadLicensesOnlyTheLinesItServed' /tmp/adr029-t1n.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr029-t1n.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnAliasSpellingIsTheSameFileToThePerLineLedger` | `internal/adversarial/ledger_test.go` | A partial read refuses an alias-spelled write to lines never served and leaves the file unchanged; a WHOLE read still licenses one; the case-only variant does both where the filesystem is case-insensitive | — | S1, S2, S3 |
| `TestAFailedAnchorDoesNotReadBackAnUnservedLine` | `internal/adversarial/ledger_test.go` | Unchanged from ADR-028, and now holds for the alias spelling too | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAnAliasSpellingIsTheSameFileToThePerLineLedger` |
| 2 — something selects it | The resolution runs for every file a plan names; the S2 mutation restores the second lookup and the fence goes red on a write that applied |
| 3 — the caller can discover it | The refusal is the ledger's existing message, naming the file, the address and the served spans |
| 4 — it is used | Contract §67 (T2) drives the symlink half through the built binary; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · cd259f2* · mutant killed · exit 1 · `internal/apply/apply.go` · the per-line gate goes back to its own exact-key lookup, so an alias spelling misses, covered() reads the miss as a caller who read nothing, and a write to lines never served applies at exit 0 — the defect this record exists to close · acceptance-sha256:bc4adbfc3ec32062baab9dcacfde2e7bc661e5b76a3bfd8b8d5abaa401be2a0d · covers:an alias-spelled hunk being refused for lines the caller was never served
- 2026-09-07 · cd259f2* · mutant killed · exit 1 · `internal/apply/apply.go` · alias recovery is dropped, so a file read WHOLE under one spelling is refused as unread under another — issue #47 undone, which is what a one-sided fixture asserting only the refusal would call success · acceptance-sha256:bc4adbfc3ec32062baab9dcacfde2e7bc661e5b76a3bfd8b8d5abaa401be2a0d · covers:a whole read still licensing an alias-spelled write

## Invariants
- Issue #47's promise holds: a file read under one spelling and written under another is NOT refused as unread, so long as the lines were served. Half B of the test is what pins it, and a mutant that refuses every alias must go red there.
- ADR-028's ordering is untouched. This record supplies the value `covered()` consults; it does not move any guard.
- `os.SameFile` stays the identity oracle. No case folding, no filesystem sniffing — that is what #47 rejected.
- `internal/read`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- ⚠ A one-sided fixture is green against "refuse every alias", which is a regression of issue #47 dressed as a fix. Both halves live in one test for that reason, and the S2 mutant is what proves they are distinguished.
- The hoisted resolution now runs for a file that does not exist, where it did not before. `sameFileEntry` returns false when `os.Stat` fails, and S5 asserts the `create` path is unaffected.

## Stop Condition

Stop and ask if closing this requires `internal/apply` to resolve symlinks itself rather than asking
`os.SameFile` — that is a different identity model, it contradicts what issue #47 chose, and it would
be a separate decision.

## Out of Scope

- The contract row — that is T2's job
- The second ledger ENTRY left behind by a successful alias-spelled write (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-07 · cd259f2* · exit 0 · `set -o pipefail …` · acceptance-sha256:bc4adbfc3ec32062baab9dcacfde2e7bc661e5b76a3bfd8b8d5abaa401be2a0d · ms:14015
- 2026-09-07 · cd259f2* · exit 0 · `set -o pipefail …` · acceptance-sha256:bc4adbfc3ec32062baab9dcacfde2e7bc661e5b76a3bfd8b8d5abaa401be2a0d · ms:15562
