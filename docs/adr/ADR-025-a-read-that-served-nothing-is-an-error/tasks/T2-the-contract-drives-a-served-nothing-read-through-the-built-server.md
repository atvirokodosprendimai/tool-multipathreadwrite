# Task ADR-025-T2: The contract drives a served-nothing read through the built server

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** Zy
**Produces:** `scripts/contract.sh` §63
**Consumes:** the served-read return flags when `len(observed) == 0` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the flag on a read that served nothing through the built binary`, `the absence of the flag on a read that served something through the built binary`, `the per-path reasons surviving in content[0] of the flagged result`

## Goal

Pin the new shape against `$MRW` — the binary the contract builds — so the promise is proved of the
server that ships rather than of the function alone.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | Adds §63, driving the built binary over stdio. A unit test proves the function; it cannot prove the shipped server returns it, which is why §53 exists and why ADR-024 has §62. |

## Ordered Steps

1. [S1] Confirm §63 is unused — `grep -c '^# 63\. ' scripts/contract.sh` returns 0 — before writing the fence that greps for it, so the clause is red at authoring time. [proof: acceptance]
2. [S2] Write §63. It drives `mrw_read` three times through `m mcp` against a fixture root and asserts the PAIRING: `specs: ["nope_dir"]` comes back `isError: true` AND still carries `-- nope_dir:` with its reason in `content[0]`; `specs: ["a.go"]` comes back with the key ABSENT; and `specs: ["a.go", "nope_dir"]` comes back with the key ABSENT, which is ADR-024's member and the case a careless `problems > 0` would break. [proof: mutation]
3. [S3] Pass the JSON through files rather than argv, as §48 and §62 record — Linux caps one argument at 131,072 bytes and a served page exceeds it. [proof: acceptance]
4. [S4] Run the whole contract script and confirm exit 0 with §62 still passing beside the new row. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 63\. ' scripts/contract.sh \
  && ./scripts/contract.sh
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§63` | `scripts/contract.sh` | The built server flags a read that served nothing, keeps the per-path reason in its text, and does NOT flag a read that served something | — | S1, S2, S3 |
| `§62` | `scripts/contract.sh` | Unchanged: the built server still sends a page unflagged and still flags a refusal | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §63 in `scripts/contract.sh` |
| 2 — something selects it | The row drives `$MRW`, the binary built at the top of the script, so it fails if the condition is in the source but not in what ships; the S2 mutation reverts the return to `false` and the row must go red |
| 3 — the caller can discover it | `isError` is the MCP protocol's field; the row asserts what a host actually receives over stdio |
| 4 — it is used | `./scripts/contract.sh` is a declared gate in `CONTRIBUTING.md` and runs on every change; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the unconditional false in the SHIPPED binary the row drives; §63 must go red, or the row proves nothing about what the server actually sends · acceptance-sha256:b5bc10e98b19e16c662552383fec67106a9f7755d817b90c315dc7e22b026c0e · covers:the flag on a read that served nothing through the built binary
- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · makes the shipped server flag every served read; §63 must go red on the sibling and missed-range calls, or the row would pass against a server that simply flags everything · acceptance-sha256:b5bc10e98b19e16c662552383fec67106a9f7755d817b90c315dc7e22b026c0e · covers:the absence of the flag on a read that served something through the built binary
- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · empties the report the shipped server sends, so the flagged answer arrives naming nothing; §63 must go red because an answer with no content and no reason is worse than the unflagged one it replaced · acceptance-sha256:b5bc10e98b19e16c662552383fec67106a9f7755d817b90c315dc7e22b026c0e · covers:the per-path reasons surviving in content[0] of the flagged result

## Invariants

- §62 keeps passing unchanged — the page, the index and the refusal shapes are ADR-024's and are not this record's to move.
- The row pairs the flagged case with the unflagged ones. A row asserting only the presence of a flag would pass against a server that flagged everything.
- The row reads its JSON from files, never argv.

## Risks

- The row could assert only the flag and not the surviving report text, which would let a change that flags the result and drops its reasons pass. Mitigated: S2 asserts `-- nope_dir:` is still in `content[0]` of the flagged result.

## Stop Condition

Stop and ask if §62 goes red — that means T1 reached a shape that served something, and the record's
scope is wrong rather than the row.

## Out of Scope

- The condition itself and its unit test — that is T1's job

## Verification Log
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5bc10e98b19e16c662552383fec67106a9f7755d817b90c315dc7e22b026c0e · ms:25013
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5bc10e98b19e16c662552383fec67106a9f7755d817b90c315dc7e22b026c0e · ms:22848
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5bc10e98b19e16c662552383fec67106a9f7755d817b90c315dc7e22b026c0e · ms:20594
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5bc10e98b19e16c662552383fec67106a9f7755d817b90c315dc7e22b026c0e · ms:26266
