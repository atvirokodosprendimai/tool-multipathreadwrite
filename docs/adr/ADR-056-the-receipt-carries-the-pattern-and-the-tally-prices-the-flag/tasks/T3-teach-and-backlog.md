# Task ADR-056-T3: teach both; correct the pre-registration's data source

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** README, AGENTS.md, BACKLOG correction
**Consumes:** T1, T2
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `receipt has pattern`, `stats prices the flag`, `backlog reads stats`

## Goal

Name `pattern` on the receipts and the pricing block in README and AGENTS.md; in BACKLOG, replace the impossible "replay" sentence with the `mrw stats --json` source, criterion unchanged.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | Safety bullet gains the two sentences. |
| `AGENTS.md` | edit | Rules bullet and the `mrw stats` entry. |
| `docs/adr/BACKLOG.md` | edit | Pre-registration data source; ADR-055 BACKLOG row for the MCP pattern closed with a receipt. |
| `cmd/mrw/main.go` | edit | `write --help` names `pattern` in the JSON receipt. |
| `cmd/mrw/writehelp_test.go` | edit | Red: help names it. |

## Ordered Steps

1. [S1] `TestWriteHelpNamesThePatternField` — RED. [proof: mutation]
2. [S2] Teach in the four files. S1 GREEN. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v -run 'TestWriteHelpNamesThePatternField' 2>&1 | tee /tmp/adr056-t3.out \
  && grep -q '^--- PASS: TestWriteHelpNamesThePatternField' /tmp/adr056-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr056-t3.out \
  && grep -q 'mrw stats --json' docs/adr/BACKLOG.md \
  && grep -q 'pricing' AGENTS.md README.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWriteHelpNamesThePatternField` | `cmd/mrw/writehelp_test.go` | `--help` says the JSON receipt carries `pattern` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `write --help` |
| 3 — the caller can discover it | README, AGENTS.md |
| 4 — it is used | the BACKLOG campaign reads the block |

## Mutation Log

_(tool-written)_

## Invariants

- The pre-registration's CRITERION is not edited; only its data source.

## Risks

| Risk | Mitigation |
|------|------------|
| Teaching drifts from the binary | the help test |

## Stop Condition

None.

## Out of Scope

- The mrw skill update happens at release, outside the record.

## Verification Log

_(tool-written)_
