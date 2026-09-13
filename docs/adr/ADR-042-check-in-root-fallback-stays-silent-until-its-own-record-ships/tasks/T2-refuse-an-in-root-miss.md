# Task ADR-042-T2: Refuse an in-root miss

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the in-root miss refusal and contract §86
**Consumes:** `check.confine` (PR #15, unchanged for out-of-root)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a missing in-root path is refused at exit 2`, `an existing package still scopes`, `prose and testdata still fall back`

## Goal

An in-root path that is not there is refused (exit 2), not answered as a whole-project PASS. Existing non-package directories still fall back. No new exit code.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `confine` refuses a missing in-root path the way it already refuses an unreadable one |
| `internal/check/check_test.go` | edit | **the ADR's Enforced-by** — a miss cannot PASS |
| `scripts/contract.sh` | edit | §15/§15c stop asserting typo-fallback; **§86** drives the built binary |
| `README.md` | edit | the check section still teaches that a typo falls back |
| `docs/adr/BACKLOG.md` | edit | the leftover is closed |

## Ordered Steps

1. [S1] Write `TestAnInRootMissIsRefusedNotASilentPass` and confirm it is RED: `Run` on `nosuchdir` / `chek.go` still returns OK on the whole-project command. [proof: acceptance]
2. [S2] Refuse a missing in-root path in `confine` before `command` runs. Use `os.IsNotExist` after `rooted.Resolve`, never `filepath.IsAbs`. Confirm S1 GREEN. [proof: mutation]
3. [S3] Invert the unit rows that still assert typo-fallback (`TestAnUnplaceableInRootScopeStillFallsBack`, `TestReadableAndMissingScopesAreUnchanged`). Leave prose/testdata fallback. [proof: acceptance]
4. [S4] Write §86 against a binary that still PASSes a miss and confirm it is RED, then rebuild and confirm it is GREEN. Pair: a real package still scopes (exit 0); a miss is exit 2 and prints no `exit_code`. [proof: mutation]
5. [S5] Teach README; receipt BACKLOG. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 86\. ' scripts/contract.sh \
  && go test ./internal/check/ -count=1 -v \
    -run 'TestAnInRootMissIsRefusedNotASilentPass' 2>&1 | tee /tmp/adr042-t2.out \
  && grep -q '^--- PASS: TestAnInRootMissIsRefusedNotASilentPass' /tmp/adr042-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr042-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/check/
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name and `# 86.`. **§86 rather than §85**: 85 is the highest on `main`.

⚠ **THE UNIT TEST ALONE CANNOT PROVE THIS.** `confine` can be correct and the binary still PASS a miss if a caller never reaches it. §86 is the row that binds, because it runs `$MRW`.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnInRootMissIsRefusedNotASilentPass` | `internal/check/check_test.go` | **the ADR's Enforced-by** — a missing in-root path is an error, nothing ran, an existing package still runs | — | S1, S2 |
| `§86` | `scripts/contract.sh` | Built binary: miss exits 2 and emits no result; a real package still scopes | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `Run` calls `confine` before `command`; §86 proves that through the real binary |
| 3 — the caller can discover it | the refusal names the path; README documents it |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry. The evidence for building it is the PR #15 inbox leftover |

## Mutation Log
<!-- filled during execution -->
- 2026-09-12 · 2f49847* · mutant killed · exit 1 · `internal/check/check.go` · drop the miss refusal so Run falls back and PASSes a typo · acceptance-sha256:7b6ac897c7f816333bce75436be211e2ea08d7f03a43f236adc19685712481fb · covers:a missing in-root path is refused at exit 2

## Invariants

- A missing in-root path is refused at exit 2. Nothing ran. No result document.
- An existing package still scopes.
- A directory of prose or `testdata` still falls back to the whole-project command.
- No new exit code. 4096 stays. ADR-019 A stands.
- `go.mod` declares exactly one requirement.

## Risks

- Refusing every unplaceable path, including prose/`testdata`. Mitigated: only `os.IsNotExist` is a miss.
- The guard is written and `command` still falls back for a miss that never reaches `confine`. Mitigated by §86 driving the built binary.

## Stop Condition

Stop if the work needs a new exit code, a `Result.Scoped` field, or a change to ADR-019 A / the 4096 handshake bound.

## Out of Scope

- Honouring a miss by reporting `Scoped` instead of refusing (permanent: boundary: M said refuse when both were allowed)
- Scope derivation for languages other than Go (deferred: docs/adr/BACKLOG.md)
- Playtrix (external: wing_playtrix)

## Verification Log
<!-- adr-verify appends here -->
- 2026-09-12 · 2f49847* · exit 0 · `set -o pipefail …` · acceptance-sha256:7b6ac897c7f816333bce75436be211e2ea08d7f03a43f236adc19685712481fb · ms:419
