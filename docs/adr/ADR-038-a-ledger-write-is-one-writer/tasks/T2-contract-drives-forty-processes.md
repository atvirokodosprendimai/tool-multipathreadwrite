# Task ADR-038-T2: Contract §76 drives 40 processes through `$MRW`

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** §76 (T2)
**Consumes:** `Record` holds the lock (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `forty concurrent reads keeping forty`, `writability still following the ledger`

## Goal

Prove the built binary, not the package, keeps every entry when 40 `mrw read` processes race, and
keep §24's safety assertion so a build that refuses every write cannot pass.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 76.` after the current highest (`75.`). 40 background `$MRW read` of distinct files; `kept` must equal 40. Then the existing §24 writes still assert `applied == kept`. Amend §24's comment that loss is accepted — do not delete the safety half. Reserve 76 with `grep -oE '^# [0-9]+\. ' scripts/contract.sh \| sort -k2,2n \| tail -1`. |

## Ordered Steps

1. [S1] Confirm `# 76.` matches nothing. Write the row against a binary that does **not** yet lock
   (or with the lock temporarily removed) and confirm it is RED: `kept < 40` on a machine that
   reproduces, which this host did (40 kept 5). [proof: mutation]
2. [S2] Rebuild with T1's lock and confirm `kept == 40` and the §24 safety writes still match.
   Pair: 40 concurrent reads keep 40; a subsequent write to each file applies exactly 40 times, not
   0 and not a silent extra. [proof: mutation]
3. [S3] Amend §24's prose so it no longer says ADR-002 accepts lost entries. Leave the
   writability-follows-the-ledger assertion. [proof: acceptance]
4. [S4] Run `./scripts/contract.sh` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 76\. ' scripts/contract.sh \
  && grep -q 'kept == 40\|kept.*eq.*N\|want "\$N" "\$kept"' scripts/contract.sh \
  && ./scripts/contract.sh
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§76` | `scripts/contract.sh` | 40 concurrent `$MRW read` keep 40 entries, and writability follows the ledger | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `# 76.` |
| 2 — something selects it | `$MRW` is the binary `contract.sh` builds; a package-only lock that `cmd/mrw` does not call fails this row |
| 3 — the caller can discover it | T3 |
| 4 — it is used | the row is the measurement |

## Mutation Log

- 2026-09-09 · 0737f60* · mutant killed · exit 1 · `internal/seen/lock_unix.go` · unlocked Record loses entries across processes; §76 must see kept < 40 · acceptance-sha256:6e6b1e1228929a515d2146936d080e7a6014b88ca8c12459ff6557366216a198

## Invariants

- §24's safety property remains: files still in the ledger are exactly the ones that can be
  written. Fail-open is 40 applied with 5 kept; refuse-everything is 0; only honouring survivors
  matches.
- Linux-only, as `contract.sh` already is. The Windows lock is T1's compile-time proof.
- Do not read an exit code through a pipe.

## Risks

- ⚠ Rewriting §24 inside a documentation edit is how a gate stops asserting what it was written
  for. This task owns the rewrite and keeps the safety half.
- A machine that never reproduced the race would have skipped §24. §76 must not skip: `kept == 40`
  is required even on a serialising host.

## Stop Condition

Stop and ask if §76 cannot be made red on this host even with the lock removed — then the row
cannot prove the lock and must not ship.

## Out of Scope

- Teaching — that is T3's job

## Verification Log
- 2026-09-09 · 0737f60* · exit 0 · `set -o pipefail …` · acceptance-sha256:6e6b1e1228929a515d2146936d080e7a6014b88ca8c12459ff6557366216a198 · ms:21259
