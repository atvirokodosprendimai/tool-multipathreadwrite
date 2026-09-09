# Task ADR-038-T3: README and the handshake stop teaching the race

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** teaching names the lock
**Consumes:** `Record` holds the lock (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the handshake omitting the race claim`, `the README omitting 40 racing reads kept 5`

## Goal

Stop telling callers that parallel CLI processes overwrite ledger entries, now that they do not.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | The "one call at a time / 40 racing reads kept 5" paragraph. MCP still serializes in-process; that is no longer a contrast with a racing CLI. |
| `internal/mcp/instructions.go` | edit | Drop *"while parallel CLI processes race"*. Keep *"one server is one writer"* / serialized if it still earns its keep as "one process, no flock wait" — or shorten. Bound stays 4096. |
| `internal/mcp/mcp_test.go` | edit | `TestTheHandshakeDoesNotTeachALedgerRace` — `instructionsText` must not contain `parallel CLI processes race`. `TestTheSurfaceSaysTheCLIIsRicher` must stay green (it already keys on `serialized` and `one fixed checkout`, not on `race`). |
| `cmd/mrw/agentsdoc_test.go` | edit | `TestReadmeDoesNotTeachALedgerRace` lives beside `TestReadmeAndSkillNameInstructions`. README must not contain `40 racing reads kept 5`. |

## Ordered Steps

1. [S1] Write `TestTheHandshakeDoesNotTeachALedgerRace` and confirm it is RED on today's
   `instructionsText`. [proof: mutation]
2. [S2] Write `TestReadmeDoesNotTeachALedgerRace` and confirm it is RED on today's README.
   [proof: mutation]
3. [S3] Edit the two documents. Confirm `len(instructionsText()) <= 4096` still holds inside
   `TestTheInstructionsTellAHostHowToAuthorAPlan` / `TestTheSurfaceSaysTheCLIIsRicher`.
   [proof: mutation]
4. [S4] Confirm `TestTheSurfaceSaysTheCLIIsRicher` is still green — do not drop `serialized` or
   `one fixed checkout` to fund the edit. [proof: acceptance]
5. [S5] `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ ./cmd/mrw/ -count=1 -v \
  -run 'TestTheHandshakeDoesNotTeachALedgerRace|TestReadmeDoesNotTeachALedgerRace|TestTheSurfaceSaysTheCLIIsRicher' 2>&1 | tee /tmp/adr038-t3.out \
  && grep -q '^--- PASS: TestTheHandshakeDoesNotTeachALedgerRace' /tmp/adr038-t3.out \
  && grep -q '^--- PASS: TestReadmeDoesNotTeachALedgerRace' /tmp/adr038-t3.out \
  && grep -q '^--- PASS: TestTheSurfaceSaysTheCLIIsRicher' /tmp/adr038-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr038-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/mcp/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheHandshakeDoesNotTeachALedgerRace` | `internal/mcp/mcp_test.go` | `instructionsText` does not contain `parallel CLI processes race` | — | S1, S3 |
| `TestReadmeDoesNotTeachALedgerRace` | `cmd/mrw/agentsdoc_test.go` | README does not contain `40 racing reads kept 5` | — | S2, S3 |
| `TestTheSurfaceSaysTheCLIIsRicher` | `internal/mcp/mcp_test.go` | Existing surface-choice claims and the 4096 bound still hold | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two new tests |
| 2 — something selects it | MCP `initialize` calls `instructionsText`; README is the install doc |
| 3 — the caller can discover it | those two surfaces |
| 4 — it is used | nothing measures this — ADR-009 refused telemetry |

## Mutation Log

- 2026-09-09 · 0737f60* · mutant killed · exit 1 · `internal/mcp/instructions.go` · restoring the handshake race clause must trip TestTheHandshakeDoesNotTeachALedgerRace · acceptance-sha256:85b58dd6cbc25b406a3e853aa47b107b6508e54f4889bed6b2412b6da109ce8e
- 2026-09-09 · 0737f60* · mutant killed · exit 1 · `README.md` · restoring the measured race sentence must trip TestReadmeDoesNotTeachALedgerRace · acceptance-sha256:85b58dd6cbc25b406a3e853aa47b107b6508e54f4889bed6b2412b6da109ce8e

## Invariants

- `maxInstructionsChars` is still the literal 4096.
- `guide.Shared()` is unchanged.
- Do not restore "40 racing reads kept 5" as history in the README's live instructions. An ADR
  may keep the measurement; the user-facing paragraph may not.

## Risks

- Funding the edit by dropping `serialized` or `one fixed checkout` turns
  `TestTheSurfaceSaysTheCLIIsRicher` red. S4 is that gate.

## Stop Condition

Stop and ask if the replacement clause cannot fit in 4096 without dropping a guarded claim. That
is the same wall ADR-027 and ADR-035 hit; do not raise the bound.

## Out of Scope

- The palace-centralised `mrw` skill (permanent: boundary: `am_update_skill` overwrites the whole
  body; this record does not take that risk)

## Verification Log
- 2026-09-09 · 0737f60* · exit 0 · `set -o pipefail …` · acceptance-sha256:85b58dd6cbc25b406a3e853aa47b107b6508e54f4889bed6b2412b6da109ce8e · ms:1141
- 2026-09-09 · 0737f60* · exit 0 · `set -o pipefail …` · acceptance-sha256:85b58dd6cbc25b406a3e853aa47b107b6508e54f4889bed6b2412b6da109ce8e · ms:812
