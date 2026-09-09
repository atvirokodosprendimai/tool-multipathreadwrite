# Task ADR-039-T2: The CLI still licenses a fitting read without ack

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** CLI fitting reads still license without ack
**Consumes:** fitting serves held pending per file (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the CLI still recording on serve`, `an MCP fitting write without ack refused`

## Goal

Pin the member this record deliberately leaves out: a CLI `mrw read` of a fitting file still
licenses a following write with no `ack`, while the same read over MCP does not.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/ack_test.go` | edit | `TestACLIReadStillLicensesWithoutAck` drives `cmd/mrw` (or the same `read.Run`+`seen.Record` the CLI uses — see step) then a write with no ack, and asserts apply. The MCP half is T1; this test is the CLI half of the same pair. |
| `cmd/mrw/main.go` | none | Selection: if this test stays green after deleting CLI `seen.Record`, the test is on the wrong caller. |

## Ordered Steps

1. [S1] Write `TestACLIReadStillLicensesWithoutAck` and confirm it is RED if T1's change was
   wrongly applied to the CLI, and GREEN against T1 as specified: `mrw read` a three-line file,
   then `mrw write` a replace of line 2 with no ack, exit 0. Drive it the way
   `internal/adversarial` already drives the CLI — do not reimplement Record. [proof: mutation]
2. [S2] Confirm T1's MCP tests still refuse the unacked fitting write. The pair is the point:
   same file, two transports, opposite licence. [proof: acceptance]
3. [S3] Run `gofmt` and `go vet` unpiped on the packages touched. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -v \
  -run 'TestACLIReadStillLicensesWithoutAck|TestAFittingReadLicensesOnlyWhatCameBack' 2>&1 \
  | tee /tmp/adr039-t2.out \
  && grep -q '^--- PASS: TestACLIReadStillLicensesWithoutAck' /tmp/adr039-t2.out \
  && grep -q '^--- PASS: TestAFittingReadLicensesOnlyWhatCameBack' /tmp/adr039-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr039-t2.out \
  && [ -z "$(gofmt -l internal/mcp cmd/mrw)" ] \
  && go vet ./internal/mcp/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACLIReadStillLicensesWithoutAck` | `internal/mcp/ack_test.go` | CLI fitting read then write, no ack, applies | — | S1 |
| `TestAFittingReadLicensesOnlyWhatCameBack` | `internal/mcp/ack_test.go` | MCP half of the pair still refuses without ack (T1) | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestACLIReadStillLicensesWithoutAck` |
| 2 — something selects it | CLI `seen.Record` at `cmd/mrw/main.go`. Deleting that Record must fail S1 |
| 3 — the caller can discover it | n/a: no declared interface — this is the absence of a requirement |
| 4 — it is used | nothing measures this yet — ADR-009; the contract script's non-MCP rows keep exercising CLI reads |

## Mutation Log

- 2026-09-09 · ebbc4be* · mutant killed · exit 1 · `cmd/mrw/main.go` · deleting CLI Record must fail TestACLIReadStillLicensesWithoutAck · acceptance-sha256:4938a66b685d66d7a138c5f425f1f12bdd7373f377b6aa95471349a92b1a30f0

## Invariants

- MCP fitting reads still license nothing until ack (T1).
- No `ack` flag is added to the CLI.
- Engine packages listed in the parent go/no-go stay byte-identical.

## Risks

- Driving `read.Run` instead of the CLI `Record` site tests the engine and misses a CLI that
  stopped recording. Mitigation: S1 names `cmd/mrw` as the caller that must go red when Record is
  deleted there.

## Stop Condition

Stop if the CLI test can only be made red by adding `ack` to `mrw read`. That is a different
record, and it is Out of Scope. Stop if pinning the CLI requires changing `internal/seen`.

## Out of Scope

- Contract §77 and handshake copy (T3)
- Requiring CLI ack (parent Out of Scope)

## Verification Log
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:4938a66b685d66d7a138c5f425f1f12bdd7373f377b6aa95471349a92b1a30f0 · ms:1094
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:4938a66b685d66d7a138c5f425f1f12bdd7373f377b6aa95471349a92b1a30f0 · ms:1365
