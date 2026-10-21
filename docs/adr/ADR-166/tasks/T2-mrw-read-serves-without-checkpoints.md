# Task ADR-166-T2: mrw_read serves without checkpoints, and says so

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `mrw_read`'s handling of a state-directory failure from `hold`, at both its sites
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the answer without markers`, `the no-checkpoint line`, `an ack cannot license it`

## Goal

Decision 2: `mrw_read` with an unwritable state directory answers with the served text, no checkpoint markers, and a top line saying no checkpoint was held, instead of JSON-RPC -32603.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | both `hold` call sites: a `statehint.About` failure returns the unmarked answer with the line |
| `internal/mcp/readonly166_test.go` | add | the test |
| `internal/mcp/statehint164_test.go` | edit | ADR-164 T3's two content rows ("a read holds its checkpoints", "a paged read holds its checkpoints") now expect an answer without checkpoints; the quiet, ack and plan rows keep their error shapes (Decision 3) |
| `scripts/contract.sh` | edit | §285 |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestMrwReadWithAnUnwritableStateServesWithoutCheckpoints (`readTool` with `XDG_STATE_HOME` under a read-only parent: no rpc error, not isError, `content[0]` begins `-- no checkpoint was held` and holds the served line, and no `-- ck ` marker; a file over the ceiling answers its first page the same way, unmarked, its `next_read` kept; `quiet`, `ack` and a `sub` plan still answer the error). [proof: acceptance]
2. [S2] Both call sites. [proof: mutation]
3. [S3] Contract §285 (the pair: with a writable state the answer carries checkpoints). [proof: human: the executor runs ./scripts/contract.sh once and reads the §285 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestMrwReadWithAnUnwritableStateServesWithoutCheckpoints$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestMrwReadWithAnUnwritableStateServesWithoutCheckpoints \(' "$out" \
  && grep -q '^# 285\. ' scripts/contract.sh
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMrwReadWithAnUnwritableStateServesWithoutCheckpoints` | `internal/mcp/readonly166_test.go` | a whole answer and a page, no markers, the line | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two branches |
| 2 — something selects it | every `mrw_read` passes `hold`; contract §285 drives `mrw mcp` |
| 3 — the caller can discover it | the line at the top of `content[0]` |
| 4 — it is used | sandboxed MCP hosts; nothing measures it |

## Invariants

- No checkpoint marker appears, so no ack can license a line.
- Any other `hold` failure is answered as today.

## Risks

- The page path builds its result before holding: the unmarked page must be the same page, `next_read` included; the test reads a file over the ceiling.

## Out of Scope

- `mrw_write` in a read-only state (permanent: boundary: a write needs the ledger, ADR-002)

## Stop Condition

Stop if `mrw_read` touches the state directory before `hold` on some path (a refusal earlier than the hold): add that site to the record first.

## Mutation Log
(empty until execute)
- 2026-10-21 · 13132ca1* · mutant killed · exit 1 · `internal/mcp/tools.go` · a whole content read answers the hold error again · acceptance-sha256:9f1fd9d571a13d0286d7113a7390ab16c4dc3eeaa39bedb61c8309065395d886
- 2026-10-21 · 13132ca1* · mutant killed · exit 1 · `internal/mcp/tools.go` · a paged content read answers the hold error again · acceptance-sha256:9f1fd9d571a13d0286d7113a7390ab16c4dc3eeaa39bedb61c8309065395d886

## Verification Log
(empty until execute)
- 2026-10-21 · 13132ca1* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f1fd9d571a13d0286d7113a7390ab16c4dc3eeaa39bedb61c8309065395d886 · ms:3427
- 2026-10-21 · 13132ca1* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f1fd9d571a13d0286d7113a7390ab16c4dc3eeaa39bedb61c8309065395d886 · ms:3039
