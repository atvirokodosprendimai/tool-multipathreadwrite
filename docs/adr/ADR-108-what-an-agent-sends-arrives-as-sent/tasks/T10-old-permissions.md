# Task ADR-108-T10: permissions issued under the old checkpoint rules are discarded

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** permissions issued under the old checkpoint rules are discarded
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `permissions issued under the old checkpoint rules are discarded`

## Goal

T1 and T7 stop new checkpoints licensing lines nobody saw, but an upgrade kept what an older binary had already issued: a sparse MCP read's span `1-100`, acknowledged into the `#mrw-seen v2` ledger or still held in `pending.json`, licensed line 50 after the upgrade as before (third Codex review of #304). The ledger header becomes `#mrw-seen v3` — its own comment says the version is bumped whenever an older file's contents can no longer be trusted — so a v2 ledger is discarded as stale with the notice naming why; and the pending store moves to `pending-v3.json`, so an old hold can no longer be acknowledged. The cost is one re-read for every caller, once.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `header`, `StaleNotice` |
| `internal/seen/seen_test.go` | edit | the deliberate-bump fixture moves to v3 and asserts v2 is stale |
| `internal/mcp/ack.go` | edit | `pendingName` |
| `internal/mcp/state077_test.go` | edit | names the store through `pendingName` |
| `internal/mcp/legacy108_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test(s) `TestPermissionsIssuedUnderTheOldCheckpointRulesAreDiscarded`; confirm RED. [proof: mutation]
2. [S2] Bump the header and rename the store; contract §206 through `$MRW`. Mutants: the header back to v2; the store back to `pending.json`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestPermissionsIssuedUnderTheOldCheckpointRulesAreDiscarded' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestPermissionsIssuedUnderTheOldCheckpointRulesAreDiscarded \(' "$out" \
  && grep -q '^# 206\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestPermissionsIssuedUnderTheOldCheckpointRulesAreDiscarded` | `internal/mcp/legacy108_test.go` | a `#mrw-seen v2` ledger holding span 1-100 loads empty and reads stale; a `pending.json` hold of 1-100, acknowledged, records nothing | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every write, which loads the ledger; every acknowledgement, which loads the store |
| 3 — the caller can discover it | the CLI prints the stale-ledger notice, naming this cause |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the third Codex review of #304 |

## Mutation Log
- 2026-10-01 · bcc2818* · mutant killed · exit 1 · `internal/seen/seen.go` · the header back to v2: an old span 1-100 licenses line 50 again · acceptance-sha256:93c09dcfce46dc4770b3aff111bc99d4de497ce408bb6572576c643120b1789c
- 2026-10-01 · bcc2818* · mutant killed · exit 1 · `internal/mcp/ack.go` · the store back to pending.json: an old hold of 1-100 is acknowledged into a licence · acceptance-sha256:93c09dcfce46dc4770b3aff111bc99d4de497ce408bb6572576c643120b1789c

## Invariants

- A ledger this build writes loads back; the format of a record is unchanged.

## Risks

- Every caller re-reads once after upgrading; the stale-ledger notice says why.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · bcc2818* · exit 1 · `set -o pipefail …` · acceptance-sha256:93c09dcfce46dc4770b3aff111bc99d4de497ce408bb6572576c643120b1789c · ms:334 · test-lock-sha256:cbd7c4b2902ccf48472fb747ec0862c302a05ec414fd85a94895687688eaf76f · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9sZWdhY3kxMDhfdGVzdC5nbwlUZXN0UGVybWlzc2lvbnNJc3N1ZWRVbmRlclRoZU9sZENoZWNrcG9pbnRSdWxlc0FyZURpc2NhcmRlZAk5OTg1MzVkZjVmZjY5MTNkNGI4YTIxOGQ3ODEwMWRhZGRiNDFjZTJjYjMyNTRjY2I3NmNiYmIzZjZlN2Q2NWE5
  ```
  --- last 8 line(s) of stdout
  === RUN   TestPermissionsIssuedUnderTheOldCheckpointRulesAreDiscarded
      legacy108_test.go:37: a v2 ledger's span 1-100 still licenses line 50 (<nil>)
      legacy108_test.go:40: a v2 ledger was not reported stale: false <nil>
      legacy108_test.go:59: a hold of 1-100 from pending.json was acknowledged into a licence (<nil>)
  --- FAIL: TestPermissionsIssuedUnderTheOldCheckpointRulesAreDiscarded (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.096s
  FAIL
  ```
- 2026-10-01 · bcc2818* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:93c09dcfce46dc4770b3aff111bc99d4de497ce408bb6572576c643120b1789c · ms:0 · test-lock-sha256:3af79b86c8ea623569351ac0cbac26909e7c11e64330f173cc62f5e1e67ecdcd · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9sZWdhY3kxMDhfdGVzdC5nbwlUZXN0UGVybWlzc2lvbnNJc3N1ZWRVbmRlclRoZU9sZENoZWNrcG9pbnRSdWxlc0FyZURpc2NhcmRlZAlmZmE2MTM1MjViNmZiMGQxZWM5MjY1MTU0ODRkZGZlNDY4ODIyYmQ4OTg0YzJhNWJmODVlODE1MjllNzgzZDdj · test-lock-kind:replace
- 2026-10-01 · human-observed · Claude's session observed the relock: after red the two ledger assertions check that a.txt is present before Covers, since a missing entry's zero Observation reads as whole-file; the red run's failures were the old binary's real acceptance of both
- 2026-10-01 · bcc2818* · exit 0 · `set -o pipefail …` · acceptance-sha256:93c09dcfce46dc4770b3aff111bc99d4de497ce408bb6572576c643120b1789c · ms:430
- 2026-10-01 · bcc2818* · exit 0 · `set -o pipefail …` · acceptance-sha256:93c09dcfce46dc4770b3aff111bc99d4de497ce408bb6572576c643120b1789c · ms:372
