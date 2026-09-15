# Task ADR-058-T2: contract §107–§109; BACKLOG shipped; teach

**Depends-on:** T1
**Covers:** none — T1 holds the spec IDs; this task proves the binary and the reservation
**Estimated scope:** S
**Owner:** unassigned
**Produces:** §107–§109; BACKLOG ADR-058 shipped
**Consumes:** `read.AstGrep` + CLI `--ast-grep` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `§107 exists`, `§108 exists`, `§109 exists`, `BACKLOG no longer says planned not executed`

## Goal

The built binary refuses a missing `ast-grep`, two finders, and zero hits the way T1's tests do. BACKLOG's reserved row is marked shipped.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | **§107–§109**. |
| `docs/adr/BACKLOG.md` | edit | Reserved ADR-058 → shipped. |
| `README.md` | edit | Flag table and MCP mapping. |
| `AGENTS.md` | edit | `ast_grep` mapping. |

## Ordered Steps

1. [S1] Confirm `# 107.` `# 108.` `# 109.` exist in `scripts/contract.sh`. [proof: acceptance]
2. [S2] Mark BACKLOG ADR-058 shipped. [proof: acceptance]
3. [S3] Teach README/AGENTS. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 107\. ' scripts/contract.sh \
  && grep -q '^# 108\. ' scripts/contract.sh \
  && grep -q '^# 109\. ' scripts/contract.sh \
  && grep -q 'Shipped 2026-09-15 as ADR-058' docs/adr/BACKLOG.md \
  && grep -q -- '--ast-grep' README.md \
  && grep -q 'ast_grep' AGENTS.md \
  && [ -z "$(gofmt -l .)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMissingAstGrepIsUsageAndNamesTheBinary` | `cmd/mrw/astgrep_test.go` | the binary T1 wired; §107 is the contract twin | — | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §107–§109 |
| 2 — something selects it | contract.sh drives `$MRW` |
| 3 — the caller can discover it | README / AGENTS |
| 4 — it is used | nothing measures this yet |

## Mutation Log
- 2026-09-15 · 9ff8e35* · mutant killed · exit 1 · `AGENTS.md` · MCP mapping must name ast_grep or the surfaces disagree · acceptance-sha256:bf44d3e838c8cf9ac448ee10c4d069a74e522491862071ecf410f01ed8411043 · covers:BACKLOG no longer says planned not executed

## Invariants

- `--grep` stays regex.
- Linux-only contract (POSIX `#!/bin/sh` fake).

## Risks

- none

## Stop Condition

Stop if BACKLOG still says "planned, not executed" after this task.

## Out of Scope

- Desktop probe (ADR-023).

## Verification Log
- 2026-09-15 · 9ff8e35* · exit 0 · `set -o pipefail …` · acceptance-sha256:bf44d3e838c8cf9ac448ee10c4d069a74e522491862071ecf410f01ed8411043 · ms:48
- 2026-09-15 · 9ff8e35* · exit 0 · `set -o pipefail …` · acceptance-sha256:bf44d3e838c8cf9ac448ee10c4d069a74e522491862071ecf410f01ed8411043 · ms:57
- 2026-09-15 · 9ff8e35* · exit 1 · `set -o pipefail …` · acceptance-sha256:bf44d3e838c8cf9ac448ee10c4d069a74e522491862071ecf410f01ed8411043 · ms:32 · test-lock-sha256:6d2e5e5270e896e5b76c9cb19c8b87e489d500cf2ce582827a9336b84d4878fe · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcF90ZXN0LmdvCVRlc3RBUHJlc2VudEFzdEdyZXBXaXRoWmVyb0hpdHNJc05vdFRoZU1pc3NpbmdCaW5hcnlQYXRoCWYzYWQzY2Y1ZmEwNTQzMDEyZjRhYjlkNjA1MDhmZDAzODJmZDc0NTQyMDU3OTJmY2FlNTdlYTY3NWQ1NDQ5ZjkKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0QXN0R3JlcE9ic2VydmVzT25seVNlcnZlZExpbmVzCWE2NWNiM2Q1MmE5ZjgyNmE3NjJmNmFmZmM2ODFhZTNmNTQyNDAxNDllYTNjN2VjNWUxNGMyM2NkN2RjZmVlYTkKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0QXN0R3JlcFNlcnZlc1Jhbmdlc1Rocm91Z2hSZWFkCTNlMzMwNjcyNjQwMDJkM2EyYzEyN2M2MTk0ZDgyNDU2MjYzMzU4MWQ4NzFmNmUzODlkMzhlYTJhZTVkMDlmMTUKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0R3JlcEFuZEFzdEdyZXBUb2dldGhlckFyZVVzYWdlCTU1MjI0MWZiYmMwNzlhODQ1YWMzNzZiNzRhOTIyM2JmZThkMWExZGQwZWYzNDE2YTc1MTRjMzBjOTIxZWVlYzgKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0TWlzc2luZ0FzdEdyZXBJc1VzYWdlQW5kTmFtZXNUaGVCaW5hcnkJM2M2ZTUwZjQ3YmQ1NzBkMTE4ZWM1MGY1M2ZmZTM0ZmFlMWMwYTVjOGU3YmMzZjNmZTNlZDUxNmM1MzAwYTAyYQ
  ```
  ```
